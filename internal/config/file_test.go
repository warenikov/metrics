package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDuration_UnmarshalJSON(t *testing.T) {
	var d duration
	require.NoError(t, d.UnmarshalJSON([]byte(`"1s500ms"`)))
	assert.Equal(t, 1500*time.Millisecond, time.Duration(d))

	assert.Error(t, d.UnmarshalJSON([]byte(`"not-a-duration"`)))
	assert.Error(t, d.UnmarshalJSON([]byte(`123`)))
}

func TestLoadFileConfig(t *testing.T) {
	t.Run("parses all fields", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.json")
		require.NoError(t, os.WriteFile(path, []byte(`{
			"address": "localhost:9090",
			"restore": false,
			"store_interval": "1s",
			"store_file": "db.txt",
			"database_dsn": "dsn",
			"crypto_key": "key.pem",
			"report_interval": "2s",
			"poll_interval": "3s"
		}`), 0o600))

		fc, err := loadFileConfig(path)
		require.NoError(t, err)
		require.NotNil(t, fc.Address)
		assert.Equal(t, "localhost:9090", *fc.Address)
		require.NotNil(t, fc.Restore)
		assert.False(t, *fc.Restore)
		require.NotNil(t, fc.StoreInterval)
		assert.Equal(t, time.Second, time.Duration(*fc.StoreInterval))
		require.NotNil(t, fc.StoreFile)
		assert.Equal(t, "db.txt", *fc.StoreFile)
		require.NotNil(t, fc.DatabaseDSN)
		assert.Equal(t, "dsn", *fc.DatabaseDSN)
		require.NotNil(t, fc.CryptoKey)
		assert.Equal(t, "key.pem", *fc.CryptoKey)
		require.NotNil(t, fc.ReportInterval)
		assert.Equal(t, 2*time.Second, time.Duration(*fc.ReportInterval))
		require.NotNil(t, fc.PollInterval)
		assert.Equal(t, 3*time.Second, time.Duration(*fc.PollInterval))
	})

	t.Run("absent keys stay nil", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.json")
		require.NoError(t, os.WriteFile(path, []byte(`{}`), 0o600))

		fc, err := loadFileConfig(path)
		require.NoError(t, err)
		assert.Nil(t, fc.Address)
		assert.Nil(t, fc.Restore)
		assert.Nil(t, fc.StoreInterval)
	})

	t.Run("missing file", func(t *testing.T) {
		_, err := loadFileConfig(filepath.Join(t.TempDir(), "missing.json"))
		assert.Error(t, err)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.json")
		require.NoError(t, os.WriteFile(path, []byte(`{invalid`), 0o600))
		_, err := loadFileConfig(path)
		assert.Error(t, err)
	})
}

func TestApplyServerFileConfig_OnlyFillsDefaults(t *testing.T) {
	defaults := Config{ServerAddr: "localhost:8080", Restore: true, StoreInterval: 300, FileStoragePath: "storage.txt"}
	cfg := defaults
	cfg.ServerAddr = "already-set-by-flag" // simulate a flag override

	addr := "from-file"
	restore := false
	fc := &fileConfig{Address: &addr, Restore: &restore}

	applyServerFileConfig(&cfg, defaults, fc)

	assert.Equal(t, "already-set-by-flag", cfg.ServerAddr, "flag-set field must not be overwritten by file")
	assert.False(t, cfg.Restore, "default-value field must be overwritten by file")
}

func TestApplyAgentFileConfig_OnlyFillsDefaults(t *testing.T) {
	defaults := Config{ServerAddr: "localhost:8080", ReportInterval: 10, PollInterval: 2}
	cfg := defaults
	cfg.PollInterval = 42 // simulate an env override

	pollFromFile := duration(9 * time.Second)
	fc := &fileConfig{PollInterval: &pollFromFile}

	applyAgentFileConfig(&cfg, defaults, fc)

	assert.Equal(t, 42, cfg.PollInterval, "env-set field must not be overwritten by file")
}
