package notifications

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/auth"
)

const (
	UserNotifiableType                             = `App\Models\User`
	TypeDepositSuccessUserNotification             = `App\Notifications\DepositSuccessUserNotification`
	TypeWithdrawalStatusUpdatedNotification        = `App\Notifications\WithdrawalStatusUpdatedNotification`
	TypeWithdrawalRequestedDevNotification         = `App\Notifications\WithdrawalRequestedDevNotification`
	TypeWithdrawalRequestedUserNotification        = `App\Notifications\WithdrawalRequestedUserNotification`
	TypeTokoCreatedNotification                    = `App\Notifications\TokoCreatedNotification`
	TypeTokoCreatedOwnerNotification               = `App\Notifications\TokoCreatedOwnerNotification`
	TypeTokoCallbackDeliveryFailedNotification     = `App\Notifications\TokoCallbackDeliveryFailedNotification`
	TypeMonthlyOperationalFeeCollectedNotification = `App\Notifications\MonthlyOperationalFeeCollectedNotification`
)

type querier interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type Severity string

const (
	SeverityInfo    Severity = "info"
	SeveritySuccess Severity = "success"
	SeverityWarning Severity = "warning"
	SeverityDanger  Severity = "danger"
)

type Action struct {
	Label string  `json:"label,omitempty"`
	URL   *string `json:"url,omitempty"`
}

type Data struct {
	Format    string         `json:"format,omitempty"`
	Title     string         `json:"title"`
	Body      string         `json:"body,omitempty"`
	Icon      string         `json:"icon,omitempty"`
	IconColor string         `json:"iconColor,omitempty"`
	Status    string         `json:"status,omitempty"`
	Action    *Action        `json:"action,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

type Record struct {
	ID        string
	Type      string
	Title     string
	Body      string
	Icon      string
	IconColor string
	Status    string
	Action    *Action
	ReadAt    *time.Time
	CreatedAt time.Time
}

type ListInput struct {
	Scope   string
	Page    int
	PerPage int
}

type ListSummary struct {
	Total          int64
	Unread         int64
	UnreadCritical int64
	UnreadWarnings int64
	UnreadSuccess  int64
}

type ListResult struct {
	Data       []Record
	Page       int
	PerPage    int
	Total      int64
	TotalPages int
	Summary    ListSummary
}

type Service struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

func NewService(db *pgxpool.Pool, logger zerolog.Logger) *Service {
	return &Service{
		db:     db,
		logger: logger.With().Str("module", "notifications").Logger(),
	}
}

func (s *Service) ListForBackoffice(ctx context.Context, actor auth.PublicUser, input ListInput) (*ListResult, error) {
	page := input.Page
	if page < 1 {
		page = 1
	}

	perPage := input.PerPage
	switch perPage {
	case 8, 10, 20, 25, 50:
	default:
		perPage = 20
	}

	scope := strings.ToLower(strings.TrimSpace(input.Scope))
	includeUnreadOnly := scope == "unread"

	baseArgs := []any{UserNotifiableType, actor.ID}
	whereClause := `
		WHERE notifiable_type = $1
			AND notifiable_id = $2
	`
	if includeUnreadOnly {
		whereClause += `
			AND read_at IS NULL
		`
	}

	var total int64
	if err := s.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM notifications
	`+whereClause, baseArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count notifications: %w", err)
	}

	summary, err := s.summary(ctx, actor.ID)
	if err != nil {
		return nil, err
	}

	args := append([]any{}, baseArgs...)
	args = append(args, perPage, (page-1)*perPage)
	rows, err := s.db.Query(ctx, `
		SELECT id::text, type, data, read_at, created_at
		FROM notifications
	`+whereClause+`
		ORDER BY created_at DESC, id DESC
		LIMIT $3
		OFFSET $4
	`, args...)
	if err != nil {
		return nil, fmt.Errorf("list notifications: %w", err)
	}
	defer rows.Close()

	items := make([]Record, 0, perPage)
	for rows.Next() {
		var (
			record  Record
			payload []byte
		)
		if err := rows.Scan(&record.ID, &record.Type, &payload, &record.ReadAt, &record.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan notification row: %w", err)
		}

		message := parseData(payload)
		record.Title = message.Title
		record.Body = message.Body
		record.Icon = message.Icon
		record.IconColor = message.IconColor
		record.Status = message.Status
		record.Action = message.Action
		items = append(items, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate notifications: %w", err)
	}

	totalPages := 0
	if total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(perPage)))
	}

	return &ListResult{
		Data:       items,
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: totalPages,
		Summary:    summary,
	}, nil
}

