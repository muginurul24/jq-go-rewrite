package withdrawals

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/auth"
	qrisintegration "github.com/mugiew/justqiuv2-rewrite/apps/api/internal/integrations/qris"
)

const (
	minimumWithdrawalAmount        int64         = 25_000
	withdrawalInquiryTTL           time.Duration = 15 * time.Minute
	withdrawalTransferType         int           = 2
	defaultWithdrawalFeePercentage int64         = 15
)

var (
	ErrTokoNotFound           = errors.New("withdrawal toko not found")
	ErrBankNotFound           = errors.New("withdrawal bank not found")
	ErrAmountTooSmall         = errors.New("withdrawal amount too small")
	ErrSettleBalanceNotEnough = errors.New("withdrawal settle balance not enough")
	ErrInquiryFailed          = errors.New("withdrawal inquiry failed")
	ErrInquiryStateNotFound   = errors.New("withdrawal inquiry state not found")
	ErrTransferFailed         = errors.New("withdrawal transfer failed")
)

type qrisClient interface {
	Inquiry(ctx context.Context, amount int64, bankCode string, accountNumber string, transferType int) (*qrisintegration.InquiryResponse, error)
	Transfer(ctx context.Context, amount int64, bankCode string, accountNumber string, transferType int, inquiryID int64) (*qrisintegration.TransferResponse, error)
}

type TokoOption struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	OwnerUsername string `json:"ownerUsername"`
	SettleBalance int64  `json:"settleBalance"`
}

type BankOption struct {
	ID            int64  `json:"id"`
	BankCode      string `json:"bankCode"`
	BankName      string `json:"bankName"`
	AccountNumber string `json:"accountNumber"`
	AccountName   string `json:"accountName"`
}

type BootstrapResult struct {
	Tokos         []TokoOption
	SelectedToko  *TokoOption
	Banks         []BankOption
	FeePercentage int64
	MinimumAmount int64
}

type InquirySummary struct {
	InquiryID                int64  `json:"inquiryId"`
	AccountName              string `json:"accountName"`
	BankName                 string `json:"bankName"`
	AccountNumber            string `json:"accountNumber"`
	Amount                   int64  `json:"amount"`
	BankFee                  int64  `json:"bankFee"`
	PlatformFee              int64  `json:"platformFee"`
	PartnerRefNo             string `json:"partnerRefNo"`
	CurrentSettleBalance     int64  `json:"currentSettleBalance"`
	EstimatedTotalDeduction  int64  `json:"estimatedTotalDeduction"`
	EstimatedRemainingSettle int64  `json:"estimatedRemainingSettle"`
	FinalTotalDeduction      int64  `json:"finalTotalDeduction"`
	FinalRemainingSettle     int64  `json:"finalRemainingSettle"`
}

type InquiryResult struct {
	SelectedToko *TokoOption    `json:"selectedToko"`
	SelectedBank *BankOption    `json:"selectedBank"`
	Inquiry      InquirySummary `json:"inquiry"`
}

type SubmitResult struct {
	SelectedToko *TokoOption          `json:"selectedToko"`
	SelectedBank *BankOption          `json:"selectedBank"`
	Inquiry      InquirySummary       `json:"inquiry"`
	Transaction  SubmittedTransaction `json:"transaction"`
}

type SubmittedTransaction struct {
	ID     int64  `json:"id"`
	Code   string `json:"code"`
	Status string `json:"status"`
	Amount int64  `json:"amount"`
}

type Service struct {
	db            *pgxpool.Pool
	redis         *redis.Client
	qris          qrisClient
	notifications notificationWriter
}

type notificationWriter interface {
	NotifyWithdrawalRequested(ctx context.Context, ownerUserID int64, ownerUsername string, tokoName string, amount int64, platformFee int64, bankFee int64, transactionCode *string) error
}

type accessibleToko struct {
	ID            int64
	UserID        int64
	Name          string
	OwnerUsername string
	SettleBalance int64
}

