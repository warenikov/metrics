package server

import (
	"encoding/json"
	"errors"
	"html/template"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
	models "metrics/internal/model"
	"metrics/internal/logger"
	"metrics/internal/service"
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

type httpAdapter struct {
	updater MetricsUpdater
	getter  MetricsGetter
	health  HealthChecker
}

func (h *httpAdapter) update(w http.ResponseWriter, r *http.Request) {
	mType := chi.URLParam(r, "type")
	mName := chi.URLParam(r, "name")
	mValue := chi.URLParam(r, "value")

	if mName == "" || mValue == "" || mType == "" {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if _, err := h.updater.ParseAndSave(r.Context(), mType, mName, mValue); err != nil {
		writeError(err, w)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *httpAdapter) updateJSON(w http.ResponseWriter, r *http.Request) {
	var metrica models.Metrics

	if err := json.NewDecoder(r.Body).Decode(&metrica); err != nil {
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
	)

	valStr := metrica.ValueString()
	if _, err := h.updater.ParseAndSave(r.Context(), metrica.MType, metrica.ID, valStr); err != nil {
		writeError(err, w)
		return
	}

	m, err := h.getter.GetMetrica(r.Context(), metrica.MType, metrica.ID)
	if err != nil {
		writeError(err, w)
		return
	}

	resp, err := json.Marshal(m)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func (h *httpAdapter) getMetricaJSON(w http.ResponseWriter, r *http.Request) {
	var metrica models.Metrics

	if err := json.NewDecoder(r.Body).Decode(&metrica); err != nil {
		logger.Log.Debug("GetMetricaJson: Decode error", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	logger.Log.Debug("GetMetricaJson: Request for",
		zap.String("ID", metrica.ID),
		zap.String("Type", metrica.MType),
	)

	found, err := h.getter.GetMetrica(r.Context(), metrica.MType, metrica.ID)
	if err != nil {
		logger.Log.Debug("GetMetricaJson error",
			zap.String("Metrica name", metrica.ID),
			zap.Error(err),
		)
		writeError(err, w)
		return
	}

	resp, err := json.Marshal(found)
	if err != nil {
		logger.Log.Debug("GetMetricaJson: Marshal error", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", models.ContentTypeJSON)
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func (h *httpAdapter) getMetrica(w http.ResponseWriter, r *http.Request) {
	mType := chi.URLParam(r, "type")
	mName := chi.URLParam(r, "name")

	if mName == "" || mType == "" {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	mm, err := h.getter.GetMetrica(r.Context(), mType, mName)
	if err != nil {
		writeError(err, w)
		return
	}

	w.Header().Set("Content-Type", models.ContentTypeText)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(mm.ValueString()))
}

func (h *httpAdapter) updateBatch(w http.ResponseWriter, r *http.Request) {
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

	if err := h.updater.UpdateBatch(r.Context(), metrics); err != nil {
		writeError(err, w)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *httpAdapter) getMetricsList(w http.ResponseWriter, r *http.Request) {
	metrics, err := h.getter.GetListMetrics(r.Context())
	if err != nil {
		logger.Log.Error("Can't get metrics", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err = tmpl.Execute(w, metrics); err != nil {
		logger.Log.Error("Can't execute template", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func (h *httpAdapter) pingDB(w http.ResponseWriter, r *http.Request) {
	if err := h.health.PingDB(); err != nil {
		writeError(err, w)
		return
	}

	w.Header().Set("Content-Type", models.ContentTypeText)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func writeError(err error, w http.ResponseWriter) {
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
