package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/config"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/modules/operationalfees"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/modules/settlements"
)

type balanceSettler interface {
	SettlePendingBalances(ctx context.Context) (*settlements.Result, error)
}

type pendingQRISExpirer interface {
	ExpireOverduePendingQRIS(ctx context.Context) (int64, error)
}

type monthlyOperationalFeeCollector interface {
	CollectMonthlyOperationalFees(ctx context.Context) (*operationalfees.Result, error)
}

const (
	settlePendingBalancesSpec = "0 16 * * 1-5"
	expireOverduePendingSpec  = "@every 1m"
	monthlyOperationalFeeSpec = "0 0 1 * *"
)

func New(cfg config.Config, logger zerolog.Logger, settler balanceSettler, expirer pendingQRISExpirer, collector monthlyOperationalFeeCollector) (*cron.Cron, error) {
	location, err := time.LoadLocation(cfg.App.Timezone)
	if err != nil {
		return nil, fmt.Errorf("load scheduler timezone: %w", err)
	}

	instance := cron.New(cron.WithLocation(location))

	_, err = instance.AddFunc(settlePendingBalancesSpec, func() {
		runLogger := logger.With().Str("job", "settle_pending_balances").Logger()
		if settler == nil {
			runLogger.Error().Msg("settler is not configured")
			return
		}

		result, err := settler.SettlePendingBalances(context.Background())
		if err != nil {
			runLogger.Error().Err(err).Msg("failed to settle pending balances")
			return
		}

		runLogger.Info().
			Int("processed_count", result.ProcessedCount).
			Int64("settled_total", result.SettledTotal).
			Msg("scheduler processed pending balances")
	})
	if err != nil {
		return nil, fmt.Errorf("register settle scheduler: %w", err)
	}

	_, err = instance.AddFunc(expireOverduePendingSpec, func() {
		runLogger := logger.With().Str("job", "expire_overdue_pending_qris").Logger()
		if expirer == nil {
			runLogger.Error().Msg("pending qris expirer is not configured")
			return
		}

		expiredCount, err := expirer.ExpireOverduePendingQRIS(context.Background())
		if err != nil {
			runLogger.Error().Err(err).Msg("failed to expire overdue pending qris transactions")
			return
		}

		if expiredCount == 0 {
			return
		}

		runLogger.Info().
			Int64("expired_count", expiredCount).
			Msg("scheduler expired overdue pending qris transactions")
	})
	if err != nil {
		return nil, fmt.Errorf("register overdue pending qris scheduler: %w", err)
	}

	_, err = instance.AddFunc(monthlyOperationalFeeSpec, func() {
		runLogger := logger.With().Str("job", "collect_monthly_operational_fees").Logger()
		if collector == nil {
			runLogger.Error().Msg("operational fee collector is not configured")
			return
		}

		result, err := collector.CollectMonthlyOperationalFees(context.Background())
		if err != nil {
			runLogger.Error().Err(err).Msg("failed to collect monthly operational fees")
			return
		}

		runLogger.Info().
			Int("processed_count", result.ProcessedCount).
			Int64("deducted_total", result.DeductedTotal).
			Msg("scheduler collected monthly operational fees")
	})
	if err != nil {
		return nil, fmt.Errorf("register monthly operational fee scheduler: %w", err)
	}

	return instance, nil
}