func (s *Service) MarkRead(ctx context.Context, actor auth.PublicUser, notificationID string) error {
	parsedID, err := uuid.Parse(strings.TrimSpace(notificationID))
	if err != nil {
		return fmt.Errorf("invalid notification id: %w", err)
	}

	tag, err := s.db.Exec(ctx, `
		UPDATE notifications
		SET read_at = COALESCE(read_at, CURRENT_TIMESTAMP),
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
			AND notifiable_type = $2
			AND notifiable_id = $3
	`, parsedID, UserNotifiableType, actor.ID)
	if err != nil {
		return fmt.Errorf("mark notification read: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}

func (s *Service) MarkAllRead(ctx context.Context, actor auth.PublicUser) (int64, error) {
	tag, err := s.db.Exec(ctx, `
		UPDATE notifications
		SET read_at = CURRENT_TIMESTAMP,
			updated_at = CURRENT_TIMESTAMP
		WHERE notifiable_type = $1
			AND notifiable_id = $2
			AND read_at IS NULL
	`, UserNotifiableType, actor.ID)
	if err != nil {
		return 0, fmt.Errorf("mark all notifications read: %w", err)
	}

	return tag.RowsAffected(), nil
}

func (s *Service) NotifyTokoCreated(ctx context.Context, ownerUserID int64, tokoName string, ownerName string) error {
	adminPayload := Data{
		Format:    "filament",
		Title:     "Toko baru dibuat",
		Body:      fmt.Sprintf("Toko \"%s\" baru saja dibuat oleh %s.", tokoName, ownerName),
		Icon:      "heroicon-o-building-storefront",
		IconColor: string(SeverityInfo),
		Status:    string(SeverityInfo),
		Action:    action("Buka toko", "/backoffice/tokos"),
	}
	if err := s.notifyRoles(ctx, []string{"dev", "superadmin"}, []int64{ownerUserID}, TypeTokoCreatedNotification, adminPayload); err != nil {
		return err
	}

	ownerPayload := Data{
		Format:    "filament",
		Title:     "Selamat! Toko berhasil dibuat",
		Body:      fmt.Sprintf("Toko \"%s\" berhasil dibuat.", tokoName),
		Icon:      "heroicon-o-check-circle",
		IconColor: string(SeveritySuccess),
		Status:    string(SeveritySuccess),
		Action:    action("Lihat toko", "/backoffice/tokos"),
	}
	return s.insertUserNotification(ctx, s.db, ownerUserID, TypeTokoCreatedOwnerNotification, ownerPayload)
}

func (s *Service) NotifyWithdrawalRequested(ctx context.Context, ownerUserID int64, ownerUsername string, tokoName string, amount int64, platformFee int64, bankFee int64, transactionCode *string) error {
	devPayload := Data{
		Format:    "filament",
		Title:     "Withdrawal Pending",
		Body:      fmt.Sprintf("Username %s toko %s baru saja melakukan withdraw dengan status pending.", ownerUsername, tokoName),
		Icon:      "heroicon-o-banknotes",
		IconColor: string(SeverityWarning),
		Status:    string(SeverityWarning),
		Action:    action("Audit transaksi", "/backoffice/transactions"),
		Metadata:  codeMetadata(transactionCode),
	}
	if err := s.notifyRoles(ctx, []string{"dev", "superadmin"}, nil, TypeWithdrawalRequestedDevNotification, devPayload); err != nil {
		return err
	}

	totalDeduction := platformFee + bankFee
	userPayload := Data{
		Format:    "filament",
		Title:     "Withdrawal Terkirim",
		Body:      fmt.Sprintf("Permintaan withdraw sebesar Rp %s dengan total potongan admin sebesar Rp %s sedang dalam proses.", formatIDR(amount), formatIDR(totalDeduction)),
		Icon:      "heroicon-o-clock",
		IconColor: string(SeverityInfo),
		Status:    string(SeverityInfo),
		Action:    action("Lihat transaksi", "/backoffice/transactions"),
		Metadata:  codeMetadata(transactionCode),
	}
	return s.insertUserNotification(ctx, s.db, ownerUserID, TypeWithdrawalRequestedUserNotification, userPayload)
}

func (s *Service) NotifyDepositSuccess(ctx context.Context, ownerUserID int64, amount int64, isNexusggrTopup bool, transactionCode *string) error {
	title := "Deposit Player Masuk"
	body := fmt.Sprintf("Pembayaran deposit pemain sebesar Rp %s telah berhasil masuk ke saldo pending.", formatIDR(amount))
	if isNexusggrTopup {
		title = "Topup NexusGGR Berhasil"
		body = fmt.Sprintf("Topup saldo NexusGGR sebesar Rp %s telah berhasil diproses.", formatIDR(amount))
	}

	payload := Data{
		Format:    "filament",
		Title:     title,
		Body:      body,
		Icon:      "heroicon-o-arrow-down-tray",
		IconColor: string(SeveritySuccess),
		Status:    string(SeveritySuccess),
		Action:    action("Lihat transaksi", "/backoffice/transactions"),
		Metadata:  codeMetadata(transactionCode),
	}

	return s.insertUserNotification(ctx, s.db, ownerUserID, TypeDepositSuccessUserNotification, payload)
}

func (s *Service) NotifyWithdrawalStatusUpdated(ctx context.Context, ownerUserID int64, amount int64, status string, transactionCode *string) error {
	upperStatus := strings.ToUpper(strings.TrimSpace(status))
	severity := SeverityDanger
	icon := "heroicon-o-x-circle"
	if strings.EqualFold(status, "success") {
		severity = SeveritySuccess
		icon = "heroicon-o-check-circle"
	}

	payload := Data{
		Format:    "filament",
		Title:     fmt.Sprintf("Withdrawal %s", upperStatus),
		Body:      fmt.Sprintf("Permintaan withdraw sebesar Rp %s berstatus: %s.", formatIDR(amount), upperStatus),
		Icon:      icon,
		IconColor: string(severity),
		Status:    string(severity),
		Action:    action("Lihat transaksi", "/backoffice/transactions"),
		Metadata:  codeMetadata(transactionCode),
	}

	return s.insertUserNotification(ctx, s.db, ownerUserID, TypeWithdrawalStatusUpdatedNotification, payload)
}

func (s *Service) NotifyCallbackDeliveryFailed(ctx context.Context, eventType string, reference string, callbackURL string, statusCode int, failure string) error {
	body := fmt.Sprintf("Callback %s untuk reference %s ke %s gagal.", strings.ToUpper(strings.TrimSpace(eventType)), reference, callbackURL)
	if statusCode > 0 {
		body = fmt.Sprintf("%s HTTP %d.", body, statusCode)
	}
	if failure != "" {
		body = fmt.Sprintf("%s %s", body, strings.TrimSpace(failure))
	}

	payload := Data{
		Format:    "filament",
		Title:     "Toko callback gagal",
		Body:      body,
		Icon:      "heroicon-o-exclamation-triangle",
		IconColor: string(SeverityDanger),
		Status:    string(SeverityDanger),
		Action:    action("Audit transaksi", "/backoffice/transactions"),
		Metadata: map[string]any{
			"eventType":   eventType,
			"reference":   reference,
			"callbackUrl": callbackURL,
			"statusCode":  statusCode,
			"failure":     failure,
		},
	}

	return s.notifyRoles(ctx, []string{"dev", "superadmin"}, nil, TypeTokoCallbackDeliveryFailedNotification, payload)
}

func (s *Service) NotifyMonthlyOperationalFeesCollected(ctx context.Context, processedCount int, deductedTotal int64) error {
	payload := Data{
		Format:    "filament",
		Title:     "Biaya operasional bulanan diproses",
		Body:      fmt.Sprintf("Potongan settle VPS dan operasional berhasil diambil dari %d toko dengan total Rp %s.", processedCount, formatIDR(deductedTotal)),
		Icon:      "heroicon-o-banknotes",
		IconColor: string(SeverityInfo),
		Status:    string(SeverityInfo),
		Action:    action("Buka dashboard", "/backoffice"),
		Metadata: map[string]any{
			"processedCount": processedCount,
			"deductedTotal":  deductedTotal,
		},
	}

	return s.notifyRoles(ctx, []string{"dev", "superadmin"}, nil, TypeMonthlyOperationalFeeCollectedNotification, payload)
}

func (s *Service) summary(ctx context.Context, userID int64) (ListSummary, error) {
	var summary ListSummary
	if err := s.db.QueryRow(ctx, `
		SELECT
			COUNT(*)::bigint AS total,
			COUNT(*) FILTER (WHERE read_at IS NULL)::bigint AS unread,
			COUNT(*) FILTER (
				WHERE read_at IS NULL
				AND COALESCE(data->>'status', '') IN ('warning', 'danger')
			)::bigint AS unread_critical,
			COUNT(*) FILTER (
				WHERE read_at IS NULL
				AND COALESCE(data->>'status', '') = 'warning'
			)::bigint AS unread_warnings,
			COUNT(*) FILTER (
				WHERE read_at IS NULL
				AND COALESCE(data->>'status', '') = 'success'
			)::bigint AS unread_success
		FROM notifications
		WHERE notifiable_type = $1
			AND notifiable_id = $2
	`, UserNotifiableType, userID).Scan(
		&summary.Total,
		&summary.Unread,
		&summary.UnreadCritical,
		&summary.UnreadWarnings,
		&summary.UnreadSuccess,
	); err != nil {
		return ListSummary{}, fmt.Errorf("notification summary: %w", err)
	}

	return summary, nil
}

func (s *Service) notifyRoles(ctx context.Context, roles []string, excludeUserIDs []int64, notificationType string, payload Data) error {
	userIDs, err := s.userIDsByRoles(ctx, roles, excludeUserIDs)
	if err != nil {
		return err
	}

	return s.insertUserNotifications(ctx, s.db, userIDs, notificationType, payload)
}

func (s *Service) userIDsByRoles(ctx context.Context, roles []string, excludeUserIDs []int64) ([]int64, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id
		FROM users
		WHERE is_active = TRUE
			AND role = ANY($1)
		ORDER BY id
	`, roles)
	if err != nil {
		return nil, fmt.Errorf("list notification users by roles: %w", err)
	}
	defer rows.Close()

	excluded := make(map[int64]struct{}, len(excludeUserIDs))
	for _, id := range excludeUserIDs {
		excluded[id] = struct{}{}
	}

	userIDs := make([]int64, 0)
	for rows.Next() {
		var userID int64
		if err := rows.Scan(&userID); err != nil {
			return nil, fmt.Errorf("scan notification user id: %w", err)
		}
		if _, blocked := excluded[userID]; blocked {
			continue
		}
		userIDs = append(userIDs, userID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate notification users by roles: %w", err)
	}

	return userIDs, nil
}

func (s *Service) insertUserNotifications(ctx context.Context, q querier, userIDs []int64, notificationType string, payload Data) error {
	deduped := make([]int64, 0, len(userIDs))
	for _, userID := range userIDs {
		if userID <= 0 || slices.Contains(deduped, userID) {
			continue
		}
		deduped = append(deduped, userID)
	}

	for _, userID := range deduped {
		if err := s.insertUserNotification(ctx, q, userID, notificationType, payload); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) insertUserNotification(ctx context.Context, q querier, userID int64, notificationType string, payload Data) error {
	if userID <= 0 {
		return nil
	}

	if strings.TrimSpace(payload.Format) == "" {
		payload.Format = "filament"
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal notification payload: %w", err)
	}

	if _, err := q.Exec(ctx, `
		INSERT INTO notifications (
			id,
			type,
			notifiable_type,
			notifiable_id,
			data,
			read_at,
			created_at,
			updated_at
		) VALUES ($1, $2, $3, $4, $5, NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, uuid.NewString(), notificationType, UserNotifiableType, userID, body); err != nil {
		return fmt.Errorf("insert notification: %w", err)
	}

	return nil
}

func parseData(raw []byte) Data {
	var payload Data
	if err := json.Unmarshal(raw, &payload); err != nil {
		return Data{}
	}

	return payload
}

func formatIDR(amount int64) string {
	raw := fmt.Sprintf("%d", amount)
	if amount == 0 {
		return "0"
	}

	negative := strings.HasPrefix(raw, "-")
	if negative {
		raw = strings.TrimPrefix(raw, "-")
	}

	parts := make([]string, 0, len(raw)/3+1)
	for len(raw) > 3 {
		parts = append([]string{raw[len(raw)-3:]}, parts...)
		raw = raw[:len(raw)-3]
	}
	parts = append([]string{raw}, parts...)

	formatted := strings.Join(parts, ".")
	if negative {
		return "-" + formatted
	}
	return formatted
}

func action(label string, url string) *Action {
	if strings.TrimSpace(label) == "" || strings.TrimSpace(url) == "" {
		return nil
	}
	cleanURL := strings.TrimSpace(url)
	return &Action{Label: strings.TrimSpace(label), URL: &cleanURL}
}

func codeMetadata(code *string) map[string]any {
	if code == nil || strings.TrimSpace(*code) == "" {
		return nil
	}
	return map[string]any{"code": strings.TrimSpace(*code)}
}
