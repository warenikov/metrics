package main

import (
	"metrics/internal/config"
	"metrics/internal/handler"
	"metrics/internal/logger"
	"metrics/internal/repository"
	"metrics/internal/server"
	"metrics/internal/service"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
)

func main() {
	cfg := config.LoadConfig()

	if err := logger.Initialize(cfg.LogLevel); err != nil {
		logger.Log.Fatal("Failed to initialize logger", zap.Error(err))
	}

	mem := repository.NewMemStorage()
	repo := repository.NewFileBackedRepo(mem, cfg.FileStoragePath, cfg.StoreInterval)

	if cfg.Restore {
		if err := repo.Load(); err != nil {
			logger.Log.Error("Failed to load metrics from file", zap.Error(err))
		}
	}

	if cfg.StoreInterval > 0 {
		go repo.RunSave()
	}

	svc := service.NewMetricsService(repo)
	h := handler.NewHandler(svc)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		if err := server.Start(cfg, h); err != nil {
			logger.Log.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	<-quit

	if err := repo.Save(); err != nil {
		logger.Log.Error("Failed to save metrics on shutdown", zap.Error(err))
	}
}
