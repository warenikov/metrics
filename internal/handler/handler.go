package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"metrics/internal/logger"
	models "metrics/internal/model"
	"metrics/internal/service"
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
		h.errorProcess(err, w)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) UpdateJson(w http.ResponseWriter, r *http.Request) {
	var metrica models.Metrics

	err := json.NewDecoder(r.Body).Decode(&metrica)
	if err != nil {
		logger.Log.Error(fmt.Sprintf("Failed to decode JSON: %v", err))
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	deltaVal := "nil"
	if metrica.Delta != nil {
		deltaVal = fmt.Sprintf("%d", *metrica.Delta)
	}

	valueVal := "nil"
	if metrica.Value != nil {
		valueVal = fmt.Sprintf("%f", *metrica.Value)
	}

	logger.Log.Info(fmt.Sprintf(
		"UPDATE: ID=%s, Type=%s, Delta=%s, Value=%s",
		metrica.ID, metrica.MType, deltaVal, valueVal,
	))
	// ------------------------------------

	valStr := metrica.ValueString()
	_, err = h.svc.ParseAndSave(metrica.MType, metrica.ID, valStr)
	if err != nil {
		h.errorProcess(err, w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

}

func (h *Handler) GetMetricaJson(w http.ResponseWriter, r *http.Request) {
	var metrica models.Metrics

	err := json.NewDecoder(r.Body).Decode(&metrica)
	if err != nil {
		logger.Log.Error(fmt.Sprintf("GetMetricaJson: Decode error: %v", err))
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	logger.Log.Info(fmt.Sprintf("GetMetricaJson: Request for ID=%s, Type=%s", metrica.ID, metrica.MType))

	foundMetrica, err := h.svc.GetMetrica(metrica.MType, metrica.ID)
	if err != nil {
		logger.Log.Warn(fmt.Sprintf("GetMetricaJson: Metric %s not found or error: %v", metrica.ID, err))
		h.errorProcess(err, w)
		return
	}

	dVal, vVal := "nil", "nil"
	if foundMetrica.Delta != nil {
		dVal = fmt.Sprintf("%d", *foundMetrica.Delta)
	}
	if foundMetrica.Value != nil {
		vVal = fmt.Sprintf("%f", *foundMetrica.Value)
	}

	logger.Log.Info(fmt.Sprintf("GetMetricaJson: Found in storage: ID=%s, Delta=%s, Value=%s",
		foundMetrica.ID, dVal, vVal))

	resp, err := json.Marshal(foundMetrica)
	if err != nil {
		logger.Log.Error(fmt.Sprintf("GetMetricaJson: Marshal error: %v", err))
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	logger.Log.Info(fmt.Sprintf("GetMetricaJson: Sending JSON: %s", string(resp)))

	w.Header().Set("Content-Type", models.ContentTypeJson)
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
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
		h.errorProcess(err, w)
		return
	}

	result := mm.ValueString()

	w.Header().Set("Content-Type", models.ContentTypeText)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(result))
}

func (h *Handler) errorProcess(err error, w http.ResponseWriter) {
	switch {
	case errors.Is(err, service.ErrTypeMetric) || errors.Is(err, service.ErrInvalidValue):
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
	case errors.Is(err, service.ErrMetricNotFound):
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
	default:
		logger.Log.Error(err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}
