// Command server runs the Tuti tutor backend: a Claude-backed streaming
// chat endpoint and a capture (screenshot) upload/list endpoint.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"tuti-server/internal/agent/claude"
	"tuti-server/internal/config"
	"tuti-server/internal/httpapi"
	"tuti-server/internal/storage/localfs"
	"tuti-server/internal/tracing/slogtracer"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	if err := run(logger); err != nil {
		logger.Error("server exited with error", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg := config.Load()

	store, err := localfs.New(cfg.StorageDir)
	if err != nil {
		return err
	}

	tutorAgent := claude.New(cfg.AnthropicModel, cfg.AnthropicAPIKey)

	server := &httpapi.Server{
		Agent:          tutorAgent,
		Store:          store,
		Tracer:         slogtracer.New(logger),
		MaxUploadBytes: cfg.MaxUploadBytes,
	}

	httpServer := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           server.Routes(),
		ReadHeaderTimeout: 10 * time.Second,
		// No overall write timeout: chat responses stream for as long as
		// the model takes to reply.
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		logger.Info("listening", "addr", httpServer.Addr, "model", cfg.AnthropicModel, "storage_dir", cfg.StorageDir)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	logger.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return httpServer.Shutdown(shutdownCtx)
}
