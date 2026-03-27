package main

import (
	"context"
	"metrics/internal/config"
	"metrics/internal/handler"
	"metrics/internal/logger"
	"metrics/internal/repository"
	"metrics/internal/server"
	"metrics/internal/service"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
)

func main() {
	cfg, err := config.LoadServerConfig()
	if err != nil {
		logger.Log.Fatal("Invalid config", zap.Error(err))
	}

	if err := logger.Initialize(cfg.LogLevel); err != nil {
		logger.Log.Fatal("Failed to initialize logger", zap.Error(err))
	}

	mem := repository.NewMemStorage()
	repo, e := repository.NewFileBackedRepo(mem, cfg.FileStoragePath, cfg.StoreInterval, cfg.Restore)
	if e != nil {
		logger.Log.Fatal("Failed to initialize repository")
	}

	if cfg.StoreInterval > 0 {
		go repo.RunSave()
	}

	svc := service.NewMetricsService(repo)
	h := handler.NewHandler(svc)
	srv := server.New(cfg, h)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := srv.Start(); err != nil {
			logger.Log.Fatal("Failed to start server", zap.Error(err))
		}
	}()
	logger.Log.Info("Server started", zap.String("host", cfg.ServerAddr))
	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Log.Error("Failed to shutdown server", zap.Error(err))
	}

	if err := repo.Save(); err != nil {
		logger.Log.Error("Failed to save metrics to file", zap.Error(err))
	}

	if err := repo.Close(); err != nil {
		logger.Log.Error("Failed to close repository", zap.Error(err))
	}
}
