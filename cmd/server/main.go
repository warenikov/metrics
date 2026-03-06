package main

import (
	"log"
	"metrics/internal/config"
	"metrics/internal/handler"
	storage "metrics/internal/repository"
	"metrics/internal/server"
	"metrics/internal/service"
)

func main() {
	//получить конфиг
	cfg := config.LoadConfig()

	repo := storage.NewMemStorage()
	svc := service.NewMetricsService(repo)
	h := handler.NewHandler(svc)

	//запустить сервер с конфигом
	err := server.Start(cfg, h)
	if err != nil {
		log.Fatal(err)
	}
}
