package server

import (
	"errors"
	"fmt"
	"log"
	"metrics/internal/config"
	models "metrics/internal/model"
	"metrics/internal/service"
	"net/http"
)

type Handler struct {
	svc service.Service
}

func MustStart(cfg *config.Config, svc service.Service) {
	h := &Handler{svc: svc}

	mux := http.NewServeMux()

	mux.HandleFunc("POST /update/{type}/{name}/{value}", h.UpdateHandler)

	addr := fmt.Sprintf("%s:%s", cfg.ServerAddr, cfg.ServerPort)

	log.Printf("Запуск сервера на http://%s", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Ошибка сервера: %v", err)
	}
}

func (h *Handler) UpdateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != models.ContentTypeText {
		http.Error(w, "Content type not supported", http.StatusBadRequest)
		return
	}

	mType := r.PathValue("type")
	mName := r.PathValue("name")
	mValue := r.PathValue("value")

	if mName == "" {
		http.Error(w, "Metric name is missing", http.StatusNotFound)
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
