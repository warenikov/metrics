package main

import (
	"metrics/internal/agent"
	"metrics/internal/config"
	"metrics/internal/logger"
	"os"

	"go.uber.org/zap"
)

func main() {
	cfg, err := config.LoadAgentConfig()
	if err != nil {
		logger.Log.Fatal("Invalid config", zap.Error(err))
	}
	agent := agent.NewMetricaAgent(cfg)
	if err := logger.Initialize(cfg.LogLevel); err != nil {
		logger.Log.Fatal("Failed to initialize logger", zap.Error(err))
	}
	fileContent, fileErr := os.ReadFile(cfg.Key)
	logger.Log.Info("key diagnostics",
		zap.String("key_value", cfg.Key),
		zap.String("file_content", string(fileContent)),
		zap.Bool("file_exists", fileErr == nil),
	)
	agent.Run()

}
