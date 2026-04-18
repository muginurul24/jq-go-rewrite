package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	transporthttp "github.com/mugiew/justqiuv2-rewrite/apps/api/internal/http"
	"github.com/rs/zerolog"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/app"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	bootLogger := zerolog.New(os.Stderr).With().Timestamp().Logger()

	runtime, err := app.New(ctx)
	if err != nil {
		bootLogger.Fatal().Err(err).Msg("bootstrap failed")
	}

	server := &http.Server{
		Addr:              ":" + runtime.Config.HTTP.Port,
		Handler:           transporthttp.NewRouter(runtime),
		ReadTimeout:       runtime.Config.HTTP.ReadTimeout,
		ReadHeaderTimeout: runtime.Config.HTTP.ReadHeaderTimeout,
		WriteTimeout:      runtime.Config.HTTP.WriteTimeout,
		IdleTimeout:       runtime.Config.HTTP.IdleTimeout,
	}

	runtime.Logger.Info().
		Str("addr", server.Addr).
		Msg("http server started")

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), runtime.Config.HTTP.ShutdownTimeout)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			runtime.Logger.Error().Err(err).Msg("graceful shutdown failed")
		}
	}()

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		runtime.Logger.Fatal().Err(err).Msg("http server crashed")
	}

	closeCtx, cancel := context.WithTimeout(context.Background(), runtime.Config.HTTP.ShutdownTimeout)
	defer cancel()

	if err := runtime.Close(closeCtx); err != nil {
		runtime.Logger.Error().Err(err).Msg("runtime close failed")
	}
}
