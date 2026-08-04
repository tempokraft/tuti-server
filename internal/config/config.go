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

	// AnthropicAPIKey overrides the SDK's default credential resolution
	// (ANTHROPIC_API_KEY env var, ANTHROPIC_AUTH_TOKEN, or an `ant auth
	// login` profile) when non-empty. Usually left empty in favor of the
	// SDK reading ANTHROPIC_API_KEY itself.
	AnthropicAPIKey string

	// AnthropicModel is the Claude model ID used for tutoring chat.
	AnthropicModel string

	// MaxUploadBytes caps the size of a single capture upload.
	MaxUploadBytes int64
}

const (
	defaultPort           = "8080"
	defaultStorageDir     = "./data/captures"
	defaultModel          = "claude-opus-5"
	defaultMaxUploadBytes = 10 << 20 // 10 MiB
)

// Load reads configuration from the environment, applying defaults for
// anything unset.
func Load() Config {
	return Config{
		Port:            getEnv("PORT", defaultPort),
		StorageDir:      getEnv("STORAGE_DIR", defaultStorageDir),
		AnthropicAPIKey: os.Getenv("ANTHROPIC_API_KEY"),
		AnthropicModel:  getEnv("ANTHROPIC_MODEL", defaultModel),
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
