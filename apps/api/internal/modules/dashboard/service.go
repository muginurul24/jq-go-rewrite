package dashboard

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/auth"
	nexusggrintegration "github.com/mugiew/justqiuv2-rewrite/apps/api/internal/integrations/nexusggr"
	qrisintegration "github.com/mugiew/justqiuv2-rewrite/apps/api/internal/integrations/qris"
)

const externalBalanceCacheTTL = 5 * time.Minute

type qrisClient interface {
	Balance(ctx context.Context) (*qrisintegration.BalanceResponse, error)
}

type nexusClient interface {
	MoneyInfo(ctx context.Context, userCode *string, allUsers bool) (*nexusggrintegration.MoneyInfoResponse, error)
}

type Service struct {
	db       *pgxpool.Pool
	cache    *redis.Client
	qris     qrisClient
	nexusggr nexusClient
	location *time.Location
	now      func() time.Time
}

type OverviewResult struct {
	GeneratedAt        string               `json:"generatedAt"`
	Role               string               `json:"role"`
	Stats              OverviewStats        `json:"stats"`
	AlertSummary       OverviewAlertSummary `json:"alertSummary"`
	Alerts             []OverviewAlert      `json:"alerts"`
	RecentTransactions []RecentTransaction  `json:"recentTransactions"`
}

type OverviewStats struct {
	PendingBalance       int64   `json:"pendingBalance"`
	SettleBalance        int64   `json:"settleBalance"`
	NexusggrBalance      int64   `json:"nexusggrBalance"`
	PlatformIncome       *int64  `json:"platformIncome,omitempty"`
	ExternalQRPending    *int64  `json:"externalQrPending,omitempty"`
	ExternalQRSettle     *int64  `json:"externalQrSettle,omitempty"`
	ExternalAgentBalance *int64  `json:"externalAgentBalance,omitempty"`
	ExternalAgentCode    *string `json:"externalAgentCode,omitempty"`
}

type RecentTransaction struct {
	ID        int64   `json:"id"`
	Code      *string `json:"code,omitempty"`
	TokoName  string  `json:"tokoName"`
	Player    *string `json:"player,omitempty"`
	Category  string  `json:"category"`
	Type      string  `json:"type"`
	Status    string  `json:"status"`
	Amount    int64   `json:"amount"`
	CreatedAt string  `json:"createdAt"`
}

type OverviewAlertSummary struct {
	UnreadNotifications   int64 `json:"unreadNotifications"`
	CriticalNotifications int64 `json:"criticalNotifications"`
	PendingOverdueQris    int64 `json:"pendingOverdueQris"`
	PendingWithdrawals    int64 `json:"pendingWithdrawals"`
	LowSettleTokos        int64 `json:"lowSettleTokos"`
	LowNexusggrTokos      int64 `json:"lowNexusggrTokos"`
}

type OverviewAlert struct {
	Key      string `json:"key"`
	Severity string `json:"severity"`
	Title    string `json:"title"`
	Body     string `json:"body"`
	Count    int64  `json:"count"`
	Href     string `json:"href"`
}

type OperationalPulseResult struct {
	GeneratedAt string                 `json:"generatedAt"`
	Role        string                 `json:"role"`
	Stats       OperationalPulseStats  `json:"stats"`
	QRIS        []TransactionSeriesRow `json:"qris"`
	Nexusggr    []TransactionSeriesRow `json:"nexusggr"`
}

type OperationalPulseStats struct {
	PendingTransactions  int64 `json:"pendingTransactions"`
	FailedTransactions7d int64 `json:"failedTransactions7d"`
	SuccessfulQRIS7d     int64 `json:"successfulQris7d"`
	SuccessfulNexusggr7d int64 `json:"successfulNexusggr7d"`
}

type TransactionSeriesRow struct {
	Date       string `json:"date"`
	Deposit    int64  `json:"deposit"`
	Withdrawal int64  `json:"withdrawal"`
}

type localTotals struct {
	Pending  int64
	Settle   int64
	Nexusggr int64
}

const lowBalanceAlertThreshold int64 = 100_000

type cachedQRBalance struct {
	Pending int64 `json:"pending"`
	Settle  int64 `json:"settle"`
}

type cachedAgentBalance struct {
	Balance int64   `json:"balance"`
	Code    *string `json:"code,omitempty"`
}

func NewService(db *pgxpool.Pool, cache *redis.Client, qris qrisClient, nexusggr nexusClient, timezone string) *Service {
	location := time.Local
	if strings.TrimSpace(timezone) != "" {
		if resolved, err := time.LoadLocation(timezone); err == nil {
			location = resolved
		}
	}

	return &Service{
		db:       db,
		cache:    cache,
		qris:     qris,
		nexusggr: nexusggr,
		location: location,
		now:      time.Now,
	}
}

