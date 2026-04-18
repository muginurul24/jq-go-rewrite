package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/app"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/modules/settlements"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/modules/transactions"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/scheduler"
	"github.com/rs/zerolog"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	bootLogger := zerolog.New(os.Stderr).With().Timestamp().Logger()

	runtime, err := app.New(ctx)
	if err != nil {
		bootLogger.Fatal().Err(err).Msg("bootstrap failed")
	}

	settlementService := settlements.NewService(runtime.DB, runtime.Logger)
	transactionService := transactions.NewService(runtime.DB)

	instance, err := scheduler.New(runtime.Config, runtime.Logger, settlementService, transactionService)
	if err != nil {
		runtime.Logger.Fatal().Err(err).Msg("scheduler bootstrap failed")
	}

	instance.Start()
	runtime.Logger.Info().
		Str("timezone", runtime.Config.App.Timezone).
		Msg("scheduler started")

	<-ctx.Done()

	stopCtx := instance.Stop()
	select {
	case <-stopCtx.Done():
	case <-time.After(runtime.Config.HTTP.ShutdownTimeout):
	}

	closeCtx, cancel := context.WithTimeout(context.Background(), runtime.Config.HTTP.ShutdownTimeout)
	defer cancel()

	if err := runtime.Close(closeCtx); err != nil {
		runtime.Logger.Error().Err(err).Msg("runtime close failed")
	}
}
