package callmanagement

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/auth"
	nexusggrintegration "github.com/mugiew/justqiuv2-rewrite/apps/api/internal/integrations/nexusggr"
)

var (
	ErrPlayerNotFound         = errors.New("call management player not found")
	ErrNoAccessiblePlayers    = errors.New("call management no accessible players")
	ErrUpstreamCallFailed     = errors.New("call management upstream call failed")
	ErrCallListUnavailable    = errors.New("call management call list unavailable")
	ErrCallHistoryUnavailable = errors.New("call management call history unavailable")
	ErrCallCancelFailed       = errors.New("call management call cancel failed")
	ErrControlRTPFailed       = errors.New("call management control rtp failed")
)

const (
	defaultHistoryOffset = 0
	defaultHistoryLimit  = 100
	callTypeCommonFree   = 1
	callTypeBuyBonusFree = 2
)

type nexusClient interface {
	CallPlayers(ctx context.Context) (*nexusggrintegration.CallPlayersResponse, error)
	CallList(ctx context.Context, providerCode string, gameCode string) (*nexusggrintegration.CallListResponse, error)
	CallApply(ctx context.Context, providerCode string, gameCode string, userCode string, callRTP int, callType int) (*nexusggrintegration.CallApplyResponse, error)
	CallHistory(ctx context.Context, offset int, limit int) (*nexusggrintegration.CallHistoryResponse, error)
	CallCancel(ctx context.Context, callID int) (*nexusggrintegration.CallCancelResponse, error)
	ControlRtp(ctx context.Context, providerCode string, userCode string, rtp float64) (*nexusggrintegration.ControlRtpResponse, error)
	ControlUsersRtp(ctx context.Context, userCodes []string, rtp float64) (*nexusggrintegration.ControlRtpResponse, error)
}

type ManagedPlayerOption struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	UserLabel string `json:"userLabel"`
	TokoName  string `json:"tokoName"`
}

type ActivePlayerRecord struct {
	PlayerID     int64  `json:"playerId"`
	Username     string `json:"username"`
	UserLabel    string `json:"userLabel"`
	TokoName     string `json:"tokoName"`
	ProviderCode string `json:"providerCode"`
	GameCode     string `json:"gameCode"`
	Bet          any    `json:"bet"`
	Balance      any    `json:"balance"`
	TotalDebit   any    `json:"totalDebit"`
	TotalCredit  any    `json:"totalCredit"`
	TargetRTP    any    `json:"targetRtp"`
	RealRTP      any    `json:"realRtp"`
}

type CallOption struct {
	RTP           any    `json:"rtp"`
	CallType      string `json:"callType"`
	CallTypeValue int    `json:"callTypeValue"`
}

type CallHistoryRecord struct {
	ID           any    `json:"id"`
	PlayerID     int64  `json:"playerId"`
	Username     string `json:"username"`
	UserLabel    string `json:"userLabel"`
	TokoName     string `json:"tokoName"`
	ProviderCode any    `json:"providerCode"`
	GameCode     any    `json:"gameCode"`
	Bet          any    `json:"bet"`
	UserPrev     any    `json:"userPrev"`
	UserAfter    any    `json:"userAfter"`
	AgentPrev    any    `json:"agentPrev"`
	AgentAfter   any    `json:"agentAfter"`
	Expect       any    `json:"expect"`
	Missed       any    `json:"missed"`
	Real         any    `json:"real"`
	RTP          any    `json:"rtp"`
	Type         any    `json:"type"`
	Status       any    `json:"status"`
	StatusLabel  string `json:"statusLabel"`
	CanCancel    bool   `json:"canCancel"`
	CreatedAt    any    `json:"createdAt"`
	UpdatedAt    any    `json:"updatedAt"`
}

type BootstrapResult struct {
	ManagedPlayers []ManagedPlayerOption `json:"managedPlayers"`
}

type ApplyInput struct {
	PlayerID      int64
	ProviderCode  string
	GameCode      string
	CallRTP       int
	CallTypeValue int
}