func (s *Service) Overview(ctx context.Context, actor auth.PublicUser) (*OverviewResult, error) {
	totals, err := s.loadLocalTotals(ctx, actor)
	if err != nil {
		return nil, err
	}

	alertSummary, err := s.loadAlertSummary(ctx, actor)
	if err != nil {
		return nil, err
	}

	recentTransactions, err := s.loadRecentTransactions(ctx, actor, 8)
	if err != nil {
		return nil, err
	}

	stats := OverviewStats{
		PendingBalance:  totals.Pending,
		SettleBalance:   totals.Settle,
		NexusggrBalance: totals.Nexusggr,
	}

	if actor.Role == "dev" {
		incomeAmount, err := s.loadPlatformIncome(ctx)
		if err != nil {
			return nil, err
		}
		if incomeAmount != nil {
			stats.PlatformIncome = incomeAmount
		}

		if qrBalance, err := s.loadExternalQRBalance(ctx); err == nil && qrBalance != nil {
			stats.ExternalQRPending = &qrBalance.Pending
			stats.ExternalQRSettle = &qrBalance.Settle
		}

		if agentBalance, err := s.loadExternalAgentBalance(ctx); err == nil && agentBalance != nil {
			stats.ExternalAgentBalance = &agentBalance.Balance
			stats.ExternalAgentCode = agentBalance.Code
		}
	}

	return &OverviewResult{
		GeneratedAt:        s.currentTime().UTC().Format(time.RFC3339Nano),
		Role:               actor.Role,
		Stats:              stats,
		AlertSummary:       alertSummary,
		Alerts:             buildOverviewAlerts(alertSummary),
		RecentTransactions: recentTransactions,
	}, nil
}

func (s *Service) OperationalPulse(ctx context.Context, actor auth.PublicUser) (*OperationalPulseResult, error) {
	rows, err := s.loadTransactionSeries(ctx, actor, 7)
	if err != nil {
		return nil, err
	}

	qrisSeries, nexusSeries := presentSeries(rows, 7, s.currentTime())

	stats, err := s.loadOperationalStats(ctx, actor, 7)
	if err != nil {
		return nil, err
	}

	return &OperationalPulseResult{
		GeneratedAt: s.currentTime().UTC().Format(time.RFC3339Nano),
		Role:        actor.Role,
		Stats:       stats,
		QRIS:        qrisSeries,
		Nexusggr:    nexusSeries,
	}, nil
}

func (s *Service) loadLocalTotals(ctx context.Context, actor auth.PublicUser) (localTotals, error) {
	scopeClause, args := scopeCondition(actor, 1)

	query := `
		SELECT
			COALESCE(SUM(b.pending), 0),
			COALESCE(SUM(b.settle), 0),
			COALESCE(SUM(b.nexusggr), 0)
		FROM balances b
		INNER JOIN tokos t ON t.id = b.toko_id
		WHERE t.deleted_at IS NULL
	` + scopeClause

	var result localTotals
	if err := s.db.QueryRow(ctx, query, args...).Scan(&result.Pending, &result.Settle, &result.Nexusggr); err != nil {
		return localTotals{}, fmt.Errorf("load dashboard local totals: %w", err)
	}

	return result, nil
}

func (s *Service) loadPlatformIncome(ctx context.Context) (*int64, error) {
	var amount *int64
	if err := s.db.QueryRow(ctx, `
		SELECT amount
		FROM incomes
		ORDER BY id ASC
		LIMIT 1
	`).Scan(&amount); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("load platform income: %w", err)
	}

	return amount, nil
}

func (s *Service) loadRecentTransactions(ctx context.Context, actor auth.PublicUser, limit int) ([]RecentTransaction, error) {
	if limit <= 0 {
		limit = 8
	}

	scopeClause, args := scopeCondition(actor, 1)
	args = append(args, limit)

	query := `
		SELECT
			tx.id,
			tx.code,
			t.name,
			tx.player,
			tx.category,
			tx.type,
			tx.status,
			tx.amount,
			tx.created_at
		FROM transactions tx
		INNER JOIN tokos t ON t.id = tx.toko_id
		WHERE t.deleted_at IS NULL
	` + scopeClause + `
		ORDER BY tx.created_at DESC
		LIMIT $` + strconv.Itoa(len(args))

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("load recent dashboard transactions: %w", err)
	}
	defer rows.Close()

	transactions := make([]RecentTransaction, 0, limit)
	for rows.Next() {
		var record RecentTransaction
		var createdAt time.Time
		if err := rows.Scan(
			&record.ID,
			&record.Code,
			&record.TokoName,
			&record.Player,
			&record.Category,
			&record.Type,
			&record.Status,
			&record.Amount,
			&createdAt,
		); err != nil {
			return nil, fmt.Errorf("scan recent dashboard transaction: %w", err)
		}

		record.CreatedAt = createdAt.UTC().Format(time.RFC3339Nano)
		transactions = append(transactions, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recent dashboard transactions: %w", err)
	}

	return transactions, nil
}

