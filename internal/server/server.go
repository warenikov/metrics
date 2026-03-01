package server

import (
	"fmt"
	"log"
	"metrics/internal/config"
	"metrics/internal/handler"
	"metrics/internal/service"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	svc service.Service
}

func MustStart(cfg *config.Config, h handler.MetricsHandler) {
	r := chi.NewRouter()
	r.Post("/update/{type}/{name}/{value}", h.Update)

	r.Get("/value/{type}/{name}", h.GetMetrica)
	r.Get("/", h.GetMetricsList)

	addr := fmt.Sprintf("%s:%s", cfg.ServerAddr, cfg.ServerPort)

	log.Printf("Запуск сервера на http://%s", addr)

	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("Ошибка сервера: %v", err)
		panic(err)
	}
}
