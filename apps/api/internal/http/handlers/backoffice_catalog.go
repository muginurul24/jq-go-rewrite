package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/modules/catalog"
)

type BackofficeCatalogHandler struct {
	service *catalog.Service
}

func NewBackofficeCatalogHandler(service *catalog.Service) *BackofficeCatalogHandler {
	return &BackofficeCatalogHandler{service: service}
}

func (h *BackofficeCatalogHandler) Providers(w http.ResponseWriter, r *http.Request) {
	response, err := h.service.ProviderList(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"message": "Failed to load providers",
		})
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *BackofficeCatalogHandler) Games(w http.ResponseWriter, r *http.Request) {
	providerCode := strings.TrimSpace(r.URL.Query().Get("provider_code"))
	if providerCode == "" {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{
			"message": "provider_code is required",
		})
		return
	}

	localized, _ := strconv.ParseBool(strings.TrimSpace(r.URL.Query().Get("localized")))
	if localized {
		response, err := h.service.GameListV2(r.Context(), providerCode)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"message": "Failed to load localized games",
			})
			return
		}

		writeJSON(w, http.StatusOK, response)
		return
	}

	response, err := h.service.GameList(r.Context(), providerCode)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"message": "Failed to load games",
		})
		return
	}

	writeJSON(w, http.StatusOK, response)
}
