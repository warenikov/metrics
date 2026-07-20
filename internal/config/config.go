// Package config loads and validates configuration for the server and agent
// binaries from command-line flags, environment variables, and an optional
// JSON config file. Priority, highest to lowest: flags > env vars > config
// file > hardcoded defaults.
package config

import (
	"flag"
	"fmt"

	"github.com/caarlos0/env/v6"
)

// Config holds all runtime configuration shared between the server and the agent.
// Each field is mapped to an environment variable via the env struct tag.
type Config struct {
	ServerAddr      string `env:"ADDRESS"`
	LogLevel        string `env:"LOG_LEVEL"`
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
	DBDSN           string `env:"DATABASE_DSN"`
	Key             string `env:"KEY"`
	CryptoKeyPath   string `env:"CRYPTO_KEY"`
	ConfigPath      string `env:"CONFIG"`
	TrustedSubnet   string `env:"TRUSTED_SUBNET"`
	GRPCAddr        string `env:"GRPC_ADDRESS"`
	AuditFile       string `env:"AUDIT_FILE"`
	AuditURL        string `env:"AUDIT_URL"`
	ReportInterval  int    `env:"REPORT_INTERVAL"`
	PollInterval    int    `env:"POLL_INTERVAL"`
	StoreInterval   uint   `env:"STORE_INTERVAL"`
	RateLimit       int    `env:"RATE_LIMIT"`
	Restore         bool   `env:"RESTORE"`
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
	fs.StringVar(&cfg.CryptoKeyPath, "crypto-key", cfg.CryptoKeyPath, "Path to RSA private key file for decrypting request bodies")
	fs.StringVar(&cfg.ConfigPath, "c", cfg.ConfigPath, "Path to JSON config file")
	fs.StringVar(&cfg.ConfigPath, "config", cfg.ConfigPath, "Path to JSON config file")
	fs.StringVar(&cfg.TrustedSubnet, "t", cfg.TrustedSubnet, "Trusted subnet (CIDR) for the X-Real-IP check")
	fs.StringVar(&cfg.GRPCAddr, "g", cfg.GRPCAddr, "gRPC listen address")
	fs.StringVar(&cfg.AuditFile, "audit-file", cfg.AuditFile, "Audit log file path")
	fs.StringVar(&cfg.AuditURL, "audit-url", cfg.AuditURL, "Audit log remote URL")
	if err := fs.Parse(flagArgs()); err != nil {
		return nil, fmt.Errorf("invalid flag: %w", err)
	}

	if cfg.ConfigPath != "" {
		fc, err := loadFileConfig(cfg.ConfigPath)
		if err != nil {
			return nil, fmt.Errorf("invalid config file: %w", err)
		}
		applyServerFileConfig(cfg, fc, visitedFlags(fs))
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
	fs.StringVar(&cfg.CryptoKeyPath, "crypto-key", cfg.CryptoKeyPath, "Path to RSA public key file for encrypting request bodies")
	fs.StringVar(&cfg.ConfigPath, "c", cfg.ConfigPath, "Path to JSON config file")
	fs.StringVar(&cfg.ConfigPath, "config", cfg.ConfigPath, "Path to JSON config file")
	fs.StringVar(&cfg.GRPCAddr, "g", cfg.GRPCAddr, "gRPC server address to send metrics to")
	fs.IntVar(&cfg.RateLimit, "l", cfg.RateLimit, "Rate limit for outgoing requests")
	if err := fs.Parse(flagArgs()); err != nil {
		return nil, fmt.Errorf("invalid flag: %w", err)
	}
	if cfg.RateLimit < 0 {
		return nil, fmt.Errorf("rate limit must be non-negative, got %d", cfg.RateLimit)
	}

	if cfg.ConfigPath != "" {
		fc, err := loadFileConfig(cfg.ConfigPath)
		if err != nil {
			return nil, fmt.Errorf("invalid config file: %w", err)
		}
		applyAgentFileConfig(cfg, fc, visitedFlags(fs))
	}

	return cfg, nil
}

func loadFromEnv(cfg *Config) error {
	return env.Parse(cfg)
}