type seriesRow struct {
	Category string
	Type     string
	Date     time.Time
	Amount   int64
}

func (s *Service) loadTransactionSeries(ctx context.Context, actor auth.PublicUser, days int) ([]seriesRow, error) {
	if days <= 0 {
		days = 7
	}

	startDate := beginningOfDay(s.currentTime()).AddDate(0, 0, -(days - 1))
	args := []any{startDate}
	scopeClause, scopeArgs := scopeCondition(actor, 2)
	args = append(args, scopeArgs...)

	query := `
		SELECT
			tx.category,
			tx.type,
			DATE(tx.created_at),
			COALESCE(SUM(tx.amount), 0)
		FROM transactions tx
		INNER JOIN tokos t ON t.id = tx.toko_id
		WHERE t.deleted_at IS NULL
			AND tx.status = 'success'
			AND tx.created_at >= $1
	` + scopeClause + `
		GROUP BY tx.category, tx.type, DATE(tx.created_at)
	`

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("load dashboard transaction series: %w", err)
	}
	defer rows.Close()

	result := make([]seriesRow, 0, days*4)
	for rows.Next() {
		var row seriesRow
		if err := rows.Scan(&row.Category, &row.Type, &row.Date, &row.Amount); err != nil {
			return nil, fmt.Errorf("scan dashboard transaction series: %w", err)
		}
		result = append(result, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate dashboard transaction series: %w", err)
	}

	return result, nil
}

func (s *Service) loadOperationalStats(ctx context.Context, actor auth.PublicUser, days int) (OperationalPulseStats, error) {
	if days <= 0 {
		days = 7
	}

	startDate := beginningOfDay(s.currentTime()).AddDate(0, 0, -(days - 1))
	args := []any{startDate}
	scopeClause, scopeArgs := scopeCondition(actor, 2)
	args = append(args, scopeArgs...)

	query := `
		SELECT
			COUNT(*) FILTER (WHERE tx.status = 'pending') AS pending_transactions,
			COUNT(*) FILTER (WHERE tx.status = 'failed' AND tx.created_at >= $1) AS failed_transactions_7d,
			COUNT(*) FILTER (WHERE tx.status = 'success' AND tx.category = 'qris' AND tx.created_at >= $1) AS successful_qris_7d,
			COUNT(*) FILTER (WHERE tx.status = 'success' AND tx.category = 'nexusggr' AND tx.created_at >= $1) AS successful_nexusggr_7d
		FROM transactions tx
		INNER JOIN tokos t ON t.id = tx.toko_id
		WHERE t.deleted_at IS NULL
	` + scopeClause

	var stats OperationalPulseStats
	if err := s.db.QueryRow(ctx, query, args...).Scan(
		&stats.PendingTransactions,
		&stats.FailedTransactions7d,
		&stats.SuccessfulQRIS7d,
		&stats.SuccessfulNexusggr7d,
	); err != nil {
		return OperationalPulseStats{}, fmt.Errorf("load dashboard operational stats: %w", err)
	}

	return stats, nil
}

