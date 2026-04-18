package security

import (
	"reflect"
	"testing"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/config"
)

func TestTrustedOriginHostsNormalizesHostsAndAddsDevDefaults(t *testing.T) {
	cfg := config.Config{
		App: config.AppConfig{
			Env: "development",
			URL: "http://localhost:8080",
		},
		Session: config.SessionConfig{
			CSRFTrustedOrigins: []string{
				"https://admin.example.com",
				"dashboard.example.com",
				"localhost:5173",
			},
		},
	}

	got := trustedOriginHosts(cfg)
	want := []string{
		"localhost:8080",
		"admin.example.com",
		"dashboard.example.com",
		"localhost:5173",
		"127.0.0.1:5173",
		"localhost:4173",
		"127.0.0.1:4173",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected trusted origins\nwant: %#v\ngot:  %#v", want, got)
	}
}

func TestTrustedOriginHostsDoesNotInjectDevDefaultsInProduction(t *testing.T) {
	cfg := config.Config{
		App: config.AppConfig{
			Env: "production",
			URL: "https://api.example.com",
		},
		Session: config.SessionConfig{
			CSRFTrustedOrigins: []string{"https://admin.example.com"},
		},
	}

	got := trustedOriginHosts(cfg)
	want := []string{"api.example.com", "admin.example.com"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected trusted origins\nwant: %#v\ngot:  %#v", want, got)
	}
}
