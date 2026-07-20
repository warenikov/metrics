// Package server wires the HTTP layer of the metrics server: routing, middleware,
// and the adapter between chi handlers and the service interfaces.
package server

import (
	"context"
	"crypto/rsa"
	"errors"
	"net"
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

// MetricsUpdater is the interface for writing metrics to the store.
type MetricsUpdater interface {
	// ParseAndSave validates and stores a single metric identified by type, id, and string value.
	ParseAndSave(ctx context.Context, mType, id, value string) (models.Metrics, error)
	// UpdateBatch atomically stores a slice of metrics.
	UpdateBatch(ctx context.Context, metrics []models.Metrics) error
}

// MetricsGetter is the interface for reading metrics from the store.
type MetricsGetter interface {
	// GetMetrica returns a single metric by type and id.
	GetMetrica(ctx context.Context, mType, id string) (*models.Metrics, error)
	// GetListMetrics returns all stored metrics.
	GetListMetrics(ctx context.Context) ([]models.Metrics, error)
}

// HealthChecker reports the availability of the underlying data store.
type HealthChecker interface {
	// PingDB returns nil if the database is reachable.
	PingDB() error
}

// Server wraps the standard library HTTP server with a pre-configured chi router.
type Server struct {
	httpServer *http.Server
}

// New creates a Server with a chi router, request logging, gzip compression,
// and optional HMAC verification, RSA decryption, and trusted-subnet
// middleware.
// broker may be nil, in which case audit logging is disabled.
// privKey may be nil, in which case incoming request bodies are assumed to
// be unencrypted.
// trustedSubnet may be nil, in which case requests are accepted regardless
// of their X-Real-IP header.
func New(cfg *config.Config, updater MetricsUpdater, getter MetricsGetter, health HealthChecker, broker *audit.Broker, privKey *rsa.PrivateKey, trustedSubnet *net.IPNet) *Server {
	r := chi.NewRouter()

	r.Use(middleware.LoggerMiddleware)
	if trustedSubnet != nil {
		r.Use(middleware.TrustedSubnetMiddleware(trustedSubnet))
	}
	if privKey != nil {
		r.Use(middleware.CryptoMiddleware(privKey))
	}
	r.Use(middleware.GzipMiddleware)
	if cfg.Key != "" {
		r.Use(middleware.HashMiddleware(cfg.Key))
	}

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

// Handler returns the HTTP handler used by the server.
// Useful for testing with httptest.NewServer.
func (s *Server) Handler() http.Handler {
	return s.httpServer.Handler
}

// Start begins listening for incoming HTTP requests. It blocks until the server
// is shut down; [http.ErrServerClosed] is treated as a clean shutdown and not returned.
func (s *Server) Start() error {
	logger.Log.Debug("Starting server on http://" + s.httpServer.Addr)
	if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Shutdown gracefully stops the server, waiting for active connections to finish
// within the deadline imposed by ctx.
func (s *Server) Shutdown(ctx context.Context) error {
	logger.Log.Info("Shutting down server...")
	return s.httpServer.Shutdown(ctx)
}