func (s *Service) loadAlertSummary(ctx context.Context, actor auth.PublicUser) (OverviewAlertSummary, error) {
	scopeClause, args := scopeCondition(actor, 1)
	args = append(args, s.currentTime().Add(-30*time.Minute))

	var summary OverviewAlertSummary
	query := `
		SELECT
			COUNT(*) FILTER (
				WHERE tx.category = 'qris'
					AND tx.type = 'deposit'
					AND tx.status = 'pending'
					AND tx.created_at < $` + strconv.Itoa(len(args)) + `
			) AS pending_overdue_qris,
			COUNT(*) FILTER (
				WHERE tx.category = 'qris'
					AND tx.type = 'withdrawal'
					AND tx.status = 'pending'
			) AS pending_withdrawals,
			COUNT(DISTINCT t.id) FILTER (
				WHERE COALESCE(b.settle, 0) < ` + strconv.FormatInt(lowBalanceAlertThreshold, 10) + `
			) AS low_settle_tokos,
			COUNT(DISTINCT t.id) FILTER (
				WHERE COALESCE(b.nexusggr, 0) < ` + strconv.FormatInt(lowBalanceAlertThreshold, 10) + `
			) AS low_nexusggr_tokos
		FROM tokos t
		LEFT JOIN balances b ON b.toko_id = t.id
		LEFT JOIN transactions tx ON tx.toko_id = t.id
		WHERE t.deleted_at IS NULL
			AND t.is_active = TRUE
	` + scopeClause

	if err := s.db.QueryRow(ctx, query, args...).Scan(
		&summary.PendingOverdueQris,
		&summary.PendingWithdrawals,
		&summary.LowSettleTokos,
		&summary.LowNexusggrTokos,
	); err != nil {
		return OverviewAlertSummary{}, fmt.Errorf("load dashboard alert summary: %w", err)
	}

	if err := s.db.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE read_at IS NULL)::bigint AS unread_notifications,
			COUNT(*) FILTER (
				WHERE read_at IS NULL
					AND COALESCE(data->>'status', '') IN ('warning', 'danger')
			)::bigint AS critical_notifications
		FROM notifications
		WHERE notifiable_type = $1
			AND notifiable_id = $2
	`, `App\Models\User`, actor.ID).Scan(&summary.UnreadNotifications, &summary.CriticalNotifications); err != nil {
		return OverviewAlertSummary{}, fmt.Errorf("load dashboard notification summary: %w", err)
	}

	return summary, nil
}

func (s *Service) loadExternalQRBalance(ctx context.Context) (*cachedQRBalance, error) {
	const cacheKey = "dashboard:qr_balance"

	if cached := s.loadCachedQRBalance(ctx, cacheKey); cached != nil {
		return cached, nil
	}

	response, err := s.qris.Balance(ctx)
	if err != nil || response == nil || strings.ToLower(strings.TrimSpace(response.Status)) != "success" {
		return nil, err
	}

	result := &cachedQRBalance{
		Pending: toInt64(response.PendingBalance),
		Settle:  toInt64(response.SettleBalance),
	}
	s.storeCachedJSON(ctx, cacheKey, result)

	return result, nil
}

func buildOverviewAlerts(summary OverviewAlertSummary) []OverviewAlert {
	alerts := make([]OverviewAlert, 0, 5)

	if summary.CriticalNotifications > 0 {
		alerts = append(alerts, OverviewAlert{
			Key:      "critical-notifications",
			Severity: "danger",
			Title:    "Notifikasi kritikal belum dibaca",
			Body:     "Tindak lanjuti callback gagal atau error operasional yang belum dibaca.",
			Count:    summary.CriticalNotifications,
			Href:     "/backoffice/notifications",
		})
	}
	if summary.PendingOverdueQris > 0 {
		alerts = append(alerts, OverviewAlert{
			Key:      "pending-overdue-qris",
			Severity: "warning",
			Title:    "QRIS pending melewati 30 menit",
			Body:     "Ada transaksi deposit QRIS yang belum beres melebihi SLA expiry legacy.",
			Count:    summary.PendingOverdueQris,
			Href:     "/backoffice/transactions",
		})
	}
	if summary.PendingWithdrawals > 0 {
		alerts = append(alerts, OverviewAlert{
			Key:      "pending-withdrawals",
			Severity: "warning",
			Title:    "Withdrawal masih pending",
			Body:     "Ada transfer keluar yang masih menunggu callback disbursement.",
			Count:    summary.PendingWithdrawals,
			Href:     "/backoffice/transactions",
		})
	}
	if summary.LowSettleTokos > 0 {
		alerts = append(alerts, OverviewAlert{
			Key:      "low-settle-tokos",
			Severity: "danger",
			Title:    "Saldo settle toko menipis",
			Body:     "Beberapa toko punya settle di bawah ambang operasional Rp 100.000.",
			Count:    summary.LowSettleTokos,
			Href:     "/backoffice/tokos",
		})
	}
	if summary.LowNexusggrTokos > 0 {
		alerts = append(alerts, OverviewAlert{
			Key:      "low-nexusggr-tokos",
			Severity: "warning",
			Title:    "Saldo NexusGGR toko menipis",
			Body:     "Beberapa toko punya pool NexusGGR lokal di bawah Rp 100.000.",
			Count:    summary.LowNexusggrTokos,
			Href:     "/backoffice/tokos",
		})
	}

	return alerts
}

func (s *Service) loadExternalAgentBalance(ctx context.Context) (*cachedAgentBalance, error) {
	const cacheKey = "dashboard:agent_balance:v2"

	if cached := s.loadCachedAgentBalance(ctx, cacheKey); cached != nil {
		return cached, nil
	}

	response, err := s.nexusggr.MoneyInfo(ctx, nil, false)
	if err != nil || response == nil || response.Status != 1 || response.Agent == nil {
		return nil, err
	}

	var code *string
	if value := strings.TrimSpace(stringValue(response.Agent["agent_code"])); value != "" {
		code = &value
	}

	result := &cachedAgentBalance{
		Balance: toInt64(response.Agent["balance"]),
		Code:    code,
	}
	s.storeCachedJSON(ctx, cacheKey, result)

	return result, nil
}

func presentSeries(rows []seriesRow, days int, referenceTime time.Time) ([]TransactionSeriesRow, []TransactionSeriesRow) {
	startDate := beginningOfDay(referenceTime).AddDate(0, 0, -(days - 1))
	qris := make([]TransactionSeriesRow, 0, days)
	nexus := make([]TransactionSeriesRow, 0, days)

	type seriesKey struct {
		Category string
		Type     string
		Date     string
	}

	index := make(map[seriesKey]int64, len(rows))
	for _, row := range rows {
		key := seriesKey{
			Category: row.Category,
			Type:     row.Type,
			Date:     row.Date.Format("2006-01-02"),
		}
		index[key] = row.Amount
	}

	for offset := 0; offset < days; offset++ {
		currentDate := startDate.AddDate(0, 0, offset)
		dateLabel := currentDate.Format("2006-01-02")

		qris = append(qris, TransactionSeriesRow{
			Date:       dateLabel,
			Deposit:    index[seriesKey{Category: "qris", Type: "deposit", Date: dateLabel}],
			Withdrawal: index[seriesKey{Category: "qris", Type: "withdrawal", Date: dateLabel}],
		})
		nexus = append(nexus, TransactionSeriesRow{
			Date:       dateLabel,
			Deposit:    index[seriesKey{Category: "nexusggr", Type: "deposit", Date: dateLabel}],
			Withdrawal: index[seriesKey{Category: "nexusggr", Type: "withdrawal", Date: dateLabel}],
		})
	}

	return qris, nexus
}

func (s *Service) loadCachedQRBalance(ctx context.Context, key string) *cachedQRBalance {
	var value cachedQRBalance
	if s.loadCachedJSON(ctx, key, &value) {
		return &value
	}
	return nil
}

func (s *Service) loadCachedAgentBalance(ctx context.Context, key string) *cachedAgentBalance {
	var value cachedAgentBalance
	if s.loadCachedJSON(ctx, key, &value) {
		return &value
	}
	return nil
}

func (s *Service) loadCachedJSON(ctx context.Context, key string, target any) bool {
	if s.cache == nil {
		return false
	}

	payload, err := s.cache.Get(ctx, key).Bytes()
	if err != nil {
		return false
	}

	return json.Unmarshal(payload, target) == nil
}

func (s *Service) storeCachedJSON(ctx context.Context, key string, value any) {
	if s.cache == nil {
		return
	}

	payload, err := json.Marshal(value)
	if err != nil {
		return
	}

	_ = s.cache.Set(ctx, key, payload, externalBalanceCacheTTL).Err()
}

func (s *Service) currentTime() time.Time {
	if s.location == nil {
		return s.now()
	}

	return s.now().In(s.location)
}

func scopeCondition(actor auth.PublicUser, parameterIndex int) (string, []any) {
	if actor.Role == "dev" || actor.Role == "superadmin" {
		return "", nil
	}

	return " AND t.user_id = $" + strconv.Itoa(parameterIndex), []any{actor.ID}
}

func beginningOfDay(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, value.Location())
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		return ""
	}
}

func toInt64(value any) int64 {
	switch typed := value.(type) {
	case nil:
		return 0
	case int64:
		return typed
	case int32:
		return int64(typed)
	case int:
		return int64(typed)
	case float64:
		return int64(math.Round(typed))
	case float32:
		return int64(math.Round(float64(typed)))
	case json.Number:
		if parsed, err := typed.Int64(); err == nil {
			return parsed
		}
		if parsed, err := typed.Float64(); err == nil {
			return int64(math.Round(parsed))
		}
	case string:
		if parsed, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64); err == nil {
			return parsed
		}
		if parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64); err == nil {
			return int64(math.Round(parsed))
		}
	}

	return 0
}
