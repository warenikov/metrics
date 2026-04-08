package server

import (
	"context"
	"errors"
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
	UpdateBatch(w http.ResponseWriter, r *http.Request)
	GetMetrica(w http.ResponseWriter, r *http.Request)
	GetMetricaJSON(w http.ResponseWriter, r *http.Request)
	GetMetricsList(w http.ResponseWriter, r *http.Request)
	PingDB(w http.ResponseWriter, r *http.Request)
}

type Server struct {
	httpServer *http.Server
}

func New(cfg *config.Config, h MetricsHandler) *Server {
	r := chi.NewRouter()

	r.Use(middleware.LoggerMiddleware)
	r.Use(middleware.GzipMiddleware)

	r.Post("/update/{type}/{name}/{value}", h.Update)
	r.Post("/update/", h.UpdateJSON)
	r.Post("/updates/", h.UpdateBatch)
	r.Post("/value/", h.GetMetricaJSON)
	r.Get("/value/{type}/{name}", h.GetMetrica)
	r.Get("/", h.GetMetricsList)

	r.Get("/ping", h.PingDB)

	return &Server{
		httpServer: &http.Server{
			Addr:    cfg.ServerAddr,
			Handler: r,
		},
	}
}

func (s *Server) Start() error {
	logger.Log.Debug("Starting server on http://" + s.httpServer.Addr)
	if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	logger.Log.Info("Shutting down server...")
	return s.httpServer.Shutdown(ctx)
}
