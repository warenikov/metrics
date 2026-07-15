package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeConfigFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func TestLoadServerConfig(t *testing.T) {
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	tests := []struct {
		envVars          map[string]string
		name             string
		expectedAddr     string
		expectedFilePath string
		args             []string
		expectedInterval uint
		wantErr          bool
		expectedRestore  bool
	}{
		{
			name:             "defaults",
			envVars:          map[string]string{},
			args:             []string{},
			expectedAddr:     "localhost:8080",
			expectedInterval: 300,
			expectedFilePath: "storage.txt",
			expectedRestore:  true,
		},
		{
			name:             "env vars",
			envVars:          map[string]string{"ADDRESS": "env:9090", "STORE_INTERVAL": "60", "FILE_STORAGE_PATH": "env.txt", "RESTORE": "false"},
			args:             []string{},
			expectedAddr:     "env:9090",
			expectedInterval: 60,
			expectedFilePath: "env.txt",
			expectedRestore:  false,
		},
		{
			name:             "cli flags override env",
			envVars:          map[string]string{"ADDRESS": "env:9090"},
			args:             []string{"-a", "flag:7777", "-i", "10", "-f", "flag.txt", "-r=false"},
			expectedAddr:     "flag:7777",
			expectedInterval: 10,
			expectedFilePath: "flag.txt",
			expectedRestore:  false,
		},
		{
			name:    "invalid env returns error",
			envVars: map[string]string{"STORE_INTERVAL": "notanumber"},
			args:    []string{},
			wantErr: true,
		},
		{
			name:    "negative interval via env returns error",
			envVars: map[string]string{"STORE_INTERVAL": "-5"},
			args:    []string{},
			wantErr: true,
		},
		{
			name:    "negative interval via flag returns error",
			envVars: map[string]string{},
			args:    []string{"-i", "-5"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			os.Args = append([]string{"test_bin"}, tt.args...)

			cfg, err := LoadServerConfig()

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedAddr, cfg.ServerAddr)
			assert.Equal(t, tt.expectedInterval, cfg.StoreInterval)
			assert.Equal(t, tt.expectedFilePath, cfg.FileStoragePath)
			assert.Equal(t, tt.expectedRestore, cfg.Restore)
		})
	}
}

func TestLoadAgentConfigRateLimit(t *testing.T) {
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	tests := []struct {
		name              string
		envVars           map[string]string
		args              []string
		wantErr           bool
		expectedRateLimit int
	}{
		{
			name:              "default zero",
			envVars:           map[string]string{},
			args:              []string{},
			expectedRateLimit: 0,
		},
		{
			name:              "positive via flag",
			args:              []string{"-l", "3"},
			expectedRateLimit: 3,
		},
		{
			name:              "positive via env",
			envVars:           map[string]string{"RATE_LIMIT": "5"},
			expectedRateLimit: 5,
		},
		{
			name:    "negative via flag returns error",
			args:    []string{"-l", "-1"},
			wantErr: true,
		},
		{
			name:    "negative via env returns error",
			envVars: map[string]string{"RATE_LIMIT": "-2"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}
			os.Args = append([]string{"test_bin"}, tt.args...)

			cfg, err := LoadAgentConfig()

			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expectedRateLimit, cfg.RateLimit)
		})
	}
}

func TestLoadAgentConfig(t *testing.T) {
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	tests := []struct {
		envVars        map[string]string
		name           string
		expectedAddr   string
		args           []string
		expectedReport int
		expectedPoll   int
		wantErr        bool
	}{
		{
			name:           "defaults",
			envVars:        map[string]string{},
			args:           []string{},
			expectedAddr:   "localhost:8080",
			expectedReport: 10,
			expectedPoll:   2,
		},
		{
			name:           "env vars",
			envVars:        map[string]string{"ADDRESS": "env:8888", "POLL_INTERVAL": "5", "REPORT_INTERVAL": "20"},
			args:           []string{},
			expectedAddr:   "env:8888",
			expectedReport: 20,
			expectedPoll:   5,
		},
		{
			name:           "cli flags override env",
			envVars:        map[string]string{"ADDRESS": "env:8888"},
			args:           []string{"-a", "flag:9999", "-r", "30", "-p", "7"},
			expectedAddr:   "flag:9999",
			expectedReport: 30,
			expectedPoll:   7,
		},
		{
			name:    "invalid env returns error",
			envVars: map[string]string{"REPORT_INTERVAL": "fast", "POLL_INTERVAL": "slow"},
			args:    []string{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			os.Args = append([]string{"test_bin"}, tt.args...)

			cfg, err := LoadAgentConfig()

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedAddr, cfg.ServerAddr)
			assert.Equal(t, tt.expectedReport, cfg.ReportInterval)
			assert.Equal(t, tt.expectedPoll, cfg.PollInterval)
		})
	}
}

