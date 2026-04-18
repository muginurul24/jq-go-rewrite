package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/config"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/seed"
)

func main() {
	var migrationsDir string
	var resetAndSeed bool
	var seedProfile string
	flag.StringVar(&migrationsDir, "dir", "./migrations", "path to goose migrations directory")
	flag.BoolVar(&resetAndSeed, "seed", false, "reset the database, rerun migrations, then seed the legacy baseline income row and default dev operator")
	flag.StringVar(&seedProfile, "seed-profile", "development", "seed profile to use when running --seed or seed command (development|demo)")
	flag.Parse()

	command := "up"
	if flag.NArg() > 0 {
		command = flag.Arg(0)
	}

	cfg, err := config.Load()
	if err != nil {
		exitf("load config: %v", err)
	}

	db, err := sql.Open("postgres", cfg.Database.DSN())
	if err != nil {
		exitf("open database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		exitf("ping database: %v", err)
	}

	if err := goose.SetDialect("postgres"); err != nil {
		exitf("set goose dialect: %v", err)
	}

	resolvedDir, err := filepath.Abs(migrationsDir)
	if err != nil {
		exitf("resolve migrations directory: %v", err)
	}

	if needsGooseVersionTable(command, resetAndSeed) {
		if _, err := goose.EnsureDBVersion(db); err != nil {
			exitf("ensure goose version table failed: %v", err)
		}
	}

	if resetAndSeed || command == "seed" {
		if strings.EqualFold(cfg.App.Env, "production") && !strings.EqualFold(os.Getenv("ALLOW_DESTRUCTIVE_SEED"), "true") {
			exitf("refusing to run destructive seed in production without ALLOW_DESTRUCTIVE_SEED=true")
		}

		if err := goose.Reset(db, resolvedDir); err != nil {
			exitf("goose reset failed: %v", err)
		}

		if err := goose.Up(db, resolvedDir); err != nil {
			exitf("goose up after reset failed: %v", err)
		}

		switch strings.ToLower(strings.TrimSpace(seedProfile)) {
		case "development", "dev", "":
			result, err := seed.Development(context.Background(), db)
			if err != nil {
				exitf("seed development data failed: %v", err)
			}

			fmt.Printf(
				"database reset + seed complete\nseed_profile=development\nuser_id=%d\nusername=%s\npassword=%s\nemail=%s\nincome.ggr=%d\nincome.fee_transaction=%d\nincome.fee_withdrawal=%d\n",
				result.UserID,
				result.Username,
				result.Password,
				result.Email,
				result.IncomeGGR,
				result.IncomeFeeTx,
				result.IncomeFeeWithdraw,
			)
		case "demo":
			result, err := seed.Demo(context.Background(), db)
			if err != nil {
				exitf("seed demo data failed: %v", err)
			}

			fmt.Printf(
				"database reset + seed complete\nseed_profile=demo\ndev.user_id=%d\ndev.username=%s\ndev.password=%s\ndev.email=%s\nowner.user_id=%d\nowner.username=%s\nowner.password=%s\nowner.email=%s\nmfa.user_id=%d\nmfa.username=%s\nmfa.password=%s\nmfa.email=%s\ntoko.id=%d\ntoko.name=%s\ntoko.token=%s\nplayer.id=%d\nplayer.username=%s\nplayer.external=%s\nincome.amount=%d\nincome.ggr=%d\nincome.fee_transaction=%d\nincome.fee_withdrawal=%d\n",
				result.DevUserID,
				result.DevUsername,
				result.DevPassword,
				result.DevEmail,
				result.OwnerUserID,
				result.OwnerUsername,
				result.OwnerPassword,
				result.OwnerEmail,
				result.MFAUserID,
				result.MFAUsername,
				result.MFAPassword,
				result.MFAEmail,
				result.TokoID,
				result.TokoName,
				result.TokoToken,
				result.PlayerID,
				result.PlayerUsername,
				result.PlayerExternal,
				result.IncomeAmount,
				result.IncomeGGR,
				result.IncomeFeeTx,
				result.IncomeFeeWithdraw,
			)
		default:
			exitf("unsupported seed profile %q (supported: development, demo)", seedProfile)
		}
		return
	}

	switch command {
	case "up":
		err = goose.Up(db, resolvedDir)
	case "down":
		err = goose.Down(db, resolvedDir)
	case "baseline":
		err = baselineDatabase(db, resolvedDir)
	case "redo":
		err = goose.Redo(db, resolvedDir)
	case "reset":
		err = goose.Reset(db, resolvedDir)
	case "status":
		err = goose.Status(db, resolvedDir)
	case "version":
		var version int64
		version, err = goose.GetDBVersion(db)
		if err == nil {
			fmt.Println(version)
		}
	default:
		exitf("unsupported migrate command %q (supported: up, down, baseline, redo, reset, status, version, seed, or --seed)", command)
	}

	if err != nil {
		exitf("goose %s failed: %v", command, err)
	}
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func baselineDatabase(db *sql.DB, migrationsDir string) error {
	currentVersion, err := goose.EnsureDBVersion(db)
	if err != nil {
		return fmt.Errorf("ensure db version: %w", err)
	}

	targetVersion, err := latestMigrationVersion(migrationsDir)
	if err != nil {
		return err
	}

	if currentVersion == targetVersion {
		fmt.Printf("database already baselined\nversion=%d\n", targetVersion)
		return nil
	}

	if currentVersion > 0 && currentVersion != targetVersion {
		return fmt.Errorf("database already has goose version %d; refusing to baseline to %d", currentVersion, targetVersion)
	}

	if _, err := db.Exec(
		fmt.Sprintf("INSERT INTO %s (version_id, is_applied) VALUES ($1, $2)", goose.DefaultTablename),
		targetVersion,
		true,
	); err != nil {
		return fmt.Errorf("insert baseline version: %w", err)
	}

	fmt.Printf("database baseline complete\nversion=%d\n", targetVersion)
	return nil
}

func latestMigrationVersion(migrationsDir string) (int64, error) {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return 0, fmt.Errorf("read migrations directory: %w", err)
	}

	var latest int64
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		version, err := goose.NumericComponent(entry.Name())
		if err != nil {
			continue
		}
		if version > latest {
			latest = version
		}
	}

	if latest == 0 {
		return 0, fmt.Errorf("no valid migrations found in %s", migrationsDir)
	}

	return latest, nil
}

func needsGooseVersionTable(command string, resetAndSeed bool) bool {
	if resetAndSeed {
		return true
	}

	switch command {
	case "seed", "down", "redo", "reset", "status":
		return true
	default:
		return false
	}
}
