package security

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/csrf"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/config"
)

func NewCSRFMiddleware(cfg config.Config) func(http.Handler) http.Handler {
	options := []csrf.Option{
		csrf.CookieName(CSRFCookieName(cfg.App.Name)),
		csrf.Path("/"),
		csrf.HttpOnly(true),
		csrf.Secure(cfg.Session.SecureCookie),
		csrf.SameSite(csrf.SameSiteMode(SameSiteMode(cfg.Session.SameSite))),
		csrf.RequestHeader("X-CSRF-Token"),
		csrf.ErrorHandler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"message":"CSRF token mismatch"}`))
		})),
	}

	if trustedOrigins := trustedOriginHosts(cfg); len(trustedOrigins) > 0 {
		options = append(options, csrf.TrustedOrigins(trustedOrigins))
	}

	return csrf.Protect([]byte(cfg.Session.CSRFSecret), options...)
}

func trustedOriginHosts(cfg config.Config) []string {
	seen := map[string]struct{}{}
	origins := make([]string, 0, 6)

	add := func(value string) {
		host := normalizeTrustedOrigin(value)
		if host == "" {
			return
		}

		if _, exists := seen[host]; exists {
			return
		}

		seen[host] = struct{}{}
		origins = append(origins, host)
	}

	add(cfg.App.URL)
	for _, value := range cfg.Session.CSRFTrustedOrigins {
		add(value)
	}

	if strings.ToLower(cfg.App.Env) != "production" {
		add("localhost:5173")
		add("127.0.0.1:5173")
		add("localhost:4173")
		add("127.0.0.1:4173")
	}

	return origins
}

func normalizeTrustedOrigin(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}

	if strings.Contains(trimmed, "://") {
		parsed, err := url.Parse(trimmed)
		if err != nil {
			return ""
		}
		return parsed.Host
	}

	return trimmed
}
