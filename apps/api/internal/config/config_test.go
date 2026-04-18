package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindDotEnvFileFromFindsParentDotEnv(t *testing.T) {
	rootDir := t.TempDir()
	appDir := filepath.Join(rootDir, "apps", "api")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatalf("mkdir app dir: %v", err)
	}

	envPath := filepath.Join(rootDir, ".env")
	if err := os.WriteFile(envPath, []byte("APP_ENV=test\n"), 0o644); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	foundPath, found := findDotEnvFileFrom(appDir)
	if !found {
		t.Fatal("expected .env to be found in parent directories")
	}

	if foundPath != envPath {
		t.Fatalf("expected env path %q, got %q", envPath, foundPath)
	}
}

func TestFindDotEnvFileFromReturnsFalseWhenMissing(t *testing.T) {
	rootDir := t.TempDir()
	appDir := filepath.Join(rootDir, "apps", "api")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatalf("mkdir app dir: %v", err)
	}

	foundPath, found := findDotEnvFileFrom(appDir)
	if found {
		t.Fatalf("expected no .env file, got %q", foundPath)
	}
}
