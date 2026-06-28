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

// Set at build time via -ldflags "-X main.buildVersion=v1.0.0 -X main.buildDate=... -X main.buildCommit=..."
var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

func printBuildInfo() {
	na := func(s string) string {
		if s == "" {
			return "N/A"
		}
		return s
	}
	fmt.Printf("Build version: %s\n", na(buildVersion))
	fmt.Printf("Build date: %s\n", na(buildDate))
	fmt.Printf("Build commit: %s\n", na(buildCommit))
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
