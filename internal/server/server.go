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
	UpdateJSON(w http.ResponseWriter, r *http.Request)
	GetMetrica(w http.ResponseWriter, r *http.Request)
	GetMetricaJSON(w http.ResponseWriter, r *http.Request)
	GetMetricsList(w http.ResponseWriter, r *http.Request)
}

func Start(cfg *config.Config, h MetricsHandler) error {
	r := chi.NewRouter()

	r.Use(middleware.LoggerMiddleware)
	r.Use(middleware.GzipMiddleware)

	r.Post("/update/{type}/{name}/{value}", h.Update)
	r.Post("/update/", h.UpdateJSON)
	r.Post("/value/", h.GetMetricaJSON)

	r.Get("/value/{type}/{name}", h.GetMetrica)
	r.Get("/", h.GetMetricsList)

	addr := cfg.ServerAddr

	logger.Log.Debug("Starting server on http://" + addr)

	if err := http.ListenAndServe(addr, r); err != nil {
		return err
	}

	return nil
}
