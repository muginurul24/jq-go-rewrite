package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/auth"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/integrations/nexusggr"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/modules/players"
)

type BackofficePlayersHandler struct {
	service     backofficePlayerService
	nexusClient backofficePlayerMoneyInfoClient
}

type backofficePlayerService interface {
	ListForBackoffice(ctx context.Context, user auth.PublicUser, input players.AdminListInput) (*players.AdminListResult, error)
	MoneyInfoForBackoffice(ctx context.Context, user auth.PublicUser, playerID int64, client players.MoneyInfoClient) (*players.MoneyInfoResult, error)
}

type backofficePlayerMoneyInfoClient interface {
	MoneyInfo(ctx context.Context, userCode *string, allUsers bool) (*nexusggr.MoneyInfoResponse, error)
}

func NewBackofficePlayersHandler(service backofficePlayerService, nexusClient backofficePlayerMoneyInfoClient) *BackofficePlayersHandler {
	return &BackofficePlayersHandler{
		service:     service,
		nexusClient: nexusClient,
	}
}

func (h *BackofficePlayersHandler) List(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "Unauthorized"})
		return
	}

	input, err := parseBackofficePlayersListInput(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": err.Error()})
		return
	}

	result, err := h.service.ListForBackoffice(r.Context(), user, input)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "Failed to load players"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": presentPlayers(result.Data),
		"meta": map[string]any{
			"page":       result.Page,
			"perPage":    result.PerPage,
			"total":      result.Total,
			"totalPages": result.TotalPages,
		},
		"filters": map[string]any{
			"tokos": result.Tokos,
		},
	})
}

func (h *BackofficePlayersHandler) MoneyInfo(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "Unauthorized"})
		return
	}

	playerID, err := strconv.ParseInt(chi.URLParam(r, "playerID"), 10, 64)
	if err != nil || playerID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid player ID"})
		return
	}

	result, err := h.service.MoneyInfoForBackoffice(r.Context(), user, playerID, h.nexusClient)
	if err != nil {
		status := http.StatusInternalServerError
		message := "Failed to load player balance"
		switch {
		case errors.Is(err, players.ErrNotFound):
			status = http.StatusNotFound
			message = "Player not found"
		case errors.Is(err, players.ErrMoneyInfoUnavailable):
			status = http.StatusBadGateway
			message = "Balance player tidak tersedia dari upstream platform."
		}
		writeJSON(w, status, map[string]string{"message": message})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"playerId":    result.PlayerID,
			"username":    result.Username,
			"extUsername": result.ExtUsername,
			"tokoName":    result.TokoName,
			"balance":     result.Balance,
			"checkedAt":   result.CheckedAt,
		},
	})
}

func parseBackofficePlayersListInput(r *http.Request) (players.AdminListInput, error) {
	page, err := parsePlayersPositiveInt(r.URL.Query().Get("page"), 1)
	if err != nil {
		return players.AdminListInput{}, fmt.Errorf("invalid page")
	}

	perPage, err := parsePlayersPositiveInt(r.URL.Query().Get("per_page"), 25)
	if err != nil {
		return players.AdminListInput{}, fmt.Errorf("invalid per_page")
	}

	tokoID, err := parsePlayersInt64Optional(r.URL.Query().Get("toko_id"))
	if err != nil {
		return players.AdminListInput{}, fmt.Errorf("invalid toko_id")
	}

	return players.AdminListInput{
		Search:  r.URL.Query().Get("search"),
		TokoID:  tokoID,
		Page:    page,
		PerPage: perPage,
	}, nil
}

func presentPlayers(records []players.AdminPlayerRecord) []map[string]any {
	result := make([]map[string]any, 0, len(records))
	for _, record := range records {
		result = append(result, map[string]any{
			"id":            record.ID,
			"username":      record.Username,
			"extUsername":   record.ExtUsername,
			"tokoId":        record.TokoID,
			"tokoName":      record.TokoName,
			"ownerUsername": record.OwnerUsername,
			"createdAt":     record.CreatedAt,
			"updatedAt":     record.UpdatedAt,
		})
	}

	return result
}

func parsePlayersPositiveInt(raw string, fallback int) (int, error) {
	if strings.TrimSpace(raw) == "" {
		return fallback, nil
	}

	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("invalid positive int")
	}

	return value, nil
}

func parsePlayersInt64Optional(raw string) (int64, error) {
	if strings.TrimSpace(raw) == "" {
		return 0, nil
	}

	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("invalid int64")
	}

	return value, nil
}
