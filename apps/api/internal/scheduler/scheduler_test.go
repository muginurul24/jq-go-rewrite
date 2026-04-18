package scheduler

import (
	"context"
	"testing"

	"github.com/rs/zerolog"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/config"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/modules/operationalfees"
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

type fakeOperationalFeeCollector struct {
	calls int
}

func (f *fakeOperationalFeeCollector) CollectMonthlyOperationalFees(_ context.Context) (*operationalfees.Result, error) {
	f.calls++
	return &operationalfees.Result{ProcessedCount: 2, DeductedTotal: 200_000}, nil
}

func TestSchedulerRegistersLegacySpecs(t *testing.T) {
	if settlePendingBalancesSpec != "0 16 * * 1-5" {
		t.Fatalf("unexpected settle spec: %s", settlePendingBalancesSpec)
	}
	if expireOverduePendingSpec != "@every 1m" {
		t.Fatalf("unexpected expire spec: %s", expireOverduePendingSpec)
	}
	if monthlyOperationalFeeSpec != "0 0 1 * *" {
		t.Fatalf("unexpected monthly operational fee spec: %s", monthlyOperationalFeeSpec)
	}
}

func TestSchedulerRegistersAndRunsAllJobs(t *testing.T) {
	settler := &fakeSettler{}
	expirer := &fakeExpirer{}
	collector := &fakeOperationalFeeCollector{}

	instance, err := New(
		config.Config{App: config.AppConfig{Timezone: "Asia/Jakarta"}},
		zerolog.Nop(),
		settler,
		expirer,
		collector,
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	entries := instance.Entries()
	if len(entries) != 3 {
		t.Fatalf("expected 3 scheduler entries, got %d", len(entries))
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
	if collector.calls != 1 {
		t.Fatalf("expected collector to run once, got %d", collector.calls)
	}
}
