package main

import (
	"context"
	"fmt"
	"metrics/internal/agent"
	"metrics/internal/config"
	"metrics/internal/logger"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
)

// Set at build time via -ldflags, see README.md.
var (
	buildVersion = "N/A"
	buildDate    = "N/A"
	buildCommit  = "N/A"
)

func printBuildInfo() {
	fmt.Printf("Build version: %s\n", buildVersion)
	fmt.Printf("Build date: %s\n", buildDate)
	fmt.Printf("Build commit: %s\n", buildCommit)
}

func main() {
	printBuildInfo()
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
