package config

import (
	"flag"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
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
			name:           "Дефолты",
			envVars:        map[string]string{},
			args:           []string{},
			expectedAddr:   "localhost:8080",
			expectedReport: 10,
			expectedPoll:   2,
		},
		{
			name:           "Приоритет ENV",
			envVars:        map[string]string{"ADDRESS": "env:8888", "POLL_INTERVAL": "5"},
			args:           []string{},
			expectedAddr:   "env:8888",
			expectedReport: 10,
			expectedPoll:   5,
		},
		{
			name:           "Приоритет Флагов (высший)",
			envVars:        map[string]string{"ADDRESS": "env:8888"},
			args:           []string{"-a", "flag:9999", "-p", "7"},
			expectedAddr:   "flag:9999",
			expectedReport: 10,
			expectedPoll:   7,
		},
		{
			name:           "Некорректный ENV (не число) — должны остаться дефолты",
			envVars:        map[string]string{"REPORT_INTERVAL": "fast", "POLL_INTERVAL": "slow"},
			args:           []string{},
			expectedAddr:   "localhost:8080",
			expectedReport: 10,
			expectedPoll:   2,
		},
		{
			name:           "Пустой ENV ADDRESS — не затирает дефолт",
			envVars:        map[string]string{"ADDRESS": ""},
			args:           []string{},
			expectedAddr:   "localhost:8080",
			expectedReport: 10,
			expectedPoll:   2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Очистка
			os.Clearenv()
			for k, v := range tt.envVars {
				_ = os.Setenv(k, v)
			}

			// Сброс флагов: используем фиксированное имя "test", чтобы не зависеть от os.Args
			flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)

			// Подмена аргументов
			os.Args = append([]string{"test_bin"}, tt.args...)

			cfg := LoadConfig()

			// Проверка всех полей структуры
			assert.Equal(t, tt.expectedAddr, cfg.ServerAddr, "Ошибка в ServerAddr")
			assert.Equal(t, tt.expectedReport, cfg.ReportInterval, "Ошибка в ReportInterval")
			assert.Equal(t, tt.expectedPoll, cfg.PollInterval, "Ошибка в PollInterval")
		})
	}
}
