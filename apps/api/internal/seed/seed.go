package seed

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const (
	defaultDevUsername          = "justqiu"
	defaultDevName              = "JustQiu Dev"
	defaultDevEmail             = "justqiu@local.test"
	defaultDevPassword          = "justqiu"
	defaultIncomeGGR            = 7
	defaultIncomeFeeTransaction = 3
	defaultIncomeFeeWithdrawal  = 15

	demoOwnerUsername  = "demo-owner"
	demoOwnerName      = "Demo Owner"
	demoOwnerEmail     = "demo-owner@local.test"
	demoOwnerPassword  = "justqiu"
	demoMFAUsername    = "justqiumfa"
	demoMFAName        = "JustQiu MFA"
	demoMFAEmail       = "justqiumfa@local.test"
	demoMFAPassword    = "justqiu"
	demoTokoName       = "Demo Toko"
	demoCallbackURL    = "https://demo.example.test/callback"
	demoPlainAPIToken  = "demo-seed-toko-api-token"
	demoBankCode       = "014"
	demoBankName       = "BCA"
	demoAccountNumber  = "9876543210"
	demoAccountName    = "DEMO OWNER"
	demoPlayerUsername = "demo-player"
	demoPlayerExternal = "demo-player-ext"
)

type DevelopmentSeedResult struct {
	UserID            int64
	Username          string
	Password          string
	Email             string
	IncomeGGR         int
	IncomeFeeTx       int
	IncomeFeeWithdraw int
}

type DemoSeedResult struct {
	DevUserID         int64
	DevUsername       string
	DevPassword       string
	DevEmail          string
	OwnerUserID       int64
	OwnerUsername     string
	OwnerPassword     string
	OwnerEmail        string
	MFAUserID         int64
	MFAUsername       string
	MFAPassword       string
	MFAEmail          string
	TokoID            int64
	TokoName          string
	TokoToken         string
	BankID            int64
	PlayerID          int64
	PlayerUsername    string
	PlayerExternal    string
	IncomeAmount      int64
	IncomeGGR         int
	IncomeFeeTx       int
	IncomeFeeWithdraw int
}

func Development(ctx context.Context, db *sql.DB) (*DevelopmentSeedResult, error) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(defaultDevPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash development seed password: %w", err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin seed transaction: %w", err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	result, err := seedDevelopmentTx(ctx, tx, string(passwordHash))
	if err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit development seed: %w", err)
	}

	return result, nil
}

