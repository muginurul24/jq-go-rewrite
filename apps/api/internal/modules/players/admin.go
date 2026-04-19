package players

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/auth"
	nexusggrintegration "github.com/mugiew/justqiuv2-rewrite/apps/api/internal/integrations/nexusggr"
)

var ErrMoneyInfoUnavailable = errors.New("money info unavailable")

type MoneyInfoClient interface {
	MoneyInfo(ctx context.Context, userCode *string, allUsers bool) (*nexusggrintegration.MoneyInfoResponse, error)
}

type AdminListInput struct {
	Search  string
	TokoID  int64
	Page    int
	PerPage int
}

type AdminTokoOption struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type AdminPlayerRecord struct {
	ID            int64
	Username      string
	ExtUsername   string
	TokoID        int64
	TokoName      string
	OwnerUsername string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type AdminListResult struct {
	Data       []AdminPlayerRecord
	Page       int
	PerPage    int
	Total      int64
	TotalPages int
	Tokos      []AdminTokoOption
}

type MoneyInfoResult struct {
	PlayerID    int64
	Username    string
	ExtUsername string
	TokoName    string
	Balance     int64
	CheckedAt   time.Time
}

func (s *Service) ListForBackoffice(ctx context.Context, user auth.PublicUser, input AdminListInput) (*AdminListResult, error) {
	page := input.Page
	if page < 1 {
		page = 1
	}

	perPage := input.PerPage
	switch perPage {
	case 10, 25, 50, 100:
	default:
		perPage = 25
	}

	whereClause, args := buildPlayerAdminWhereClause(user, input)

	countQuery := `
		SELECT COUNT(*)
		FROM players p
		INNER JOIN tokos t ON t.id = p.toko_id
		INNER JOIN users u ON u.id = t.user_id
		WHERE p.deleted_at IS NULL
			AND t.deleted_at IS NULL
	` + whereClause

	var total int64
	if err := s.repository.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count players: %w", err)
	}

	listArgs := append([]any{}, args...)
	listArgs = append(listArgs, perPage, (page-1)*perPage)

	listQuery := `
		SELECT
			p.id,
			p.username,
			p.ext_username,
			t.id,
			t.name,
			u.username,
			p.created_at,
			p.updated_at
		FROM players p
		INNER JOIN tokos t ON t.id = p.toko_id
		INNER JOIN users u ON u.id = t.user_id
		WHERE p.deleted_at IS NULL
			AND t.deleted_at IS NULL
	` + whereClause + `
		ORDER BY p.created_at DESC, p.id DESC
		LIMIT $` + fmt.Sprintf("%d", len(args)+1) + `
		OFFSET $` + fmt.Sprintf("%d", len(args)+2)

	rows, err := s.repository.db.Query(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, fmt.Errorf("list players: %w", err)
	}
	defer rows.Close()

	records := make([]AdminPlayerRecord, 0, perPage)
	for rows.Next() {
		var record AdminPlayerRecord
		if err := rows.Scan(
			&record.ID,
			&record.Username,
			&record.ExtUsername,
			&record.TokoID,
			&record.TokoName,
			&record.OwnerUsername,
			&record.CreatedAt,
			&record.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan player row: %w", err)
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate players: %w", err)
	}

	tokos, err := s.accessibleTokos(ctx, user)
	if err != nil {
		return nil, err
	}

	totalPages := 0
	if total > 0 {
		totalPages = int((total + int64(perPage) - 1) / int64(perPage))
	}

	return &AdminListResult{
		Data:       records,
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: totalPages,
		Tokos:      tokos,
	}, nil
}

func (s *Service) MoneyInfoForBackoffice(ctx context.Context, user auth.PublicUser, playerID int64, client MoneyInfoClient) (*MoneyInfoResult, error) {
	player, err := s.findVisibleByID(ctx, user, playerID)
	if err != nil {
		return nil, err
	}

	response, err := client.MoneyInfo(ctx, &player.ExtUsername, false)
	if err != nil || response.Status != 1 || response.User == nil {
		return nil, ErrMoneyInfoUnavailable
	}

	return &MoneyInfoResult{
		PlayerID:    player.ID,
		Username:    player.Username,
		ExtUsername: player.ExtUsername,
		TokoName:    player.TokoName,
		Balance:     extractBalance(response.User),
		CheckedAt:   time.Now(),
	}, nil
}

type adminPlayerDetail struct {
	ID          int64
	Username    string
	ExtUsername string
	TokoName    string
}

func (s *Service) findVisibleByID(ctx context.Context, user auth.PublicUser, playerID int64) (*adminPlayerDetail, error) {
	where := ""
	args := []any{playerID}

	if !isGlobalRole(user.Role) {
		args = append(args, user.ID)
		where = " AND t.user_id = $2"
	}

	query := `
		SELECT
			p.id,
			p.username,
			p.ext_username,
			t.name
		FROM players p
		INNER JOIN tokos t ON t.id = p.toko_id
		WHERE p.id = $1
			AND p.deleted_at IS NULL
			AND t.deleted_at IS NULL
	` + where + `
		LIMIT 1
	`

	var detail adminPlayerDetail
	err := s.repository.db.QueryRow(ctx, query, args...).Scan(
		&detail.ID,
		&detail.Username,
		&detail.ExtUsername,
		&detail.TokoName,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find visible player: %w", err)
	}

	return &detail, nil
}

func (s *Service) accessibleTokos(ctx context.Context, user auth.PublicUser) ([]AdminTokoOption, error) {
	query := `
		SELECT t.id, t.name
		FROM tokos t
		WHERE t.deleted_at IS NULL
	`
	args := []any{}
	if !isGlobalRole(user.Role) {
		query += " AND t.user_id = $1"
		args = append(args, user.ID)
	}
	query += " ORDER BY t.name ASC"

	rows, err := s.repository.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list accessible tokos: %w", err)
	}
	defer rows.Close()

	options := make([]AdminTokoOption, 0, 8)
	for rows.Next() {
		var option AdminTokoOption
		if err := rows.Scan(&option.ID, &option.Name); err != nil {
			return nil, fmt.Errorf("scan accessible toko: %w", err)
		}
		options = append(options, option)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate accessible tokos: %w", err)
	}

	return options, nil
}