type cachedInquiry struct {
	ActorID       int64     `json:"actorId"`
	TokoID        int64     `json:"tokoId"`
	BankID        int64     `json:"bankId"`
	Amount        int64     `json:"amount"`
	InquiryID     int64     `json:"inquiryId"`
	AccountName   string    `json:"accountName"`
	BankName      string    `json:"bankName"`
	AccountNumber string    `json:"accountNumber"`
	BankFee       int64     `json:"bankFee"`
	PlatformFee   int64     `json:"platformFee"`
	PartnerRefNo  string    `json:"partnerRefNo"`
	CreatedAt     time.Time `json:"createdAt"`
}

func NewService(db *pgxpool.Pool, redisClient *redis.Client, qrisClient qrisClient) *Service {
	return &Service{
		db:    db,
		redis: redisClient,
		qris:  qrisClient,
	}
}

func (s *Service) WithNotifications(service notificationWriter) *Service {
	s.notifications = service
	return s
}

func (s *Service) Bootstrap(ctx context.Context, actor auth.PublicUser, selectedTokoID *int64) (*BootstrapResult, error) {
	tokos, err := s.accessibleTokos(ctx, actor)
	if err != nil {
		return nil, err
	}

	feePercentage, err := s.withdrawalFeePercentage(ctx)
	if err != nil {
		return nil, err
	}

	var selected *accessibleToko
	if len(tokos) > 0 {
		selected = &tokos[0]
	}

	if selectedTokoID != nil {
		for index := range tokos {
			if tokos[index].ID == *selectedTokoID {
				selected = &tokos[index]
				break
			}
		}
	}

	var banks []BankOption
	if selected != nil {
		banks, err = s.banksForUser(ctx, selected.UserID)
		if err != nil {
			return nil, err
		}
	}

	return &BootstrapResult{
		Tokos:         presentTokos(tokos),
		SelectedToko:  presentToko(selected),
		Banks:         banks,
		FeePercentage: feePercentage,
		MinimumAmount: minimumWithdrawalAmount,
	}, nil
}

func (s *Service) Inquiry(ctx context.Context, actor auth.PublicUser, tokoID int64, bankID int64, amount int64) (*InquiryResult, error) {
	if amount < minimumWithdrawalAmount {
		return nil, ErrAmountTooSmall
	}

	toko, err := s.findAccessibleToko(ctx, actor, tokoID)
	if err != nil {
		return nil, err
	}

	bank, err := s.findBankForUser(ctx, toko.UserID, bankID)
	if err != nil {
		return nil, err
	}

	platformFee, err := s.calculatePlatformFee(ctx, amount)
	if err != nil {
		return nil, err
	}

	estimatedTotal := amount + platformFee
	if toko.SettleBalance < estimatedTotal {
		return nil, ErrSettleBalanceNotEnough
	}

	response, err := s.qris.Inquiry(ctx, amount, bank.BankCode, bank.AccountNumber, withdrawalTransferType)
	if err != nil || response == nil || !response.Status || response.Data == nil {
		return nil, ErrInquiryFailed
	}

	inquiry := InquirySummary{
		InquiryID:                response.Data.InquiryID,
		AccountName:              strings.TrimSpace(response.Data.AccountName),
		BankName:                 strings.TrimSpace(response.Data.BankName),
		AccountNumber:            strings.TrimSpace(response.Data.AccountNumber),
		Amount:                   amount,
		BankFee:                  response.Data.Fee,
		PlatformFee:              platformFee,
		PartnerRefNo:             strings.TrimSpace(response.Data.PartnerRefNo),
		CurrentSettleBalance:     toko.SettleBalance,
		EstimatedTotalDeduction:  estimatedTotal,
		EstimatedRemainingSettle: toko.SettleBalance - estimatedTotal,
	}
	inquiry.FinalTotalDeduction = amount + inquiry.BankFee + inquiry.PlatformFee
	inquiry.FinalRemainingSettle = toko.SettleBalance - inquiry.FinalTotalDeduction

	if inquiry.AccountName == "" || inquiry.BankName == "" || inquiry.AccountNumber == "" || inquiry.PartnerRefNo == "" || inquiry.InquiryID <= 0 {
		return nil, ErrInquiryFailed
	}

	if toko.SettleBalance < inquiry.FinalTotalDeduction {
		return nil, ErrSettleBalanceNotEnough
	}

	if err := s.storeInquiry(ctx, actor, toko.ID, bank.ID, inquiry); err != nil {
		return nil, err
	}

	return &InquiryResult{
		SelectedToko: presentToko(toko),
		SelectedBank: bank,
		Inquiry:      inquiry,
	}, nil
}

