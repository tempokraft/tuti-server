// Package config loads server configuration from environment variables.
package config

import (
	"os"
	"strconv"
)

// Config holds all runtime configuration for the server.
type Config struct {
	// Port is the TCP port the HTTP server listens on.
	Port string

	// StorageDir is the local directory captures are written to.
	StorageDir string

	// SessionDir is the local directory Snap & Solve / solve session state
	// is persisted to, so it survives a restart.
	SessionDir string

	// AnalysisBackend selects the vision LLM backend: "openai" (default)
	// or "anthropic". See analysis.OpenAI / analysis.Anthropic.
	AnalysisBackend string

	// AnalysisModel is the backend-specific model ID (e.g. "claude-opus-5"
	// or "gpt-5.2") used for photo analysis.
	AnalysisModel string

	// AnthropicAPIKey overrides the Anthropic SDK's default credential
	// resolution (ANTHROPIC_API_KEY env var, ANTHROPIC_AUTH_TOKEN, or an
	// `ant auth login` profile) when non-empty. Usually left empty in
	// favor of the SDK reading ANTHROPIC_API_KEY itself. Only consulted
	// when AnalysisBackend is "anthropic".
	AnthropicAPIKey string

	// OpenAIAPIKey overrides the OpenAI SDK's default credential
	// resolution (OPENAI_API_KEY env var) when non-empty. Only consulted
	// when AnalysisBackend is "openai".
	OpenAIAPIKey string

	// MaxUploadBytes caps the size of a single capture upload.
	MaxUploadBytes int64
}

const (
	defaultPort            = "8080"
	defaultStorageDir      = "./data/captures"
	defaultSessionDir      = "./data/sessions"
	defaultAnalysisBackend = "openai"
	defaultAnthropicModel  = "claude-opus-5"
	defaultOpenAIModel     = "gpt-5.2"
	defaultMaxUploadBytes  = 10 << 20 // 10 MiB
)

// Load reads configuration from the environment, applying defaults for
// anything unset.
func Load() Config {
	backend := getEnv("ANALYSIS_BACKEND", defaultAnalysisBackend)

	defaultModel := defaultAnthropicModel
	if backend == "openai" {
		defaultModel = defaultOpenAIModel
	}

	return Config{
		Port:            getEnv("PORT", defaultPort),
		StorageDir:      getEnv("STORAGE_DIR", defaultStorageDir),
		AnalysisBackend: backend,
		AnalysisModel:   getEnv("ANALYSIS_MODEL", defaultModel),
		AnthropicAPIKey: os.Getenv("ANTHROPIC_API_KEY"),
		OpenAIAPIKey:    os.Getenv("OPENAI_API_KEY"),
		MaxUploadBytes:  getEnvInt64("MAX_UPLOAD_BYTES", defaultMaxUploadBytes),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt64(key string, fallback int64) int64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return fallback
	}
	return n
}
