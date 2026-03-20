package config

import (
	"flag"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadServerConfig(t *testing.T) {
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	tests := []struct {
		name             string
		envVars          map[string]string
		args             []string
		expectedAddr     string
		expectedInterval int
		expectedFilePath string
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
			name:             "invalid env stays default",
			envVars:          map[string]string{"STORE_INTERVAL": "notanumber"},
			args:             []string{},
			expectedAddr:     "localhost:8080",
			expectedInterval: 300,
			expectedFilePath: "storage.txt",
			expectedRestore:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)
			os.Args = append([]string{"test_bin"}, tt.args...)

			cfg := LoadServerConfig()

			assert.Equal(t, tt.expectedAddr, cfg.ServerAddr)
			assert.Equal(t, tt.expectedInterval, cfg.StoreInterval)
			assert.Equal(t, tt.expectedFilePath, cfg.FileStoragePath)
			assert.Equal(t, tt.expectedRestore, cfg.Restore)
		})
	}
}

func TestLoadAgentConfig(t *testing.T) {
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	tests := []struct {
		name           string
		envVars        map[string]string
		args           []string
		expectedAddr   string
		expectedReport int
		expectedPoll   int
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
			name:           "invalid env stays default",
			envVars:        map[string]string{"REPORT_INTERVAL": "fast", "POLL_INTERVAL": "slow"},
			args:           []string{},
			expectedAddr:   "localhost:8080",
			expectedReport: 10,
			expectedPoll:   2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)
			os.Args = append([]string{"test_bin"}, tt.args...)

			cfg := LoadAgentConfig()

			assert.Equal(t, tt.expectedAddr, cfg.ServerAddr)
			assert.Equal(t, tt.expectedReport, cfg.ReportInterval)
			assert.Equal(t, tt.expectedPoll, cfg.PollInterval)
		})
	}
}
