package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// duration unmarshals a JSON string like "1s" (time.Duration syntax) into a
// time.Duration. The config file expresses intervals this way, even though
// [Config] itself stores them as plain integer seconds.
type duration time.Duration

func (d *duration) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	parsed, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", s, err)
	}
	*d = duration(parsed)
	return nil
}

// fileConfig mirrors the on-disk JSON config format shared by the server and
// agent (each uses the subset of fields relevant to it). Pointer fields
// distinguish a key that is absent from the file (nil — env/flag/hardcoded
// value applies) from one present with a zero value (false, "").
type fileConfig struct {
	Address        *string   `json:"address"`
	StoreFile      *string   `json:"store_file"`
	DatabaseDSN    *string   `json:"database_dsn"`
	CryptoKey      *string   `json:"crypto_key"`
	StoreInterval  *duration `json:"store_interval"`
	ReportInterval *duration `json:"report_interval"`
	PollInterval   *duration `json:"poll_interval"`
	Restore        *bool     `json:"restore"`
}

// loadFileConfig reads and parses the JSON config file at path.
func loadFileConfig(path string) (*fileConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file %s: %w", path, err)
	}
	var fc fileConfig
	if err := json.Unmarshal(data, &fc); err != nil {
		return nil, fmt.Errorf("parse config file %s: %w", path, err)
	}
	return &fc, nil
}

// applyServerFileConfig overlays fc onto cfg, but only for fields still at
// their hardcoded default (i.e. not already set by an env var or flag) —
// this gives the config file the lowest priority of the three sources.
func applyServerFileConfig(cfg *Config, defaults Config, fc *fileConfig) {
	if fc.Address != nil && cfg.ServerAddr == defaults.ServerAddr {
		cfg.ServerAddr = *fc.Address
	}
	if fc.Restore != nil && cfg.Restore == defaults.Restore {
		cfg.Restore = *fc.Restore
	}
	if fc.StoreInterval != nil && cfg.StoreInterval == defaults.StoreInterval {
		cfg.StoreInterval = uint(time.Duration(*fc.StoreInterval) / time.Second)
	}
	if fc.StoreFile != nil && cfg.FileStoragePath == defaults.FileStoragePath {
		cfg.FileStoragePath = *fc.StoreFile
	}
	if fc.DatabaseDSN != nil && cfg.DBDSN == defaults.DBDSN {
		cfg.DBDSN = *fc.DatabaseDSN
	}
	if fc.CryptoKey != nil && cfg.CryptoKeyPath == defaults.CryptoKeyPath {
		cfg.CryptoKeyPath = *fc.CryptoKey
	}
}

// applyAgentFileConfig overlays fc onto cfg, but only for fields still at
// their hardcoded default (i.e. not already set by an env var or flag) —
// this gives the config file the lowest priority of the three sources.
func applyAgentFileConfig(cfg *Config, defaults Config, fc *fileConfig) {
	if fc.Address != nil && cfg.ServerAddr == defaults.ServerAddr {
		cfg.ServerAddr = *fc.Address
	}
	if fc.ReportInterval != nil && cfg.ReportInterval == defaults.ReportInterval {
		cfg.ReportInterval = int(time.Duration(*fc.ReportInterval) / time.Second)
	}
	if fc.PollInterval != nil && cfg.PollInterval == defaults.PollInterval {
		cfg.PollInterval = int(time.Duration(*fc.PollInterval) / time.Second)
	}
	if fc.CryptoKey != nil && cfg.CryptoKeyPath == defaults.CryptoKeyPath {
		cfg.CryptoKeyPath = *fc.CryptoKey
	}
}