func buildPlayerAdminWhereClause(user auth.PublicUser, input AdminListInput) (string, []any) {
	parts := make([]string, 0, 3)
	args := make([]any, 0, 3)

	if search := strings.TrimSpace(input.Search); search != "" {
		args = append(args, "%"+search+"%")
		index := len(args)
		parts = append(parts, fmt.Sprintf(
			"AND (p.username ILIKE $%[1]d OR p.ext_username ILIKE $%[1]d OR t.name ILIKE $%[1]d OR u.username ILIKE $%[1]d)",
			index,
		))
	}

	if input.TokoID > 0 {
		args = append(args, input.TokoID)
		parts = append(parts, fmt.Sprintf("AND p.toko_id = $%d", len(args)))
	}

	if !isGlobalRole(user.Role) {
		args = append(args, user.ID)
		parts = append(parts, fmt.Sprintf("AND t.user_id = $%d", len(args)))
	}

	return "\n" + strings.Join(parts, "\n"), args
}

func extractBalance(userPayload map[string]any) int64 {
	if rawBalance, ok := userPayload["balance"]; ok {
		switch typed := rawBalance.(type) {
		case int64:
			return typed
		case int32:
			return int64(typed)
		case int:
			return int64(typed)
		case float32:
			return int64(typed)
		case float64:
			return int64(typed)
		case string:
			parsedFloat, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
			if err == nil {
				return int64(parsedFloat)
			}
			parsed, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
			if err == nil {
				return parsed
			}
		}
	}

	return 0
}

func isGlobalRole(role string) bool {
	return role == "dev" || role == "superadmin"
}