func (s *Service) Submit(ctx context.Context, actor auth.PublicUser, tokoID int64, bankID int64, amount int64, inquiryID int64) (*SubmitResult, error) {
	if amount < minimumWithdrawalAmount {
		return nil, ErrAmountTooSmall
	}

	toko, err := s.findAccessibleToko(ctx, actor, tokoID)
	if err != nil {
		return nil, err
	}

	bank, err := s.findBankForUser(ctx, toko.UserID, bankID)
	if err != nil {
		return nil, err
	}

	cached, err := s.loadInquiry(ctx, actor, toko.ID, inquiryID)
	if err != nil {
		return nil, err
	}

	if cached.BankID != bank.ID || cached.Amount != amount {
		return nil, ErrInquiryStateNotFound
	}

	finalTotal := cached.Amount + cached.BankFee + cached.PlatformFee
	if toko.SettleBalance < finalTotal {
		return nil, ErrSettleBalanceNotEnough
	}

	response, err := s.qris.Transfer(ctx, amount, bank.BankCode, bank.AccountNumber, withdrawalTransferType, inquiryID)
	if err != nil || response == nil || !response.Status {
		return nil, ErrTransferFailed
	}

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin withdrawal tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var balanceID int64
	var currentSettle int64
	if err := tx.QueryRow(ctx, `
		SELECT id, COALESCE(settle, 0)
		FROM balances
		WHERE toko_id = $1
		FOR UPDATE
	`, toko.ID).Scan(&balanceID, &currentSettle); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSettleBalanceNotEnough
		}
		return nil, fmt.Errorf("lock withdrawal balance: %w", err)
	}

	if currentSettle < finalTotal {
		return nil, ErrSettleBalanceNotEnough
	}

	if _, err := tx.Exec(ctx, `
		UPDATE balances
		SET settle = settle - $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`, balanceID, finalTotal); err != nil {
		return nil, fmt.Errorf("decrement settle balance: %w", err)
	}

	notePayload, err := json.Marshal(map[string]any{
		"purpose":        "withdrawal",
		"bank_id":        bank.ID,
		"bank_name":      bank.BankName,
		"account_number": bank.AccountNumber,
		"account_name":   cached.AccountName,
		"fee":            cached.BankFee,
		"platform_fee":   cached.PlatformFee,
		"inquiry_id":     cached.InquiryID,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal withdrawal note: %w", err)
	}

	var transactionID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO transactions (
			toko_id,
			category,
			type,
			status,
			amount,
			code,
			note,
			created_at,
			updated_at
		) VALUES ($1, 'qris', 'withdrawal', 'pending', $2, $3, $4, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id
	`, toko.ID, amount, cached.PartnerRefNo, string(notePayload)).Scan(&transactionID); err != nil {
		return nil, fmt.Errorf("insert withdrawal transaction: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit withdrawal tx: %w", err)
	}

	if err := s.deleteInquiry(ctx, actor, toko.ID, inquiryID); err != nil {
		return nil, err
	}

	if s.notifications != nil {
		partnerRefNo := cached.PartnerRefNo
		_ = s.notifications.NotifyWithdrawalRequested(
			ctx,
			toko.UserID,
			toko.OwnerUsername,
			toko.Name,
			amount,
			cached.PlatformFee,
			cached.BankFee,
			&partnerRefNo,
		)
	}

	inquiry := InquirySummary{
		InquiryID:                cached.InquiryID,
		AccountName:              cached.AccountName,
		BankName:                 cached.BankName,
		AccountNumber:            cached.AccountNumber,
		Amount:                   cached.Amount,
		BankFee:                  cached.BankFee,
		PlatformFee:              cached.PlatformFee,
		PartnerRefNo:             cached.PartnerRefNo,
		CurrentSettleBalance:     toko.SettleBalance,
		EstimatedTotalDeduction:  cached.Amount + cached.PlatformFee,
		EstimatedRemainingSettle: toko.SettleBalance - (cached.Amount + cached.PlatformFee),
		FinalTotalDeduction:      finalTotal,
		FinalRemainingSettle:     toko.SettleBalance - finalTotal,
	}

	return &SubmitResult{
		SelectedToko: presentToko(toko),
		SelectedBank: bank,
		Inquiry:      inquiry,
		Transaction: SubmittedTransaction{
			ID:     transactionID,
			Code:   cached.PartnerRefNo,
			Status: "pending",
			Amount: amount,
		},
	}, nil
}

func (s *Service) accessibleTokos(ctx context.Context, actor auth.PublicUser) ([]accessibleToko, error) {
	query := `
		SELECT
			t.id,
			t.user_id,
			t.name,
			u.username,
			COALESCE(b.settle, 0) AS settle_balance
		FROM tokos t
		INNER JOIN users u ON u.id = t.user_id
		LEFT JOIN balances b ON b.toko_id = t.id
		WHERE t.deleted_at IS NULL
			AND t.is_active = TRUE
	`
	args := []any{}

	if !isGlobalRole(actor.Role) {
		query += " AND t.user_id = $1"
		args = append(args, actor.ID)
	}

	query += " ORDER BY t.name ASC, t.id ASC"

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list withdrawal tokos: %w", err)
	}
	defer rows.Close()

	result := make([]accessibleToko, 0, 8)
	for rows.Next() {
		var item accessibleToko
		if err := rows.Scan(&item.ID, &item.UserID, &item.Name, &item.OwnerUsername, &item.SettleBalance); err != nil {
			return nil, fmt.Errorf("scan withdrawal toko: %w", err)
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate withdrawal tokos: %w", err)
	}

	return result, nil
}

func (s *Service) findAccessibleToko(ctx context.Context, actor auth.PublicUser, tokoID int64) (*accessibleToko, error) {
	tokos, err := s.accessibleTokos(ctx, actor)
	if err != nil {
		return nil, err
	}

	for index := range tokos {
		if tokos[index].ID == tokoID {
			return &tokos[index], nil
		}
	}

	return nil, ErrTokoNotFound
}

func (s *Service) banksForUser(ctx context.Context, userID int64) ([]BankOption, error) {
	rows, err := s.db.Query(ctx, `
		SELECT
			id,
			bank_code,
			bank_name,
			account_number,
			account_name
		FROM banks
		WHERE user_id = $1
			AND deleted_at IS NULL
		ORDER BY created_at DESC, id DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list withdrawal banks: %w", err)
	}
	defer rows.Close()

	result := make([]BankOption, 0, 8)
	for rows.Next() {
		var item BankOption
		if err := rows.Scan(&item.ID, &item.BankCode, &item.BankName, &item.AccountNumber, &item.AccountName); err != nil {
			return nil, fmt.Errorf("scan withdrawal bank: %w", err)
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate withdrawal banks: %w", err)
	}

	return result, nil
}

func (s *Service) findBankForUser(ctx context.Context, userID int64, bankID int64) (*BankOption, error) {
	var item BankOption
	if err := s.db.QueryRow(ctx, `
		SELECT
			id,
			bank_code,
			bank_name,
			account_number,
			account_name
		FROM banks
		WHERE id = $1
			AND user_id = $2
			AND deleted_at IS NULL
	`, bankID, userID).Scan(&item.ID, &item.BankCode, &item.BankName, &item.AccountNumber, &item.AccountName); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBankNotFound
		}
		return nil, fmt.Errorf("find withdrawal bank: %w", err)
	}

	return &item, nil
}

