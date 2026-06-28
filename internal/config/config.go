// Package config loads and validates configuration for the server and agent
// binaries from command-line flags and environment variables.
// Environment variables take precedence over default values; flags override env vars.
package config

import (
	"flag"
	"fmt"

	"github.com/caarlos0/env/v6"
)

// Config holds all runtime configuration shared between the server and the agent.
// Each field is mapped to an environment variable via the env struct tag.
type Config struct {
	// ServerAddr is the TCP address the HTTP server listens on (host:port).
	ServerAddr string `env:"ADDRESS"`
	// ReportInterval is how often (in seconds) the agent sends metrics to the server.
	ReportInterval int `env:"REPORT_INTERVAL"`
	// PollInterval is how often (in seconds) the agent polls runtime metrics.
	PollInterval int `env:"POLL_INTERVAL"`
	// LogLevel is the zap log level (debug, info, warn, error).
	LogLevel string `env:"LOG_LEVEL"`
	// StoreInterval is how often (in seconds) the server flushes metrics to the file store.
	// Zero means synchronous (write-through) mode.
	StoreInterval uint `env:"STORE_INTERVAL"`
	// FileStoragePath is the path to the persistent JSON metrics file.
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
	// Restore controls whether metrics are loaded from FileStoragePath on server startup.
	Restore bool `env:"RESTORE"`
	// DBDSN is the PostgreSQL connection string. Empty disables database storage.
	DBDSN string `env:"DATABASE_DSN"`
	// Key is the HMAC-SHA256 signing key for request/response integrity verification.
	Key string `env:"KEY"`
	// RateLimit caps the number of parallel outgoing HTTP connections used by the agent.
	RateLimit int `env:"RATE_LIMIT"`
	// AuditFile is the path to the append-only audit log file. Empty disables file auditing.
	AuditFile string `env:"AUDIT_FILE"`
	// AuditURL is the remote endpoint that receives audit events via HTTP POST. Empty disables remote auditing.
	AuditURL string `env:"AUDIT_URL"`
}

// LoadServerConfig loads the server configuration from environment variables
// and command-line flags, applying the following defaults:
//   - ADDRESS:           localhost:8080
//   - LOG_LEVEL:         info
//   - STORE_INTERVAL:    300 (seconds)
//   - FILE_STORAGE_PATH: storage.txt
//   - RESTORE:           true
func LoadServerConfig() (*Config, error) {
	cfg := &Config{
		ServerAddr:      "localhost:8080",
		LogLevel:        "info",
		StoreInterval:   300,
		FileStoragePath: "storage.txt",
		Restore:         true,
		DBDSN:           "",
		Key:             "",
	}

	if err := loadFromEnv(cfg); err != nil {
		return nil, fmt.Errorf("invalid environment config: %w", err)
	}

	fs := flag.NewFlagSet("server", flag.ContinueOnError)
	fs.StringVar(&cfg.ServerAddr, "a", cfg.ServerAddr, "HTTP address")
	fs.StringVar(&cfg.FileStoragePath, "f", cfg.FileStoragePath, "File storage path")
	fs.BoolVar(&cfg.Restore, "r", cfg.Restore, "Restore metrics from file")
	fs.StringVar(&cfg.LogLevel, "l", cfg.LogLevel, "Log level")
	fs.UintVar(&cfg.StoreInterval, "i", cfg.StoreInterval, "Store interval in seconds")
	fs.StringVar(&cfg.DBDSN, "d", cfg.DBDSN, "Database DNS string")
	fs.StringVar(&cfg.Key, "k", cfg.Key, "Key for signing")
	fs.StringVar(&cfg.AuditFile, "audit-file", cfg.AuditFile, "Audit log file path")
	fs.StringVar(&cfg.AuditURL, "audit-url", cfg.AuditURL, "Audit log remote URL")
	if err := fs.Parse(flagArgs()); err != nil {
		return nil, fmt.Errorf("invalid flag: %w", err)
	}

	return cfg, nil
}

// LoadAgentConfig loads the agent configuration from environment variables
// and command-line flags, applying the following defaults:
//   - ADDRESS:         localhost:8080
//   - REPORT_INTERVAL: 10 (seconds)
//   - POLL_INTERVAL:   2 (seconds)
//   - LOG_LEVEL:       info
func LoadAgentConfig() (*Config, error) {
	cfg := &Config{
		ServerAddr:     "localhost:8080",
		ReportInterval: 10,
		PollInterval:   2,
		LogLevel:       "info",
	}

	if err := loadFromEnv(cfg); err != nil {
		return nil, fmt.Errorf("invalid environment config: %w", err)
	}

	fs := flag.NewFlagSet("agent", flag.ContinueOnError)
	fs.StringVar(&cfg.ServerAddr, "a", cfg.ServerAddr, "HTTP address")
	fs.IntVar(&cfg.ReportInterval, "r", cfg.ReportInterval, "Reporting interval in seconds")
	fs.IntVar(&cfg.PollInterval, "p", cfg.PollInterval, "Polling interval in seconds")
	fs.StringVar(&cfg.Key, "k", cfg.Key, "Key for signing")
	fs.IntVar(&cfg.RateLimit, "l", cfg.RateLimit, "Rate limit for outgoing requests")
	if err := fs.Parse(flagArgs()); err != nil {
		return nil, fmt.Errorf("invalid flag: %w", err)
	}
	if cfg.RateLimit < 0 {
		return nil, fmt.Errorf("rate limit must be non-negative, got %d", cfg.RateLimit)
	}

	return cfg, nil
}

func loadFromEnv(cfg *Config) error {
	return env.Parse(cfg)
}
