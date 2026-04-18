package nexusplayers

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oklog/ulid/v2"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/auth"
	nexusggrintegration "github.com/mugiew/justqiuv2-rewrite/apps/api/internal/integrations/nexusggr"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/modules/players"
)

var (
	ErrDuplicateUsername             = errors.New("duplicate username")
	ErrInsufficientNexusBalance      = errors.New("insufficient nexusggr balance")
	ErrCreateUserUpstreamFailure     = errors.New("create user upstream failure")
	ErrDepositUpstreamFailure        = errors.New("deposit upstream failure")
	ErrWithdrawResetUpstreamFailure  = errors.New("withdraw reset upstream failure")
	ErrTransferStatusUpstreamFailure = errors.New("transfer status upstream failure")
	ErrWithdrawMoneyInfoUpstream     = errors.New("withdraw money info upstream failure")
	ErrWithdrawUpstreamFailure       = errors.New("withdraw upstream failure")
	ErrUpstreamUserInsufficientFunds = errors.New("upstream user insufficient balance")
)

type nexusClient interface {
	UserCreate(ctx context.Context, userCode string) (*nexusggrintegration.UserCreateResponse, error)
	UserDeposit(ctx context.Context, userCode string, amount int64, agentSign *string) (*nexusggrintegration.UserBalanceMutationResponse, error)
	UserWithdraw(ctx context.Context, userCode string, amount int64, agentSign *string) (*nexusggrintegration.UserBalanceMutationResponse, error)
	UserWithdrawReset(ctx context.Context, userCode *string, allUsers bool) (*nexusggrintegration.UserWithdrawResetResponse, error)
	TransferStatus(ctx context.Context, userCode string, agentSign string) (*nexusggrintegration.TransferStatusResponse, error)
	MoneyInfo(ctx context.Context, userCode *string, allUsers bool) (*nexusggrintegration.MoneyInfoResponse, error)
}

type Service struct {
	db               *pgxpool.Pool
	playerRepository *players.Repository
	nexusClient      nexusClient
}

type CreateUserResult struct {
	Username string
}

type MutationResult struct {
	Username     string
	AgentBalance int64
	UserBalance  any
}

type WithdrawResetUser struct {
	Username       string
	WithdrawAmount any
	Balance        any
}

type WithdrawResetResult struct {
	AgentBalance any
	User         *WithdrawResetUser
	UserList     []WithdrawResetUser
}

type TransferStatusResult struct {
	Amount       any
	Type         any
	AgentBalance any
	Username     string
	UserBalance  any
}

func NewService(db *pgxpool.Pool, playerRepository *players.Repository, nexusClient nexusClient) *Service {
	return &Service{
		db:               db,
		playerRepository: playerRepository,
		nexusClient:      nexusClient,
	}
}

func (s *Service) CreateUser(ctx context.Context, toko auth.Toko, username string) (*CreateUserResult, error) {
	normalizedUsername := strings.ToLower(strings.TrimSpace(username))

	if _, err := s.playerRepository.FindByUsername(ctx, toko.ID, normalizedUsername); err == nil {
		return nil, ErrDuplicateUsername
	} else if !errors.Is(err, players.ErrNotFound) {
		return nil, err
	}

	extUsername := strings.ToLower(ulid.MustNew(ulid.Timestamp(time.Now().UTC()), rand.Reader).String())
	upstreamResponse, err := s.nexusClient.UserCreate(ctx, extUsername)
	if err != nil || upstreamResponse.Status != 1 {
		return nil, ErrCreateUserUpstreamFailure
	}

	if _, err := s.playerRepository.Create(ctx, toko.ID, normalizedUsername, extUsername); err != nil {
		return nil, err
	}

	return &CreateUserResult{Username: normalizedUsername}, nil
}

