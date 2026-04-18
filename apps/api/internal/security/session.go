package security

import (
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/alexedwards/scs/goredisstore"
	"github.com/alexedwards/scs/v2"
	"github.com/redis/go-redis/v9"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/config"
)

var nonSlugCharacters = regexp.MustCompile(`[^a-z0-9]+`)

func NewSessionManager(cfg config.Config, client *redis.Client) *scs.SessionManager {
	sessionManager := scs.New()
	sessionManager.Store = goredisstore.New(client)
	sessionManager.Lifetime = time.Duration(cfg.Session.LifetimeMinutes) * time.Minute
	sessionManager.Cookie.Name = cookiePrefix(cfg.App.Name) + "_session"
	sessionManager.Cookie.HttpOnly = true
	sessionManager.Cookie.Path = "/"
	sessionManager.Cookie.Persist = false
	sessionManager.Cookie.SameSite = SameSiteMode(cfg.Session.SameSite)
	sessionManager.Cookie.Secure = cfg.Session.SecureCookie
	sessionManager.Cookie.Domain = cfg.Session.Domain

	return sessionManager
}

func SameSiteMode(value string) http.SameSite {
	switch strings.ToLower(value) {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}

func CSRFCookieName(appName string) string {
	return cookiePrefix(appName) + "_csrf"
}

func cookiePrefix(appName string) string {
	slug := strings.ToLower(strings.TrimSpace(appName))
	slug = nonSlugCharacters.ReplaceAllString(slug, "_")
	slug = strings.Trim(slug, "_")
	if slug == "" {
		return "justqiu"
	}
	return slug
}
