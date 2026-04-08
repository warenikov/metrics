package handler

import (
	"context"
	"encoding/json"
	"errors"
	"html/template"
	"metrics/internal/logger"
	models "metrics/internal/model"
	"metrics/internal/service"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
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
	ParseAndSave(ctx context.Context, mType, id, value string) (models.Metrics, error)
	GetMetrica(ctx context.Context, mType, id string) (*models.Metrics, error)
	GetListMetrics(ctx context.Context) ([]models.Metrics, error)
	UpdateBatch(ctx context.Context, metrics []models.Metrics) error
	PingDB() error
}

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) GetMetricsList(w http.ResponseWriter, r *http.Request) {
	metrics, err := h.svc.GetListMetrics(r.Context())
	if err != nil {
		logger.Log.Error("Can't get metrics", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err = tmpl.Execute(w, metrics); err != nil {
		logger.Log.Error("Can't execute template", zap.Error(err))
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

	_, err := h.svc.ParseAndSave(r.Context(), mType, mName, mValue)
	if err != nil {
		h.errorProcess(err, w)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) UpdateJSON(w http.ResponseWriter, r *http.Request) {
	var metrica models.Metrics

	err := json.NewDecoder(r.Body).Decode(&metrica)
	if err != nil {
		logger.Log.Debug("Failed to decode JSON", zap.Error(err))
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	deltaVal := "nil"
	if metrica.Delta != nil {
		deltaVal = strconv.FormatInt(int64(*metrica.Delta), 10)
	}

	valueVal := "nil"
	if metrica.Value != nil {
		valueVal = strconv.FormatFloat(*metrica.Value, 'f', -1, 64)
	}
	logger.Log.Debug("Update:",
		zap.String("ID", metrica.ID),
		zap.String("Type", metrica.MType),
		zap.String("Delta", deltaVal),
		zap.String("Value", valueVal),
		zap.Error(err),
	)

	valStr := metrica.ValueString()
	_, err = h.svc.ParseAndSave(r.Context(), metrica.MType, metrica.ID, valStr)
	if err != nil {
		h.errorProcess(err, w)
		return
	}

	m, e := h.svc.GetMetrica(r.Context(), metrica.MType, metrica.ID)
	if e != nil {
		h.errorProcess(e, w)
		return
	}

	resp, e := json.Marshal(m)

	if e != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(resp)

}

func (h *Handler) GetMetricaJSON(w http.ResponseWriter, r *http.Request) {
	var metrica models.Metrics

	err := json.NewDecoder(r.Body).Decode(&metrica)
	if err != nil {
		logger.Log.Debug("GetMetricaJson: Decode error", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	logger.Log.Debug("GetMetricaJson: Request for",
		zap.String("ID", metrica.ID),
		zap.String("Type", metrica.MType),
	)
	foundMetrica, err := h.svc.GetMetrica(r.Context(), metrica.MType, metrica.ID)
	if err != nil {
		logger.Log.Debug("GetMetricaJson error",
			zap.String("Metrica name", metrica.ID),
			zap.Error(err),
		)
		h.errorProcess(err, w)
		return
	}

	dVal, vVal := "nil", "nil"
	if foundMetrica.Delta != nil {
		dVal = strconv.FormatInt(*foundMetrica.Delta, 10)
	}
	if foundMetrica.Value != nil {
		vVal = strconv.FormatFloat(*foundMetrica.Value, 'f', -1, 64)
	}

	logger.Log.Debug("GetMetricaJson: Found in storage: ",
		zap.String("ID", foundMetrica.ID),
		zap.String("delta", dVal),
		zap.String("val", vVal),
	)

	resp, err := json.Marshal(foundMetrica)
	if err != nil {
		logger.Log.Debug("GetMetricaJson: Marshal error",
			zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	logger.Log.Debug("GetMetricaJson: Sending JSON",
		zap.String("Json", string(resp)),
	)

	w.Header().Set("Content-Type", models.ContentTypeJSON)
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

	mm, err := h.svc.GetMetrica(r.Context(), mType, mName)

	if err != nil {
		h.errorProcess(err, w)
		return
	}

	result := mm.ValueString()

	w.Header().Set("Content-Type", models.ContentTypeText)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(result))
}

func (h *Handler) UpdateBatch(w http.ResponseWriter, r *http.Request) {
	var metrics []models.Metrics
	if err := json.NewDecoder(r.Body).Decode(&metrics); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if len(metrics) == 0 {
		w.WriteHeader(http.StatusOK)
		return
	}

	if err := h.svc.UpdateBatch(r.Context(), metrics); err != nil {
		h.errorProcess(err, w)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) PingDB(w http.ResponseWriter, r *http.Request) {
	err := h.svc.PingDB()
	if err != nil {
		h.errorProcess(err, w)
		return
	}

	w.Header().Set("Content-Type", models.ContentTypeText)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
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
