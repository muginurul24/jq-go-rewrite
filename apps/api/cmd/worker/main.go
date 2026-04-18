package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/app"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/queue"
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

	server, mux := queue.NewServer(queue.Dependencies{
		Config: runtime.Config,
		Logger: runtime.Logger,
		DB:     runtime.DB,
		Queue:  runtime.Queue,
	})

	if err := server.Start(mux); err != nil {
		runtime.Logger.Fatal().Err(err).Msg("worker failed to start")
	}

	runtime.Logger.Info().
		Int("concurrency", runtime.Config.Queue.WorkerConcurrency).
		Msg("worker started")

	<-ctx.Done()

	server.Shutdown()

	closeCtx, cancel := context.WithTimeout(context.Background(), runtime.Config.HTTP.ShutdownTimeout)
	defer cancel()

	if err := runtime.Close(closeCtx); err != nil {
		runtime.Logger.Error().Err(err).Msg("runtime close failed")
	}
}
