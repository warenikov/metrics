package main

import (
	"context"
	"fmt"
	"metrics/internal/audit"
	"metrics/internal/config"
	"metrics/internal/config/db"
	"metrics/internal/logger"
	"metrics/internal/repository"
	"metrics/internal/server"
	"metrics/internal/service"
	"metrics/migrations"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	if cfg.DBDSN != "" {
		var dbErr error
		pgxDB, dbErr = db.Connect(cfg.DBDSN)
		if dbErr != nil {
			logger.Log.Fatal("Failed to connect to database", zap.Error(dbErr))
		}
		defer func() { _ = pgxDB.Conn.Close() }()

		pgRepo, pgErr := repository.NewPostgresRepo(pgxDB.Conn, migrations.FS)
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

	var auditObservers []audit.Observer
	var fileObserver *audit.FileObserver
	if cfg.AuditFile != "" {
		fo, foErr := audit.NewFileObserver(cfg.AuditFile)
		if foErr != nil {
			logger.Log.Fatal("Failed to open audit file", zap.Error(foErr))
		}
		fileObserver = fo
		auditObservers = append(auditObservers, fo)
	}
	if cfg.AuditURL != "" {
		auditObservers = append(auditObservers, audit.NewHTTPObserver(cfg.AuditURL))
	}
	var broker *audit.Broker
	if len(auditObservers) > 0 {
		broker = audit.NewBroker(auditObservers...)
		logger.Log.Info("Audit enabled", zap.Int("sinks", len(auditObservers)))
	}

	srv := server.New(cfg, svc, svc, svc, broker)

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
	if broker != nil {
		broker.Close()
	}
	if fileObserver != nil {
		if err := fileObserver.Close(); err != nil {
			logger.Log.Error("Failed to close audit file", zap.Error(err))
		}
	}
}
