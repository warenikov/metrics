package server

import (
	"metrics/internal/config"
	"metrics/internal/logger"
	"metrics/internal/middleware"
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

	r.Use(middleware.LoggerMiddleware)

	r.Post("/update/{type}/{name}/{value}", h.Update)

	r.Get("/value/{type}/{name}", h.GetMetrica)
	r.Get("/", h.GetMetricsList)

	addr := cfg.ServerAddr

	logger.Log.Info("Starting server on http://" + addr)

	if err := http.ListenAndServe(addr, r); err != nil {
		return err
	}

	return nil
}
