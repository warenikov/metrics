package handler

import (
	"errors"
	"fmt"
	models "metrics/internal/model"
	"metrics/internal/service"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

type MetricsHandler interface {
	Update(w http.ResponseWriter, r *http.Request)
	GetMetrica(w http.ResponseWriter, r *http.Request)
	GetMetricsList(w http.ResponseWriter, r *http.Request)
}

type handler struct {
	svc service.Service
}

func (h *handler) GetMetricsList(w http.ResponseWriter, r *http.Request) {
	metrics, err := h.svc.GetListMetrics()
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	var sb strings.Builder
	sb.WriteString("<html><head><title>Metrics</title></head><body>")
	sb.WriteString("<h1>Current Metrics</h1><ul>")

	for _, m := range metrics {
		fmt.Fprintf(&sb, "<li>%s (%s): %s</li>", m.ID, m.MType, m.ValueString())
	}

	sb.WriteString("</ul></body></html>")

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(sb.String()))
}

func NewHandler(svc service.Service) MetricsHandler {
	return &handler{svc: svc}
}

func (h *handler) Update(w http.ResponseWriter, r *http.Request) {
	//if r.Header.Get("Content-Type") != models.ContentTypeText {
	//	http.Error(w, "Content type not supported", http.StatusBadRequest)
	//	return
	//}

	mType := chi.URLParam(r, "type")
	mName := chi.URLParam(r, "name")
	mValue := chi.URLParam(r, "value")

	if mName == "" {
		http.Error(w, "Metric name is missing", http.StatusBadRequest)
		return
	}

	if mType == "" {
		http.Error(w, "Metric type is missing", http.StatusBadRequest)
		return
	}

	if mValue == "" {
		http.Error(w, "Metric value is missing", http.StatusBadRequest)
		return
	}

	_, err := h.svc.ParseAndSave(mType, mName, mValue)
	if err != nil {
		switch {
		case errors.Is(err, models.ErrInvalidMetricType):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, models.ErrInvalidValue):
			http.Error(w, "Bad request: value must be a number", http.StatusBadRequest)
		default:
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *handler) GetMetrica(w http.ResponseWriter, r *http.Request) {
	mType := chi.URLParam(r, "type")
	mName := chi.URLParam(r, "name")

	if mName == "" {
		http.Error(w, "Metric name is missing", http.StatusBadRequest)
		return
	}

	if mType == "" {
		http.Error(w, "Metric type is missing", http.StatusBadRequest)
		return
	}

	mm, e := h.svc.GetMetrica(mType, mName)

	if e != nil {
		switch {
		case errors.Is(e, models.ErrInvalidMetricType):
			http.Error(w, e.Error(), http.StatusBadRequest)
		case errors.Is(e, models.ErrInvalidValue):
			http.Error(w, "Bad request", http.StatusBadRequest)
		case errors.Is(e, models.ErrMetricNotFound):
			http.Error(w, "Bad request", http.StatusNotFound)
		default:
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	result := mm.ValueString()

	w.Header().Set("Content-Type", models.ContentTypeText)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(result))
}
