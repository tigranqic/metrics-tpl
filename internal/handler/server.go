package handler

import (
	"html/template"
	"net/http"
	"strconv"

	"github.com/tigranqic/metrics-tpl/internal/middleware"
	"github.com/tigranqic/metrics-tpl/internal/repository"
	"go.uber.org/zap"

	"encoding/json"
	"io"

	"github.com/go-chi/chi/v5"

	"database/sql"

	models "github.com/tigranqic/metrics-tpl/internal/model"
)

type Handler struct {
	store repository.Storage
	db    *sql.DB
	log   *zap.Logger
	key   string
}

func NewHandler(store repository.Storage, db *sql.DB, log *zap.Logger, key string) *Handler {
	return &Handler{
		store: store,
		db:    db,
		log:   log,
		key:   key,
	}
}

func (h *Handler) Router() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.GzipDecompress)
	r.Use(middleware.NewHashMiddleware(h.key, h.log).Handle)
	r.Use(middleware.GzipCompress)

	r.Get("/", h.listMetricsHandler)
	r.Get("/ping", h.pingHandler)
	r.Get("/health", h.healthHandler)
	r.Get("/value/{type}/{name}", h.getMetricValueHandler)
	r.Post("/update/{type}/{name}/{value}", h.updateMetricHandler)
	r.Post("/updates/", h.updateMetricsBatchHandler)
	r.Post("/update/", h.updateMetricJSONHandler)
	r.Post("/value/", h.getMetricValueJSONHandler)
	r.Post("/update", h.updateMetricJSONHandler)
	r.Post("/value", h.getMetricValueJSONHandler)

	return r
}

