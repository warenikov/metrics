package main

import (
	"context"
	"crypto/rsa"
	"fmt"
	"metrics/internal/agent"
	"metrics/internal/config"
	"metrics/internal/logger"
	pb "metrics/internal/proto"
	"metrics/pkg/crypto"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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
	var pubKey *rsa.PublicKey
	if cfg.CryptoKeyPath != "" {
		key, keyErr := crypto.LoadPublicKey(cfg.CryptoKeyPath)
		if keyErr != nil {
			logger.Log.Fatal("Invalid crypto key", zap.Error(keyErr))
		}
		pubKey = key
	}

	var grpcClient pb.MetricsClient
	if cfg.GRPCAddr != "" {
		conn, connErr := grpc.NewClient(cfg.GRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if connErr != nil {
			logger.Log.Fatal("Failed to create gRPC client", zap.Error(connErr))
		}
		defer func() { _ = conn.Close() }()
		grpcClient = pb.NewMetricsClient(conn)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	a := agent.NewMetricaAgent(cfg, pubKey, grpcClient)
	a.Run(ctx)
}
