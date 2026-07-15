package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONDuration_UnmarshalJSON(t *testing.T) {
	var d jsonDuration
	require.NoError(t, d.UnmarshalJSON([]byte(`"90s"`)))
	assert.Equal(t, 90*time.Second, time.Duration(d))

	require.NoError(t, d.UnmarshalJSON([]byte(`"2m"`)))
	assert.Equal(t, 2*time.Minute, time.Duration(d))

	assert.Error(t, d.UnmarshalJSON([]byte(`"not-a-duration"`)))
	assert.Error(t, d.UnmarshalJSON([]byte(`123`)))

	// time.ParseDuration happily accepts a leading "-"; without this check the
	// negative value would later silently wrap around to a huge uint when
	// applyServerFileConfig converts it to Config.StoreInterval.
	assert.Error(t, d.UnmarshalJSON([]byte(`"-5s"`)))

	// Sub-second precision would otherwise be silently truncated when
	// converted to Config's plain integer-seconds fields (e.g. "1500ms" -> 1),
	// so it's rejected outright instead of guessing what the user meant.
	assert.Error(t, d.UnmarshalJSON([]byte(`"1500ms"`)))
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

func TestApplyServerFileConfig_SkipsExplicitlySetFields(t *testing.T) {
	cfg := &Config{ServerAddr: "localhost:8080", Restore: true, StoreInterval: 300, FileStoragePath: "storage.txt"}

	addr := "from-file"
	restore := false
	fc := &fileConfig{Address: &addr, Restore: &restore}

	// "a" was explicitly passed on the command line (simulated via
	// explicitFlags), even though its value happens to equal the default.
	applyServerFileConfig(cfg, fc, map[string]bool{"a": true})

	assert.Equal(t, "localhost:8080", cfg.ServerAddr, "flag-set field must not be overwritten by file, even if its value equals the default")
	assert.False(t, cfg.Restore, "field not set by flag/env must be overwritten by file")
}

func TestApplyServerFileConfig_EnvVarWins(t *testing.T) {
	t.Setenv("ADDRESS", "localhost:8080") // env-set, value happens to equal the default

	cfg := &Config{ServerAddr: "localhost:8080"}
	addr := "from-file"
	fc := &fileConfig{Address: &addr}

	applyServerFileConfig(cfg, fc, nil)

	assert.Equal(t, "localhost:8080", cfg.ServerAddr, "env-set field must not be overwritten by file, even if its value equals the default")
}

func TestApplyAgentFileConfig_SkipsExplicitlySetFields(t *testing.T) {
	cfg := &Config{ServerAddr: "localhost:8080", ReportInterval: 10, PollInterval: 2}

	pollFromFile := jsonDuration(9 * time.Second)
	fc := &fileConfig{PollInterval: &pollFromFile}

	t.Setenv("POLL_INTERVAL", "2") // env-set, value happens to equal the default

	applyAgentFileConfig(cfg, fc, nil)

	assert.Equal(t, 2, cfg.PollInterval, "env-set field must not be overwritten by file, even if its value equals the default")
}
