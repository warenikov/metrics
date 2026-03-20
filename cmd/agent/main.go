package main

import (
	"metrics/internal/agent"
	"metrics/internal/config"
	"metrics/internal/logger"

	"go.uber.org/zap"
)

func main() {
	cfg := config.LoadAgentConfig()
	agent := agent.NewMetricaAgent(cfg)
	//инициализируем логгер
	if err := logger.Initialize(cfg.LogLevel); err != nil {
		logger.Log.Fatal("Failed to initialize logger", zap.Error(err))
	}
	agent.Run()

}
