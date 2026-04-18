package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/security"
)

var usernameOrEmailPattern = regexp.MustCompile(`^\S+@\S+\.\S+$|^[a-zA-Z0-9_]+$`)

type Service struct {
	repository *Repository
	cipher     *security.StringCipher
}

func NewService(repository *Repository, cipher *security.StringCipher) *Service {
	return &Service{
		repository: repository,
		cipher:     cipher,
	}
}

func (s *Service) AuthenticateCredentials(ctx context.Context, login string, password string) (*User, error) {
	user, err := s.repository.FindUserByLogin(ctx, strings.TrimSpace(login))
	if err != nil {
		return nil, err
	}

	if !user.IsActive {
		return nil, ErrInactiveUser
	}

	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return nil, ErrInvalidCredentials
	}

	return user, nil
}

func (s *Service) AuthenticateUser(ctx context.Context, login string, password string) (*PublicUser, error) {
	user, err := s.AuthenticateCredentials(ctx, login, password)
	if err != nil {
		return nil, err
	}

	publicUser := user.Public()
	return &publicUser, nil
}

func (s *Service) RegisterUser(ctx context.Context, input RegisterInput) (*PublicUser, error) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user, err := s.repository.CreateUser(ctx, input, string(passwordHash))
	if err != nil {
		return nil, err
	}

	publicUser := user.Public()
	return &publicUser, nil
}

func (s *Service) FindUserByID(ctx context.Context, id int64) (*PublicUser, error) {
	user, err := s.repository.FindUserByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if !user.IsActive {
		return nil, ErrInactiveUser
	}

	publicUser := user.Public()
	return &publicUser, nil
}

func (s *Service) AuthenticateTokoToken(ctx context.Context, bearerToken string) (*Toko, error) {
	tokenID, tokenHash := parseBearerToken(bearerToken)

	var (
		accessToken *AccessToken
		err         error
	)

	if tokenID > 0 {
		accessToken, err = s.repository.FindAccessTokenByIDAndHash(ctx, tokenID, tokenHash)
	} else {
		accessToken, err = s.repository.FindAccessTokenByHash(ctx, tokenHash)
	}
	if err != nil {
		return nil, err
	}

	if accessToken.ExpiresAt != nil && accessToken.ExpiresAt.Before(time.Now()) {
		return nil, ErrUnauthorized
	}

	if normalizeTokenableType(accessToken.TokenableType) != TokenableTypeToko {
		return nil, ErrForbidden
	}

	if err := s.repository.TouchPersonalAccessToken(ctx, accessToken.ID, time.Now()); err != nil {
		return nil, err
	}

	return s.repository.FindTokoByID(ctx, accessToken.TokenableID)
}

func ValidLoginIdentifier(value string) bool {
	return usernameOrEmailPattern.MatchString(strings.TrimSpace(value))
}

func parseBearerToken(bearerToken string) (int64, string) {
	parts := strings.SplitN(strings.TrimSpace(bearerToken), "|", 2)
	if len(parts) != 2 {
		return 0, sha256Hex(bearerToken)
	}

	tokenID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, sha256Hex(parts[1])
	}

	return tokenID, sha256Hex(parts[1])
}

func sha256Hex(value string) string {
	hash := sha256.Sum256([]byte(value))
	return hex.EncodeToString(hash[:])
}

func normalizeTokenableType(value string) string {
	normalized := strings.TrimSpace(value)
	normalized = strings.ReplaceAll(normalized, `\\`, `\`)
	return normalized
}