func (h *Handler) pingHandler(w http.ResponseWriter, r *http.Request) {
	if err := h.db.Ping(); err != nil {
		h.log.Error("DB connection erro (pingHandler)", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) updateMetricHandler(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	name := chi.URLParam(r, "name")
	value := chi.URLParam(r, "value")

	if name == "" {
		h.log.Error("metric name not found", zap.String("name", name))
		http.Error(w, "metric name not found", http.StatusNotFound)
		return
	}

	if metricType != "gauge" && metricType != "counter" {
		h.log.Error("invalid metric type (updateMetricHandler)", zap.String("type", metricType))
		http.Error(w, "invalid metric type", http.StatusBadRequest)
		return
	}

	switch metricType {
	case "gauge":
		if _, err := strconv.ParseFloat(value, 64); err != nil {
			h.log.Error("invalid gauge value (updateMetricHandler)", zap.Error(err))
			http.Error(w, "invalid gauge value", http.StatusBadRequest)
			return
		}
	case "counter":
		if _, err := strconv.ParseInt(value, 10, 64); err != nil {
			h.log.Error("invalid counter value (updateMetricHandler)", zap.Error(err))
			http.Error(w, "invalid counter value", http.StatusBadRequest)
			return
		}
	}

	if err := h.store.Update(metricType, name, value); err != nil {
		h.log.Error("internal server error (updateMetricHandler)", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) getMetricValueHandler(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	name := chi.URLParam(r, "name")

	var result string
	switch metricType {
	case "gauge":
		val, err := h.store.GetGauge(name)
		if err != nil {
			h.log.Error("metric not found (getMetricValueHandler)", zap.Error(err))
			http.Error(w, "metric not found", http.StatusNotFound)
			return
		}
		result = strconv.FormatFloat(val, 'f', -1, 64)
	case "counter":
		val, err := h.store.GetCounter(name)
		if err != nil {
			h.log.Error("metric not found (getMetricValueHandler)", zap.Error(err))
			http.Error(w, "metric not found", http.StatusNotFound)
			return
		}
		result = strconv.FormatInt(val, 10)
	default:
		h.log.Error("invalid metric type (getMetricValueHandler)", zap.String("type", metricType))
		http.Error(w, "invalid metric type", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(result))
}

func (h *Handler) updateMetricsBatchHandler(w http.ResponseWriter, r *http.Request) {
	var metrics []models.Metrics

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.log.Error("failed to read body (batch)", zap.Error(err))
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}
	defer func() {
		_ = r.Body.Close()
	}()

	if err := json.Unmarshal(body, &metrics); err != nil {
		h.log.Error("failed to decode JSON (batch)", zap.Error(err))
		http.Error(w, "failed to decode JSON", http.StatusBadRequest)
		return
	}

	if len(metrics) == 0 {
		h.log.Error("empty batch", zap.String("metrics len", strconv.Itoa(len(metrics))))
		http.Error(w, "empty batch", http.StatusBadRequest)
		return
	}

	if err := h.store.UpdateBatch(metrics); err != nil {
		h.log.Error("failed to update metrics batch", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) listMetricsHandler(w http.ResponseWriter, r *http.Request) {
	type Metric struct {
		ID    string
		MType string
		Value string
	}

	all, err := h.store.GetAll()
	if err != nil {
		h.log.Error("failed to retrieve metrics", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	var metrics []Metric

	for _, m := range all {
		switch m.MType {
		case "gauge":
			if m.Value != nil {
				metrics = append(metrics, Metric{
					ID:    m.ID,
					MType: m.MType,
					Value: strconv.FormatFloat(*m.Value, 'f', -1, 64),
				})
			}
		case "counter":
			if m.Delta != nil {
				metrics = append(metrics, Metric{
					ID:    m.ID,
					MType: m.MType,
					Value: strconv.FormatInt(*m.Delta, 10),
				})
			}
		}
	}

	tmpl := `
<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"><title>Metrics</title></head>
<body>
	<h1>Metrics</h1>
	<ul>
		{{range .}}
			<li>{{.ID}} ({{.MType}}): {{.Value}}</li>
		{{else}}
			<li>No metrics found</li>
		{{end}}
	</ul>
</body>
</html>
`
	t := template.Must(template.New("metrics").Parse(tmpl))

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if err := t.Execute(w, metrics); err != nil {
		h.log.Error("failed to render template", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func (h *Handler) updateMetricJSONHandler(w http.ResponseWriter, r *http.Request) {
	var m models.Metrics
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.log.Error("failed to read body", zap.Error(err))
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}
	defer func() {
		_ = r.Body.Close()
	}()

	if err := json.Unmarshal(body, &m); err != nil {
		h.log.Error("failed to decode JSON", zap.Error(err))
		http.Error(w, "failed to decode JSON", http.StatusBadRequest)
		return
	}

	if m.ID == "" || (m.MType != "gauge" && m.MType != "counter") {
		h.log.Error("invalid metric data", zap.Error(err))
		http.Error(w, "invalid metric data", http.StatusBadRequest)
		return
	}

	var valStr string
	switch m.MType {
	case "gauge":
		if m.Value == nil {
			h.log.Error("missing gauge value", zap.Error(err))
			http.Error(w, "missing gauge value", http.StatusBadRequest)
			return
		}
		valStr = strconv.FormatFloat(*m.Value, 'f', -1, 64)
	case "counter":
		if m.Delta == nil {
			h.log.Error("missing counter delta", zap.Error(err))
			http.Error(w, "missing counter delta", http.StatusBadRequest)
			return
		}
		valStr = strconv.FormatInt(*m.Delta, 10)
	}

	if err := h.store.Update(m.MType, m.ID, valStr); err != nil {
		h.log.Error("failed to update metric", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(m)
}

func (h *Handler) getMetricValueJSONHandler(w http.ResponseWriter, r *http.Request) {
	var req models.Metrics
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Error("invalid JSON (getMetricValueJSONHandler)", zap.Error(err))
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	defer func() {
		_ = r.Body.Close()
	}()

	if req.ID == "" || (req.MType != "gauge" && req.MType != "counter") {
		h.log.Error("invalid request data (getMetricValueJSONHandler)")
		http.Error(w, "invalid request data", http.StatusBadRequest)
		return
	}

	resp := models.Metrics{ID: req.ID, MType: req.MType}

	switch req.MType {
	case "gauge":
		val, err := h.store.GetGauge(req.ID)
		if err != nil {
			h.log.Error("metric not found gauge (getMetricValueJSONHandler)", zap.Error(err))
			http.Error(w, "metric not found", http.StatusNotFound)
			return
		}
		resp.Value = &val
	case "counter":
		val, err := h.store.GetCounter(req.ID)
		if err != nil {
			h.log.Error("metric not found counter (getMetricValueJSONHandler)", zap.Error(err))
			http.Error(w, "metric not found", http.StatusNotFound)
			return
		}
		resp.Delta = &val
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}