type ControlRTPInput struct {
	PlayerID     int64
	ProviderCode string
	RTP          float64
}

type Service struct {
	db     *pgxpool.Pool
	client nexusClient
}

type playerRecord struct {
	ID            int64
	TokoID        int64
	Username      string
	ExtUsername   string
	TokoName      string
	OwnerUsername string
}

func NewService(db *pgxpool.Pool, client nexusClient) *Service {
	return &Service{
		db:     db,
		client: client,
	}
}

func (s *Service) Bootstrap(ctx context.Context, actor auth.PublicUser) (*BootstrapResult, error) {
	players, err := s.accessiblePlayers(ctx, actor)
	if err != nil {
		return nil, err
	}

	return &BootstrapResult{
		ManagedPlayers: presentManagedPlayers(players),
	}, nil
}

func (s *Service) ActivePlayers(ctx context.Context, actor auth.PublicUser) ([]ActivePlayerRecord, error) {
	players, err := s.accessiblePlayers(ctx, actor)
	if err != nil {
		return nil, err
	}

	response, err := s.client.CallPlayers(ctx)
	if err != nil || response == nil || response.Status != 1 {
		return nil, ErrUpstreamCallFailed
	}

	playersByExt := make(map[string]playerRecord, len(players))
	for _, player := range players {
		playersByExt[strings.ToLower(player.ExtUsername)] = player
	}

	records := make([]ActivePlayerRecord, 0, len(response.Data))
	for _, item := range response.Data {
		extUsername := strings.ToLower(stringValue(item["user_code"]))
		player, ok := playersByExt[extUsername]
		if !ok {
			continue
		}

		records = append(records, ActivePlayerRecord{
			PlayerID:     player.ID,
			Username:     player.Username,
			UserLabel:    playerLabel(player, players),
			TokoName:     player.TokoName,
			ProviderCode: stringValue(item["provider_code"]),
			GameCode:     stringValue(item["game_code"]),
			Bet:          item["bet"],
			Balance:      item["balance"],
			TotalDebit:   item["total_debit"],
			TotalCredit:  item["total_credit"],
			TargetRTP:    item["target_rtp"],
			RealRTP:      item["real_rtp"],
		})
	}

	sort.SliceStable(records, func(i, j int) bool {
		if records[i].UserLabel == records[j].UserLabel {
			return records[i].PlayerID < records[j].PlayerID
		}
		return records[i].UserLabel < records[j].UserLabel
	})

	return records, nil
}

func (s *Service) CallList(ctx context.Context, providerCode string, gameCode string) ([]CallOption, error) {
	response, err := s.client.CallList(ctx, strings.TrimSpace(providerCode), strings.TrimSpace(gameCode))
	if err != nil || response == nil || response.Status != 1 {
		return nil, ErrCallListUnavailable
	}

	result := make([]CallOption, 0, len(response.Calls))
	for _, item := range response.Calls {
		callType := stringValue(item["call_type"])
		result = append(result, CallOption{
			RTP:           item["rtp"],
			CallType:      callType,
			CallTypeValue: callTypeValue(callType),
		})
	}

	return result, nil
}

func (s *Service) Apply(ctx context.Context, actor auth.PublicUser, input ApplyInput) (any, error) {
	player, err := s.findAccessiblePlayer(ctx, actor, input.PlayerID)
	if err != nil {
		return nil, err
	}

	response, err := s.client.CallApply(
		ctx,
		strings.TrimSpace(input.ProviderCode),
		strings.TrimSpace(input.GameCode),
		player.ExtUsername,
		input.CallRTP,
		input.CallTypeValue,
	)
	if err != nil || response == nil || response.Status != 1 {
		return nil, ErrUpstreamCallFailed
	}

	calledMoney := intValue(response.CalledMoney)
	if err := s.decrementNexusBalance(ctx, player.TokoID, calledMoney); err != nil {
		return nil, err
	}

	return response.CalledMoney, nil
}

