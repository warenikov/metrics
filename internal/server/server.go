package server

import (
	"log"
	"metrics/internal/config"
	"metrics/internal/service"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	svc service.MetricsService
}

type MetricsHandler interface {
	Update(w http.ResponseWriter, r *http.Request)
	GetMetrica(w http.ResponseWriter, r *http.Request)
	GetMetricsList(w http.ResponseWriter, r *http.Request)
}

func Start(cfg *config.Config, h MetricsHandler) error {
	r := chi.NewRouter()
	r.Post("/update/{type}/{name}/{value}", h.Update)

	r.Get("/value/{type}/{name}", h.GetMetrica)
	r.Get("/", h.GetMetricsList)

	addr := cfg.ServerAddr

	log.Printf("Try start server on http://%s", addr)

	if err := http.ListenAndServe(addr, r); err != nil {
		return err
	}

	return nil
}
