package scheduler

import (
	"context"
	"testing"

	"github.com/rs/zerolog"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/config"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/modules/settlements"
)

type fakeSettler struct {
	calls int
}

func (f *fakeSettler) SettlePendingBalances(_ context.Context) (*settlements.Result, error) {
	f.calls++
	return &settlements.Result{ProcessedCount: 1, SettledTotal: 70}, nil
}

type fakeExpirer struct {
	calls int
}

func (f *fakeExpirer) ExpireOverduePendingQRIS(_ context.Context) (int64, error) {
	f.calls++
	return 1, nil
}

func TestSchedulerRegistersLegacySpecs(t *testing.T) {
	if settlePendingBalancesSpec != "0 16 * * 1-5" {
		t.Fatalf("unexpected settle spec: %s", settlePendingBalancesSpec)
	}
	if expireOverduePendingSpec != "@every 1m" {
		t.Fatalf("unexpected expire spec: %s", expireOverduePendingSpec)
	}
}

func TestSchedulerRegistersAndRunsBothJobs(t *testing.T) {
	settler := &fakeSettler{}
	expirer := &fakeExpirer{}

	instance, err := New(
		config.Config{App: config.AppConfig{Timezone: "Asia/Jakarta"}},
		zerolog.Nop(),
		settler,
		expirer,
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	entries := instance.Entries()
	if len(entries) != 2 {
		t.Fatalf("expected 2 scheduler entries, got %d", len(entries))
	}

	for _, entry := range entries {
		entry.Job.Run()
	}

	if settler.calls != 1 {
		t.Fatalf("expected settler to run once, got %d", settler.calls)
	}
	if expirer.calls != 1 {
		t.Fatalf("expected expirer to run once, got %d", expirer.calls)
	}
}