func (s *Service) History(ctx context.Context, actor auth.PublicUser, offset int, limit int) ([]CallHistoryRecord, error) {
	if offset < 0 {
		offset = defaultHistoryOffset
	}
	if limit <= 0 {
		limit = defaultHistoryLimit
	}

	players, err := s.accessiblePlayers(ctx, actor)
	if err != nil {
		return nil, err
	}

	response, err := s.client.CallHistory(ctx, offset, limit)
	if err != nil || response == nil || response.Status != 1 {
		return nil, ErrCallHistoryUnavailable
	}

	playersByExt := make(map[string]playerRecord, len(players))
	for _, player := range players {
		playersByExt[strings.ToLower(player.ExtUsername)] = player
	}

	records := make([]CallHistoryRecord, 0, len(response.Data))
	for _, item := range response.Data {
		extUsername := strings.ToLower(stringValue(item["user_code"]))
		player, ok := playersByExt[extUsername]
		if !ok {
			continue
		}

		statusNumber := intValue(item["status"])
		records = append(records, CallHistoryRecord{
			ID:           item["id"],
			PlayerID:     player.ID,
			Username:     player.Username,
			UserLabel:    playerLabel(player, players),
			TokoName:     player.TokoName,
			ProviderCode: item["provider_code"],
			GameCode:     item["game_code"],
			Bet:          item["bet"],
			UserPrev:     item["user_prev"],
			UserAfter:    item["user_after"],
			AgentPrev:    item["agent_prev"],
			AgentAfter:   item["agent_after"],
			Expect:       item["expect"],
			Missed:       item["missed"],
			Real:         item["real"],
			RTP:          item["rtp"],
			Type:         item["type"],
			Status:       item["status"],
			StatusLabel:  callStatusLabel(statusNumber),
			CanCancel:    statusNumber == 0,
			CreatedAt:    item["created_at"],
			UpdatedAt:    item["updated_at"],
		})
	}

	sort.SliceStable(records, func(i, j int) bool {
		return timeString(records[i].CreatedAt) > timeString(records[j].CreatedAt)
	})

	return records, nil
}

func (s *Service) Cancel(ctx context.Context, callID int) (any, error) {
	response, err := s.client.CallCancel(ctx, callID)
	if err != nil || response == nil || response.Status != 1 {
		return nil, ErrCallCancelFailed
	}

	return response.CanceledMoney, nil
}

func (s *Service) ControlRTP(ctx context.Context, actor auth.PublicUser, input ControlRTPInput) (any, error) {
	player, err := s.findAccessiblePlayer(ctx, actor, input.PlayerID)
	if err != nil {
		return nil, err
	}

	response, err := s.client.ControlRtp(ctx, strings.TrimSpace(input.ProviderCode), player.ExtUsername, input.RTP)
	if err != nil || response == nil || response.Status != 1 {
		return nil, ErrControlRTPFailed
	}

	return response.ChangedRTP, nil
}

func (s *Service) ControlUsersRTP(ctx context.Context, actor auth.PublicUser, rtp float64) (any, error) {
	players, err := s.accessiblePlayers(ctx, actor)
	if err != nil {
		return nil, err
	}
	if len(players) == 0 {
		return nil, ErrNoAccessiblePlayers
	}

	extUsernames := make([]string, 0, len(players))
	seen := make(map[string]struct{}, len(players))
	for _, player := range players {
		if _, exists := seen[player.ExtUsername]; exists {
			continue
		}
		seen[player.ExtUsername] = struct{}{}
		extUsernames = append(extUsernames, player.ExtUsername)
	}

	response, err := s.client.ControlUsersRtp(ctx, extUsernames, rtp)
	if err != nil || response == nil || response.Status != 1 {
		return nil, ErrControlRTPFailed
	}

	return response.ChangedRTP, nil
}

