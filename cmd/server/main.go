// Command server runs the Tuti tutor backend: an HTTP/JSON binding of the
// TutiService RPCs (Snap & Solve flow, captures, lessons, analysis)
// defined in tuti/proto/tuti_service.proto.
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

	"tuti-server/internal/analysis"
	"tuti-server/internal/config"
	"tuti-server/internal/httpapi"
	"tuti-server/internal/session"
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

	apiKey := cfg.AnthropicAPIKey
	if cfg.AnalysisBackend == analysis.OpenAI {
		apiKey = cfg.OpenAIAPIKey
	}
	apiKeyPresent := apiKey != ""

	logger.Info("configuration loaded",
		"port", cfg.Port,
		"storage_dir", cfg.StorageDir,
		"max_upload_bytes", cfg.MaxUploadBytes,
		"provider", cfg.AnalysisBackend,
		"provider_model", cfg.AnalysisModel,
		"provider_api_key_present", apiKeyPresent,
	)
	if !apiKeyPresent {
		// Not necessarily fatal: the SDKs also resolve credentials from
		// ANTHROPIC_AUTH_TOKEN / an `ant auth login` profile (Anthropic) or
		// other env-based auth the Config type doesn't track — but the
		// common case is a missing key, so surface it loudly rather than
		// let every analysis call fail with an opaque auth error.
		logger.Warn("no API key configured for the active provider; "+
			"relying on the SDK's own credential resolution",
			"provider", cfg.AnalysisBackend)
	}

	store, err := localfs.New(cfg.StorageDir)
	if err != nil {
		return err
	}

	analyzer, err := analysis.New(analysis.Config{
		Backend: cfg.AnalysisBackend,
		Model:   cfg.AnalysisModel,
		APIKey:  apiKey,
	})
	if err != nil {
		return err
	}

	server := &httpapi.Server{
		Store:          store,
		Analyzer:       analyzer,
		SnapStore:      session.NewSnapStore(),
		SolveStore:     session.NewSolveStore(),
		Tracer:         slogtracer.New(logger),
		MaxUploadBytes: cfg.MaxUploadBytes,
	}

	httpServer := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           server.Routes(),
		ReadHeaderTimeout: 10 * time.Second,
		// Generous but finite: AnalyzeAssets/SubmitSnapResponse call out to
		// a vision model, but every RPC is now a bounded request/response
		// (no open-ended streaming, unlike the old chat endpoint).
		WriteTimeout: 120 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		logger.Info("listening", "addr", httpServer.Addr)
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
