package handler

import (
	"net/http"
	"strings"

	"github.com/tigranqic/metrics-tpl/internal/repository"
)

type Handler struct {
	store repository.Storage
}

func NewHandler(store repository.Storage) *Handler {
	return &Handler{store: store}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/ping" && r.Method == http.MethodGet {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !strings.HasPrefix(r.URL.Path, "/update/") {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/update/"), "/")
	if len(parts) != 3 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	metricType, name, value := parts[0], parts[1], parts[2]

	if name == "" {
		http.Error(w, "metric name not found", http.StatusNotFound)
		return
	}

	if metricType != "gauge" && metricType != "counter" {
		http.Error(w, "invalid metric type", http.StatusBadRequest)
		return
	}

	if err := h.store.Update(metricType, name, value); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}
