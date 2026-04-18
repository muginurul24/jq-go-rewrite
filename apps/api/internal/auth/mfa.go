package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

type MFASetup struct {
	Secret        string
	OTPAuthURL    string
	RecoveryCodes []string
}

func (s *Service) BeginMFASetup(user PublicUser) (*MFASetup, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "JustQiu Control",
		AccountName: user.Email,
		Period:      30,
		Digits:      otp.DigitsSix,
		Algorithm:   otp.AlgorithmSHA1,
	})
	if err != nil {
		return nil, fmt.Errorf("generate totp secret: %w", err)
	}

	recoveryCodes, err := generateRecoveryCodes(8)
	if err != nil {
		return nil, fmt.Errorf("generate recovery codes: %w", err)
	}

	return &MFASetup{
		Secret:        key.Secret(),
		OTPAuthURL:    key.URL(),
		RecoveryCodes: recoveryCodes,
	}, nil
}

func BuildOTPAuthURL(user PublicUser, secret string) string {
	issuer := "JustQiu Control"
	account := user.Email
	label := url.PathEscape(issuer + ":" + account)

	query := url.Values{}
	query.Set("secret", secret)
	query.Set("issuer", issuer)
	query.Set("algorithm", "SHA1")
	query.Set("digits", "6")
	query.Set("period", "30")

	return "otpauth://totp/" + label + "?" + query.Encode()
}

func (s *Service) ConfirmMFASetup(ctx context.Context, userID int64, secret string, code string, recoveryCodes []string) error {
	user, err := s.repository.FindUserByID(ctx, userID)
	if err != nil {
		return err
	}

	if user.MFAEnabled() {
		return ErrMFAAlreadyEnabled
	}

	if !validateTOTP(secret, code) {
		return ErrMFAInvalidCode
	}

	if s.cipher == nil {
		return fmt.Errorf("mfa cipher is not configured")
	}

	encryptedSecret, err := s.cipher.EncryptString(secret)
	if err != nil {
		return fmt.Errorf("encrypt mfa secret: %w", err)
	}

	payload, err := json.Marshal(recoveryCodes)
	if err != nil {
		return fmt.Errorf("marshal recovery codes: %w", err)
	}

	encryptedRecoveryCodes, err := s.cipher.EncryptString(string(payload))
	if err != nil {
		return fmt.Errorf("encrypt recovery codes: %w", err)
	}

	return s.repository.UpdateUserMFA(ctx, userID, encryptedSecret, encryptedRecoveryCodes)
}

func (s *Service) VerifyLoginMFA(ctx context.Context, userID int64, code string) (*PublicUser, error) {
	user, err := s.repository.FindUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if !user.IsActive {
		return nil, ErrInactiveUser
	}

	if !user.MFAEnabled() {
		return nil, ErrMFANotEnabled
	}

	if err := s.verifyAndMaybeConsumeMFA(ctx, user, code); err != nil {
		return nil, err
	}

	publicUser := user.Public()
	return &publicUser, nil
}

func (s *Service) DisableMFA(ctx context.Context, userID int64, code string) error {
	user, err := s.repository.FindUserByID(ctx, userID)
	if err != nil {
		return err
	}

	if !user.MFAEnabled() {
		return ErrMFANotEnabled
	}

	if err := s.verifyAndMaybeConsumeMFA(ctx, user, code); err != nil {
		return err
	}

	return s.repository.ClearUserMFA(ctx, userID)
}

func (s *Service) verifyAndMaybeConsumeMFA(ctx context.Context, user *User, code string) error {
	secret, recoveryCodes, err := s.loadMFAState(user)
	if err != nil {
		return err
	}

	normalizedCode := strings.ToUpper(strings.TrimSpace(code))
	if validateTOTP(secret, normalizedCode) {
		return nil
	}

	for index, recoveryCode := range recoveryCodes {
		if recoveryCode != normalizedCode {
			continue
		}

		remaining := append(recoveryCodes[:index:index], recoveryCodes[index+1:]...)
		if err := s.storeRecoveryCodes(ctx, user.ID, secret, remaining); err != nil {
			return err
		}
		return nil
	}

	return ErrMFAInvalidCode
}

func (s *Service) loadMFAState(user *User) (string, []string, error) {
	if s.cipher == nil {
		return "", nil, fmt.Errorf("mfa cipher is not configured")
	}

	if user.MFASecret == nil || user.MFARecovery == nil {
		return "", nil, ErrMFANotEnabled
	}

	secret, err := s.cipher.DecryptString(*user.MFASecret)
	if err != nil {
		return "", nil, fmt.Errorf("decrypt mfa secret: %w", err)
	}

	payload, err := s.cipher.DecryptString(*user.MFARecovery)
	if err != nil {
		return "", nil, fmt.Errorf("decrypt recovery codes: %w", err)
	}

	var recoveryCodes []string
	if err := json.Unmarshal([]byte(payload), &recoveryCodes); err != nil {
		return "", nil, fmt.Errorf("unmarshal recovery codes: %w", err)
	}

	return secret, recoveryCodes, nil
}

func (s *Service) storeRecoveryCodes(ctx context.Context, userID int64, secret string, recoveryCodes []string) error {
	if s.cipher == nil {
		return fmt.Errorf("mfa cipher is not configured")
	}

	payload, err := json.Marshal(recoveryCodes)
	if err != nil {
		return fmt.Errorf("marshal recovery codes: %w", err)
	}

	encryptedSecret, err := s.cipher.EncryptString(secret)
	if err != nil {
		return fmt.Errorf("encrypt mfa secret: %w", err)
	}

	encryptedRecoveryCodes, err := s.cipher.EncryptString(string(payload))
	if err != nil {
		return fmt.Errorf("encrypt recovery codes: %w", err)
	}

	return s.repository.UpdateUserMFA(ctx, userID, encryptedSecret, encryptedRecoveryCodes)
}

func validateTOTP(secret string, code string) bool {
	secret = strings.TrimSpace(secret)
	code = strings.TrimSpace(code)
	if secret == "" || code == "" {
		return false
	}

	return totp.Validate(code, secret)
}

func generateRecoveryCodes(count int) ([]string, error) {
	if count <= 0 {
		count = 8
	}

	codes := make([]string, 0, count)
	for len(codes) < count {
		partA, err := randomHex(4)
		if err != nil {
			return nil, err
		}
		partB, err := randomHex(4)
		if err != nil {
			return nil, err
		}

		codes = append(codes, strings.ToUpper(partA)+"-"+strings.ToUpper(partB))
	}

	return codes, nil
}

func randomHex(byteLength int) (string, error) {
	payload := make([]byte, byteLength)
	if _, err := rand.Read(payload); err != nil {
		return "", fmt.Errorf("read random bytes: %w", err)
	}

	return hex.EncodeToString(payload), nil
}
