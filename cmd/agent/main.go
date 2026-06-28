package main

import (
	"context"
	"metrics/internal/agent"
	"metrics/internal/config"
	"metrics/internal/logger"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
)

func main() {
	cfg, err := config.LoadAgentConfig()
	if err != nil {
		logger.Log.Fatal("Invalid config", zap.Error(err))
	}
	if err := logger.Initialize(cfg.LogLevel); err != nil {
		logger.Log.Fatal("Failed to initialize logger", zap.Error(err))
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	a := agent.NewMetricaAgent(cfg)
	a.Run(ctx)
}
