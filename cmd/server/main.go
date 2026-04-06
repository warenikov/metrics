package main

import (
	"context"
	"metrics/internal/config"
	"metrics/internal/config/db"
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

	var (
		repo     service.Repository
		fileRepo *repository.FileBackedRepo
		pgxDB    *db.PgxDB
	)

	if cfg.DbDNS != "" {
		var dbErr error
		pgxDB, dbErr = db.Connect(cfg.DbDNS)
		if dbErr != nil {
			logger.Log.Fatal("Failed to connect to database", zap.Error(dbErr))
		}
		defer pgxDB.Conn.Close()

		pgRepo, pgErr := repository.NewPostgresRepo(pgxDB.Conn)
		if pgErr != nil {
			logger.Log.Fatal("Failed to initialize postgres repository", zap.Error(pgErr))
		}
		repo = pgRepo
		logger.Log.Info("Using PostgreSQL repository")
	} else if cfg.FileStoragePath != "" {
		mem := repository.NewMemStorage()
		var fileErr error
		fileRepo, fileErr = repository.NewFileBackedRepo(mem, cfg.FileStoragePath, cfg.StoreInterval, cfg.Restore)
		if fileErr != nil {
			logger.Log.Fatal("Failed to initialize file repository", zap.Error(fileErr))
		}
		if cfg.StoreInterval > 0 {
			go fileRepo.RunSave()
		}
		repo = fileRepo
		logger.Log.Info("Using file-backed repository", zap.String("path", cfg.FileStoragePath))
	} else {
		repo = repository.NewMemStorage()
		logger.Log.Info("Using in-memory repository")
	}

	svc := service.NewMetricsService(repo, pgxDB)
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

	if fileRepo != nil {
		if err := fileRepo.Save(); err != nil {
			logger.Log.Error("Failed to save metrics to file", zap.Error(err))
		}
		if err := fileRepo.Close(); err != nil {
			logger.Log.Error("Failed to close repository", zap.Error(err))
		}
	}
}
