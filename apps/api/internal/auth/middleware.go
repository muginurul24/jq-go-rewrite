package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/alexedwards/scs/v2"
)

func BackofficeSessionAuth(sessionManager *scs.SessionManager, service *Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := sessionManager.GetInt64(r.Context(), SessionUserIDKey)
			if userID == 0 {
				writeUnauthorized(w)
				return
			}

			user, err := service.FindUserByID(r.Context(), userID)
			if err != nil {
				_ = sessionManager.Destroy(r.Context())
				writeUnauthorized(w)
				return
			}

			next.ServeHTTP(w, r.WithContext(WithCurrentUser(r.Context(), *user)))
		})
	}
}

func TokoBearerAuth(service *Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := strings.TrimSpace(r.Header.Get("Authorization"))
			if !strings.HasPrefix(strings.ToLower(header), "bearer ") {
				writeUnauthorized(w)
				return
			}

			token := strings.TrimSpace(header[7:])
			if token == "" {
				writeUnauthorized(w)
				return
			}

			toko, err := service.AuthenticateTokoToken(r.Context(), token)
			if err != nil {
				if errors.Is(err, ErrForbidden) {
					writeForbidden(w)
					return
				}
				writeUnauthorized(w)
				return
			}

			next.ServeHTTP(w, r.WithContext(WithCurrentToko(r.Context(), *toko)))
		})
	}
}

func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"message": "Unauthorized",
	})
}

func writeForbidden(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusForbidden)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"message": "Forbidden",
	})
}