func TestLoadServerConfig_File(t *testing.T) {
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	fileContent := `{
		"address": "file:9000",
		"restore": false,
		"store_interval": "15s",
		"store_file": "file.db",
		"database_dsn": "file-dsn",
		"crypto_key": "file-key.pem",
		"trusted_subnet": "192.168.1.0/24"
	}`

	t.Run("file fills in unset values", func(t *testing.T) {
		path := writeConfigFile(t, fileContent)
		os.Args = []string{"test_bin", "-c", path}

		cfg, err := LoadServerConfig()
		require.NoError(t, err)
		assert.Equal(t, "file:9000", cfg.ServerAddr)
		assert.Equal(t, false, cfg.Restore)
		assert.Equal(t, uint(15), cfg.StoreInterval)
		assert.Equal(t, "file.db", cfg.FileStoragePath)
		assert.Equal(t, "file-dsn", cfg.DBDSN)
		assert.Equal(t, "file-key.pem", cfg.CryptoKeyPath)
		assert.Equal(t, "192.168.1.0/24", cfg.TrustedSubnet)
	})

	t.Run("-t flag sets trusted subnet", func(t *testing.T) {
		os.Args = []string{"test_bin", "-t", "10.0.0.0/8"}

		cfg, err := LoadServerConfig()
		require.NoError(t, err)
		assert.Equal(t, "10.0.0.0/8", cfg.TrustedSubnet)
	})

	t.Run("TRUSTED_SUBNET env var sets trusted subnet", func(t *testing.T) {
		t.Setenv("TRUSTED_SUBNET", "172.16.0.0/12")
		os.Args = []string{"test_bin"}

		cfg, err := LoadServerConfig()
		require.NoError(t, err)
		assert.Equal(t, "172.16.0.0/12", cfg.TrustedSubnet)
	})

	t.Run("empty trusted subnet by default", func(t *testing.T) {
		os.Args = []string{"test_bin"}

		cfg, err := LoadServerConfig()
		require.NoError(t, err)
		assert.Equal(t, "", cfg.TrustedSubnet)
	})

	t.Run("--config long flag works the same as -c", func(t *testing.T) {
		path := writeConfigFile(t, fileContent)
		os.Args = []string{"test_bin", "-config", path}

		cfg, err := LoadServerConfig()
		require.NoError(t, err)
		assert.Equal(t, "file:9000", cfg.ServerAddr)
	})

	t.Run("CONFIG env var also selects the file", func(t *testing.T) {
		path := writeConfigFile(t, fileContent)
		t.Setenv("CONFIG", path)
		os.Args = []string{"test_bin"}

		cfg, err := LoadServerConfig()
		require.NoError(t, err)
		assert.Equal(t, "file:9000", cfg.ServerAddr)
	})

	t.Run("flag overrides file", func(t *testing.T) {
		path := writeConfigFile(t, fileContent)
		os.Args = []string{"test_bin", "-c", path, "-a", "flag:1111"}

		cfg, err := LoadServerConfig()
		require.NoError(t, err)
		assert.Equal(t, "flag:1111", cfg.ServerAddr)
		assert.Equal(t, false, cfg.Restore, "fields not set by flag still come from file")
	})

	t.Run("env overrides file", func(t *testing.T) {
		path := writeConfigFile(t, fileContent)
		t.Setenv("ADDRESS", "env:2222")
		os.Args = []string{"test_bin", "-c", path}

		cfg, err := LoadServerConfig()
		require.NoError(t, err)
		assert.Equal(t, "env:2222", cfg.ServerAddr)
		assert.Equal(t, "file.db", cfg.FileStoragePath, "fields not set by env still come from file")
	})

	t.Run("flag matching the default still wins over file", func(t *testing.T) {
		// Regression test: file priority must be based on whether the flag was
		// actually passed (flag.Visit), not on comparing the resulting value
		// against the hardcoded default — otherwise a flag whose value happens
		// to equal the default would be silently overridden by the file.
		path := writeConfigFile(t, fileContent)
		os.Args = []string{"test_bin", "-c", path, "-a", "localhost:8080"}

		cfg, err := LoadServerConfig()
		require.NoError(t, err)
		assert.Equal(t, "localhost:8080", cfg.ServerAddr, "explicitly-passed flag must win even though its value equals the default")
	})

	t.Run("missing file returns error", func(t *testing.T) {
		os.Args = []string{"test_bin", "-c", filepath.Join(t.TempDir(), "missing.json")}
		_, err := LoadServerConfig()
		require.Error(t, err)
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		path := writeConfigFile(t, "not json")
		os.Args = []string{"test_bin", "-c", path}
		_, err := LoadServerConfig()
		require.Error(t, err)
	})

	t.Run("invalid duration returns error", func(t *testing.T) {
		path := writeConfigFile(t, `{"store_interval": "not-a-duration"}`)
		os.Args = []string{"test_bin", "-c", path}
		_, err := LoadServerConfig()
		require.Error(t, err)
	})

	t.Run("negative duration returns error instead of wrapping to a huge uint", func(t *testing.T) {
		path := writeConfigFile(t, `{"store_interval": "-5s"}`)
		os.Args = []string{"test_bin", "-c", path}
		_, err := LoadServerConfig()
		require.Error(t, err)
	})
}

func TestLoadAgentConfig_File(t *testing.T) {
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	fileContent := `{
		"address": "file:9000",
		"report_interval": "20s",
		"poll_interval": "4s",
		"crypto_key": "file-pub.pem"
	}`

	t.Run("file fills in unset values", func(t *testing.T) {
		path := writeConfigFile(t, fileContent)
		os.Args = []string{"test_bin", "-c", path}

		cfg, err := LoadAgentConfig()
		require.NoError(t, err)
		assert.Equal(t, "file:9000", cfg.ServerAddr)
		assert.Equal(t, 20, cfg.ReportInterval)
		assert.Equal(t, 4, cfg.PollInterval)
		assert.Equal(t, "file-pub.pem", cfg.CryptoKeyPath)
	})

	t.Run("flag overrides file, env overrides file", func(t *testing.T) {
		path := writeConfigFile(t, fileContent)
		t.Setenv("POLL_INTERVAL", "9")
		os.Args = []string{"test_bin", "-c", path, "-r", "99"}

		cfg, err := LoadAgentConfig()
		require.NoError(t, err)
		assert.Equal(t, 99, cfg.ReportInterval, "flag wins over file")
		assert.Equal(t, 9, cfg.PollInterval, "env wins over file")
		assert.Equal(t, "file:9000", cfg.ServerAddr, "file still applies where nothing else set it")
	})
}