func (s *Service) withdrawalFeePercentage(ctx context.Context) (int64, error) {
	var fee int64
	if err := s.db.QueryRow(ctx, `
		SELECT COALESCE(fee_withdrawal, 0)
		FROM incomes
		ORDER BY id ASC
		LIMIT 1
	`).Scan(&fee); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return defaultWithdrawalFeePercentage, nil
		}
		return 0, fmt.Errorf("read withdrawal fee percentage: %w", err)
	}
	if fee <= 0 {
		return defaultWithdrawalFeePercentage, nil
	}
	return fee, nil
}

func (s *Service) calculatePlatformFee(ctx context.Context, amount int64) (int64, error) {
	feePercentage, err := s.withdrawalFeePercentage(ctx)
	if err != nil {
		return 0, err
	}

	if amount <= 0 || feePercentage <= 0 {
		return 0, nil
	}

	return int64(math.Round((float64(amount) * float64(feePercentage)) / 100)), nil
}

func (s *Service) storeInquiry(ctx context.Context, actor auth.PublicUser, tokoID int64, bankID int64, inquiry InquirySummary) error {
	if s.redis == nil {
		return fmt.Errorf("redis client is not configured")
	}

	payload, err := json.Marshal(cachedInquiry{
		ActorID:       actor.ID,
		TokoID:        tokoID,
		BankID:        bankID,
		Amount:        inquiry.Amount,
		InquiryID:     inquiry.InquiryID,
		AccountName:   inquiry.AccountName,
		BankName:      inquiry.BankName,
		AccountNumber: inquiry.AccountNumber,
		BankFee:       inquiry.BankFee,
		PlatformFee:   inquiry.PlatformFee,
		PartnerRefNo:  inquiry.PartnerRefNo,
		CreatedAt:     time.Now().UTC(),
	})
	if err != nil {
		return fmt.Errorf("marshal withdrawal inquiry state: %w", err)
	}

	if err := s.redis.Set(ctx, s.inquiryKey(actor.ID, tokoID, inquiry.InquiryID), payload, withdrawalInquiryTTL).Err(); err != nil {
		return fmt.Errorf("store withdrawal inquiry state: %w", err)
	}

	return nil
}