func (s *Service) Deposit(ctx context.Context, toko auth.Toko, username string, amount int64, agentSign *string) (*MutationResult, error) {
	player, err := s.playerRepository.FindByUsername(ctx, toko.ID, username)
	if err != nil {
		return nil, err
	}

	currentBalance, err := s.currentNexusBalance(ctx, toko.ID)
	if err != nil {
		return nil, err
	}
	if currentBalance <= amount {
		return nil, ErrInsufficientNexusBalance
	}

	upstreamResponse, err := s.nexusClient.UserDeposit(ctx, player.ExtUsername, amount, agentSign)
	if err != nil || upstreamResponse.Status != 1 {
		return nil, ErrDepositUpstreamFailure
	}

	noteJSON, err := json.Marshal(map[string]any{
		"method":       "user_deposit",
		"user_balance": upstreamResponse.UserBalance,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal deposit note: %w", err)
	}

	updatedBalance, err := s.applyMutation(ctx, mutationInput{
		TokoID:         toko.ID,
		PlayerUsername: player.Username,
		ExtUsername:    player.ExtUsername,
		Type:           "deposit",
		Amount:         amount,
		Code:           trimOptionalString(agentSign),
		Note:           string(noteJSON),
		Delta:          -amount,
	})
	if err != nil {
		return nil, err
	}

	return &MutationResult{
		Username:     player.Username,
		AgentBalance: updatedBalance,
		UserBalance:  upstreamResponse.UserBalance,
	}, nil
}

func (s *Service) Withdraw(ctx context.Context, toko auth.Toko, username string, amount int64, agentSign *string) (*MutationResult, error) {
	player, err := s.playerRepository.FindByUsername(ctx, toko.ID, username)
	if err != nil {
		return nil, err
	}

	upstreamBalanceResponse, err := s.nexusClient.MoneyInfo(ctx, &player.ExtUsername, false)
	if err != nil || upstreamBalanceResponse.Status != 1 {
		return nil, ErrWithdrawMoneyInfoUpstream
	}

	if extractUserBalance(upstreamBalanceResponse.User) < amount {
		return nil, ErrUpstreamUserInsufficientFunds
	}

	upstreamResponse, err := s.nexusClient.UserWithdraw(ctx, player.ExtUsername, amount, agentSign)
	if err != nil || upstreamResponse.Status != 1 {
		return nil, ErrWithdrawUpstreamFailure
	}

	noteJSON, err := json.Marshal(map[string]any{
		"method":       "user_withdraw",
		"user_balance": upstreamResponse.UserBalance,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal withdraw note: %w", err)
	}

	updatedBalance, err := s.applyMutation(ctx, mutationInput{
		TokoID:         toko.ID,
		PlayerUsername: player.Username,
		ExtUsername:    player.ExtUsername,
		Type:           "withdrawal",
		Amount:         amount,
		Code:           trimOptionalString(agentSign),
		Note:           string(noteJSON),
		Delta:          amount,
	})
	if err != nil {
		return nil, err
	}

	return &MutationResult{
		Username:     player.Username,
		AgentBalance: updatedBalance,
		UserBalance:  upstreamResponse.UserBalance,
	}, nil
}

func (s *Service) WithdrawReset(ctx context.Context, toko auth.Toko, username *string, allUsers bool) (*WithdrawResetResult, error) {
	var (
		player   *players.Player
		userCode *string
		err      error
	)

	if username != nil && strings.TrimSpace(*username) != "" {
		player, err = s.playerRepository.FindByUsername(ctx, toko.ID, *username)
		if err != nil {
			return nil, err
		}
		userCode = &player.ExtUsername
	}

	upstreamResponse, err := s.nexusClient.UserWithdrawReset(ctx, userCode, allUsers)
	if err != nil || upstreamResponse.Status != 1 {
		return nil, ErrWithdrawResetUpstreamFailure
	}

	usernameMap, err := s.playerRepository.UsernameMapForToko(ctx, toko.ID)
	if err != nil {
		return nil, err
	}

	if err := s.createWithdrawResetTransactions(ctx, toko.ID, upstreamResponse, usernameMap, allUsers); err != nil {
		return nil, err
	}

	result := &WithdrawResetResult{
		AgentBalance: nil,
	}
	if upstreamResponse.Agent != nil {
		result.AgentBalance = upstreamResponse.Agent["balance"]
	}

	if upstreamResponse.User != nil {
		if mappedUser := mapWithdrawResetRecord(upstreamResponse.User, usernameMap); mappedUser != nil {
			result.User = mappedUser
		}
	}

	if len(upstreamResponse.UserList) > 0 {
		result.UserList = mapWithdrawResetRecords(upstreamResponse.UserList, usernameMap)
	}

	return result, nil
}

func (s *Service) TransferStatus(ctx context.Context, toko auth.Toko, username string, agentSign string) (*TransferStatusResult, error) {
	player, err := s.playerRepository.FindByUsername(ctx, toko.ID, username)
	if err != nil {
		return nil, err
	}

	upstreamResponse, err := s.nexusClient.TransferStatus(ctx, player.ExtUsername, agentSign)
	if err != nil || upstreamResponse.Status != 1 {
		return nil, ErrTransferStatusUpstreamFailure
	}

	return &TransferStatusResult{
		Amount:       upstreamResponse.Amount,
		Type:         upstreamResponse.Type,
		AgentBalance: upstreamResponse.AgentBalance,
		Username:     player.Username,
		UserBalance:  upstreamResponse.UserBalance,
	}, nil
}

type mutationInput struct {
	TokoID         int64
	PlayerUsername string
	ExtUsername    string
	Type           string
	Amount         int64
	Code           *string
	Note           string
	Delta          int64
}

func (s *Service) applyMutation(ctx context.Context, input mutationInput) (int64, error) {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, fmt.Errorf("begin mutation transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	balanceID, nexusggrBalance, err := s.lockBalance(ctx, tx, input.TokoID)
	if err != nil {
		return 0, err
	}

	updatedBalance := nexusggrBalance + input.Delta

	if _, err := tx.Exec(ctx, `
		UPDATE balances
		SET nexusggr = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`, balanceID, updatedBalance); err != nil {
		return 0, fmt.Errorf("update nexusggr balance: %w", err)
	}

	if _, err := tx.Exec(ctx, `
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
		) VALUES ($1, $2, $3, 'nexusggr', $4, 'success', $5, $6, $7, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, input.TokoID, input.PlayerUsername, input.ExtUsername, input.Type, input.Amount, input.Code, input.Note); err != nil {
		return 0, fmt.Errorf("insert nexusggr transaction: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit mutation transaction: %w", err)
	}

	return updatedBalance, nil
}

func (s *Service) currentNexusBalance(ctx context.Context, tokoID int64) (int64, error) {
	const query = `
		SELECT nexusggr::bigint
		FROM balances
		WHERE toko_id = $1
		ORDER BY id
		LIMIT 1
	`

	var balance int64
	err := s.db.QueryRow(ctx, query, tokoID).Scan(&balance)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("load current nexusggr balance: %w", err)
	}

	return balance, nil
}

func (s *Service) lockBalance(ctx context.Context, tx pgx.Tx, tokoID int64) (int64, int64, error) {
	const selectQuery = `
		SELECT id, nexusggr::bigint
		FROM balances
		WHERE toko_id = $1
		ORDER BY id
		LIMIT 1
		FOR UPDATE
	`

	var (
		balanceID int64
		balance   int64
	)

	err := tx.QueryRow(ctx, selectQuery, tokoID).Scan(&balanceID, &balance)
	if err == nil {
		return balanceID, balance, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, 0, fmt.Errorf("lock balance: %w", err)
	}

	err = tx.QueryRow(ctx, `
		INSERT INTO balances (toko_id, settle, pending, nexusggr, created_at, updated_at)
		VALUES ($1, 0, 0, 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id, nexusggr::bigint
	`, tokoID).Scan(&balanceID, &balance)
	if err != nil {
		return 0, 0, fmt.Errorf("create missing balance row: %w", err)
	}

	return balanceID, balance, nil
}

func extractUserBalance(record map[string]any) int64 {
	if record == nil {
		return 0
	}

	return toInt64(record["balance"])
}

func toInt64(value any) int64 {
	switch typedValue := value.(type) {
	case int:
		return int64(typedValue)
	case int32:
		return int64(typedValue)
	case int64:
		return typedValue
	case float32:
		return int64(typedValue)
	case float64:
		return int64(typedValue)
	case json.Number:
		parsed, err := typedValue.Int64()
		if err == nil {
			return parsed
		}
	case string:
		parsed := json.Number(typedValue)
		if value, err := parsed.Int64(); err == nil {
			return value
		}
	}

	return 0
}

func trimOptionalString(value *string) *string {
	if value == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}

	return &trimmed
}

func mapWithdrawResetRecord(record map[string]any, usernameMap map[string]string) *WithdrawResetUser {
	externalUsername, ok := record["user_code"].(string)
	if !ok {
		return nil
	}

	username, exists := usernameMap[externalUsername]
	if !exists {
		return nil
	}

	return &WithdrawResetUser{
		Username:       username,
		WithdrawAmount: record["withdraw_amount"],
		Balance:        record["balance"],
	}
}

func mapWithdrawResetRecords(records []map[string]any, usernameMap map[string]string) []WithdrawResetUser {
	mapped := make([]WithdrawResetUser, 0, len(records))
	for _, record := range records {
		mappedRecord := mapWithdrawResetRecord(record, usernameMap)
		if mappedRecord == nil {
			continue
		}
		mapped = append(mapped, *mappedRecord)
	}

	return mapped
}

func (s *Service) createWithdrawResetTransactions(
	ctx context.Context,
	tokoID int64,
	response *nexusggrintegration.UserWithdrawResetResponse,
	usernameMap map[string]string,
	allUsers bool,
) error {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin withdraw reset transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	processed := make(map[string]struct{})
	records := []map[string]any{}
	if response.User != nil {
		records = append(records, response.User)
	}
	records = append(records, response.UserList...)

	for _, record := range records {
		externalUsername, ok := record["user_code"].(string)
		if !ok {
			continue
		}
		if _, exists := processed[externalUsername]; exists {
			continue
		}
		processed[externalUsername] = struct{}{}

		username, exists := usernameMap[externalUsername]
		if !exists {
			continue
		}

		noteJSON, err := json.Marshal(map[string]any{
			"method":       "user_withdraw_reset",
			"scope":        withdrawResetScope(allUsers),
			"user_balance": record["balance"],
		})
		if err != nil {
			return fmt.Errorf("marshal withdraw reset note: %w", err)
		}

		if _, err := tx.Exec(ctx, `
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
			) VALUES ($1, $2, $3, 'nexusggr', 'withdrawal', 'success', $4, NULL, $5, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`, tokoID, username, externalUsername, toInt64(record["withdraw_amount"]), string(noteJSON)); err != nil {
			return fmt.Errorf("insert withdraw reset transaction: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit withdraw reset transaction: %w", err)
	}

	return nil
}

func withdrawResetScope(allUsers bool) string {
	if allUsers {
		return "all_users"
	}

	return "single_user"
}
