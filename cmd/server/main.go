package main

import (
	"metrics/internal/config"
	storage "metrics/internal/repository"
	"metrics/internal/server"
	"metrics/internal/service"
)

func main() {
	//получить конфиг
	cfg := config.LoadConfig()

	repo := storage.NewMemStorage()
	svc := service.NewMetricsService(repo)

	//запустить сервер с конфигом
	server.MustStart(cfg, svc)
}
