package handler

import (
	"net/http"
	"strings"

	"github.com/tigranqic/metrics-tpl/internal/repository"
)

type Handler struct {
	store *repository.MemStorage
}

func NewHandler(store *repository.MemStorage) *Handler {
	return &Handler{store: store}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
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
