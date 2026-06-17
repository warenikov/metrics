package server

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"metrics/internal/audit"
	"metrics/internal/config"
	"metrics/internal/logger"
	"metrics/internal/middleware"
	models "metrics/internal/model"
)

const (
	readTimeout  = 5 * time.Second
	writeTimeout = 10 * time.Second
	idleTimeout  = 120 * time.Second
)

type MetricsUpdater interface {
	ParseAndSave(ctx context.Context, mType, id, value string) (models.Metrics, error)
	UpdateBatch(ctx context.Context, metrics []models.Metrics) error
}

type MetricsGetter interface {
	GetMetrica(ctx context.Context, mType, id string) (*models.Metrics, error)
	GetListMetrics(ctx context.Context) ([]models.Metrics, error)
}

type HealthChecker interface {
	PingDB() error
}

type Server struct {
	httpServer *http.Server
}

func New(cfg *config.Config, updater MetricsUpdater, getter MetricsGetter, health HealthChecker, broker *audit.Broker) *Server {
	r := chi.NewRouter()

	r.Use(middleware.LoggerMiddleware)
	r.Use(middleware.GzipMiddleware)
	r.Use(middleware.HashMiddleware(cfg.Key))

	h := &httpAdapter{updater: updater, getter: getter, health: health, broker: broker}

	r.Post("/update/{type}/{name}/{value}", h.update)
	r.Post("/update/", h.updateJSON)
	r.Post("/updates/", h.updateBatch)
	r.Post("/value/", h.getMetricaJSON)
	r.Get("/value/{type}/{name}", h.getMetrica)
	r.Get("/", h.getMetricsList)
	r.Get("/ping", h.pingDB)

	return &Server{
		httpServer: &http.Server{
			Addr:         cfg.ServerAddr,
			Handler:      r,
			ReadTimeout:  readTimeout,
			WriteTimeout: writeTimeout,
			IdleTimeout:  idleTimeout,
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
