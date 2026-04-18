package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/auth"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/modules/balances"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/modules/catalog"
)

type TokoAPIHandler struct {
	balanceService balanceLookupService
	catalogService catalogLookupService
}

type merchantActiveRequest struct {
	Label  string  `json:"label"`
	Client *string `json:"client"`
}

type providerCodeRequest struct {
	ProviderCode string `json:"provider_code"`
}

type balanceLookupService interface {
	GetOrCreateForToko(ctx context.Context, tokoID int64) (*balances.Balance, error)
}

type catalogLookupService interface {
	ProviderList(ctx context.Context) (*catalog.ProviderListResponse, error)
	GameList(ctx context.Context, providerCode string) (*catalog.GameListResponse, error)
	GameListV2(ctx context.Context, providerCode string) (*catalog.GameListV2Response, error)
}

func NewTokoAPIHandler(balanceService balanceLookupService, catalogService catalogLookupService) *TokoAPIHandler {
	return &TokoAPIHandler{
		balanceService: balanceService,
		catalogService: catalogService,
	}
}

func (h *TokoAPIHandler) ProviderList(w http.ResponseWriter, r *http.Request) {
	response, err := h.catalogService.ProviderList(r.Context())
	if err != nil {
		writeCatalogError(w, err, "Failed to get provider list from upstream platform")
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *TokoAPIHandler) GameList(w http.ResponseWriter, r *http.Request) {
	providerCode, ok := decodeProviderCode(w, r)
	if !ok {
		return
	}

	response, err := h.catalogService.GameList(r.Context(), providerCode)
	if err != nil {
		writeCatalogError(w, err, "Failed to get game list from upstream platform")
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *TokoAPIHandler) GameListV2(w http.ResponseWriter, r *http.Request) {
	providerCode, ok := decodeProviderCode(w, r)
	if !ok {
		return
	}

	response, err := h.catalogService.GameListV2(r.Context(), providerCode)
	if err != nil {
		writeCatalogError(w, err, "Failed to get localized game list from upstream platform")
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *TokoAPIHandler) MerchantActive(w http.ResponseWriter, r *http.Request) {
	var request merchantActiveRequest
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, authErrorResponse{Message: "Invalid JSON payload"})
		return
	}

	errorsByField := map[string]string{}
	label := strings.TrimSpace(request.Label)
	if label == "" {
		errorsByField["label"] = "Label is required."
	} else if len(label) > 255 {
		errorsByField["label"] = "Label must not exceed 255 characters."
	}

	if request.Client != nil && len(strings.TrimSpace(*request.Client)) > 50 {
		errorsByField["client"] = "Client must not exceed 50 characters."
	}

	if len(errorsByField) > 0 {
		writeJSON(w, http.StatusUnprocessableEntity, authErrorResponse{
			Message: "Validation failed",
			Errors:  errorsByField,
		})
		return
	}

	toko, ok := auth.CurrentToko(r.Context())
	if !ok {
		writeJSON(w, http.StatusForbidden, authErrorResponse{Message: "Forbidden"})
		return
	}

	balance, err := h.balanceService.GetOrCreateForToko(r.Context(), toko.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, authErrorResponse{Message: "Failed to load balance"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"store": map[string]any{
			"name":         toko.Name,
			"callback_url": toko.CallbackURL,
			"token":        toko.Token,
		},
		"balance": map[string]any{
			"nexusggr": balance.NexusGGR,
			"pending":  balance.Pending,
			"settle":   balance.Settle,
		},
	})
}

func (h *TokoAPIHandler) Balance(w http.ResponseWriter, r *http.Request) {
	toko, ok := auth.CurrentToko(r.Context())
	if !ok {
		writeJSON(w, http.StatusForbidden, authErrorResponse{Message: "Forbidden"})
		return
	}

	balance, err := h.balanceService.GetOrCreateForToko(r.Context(), toko.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, authErrorResponse{Message: "Failed to load balance"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success":          true,
		"pending_balance":  balance.Pending,
		"settle_balance":   balance.Settle,
		"nexusggr_balance": balance.NexusGGR,
	})
}

func decodeProviderCode(w http.ResponseWriter, r *http.Request) (string, bool) {
	var request providerCodeRequest
	if err := decodeJSON(r, &request); err != nil {
		writeJSON(w, http.StatusBadRequest, authErrorResponse{Message: "Invalid JSON payload"})
		return "", false
	}

	providerCode := strings.TrimSpace(request.ProviderCode)
	switch {
	case providerCode == "":
		writeJSON(w, http.StatusUnprocessableEntity, authErrorResponse{
			Message: "Validation failed",
			Errors: map[string]string{
				"provider_code": "Provider code is required.",
			},
		})
		return "", false
	case len(providerCode) > 50:
		writeJSON(w, http.StatusUnprocessableEntity, authErrorResponse{
			Message: "Validation failed",
			Errors: map[string]string{
				"provider_code": "Provider code must not exceed 50 characters.",
			},
		})
		return "", false
	}

	return providerCode, true
}

func writeCatalogError(w http.ResponseWriter, err error, fallbackMessage string) {
	statusCode := http.StatusInternalServerError
	response := authErrorResponse{Message: fallbackMessage}

	if errors.Is(err, catalog.ErrUpstreamFailure) {
		writeJSON(w, statusCode, response)
		return
	}

	writeJSON(w, statusCode, response)
}