func (s *Service) accessiblePlayers(ctx context.Context, actor auth.PublicUser) ([]playerRecord, error) {
	query := `
		SELECT
			p.id,
			p.toko_id,
			p.username,
			p.ext_username,
			t.name,
			u.username
		FROM players p
		INNER JOIN tokos t ON t.id = p.toko_id
		INNER JOIN users u ON u.id = t.user_id
		WHERE p.deleted_at IS NULL
			AND t.deleted_at IS NULL
	`
	args := []any{}

	if !isGlobalRole(actor.Role) {
		query += " AND t.user_id = $1"
		args = append(args, actor.ID)
	}

	query += " ORDER BY p.username ASC, p.id ASC"

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list accessible players for call management: %w", err)
	}
	defer rows.Close()

	result := make([]playerRecord, 0, 32)
	for rows.Next() {
		var item playerRecord
		if err := rows.Scan(&item.ID, &item.TokoID, &item.Username, &item.ExtUsername, &item.TokoName, &item.OwnerUsername); err != nil {
			return nil, fmt.Errorf("scan call management player: %w", err)
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate call management players: %w", err)
	}

	return result, nil
}

func (s *Service) findAccessiblePlayer(ctx context.Context, actor auth.PublicUser, playerID int64) (*playerRecord, error) {
	players, err := s.accessiblePlayers(ctx, actor)
	if err != nil {
		return nil, err
	}

	for _, player := range players {
		if player.ID == playerID {
			item := player
			return &item, nil
		}
	}

	return nil, ErrPlayerNotFound
}

func (s *Service) decrementNexusBalance(ctx context.Context, tokoID int64, amount int64) error {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin call apply tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var balanceID int64
	if err := tx.QueryRow(ctx, `
		SELECT id
		FROM balances
		WHERE toko_id = $1
		ORDER BY id
		LIMIT 1
		FOR UPDATE
	`, tokoID).Scan(&balanceID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			if err := tx.QueryRow(ctx, `
				INSERT INTO balances (toko_id, settle, pending, nexusggr, created_at, updated_at)
				VALUES ($1, 0, 0, 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
				RETURNING id
			`, tokoID).Scan(&balanceID); err != nil {
				return fmt.Errorf("create missing balance for call apply: %w", err)
			}
		} else {
			return fmt.Errorf("lock balance for call apply: %w", err)
		}
	}

	if _, err := tx.Exec(ctx, `
		UPDATE balances
		SET nexusggr = nexusggr - $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`, balanceID, amount); err != nil {
		return fmt.Errorf("decrement nexusggr balance for call apply: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit call apply tx: %w", err)
	}

	return nil
}

func presentManagedPlayers(players []playerRecord) []ManagedPlayerOption {
	result := make([]ManagedPlayerOption, 0, len(players))
	for _, player := range players {
		result = append(result, ManagedPlayerOption{
			ID:        player.ID,
			Username:  player.Username,
			UserLabel: playerLabel(player, players),
			TokoName:  player.TokoName,
		})
	}
	return result
}

func playerLabel(target playerRecord, players []playerRecord) string {
	duplicates := 0
	for _, item := range players {
		if item.Username == target.Username {
			duplicates++
			if duplicates > 1 {
				return target.Username + " (" + target.TokoName + ")"
			}
		}
	}

	return target.Username
}

func callTypeValue(value string) int {
	lowered := strings.ToLower(strings.TrimSpace(value))
	if strings.Contains(lowered, "bonus") {
		return callTypeBuyBonusFree
	}
	return callTypeCommonFree
}

func callStatusLabel(status int64) string {
	switch status {
	case 0:
		return "Waiting"
	case 1:
		return "Processing"
	case 2:
		return "Finished"
	case 3:
		return "Rejected by Game Server"
	case 4:
		return "Canceled"
	default:
		return "Unknown"
	}
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case []byte:
		return strings.TrimSpace(string(typed))
	default:
		return ""
	}
}

func intValue(value any) int64 {
	switch typed := value.(type) {
	case int:
		return int64(typed)
	case int32:
		return int64(typed)
	case int64:
		return typed
	case float64:
		return int64(typed)
	default:
		return 0
	}
}

func timeString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return ""
	}
}

func isGlobalRole(role string) bool {
	return role == "dev" || role == "superadmin"
}
