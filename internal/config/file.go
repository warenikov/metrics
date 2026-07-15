package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"
)

// jsonDuration unmarshals a JSON string like "1s" (time.Duration syntax)
// into a time.Duration. The config file expresses intervals this way, even
// though [Config] itself stores them as plain integer seconds — so only
// whole-second, non-negative durations are accepted.
type jsonDuration time.Duration

func (d *jsonDuration) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	parsed, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", s, err)
	}
	if parsed < 0 {
		return fmt.Errorf("duration %q must not be negative", s)
	}
	if parsed%time.Second != 0 {
		return fmt.Errorf("duration %q must be a whole number of seconds", s)
	}
	*d = jsonDuration(parsed)
	return nil
}

// fileConfig mirrors the on-disk JSON config format shared by the server and
// agent (each uses the subset of fields relevant to it). Pointer fields
// distinguish a key that is absent from the file (nil — env/flag/hardcoded
// value applies) from one present with a zero value (false, "").
type fileConfig struct {
	Address        *string       `json:"address"`
	StoreFile      *string       `json:"store_file"`
	DatabaseDSN    *string       `json:"database_dsn"`
	CryptoKey      *string       `json:"crypto_key"`
	TrustedSubnet  *string       `json:"trusted_subnet"`
	StoreInterval  *jsonDuration `json:"store_interval"`
	ReportInterval *jsonDuration `json:"report_interval"`
	PollInterval   *jsonDuration `json:"poll_interval"`
	Restore        *bool         `json:"restore"`
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

// visitedFlags returns the set of flag names that were explicitly passed on
// the command line, as reported by [flag.FlagSet.Visit] (which — unlike
// VisitAll — only visits flags actually set, not ones left at their
// default).
func visitedFlags(fs *flag.FlagSet) map[string]bool {
	set := make(map[string]bool)
	fs.Visit(func(f *flag.Flag) { set[f.Name] = true })
	return set
}

// isSet reports whether a config field was explicitly provided via env var
// or command-line flag, as opposed to being left at its hardcoded default.
// Checking actual provenance (rather than comparing the resulting value
// against the default) means a flag or env var whose value happens to equal
// the default still correctly wins over the config file.
func isSet(explicitFlags map[string]bool, envVar string, flagNames ...string) bool {
	if _, ok := os.LookupEnv(envVar); ok {
		return true
	}
	for _, name := range flagNames {
		if explicitFlags[name] {
			return true
		}
	}
	return false
}

// applyServerFileConfig overlays fc onto cfg, but only for fields not
// explicitly set via env var or flag — this gives the config file the
// lowest priority of the three sources.
func applyServerFileConfig(cfg *Config, fc *fileConfig, explicitFlags map[string]bool) {
	if fc.Address != nil && !isSet(explicitFlags, "ADDRESS", "a") {
		cfg.ServerAddr = *fc.Address
	}
	if fc.Restore != nil && !isSet(explicitFlags, "RESTORE", "r") {
		cfg.Restore = *fc.Restore
	}
	if fc.StoreInterval != nil && !isSet(explicitFlags, "STORE_INTERVAL", "i") {
		cfg.StoreInterval = uint(time.Duration(*fc.StoreInterval) / time.Second)
	}
	if fc.StoreFile != nil && !isSet(explicitFlags, "FILE_STORAGE_PATH", "f") {
		cfg.FileStoragePath = *fc.StoreFile
	}
	if fc.DatabaseDSN != nil && !isSet(explicitFlags, "DATABASE_DSN", "d") {
		cfg.DBDSN = *fc.DatabaseDSN
	}
	if fc.CryptoKey != nil && !isSet(explicitFlags, "CRYPTO_KEY", "crypto-key") {
		cfg.CryptoKeyPath = *fc.CryptoKey
	}
	if fc.TrustedSubnet != nil && !isSet(explicitFlags, "TRUSTED_SUBNET", "t") {
		cfg.TrustedSubnet = *fc.TrustedSubnet
	}
}

// applyAgentFileConfig overlays fc onto cfg, but only for fields not
// explicitly set via env var or flag — this gives the config file the
// lowest priority of the three sources.
func applyAgentFileConfig(cfg *Config, fc *fileConfig, explicitFlags map[string]bool) {
	if fc.Address != nil && !isSet(explicitFlags, "ADDRESS", "a") {
		cfg.ServerAddr = *fc.Address
	}
	if fc.ReportInterval != nil && !isSet(explicitFlags, "REPORT_INTERVAL", "r") {
		cfg.ReportInterval = int(time.Duration(*fc.ReportInterval) / time.Second)
	}
	if fc.PollInterval != nil && !isSet(explicitFlags, "POLL_INTERVAL", "p") {
		cfg.PollInterval = int(time.Duration(*fc.PollInterval) / time.Second)
	}
	if fc.CryptoKey != nil && !isSet(explicitFlags, "CRYPTO_KEY", "crypto-key") {
		cfg.CryptoKeyPath = *fc.CryptoKey
	}
}