func Demo(ctx context.Context, db *sql.DB) (*DemoSeedResult, error) {
	devPasswordHash, err := bcrypt.GenerateFromPassword([]byte(defaultDevPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash development seed password: %w", err)
	}

	ownerPasswordHash, err := bcrypt.GenerateFromPassword([]byte(demoOwnerPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash owner seed password: %w", err)
	}

	mfaPasswordHash, err := bcrypt.GenerateFromPassword([]byte(demoMFAPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash mfa seed password: %w", err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin demo seed transaction: %w", err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	baseResult, err := seedDevelopmentTx(ctx, tx, string(devPasswordHash))
	if err != nil {
		return nil, err
	}

	if _, err = tx.ExecContext(ctx, `
		DELETE FROM users
		WHERE username = $1 OR LOWER(email) = LOWER($2)
	`, demoOwnerUsername, demoOwnerEmail); err != nil {
		return nil, fmt.Errorf("delete existing demo owner: %w", err)
	}

	if _, err = tx.ExecContext(ctx, `
		DELETE FROM users
		WHERE username = $1 OR LOWER(email) = LOWER($2)
	`, demoMFAUsername, demoMFAEmail); err != nil {
		return nil, fmt.Errorf("delete existing demo mfa user: %w", err)
	}

	var ownerUserID int64
	if err = tx.QueryRowContext(ctx, `
		INSERT INTO users (
			username,
			name,
			email,
			email_verified_at,
			password,
			role,
			is_active,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP, $4, 'admin', TRUE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id
	`, demoOwnerUsername, demoOwnerName, demoOwnerEmail, string(ownerPasswordHash)).Scan(&ownerUserID); err != nil {
		return nil, fmt.Errorf("insert demo owner: %w", err)
	}

	var mfaUserID int64
	if err = tx.QueryRowContext(ctx, `
		INSERT INTO users (
			username,
			name,
			email,
			email_verified_at,
			password,
			role,
			is_active,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP, $4, 'dev', TRUE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id
	`, demoMFAUsername, demoMFAName, demoMFAEmail, string(mfaPasswordHash)).Scan(&mfaUserID); err != nil {
		return nil, fmt.Errorf("insert demo mfa user: %w", err)
	}

	var tokoID int64
	if err = tx.QueryRowContext(ctx, `
		INSERT INTO tokos (
			user_id,
			name,
			callback_url,
			is_active,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, TRUE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id
	`, ownerUserID, demoTokoName, demoCallbackURL).Scan(&tokoID); err != nil {
		return nil, fmt.Errorf("insert demo toko: %w", err)
	}

	if _, err = tx.ExecContext(ctx, `
		INSERT INTO balances (
			toko_id,
			settle,
			pending,
			nexusggr,
			created_at,
			updated_at
		)
		VALUES ($1, 420000, 85000, 900000, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, tokoID); err != nil {
		return nil, fmt.Errorf("insert demo balance: %w", err)
	}

	tokoToken, err := issueSeedTokoTokenTx(ctx, tx, tokoID)
	if err != nil {
		return nil, err
	}

	var bankID int64
	if err = tx.QueryRowContext(ctx, `
		INSERT INTO banks (
			user_id,
			bank_code,
			bank_name,
			account_number,
			account_name,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id
	`, ownerUserID, demoBankCode, demoBankName, demoAccountNumber, demoAccountName).Scan(&bankID); err != nil {
		return nil, fmt.Errorf("insert demo bank: %w", err)
	}

	var playerID int64
	if err = tx.QueryRowContext(ctx, `
		INSERT INTO players (
			toko_id,
			username,
			ext_username,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id
	`, tokoID, demoPlayerUsername, demoPlayerExternal).Scan(&playerID); err != nil {
		return nil, fmt.Errorf("insert demo player: %w", err)
	}

	if _, err = tx.ExecContext(ctx, `
		UPDATE incomes
		SET amount = 65000, updated_at = CURRENT_TIMESTAMP
		WHERE id = 1
	`); err != nil {
		return nil, fmt.Errorf("update demo income amount: %w", err)
	}

	if _, err = tx.ExecContext(ctx, `
		INSERT INTO transactions (
			toko_id,
			player,
			external_player,
			category,
			type,
			status,
			amount,
			code,
			note,
			created_at,
			updated_at
		) VALUES
			($1, $2, $3, 'qris', 'deposit', 'success', 150000, 'QRIS-DEPOSIT-001', '{"purpose":"generate"}', CURRENT_TIMESTAMP - INTERVAL '1 day', CURRENT_TIMESTAMP - INTERVAL '1 day'),
			($1, $2, $3, 'nexusggr', 'deposit', 'success', 200000, 'NG-DEPOSIT-001', '{"method":"user_deposit"}', CURRENT_TIMESTAMP - INTERVAL '2 day', CURRENT_TIMESTAMP - INTERVAL '2 day'),
			($1, $2, $3, 'nexusggr', 'withdrawal', 'success', 75000, 'NG-WITHDRAW-001', '{"method":"user_withdraw"}', CURRENT_TIMESTAMP - INTERVAL '3 day', CURRENT_TIMESTAMP - INTERVAL '3 day'),
			($1, NULL, NULL, 'qris', 'withdrawal', 'pending', 50000, 'QRIS-WD-001', '{"purpose":"withdrawal","bank_name":"BCA"}', CURRENT_TIMESTAMP - INTERVAL '4 hour', CURRENT_TIMESTAMP - INTERVAL '4 hour'),
			($1, $2, $3, 'qris', 'deposit', 'failed', 25000, 'QRIS-DEPOSIT-FAILED', '{"purpose":"generate"}', CURRENT_TIMESTAMP - INTERVAL '5 day', CURRENT_TIMESTAMP - INTERVAL '5 day'),
			($1, $2, $3, 'qris', 'deposit', 'expired', 30000, 'QRIS-DEPOSIT-EXPIRED', '{"purpose":"generate"}', CURRENT_TIMESTAMP - INTERVAL '6 day', CURRENT_TIMESTAMP - INTERVAL '6 day')
	`, tokoID, demoPlayerUsername, demoPlayerExternal); err != nil {
		return nil, fmt.Errorf("insert demo transactions: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit demo seed: %w", err)
	}

	return &DemoSeedResult{
		DevUserID:         baseResult.UserID,
		DevUsername:       baseResult.Username,
		DevPassword:       baseResult.Password,
		DevEmail:          baseResult.Email,
		OwnerUserID:       ownerUserID,
		OwnerUsername:     demoOwnerUsername,
		OwnerPassword:     demoOwnerPassword,
		OwnerEmail:        demoOwnerEmail,
		MFAUserID:         mfaUserID,
		MFAUsername:       demoMFAUsername,
		MFAPassword:       demoMFAPassword,
		MFAEmail:          demoMFAEmail,
		TokoID:            tokoID,
		TokoName:          demoTokoName,
		TokoToken:         tokoToken,
		BankID:            bankID,
		PlayerID:          playerID,
		PlayerUsername:    demoPlayerUsername,
		PlayerExternal:    demoPlayerExternal,
		IncomeAmount:      65000,
		IncomeGGR:         baseResult.IncomeGGR,
		IncomeFeeTx:       baseResult.IncomeFeeTx,
		IncomeFeeWithdraw: baseResult.IncomeFeeWithdraw,
	}, nil
}

func seedDevelopmentTx(ctx context.Context, tx *sql.Tx, passwordHash string) (*DevelopmentSeedResult, error) {
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM users
		WHERE username = $1 OR LOWER(email) = LOWER($2)
	`, defaultDevUsername, defaultDevEmail); err != nil {
		return nil, fmt.Errorf("delete existing development user: %w", err)
	}

	var userID int64
	if err := tx.QueryRowContext(ctx, `
		INSERT INTO users (
			username,
			name,
			email,
			email_verified_at,
			password,
			role,
			is_active,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP, $4, 'dev', TRUE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id
	`, defaultDevUsername, defaultDevName, defaultDevEmail, passwordHash).Scan(&userID); err != nil {
		return nil, fmt.Errorf("insert development user: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO incomes (ggr, fee_transaction, fee_withdrawal, amount, created_at, updated_at)
		VALUES ($1, $2, $3, 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, defaultIncomeGGR, defaultIncomeFeeTransaction, defaultIncomeFeeWithdrawal); err != nil {
		return nil, fmt.Errorf("seed income row: %w", err)
	}

	return &DevelopmentSeedResult{
		UserID:            userID,
		Username:          defaultDevUsername,
		Password:          defaultDevPassword,
		Email:             defaultDevEmail,
		IncomeGGR:         defaultIncomeGGR,
		IncomeFeeTx:       defaultIncomeFeeTransaction,
		IncomeFeeWithdraw: defaultIncomeFeeWithdrawal,
	}, nil
}

func issueSeedTokoTokenTx(ctx context.Context, tx *sql.Tx, tokoID int64) (string, error) {
	var tokenID int64
	if err := tx.QueryRowContext(ctx, `
		INSERT INTO personal_access_tokens (
			tokenable_type,
			tokenable_id,
			name,
			token,
			abilities,
			created_at,
			updated_at
		)
		VALUES ('App\\Models\\Toko', $1, 'api', $2, NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id
	`, tokoID, sha256Hex(demoPlainAPIToken)).Scan(&tokenID); err != nil {
		return "", fmt.Errorf("insert demo toko token: %w", err)
	}

	composedToken := fmt.Sprintf("%d|%s", tokenID, demoPlainAPIToken)
	if _, err := tx.ExecContext(ctx, `
		UPDATE tokos
		SET token = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`, tokoID, composedToken); err != nil {
		return "", fmt.Errorf("persist demo toko token: %w", err)
	}

	return composedToken, nil
}

func sha256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
