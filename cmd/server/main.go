package main

import (
	"metrics/internal/config"
	"metrics/internal/handler"
	"metrics/internal/logger"
	storage "metrics/internal/repository"
	"metrics/internal/server"
	"metrics/internal/service"

	"go.uber.org/zap"
)

func main() {
	//получить конфиг
	cfg := config.LoadConfig()

	repo := storage.NewMemStorage()
	svc := service.NewMetricsService(repo)
	h := handler.NewHandler(svc)

	//инициализируем логгер
	if err := logger.Initialize(cfg.LogLevel); err != nil {
		logger.Log.Fatal("Failed to initialize logger", zap.Error(err))
	}

	//запустить сервер с конфигом
	err := server.Start(cfg, h)
	if err != nil {
		logger.Log.Fatal("Failed to server", zap.Error(err))
	}
}
