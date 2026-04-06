package config

import (
	"flag"
	"fmt"

	"github.com/caarlos0/env/v6"
)

type Config struct {
	ServerAddr      string `env:"ADDRESS"`
	ReportInterval  int    `env:"REPORT_INTERVAL"`
	PollInterval    int    `env:"POLL_INTERVAL"`
	LogLevel        string `env:"LOG_LEVEL"`
	StoreInterval   uint   `env:"STORE_INTERVAL"`
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
	Restore         bool   `env:"RESTORE"`
	DBDNS           string `env:"DATABASE_DSN"`
}

func LoadServerConfig() (*Config, error) {
	cfg := &Config{
		ServerAddr:      "localhost:8080",
		LogLevel:        "info",
		StoreInterval:   300,
		FileStoragePath: "storage.txt",
		Restore:         true,
		DBDNS:           "",
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
	fs.StringVar(&cfg.DBDNS, "d", cfg.DBDNS, "Database DNS string")
	if err := fs.Parse(flagArgs()); err != nil {
		return nil, fmt.Errorf("invalid flag: %w", err)
	}

	return cfg, nil
}

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
	fs.StringVar(&cfg.LogLevel, "l", cfg.LogLevel, "Log level")
	if err := fs.Parse(flagArgs()); err != nil {
		return nil, fmt.Errorf("invalid flag: %w", err)
	}

	return cfg, nil
}

func loadFromEnv(cfg *Config) error {
	return env.Parse(cfg)
}