func (s *Service) loadInquiry(ctx context.Context, actor auth.PublicUser, tokoID int64, inquiryID int64) (*cachedInquiry, error) {
	if s.redis == nil {
		return nil, fmt.Errorf("redis client is not configured")
	}

	raw, err := s.redis.Get(ctx, s.inquiryKey(actor.ID, tokoID, inquiryID)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrInquiryStateNotFound
		}
		return nil, fmt.Errorf("load withdrawal inquiry state: %w", err)
	}

	var payload cachedInquiry
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("unmarshal withdrawal inquiry state: %w", err)
	}

	if payload.ActorID != actor.ID || payload.TokoID != tokoID || payload.InquiryID != inquiryID {
		return nil, ErrInquiryStateNotFound
	}

	return &payload, nil
}

func (s *Service) deleteInquiry(ctx context.Context, actor auth.PublicUser, tokoID int64, inquiryID int64) error {
	if s.redis == nil {
		return nil
	}

	if err := s.redis.Del(ctx, s.inquiryKey(actor.ID, tokoID, inquiryID)).Err(); err != nil {
		return fmt.Errorf("delete withdrawal inquiry state: %w", err)
	}
	return nil
}

func (s *Service) inquiryKey(actorID int64, tokoID int64, inquiryID int64) string {
	return fmt.Sprintf("withdrawal:inquiry:%d:%d:%d", actorID, tokoID, inquiryID)
}

func presentTokos(items []accessibleToko) []TokoOption {
	result := make([]TokoOption, 0, len(items))
	for _, item := range items {
		result = append(result, TokoOption{
			ID:            item.ID,
			Name:          item.Name,
			OwnerUsername: item.OwnerUsername,
			SettleBalance: item.SettleBalance,
		})
	}

	return result
}

func presentToko(item *accessibleToko) *TokoOption {
	if item == nil {
		return nil
	}

	return &TokoOption{
		ID:            item.ID,
		Name:          item.Name,
		OwnerUsername: item.OwnerUsername,
		SettleBalance: item.SettleBalance,
	}
}

func isGlobalRole(role string) bool {
	return role == "dev" || role == "superadmin"
}
