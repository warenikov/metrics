package handler

import (
	"errors"
	"html/template"
	"log"
	models "metrics/internal/model"
	"net/http"

	"github.com/go-chi/chi/v5"
)

const metricsTemplate = `
<!DOCTYPE html>
<html>
<head>
    <title>Metrics</title>
</head>
<body>
    <h1>Current Metrics</h1>
    <ul>
        {{range .}}
        <li>{{.ID}} ({{.MType}}): {{.ValueString}}</li>
        {{end}}
    </ul>
</body>
</html>`

var tmpl = template.Must(template.New("metrics").Parse(metricsTemplate))

type Service interface {
	ParseAndSave(mType, id, value string) (models.Metrics, error)
	GetMetrica(mType, id string) (*models.Metrics, error)
	GetListMetrics() ([]models.Metrics, error)
}

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) GetMetricsList(w http.ResponseWriter, r *http.Request) {
	metrics, err := h.svc.GetListMetrics()
	if err != nil {
		log.Printf("Failed to get metrics list: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err = tmpl.Execute(w, metrics); err != nil {
		log.Printf("failed to execute template: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	mType := chi.URLParam(r, "type")
	mName := chi.URLParam(r, "name")
	mValue := chi.URLParam(r, "value")

	if mName == "" || mValue == "" || mType == "" {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	_, err := h.svc.ParseAndSave(mType, mName, mValue)
	if err != nil {
		switch {
		case errors.Is(err, models.ErrInvalidMetricType) || errors.Is(err, models.ErrInvalidValue):
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		case errors.Is(err, models.ErrMetricNotFound):
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		default:
			log.Printf("Internal Server Error %v", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) GetMetrica(w http.ResponseWriter, r *http.Request) {
	mType := chi.URLParam(r, "type")
	mName := chi.URLParam(r, "name")

	if mName == "" || mType == "" {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	mm, err := h.svc.GetMetrica(mType, mName)

	if err != nil {
		switch {
		case errors.Is(err, models.ErrInvalidMetricType) || errors.Is(err, models.ErrInvalidValue):
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		case errors.Is(err, models.ErrMetricNotFound):
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		default:
			log.Printf("Internal Server Error %v", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	result := mm.ValueString()

	w.Header().Set("Content-Type", models.ContentTypeText)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(result))
}
