package handler

import (
	"html/template"
	"net/http"
	"strconv"

	"github.com/tigranqic/metrics-tpl/internal/repository"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	store repository.Storage
}

func NewHandler(store repository.Storage) *Handler {
	return &Handler{store: store}
}

func (h *Handler) Router() http.Handler {
	r := chi.NewRouter()

	r.Get("/", h.listMetricsHandler)
	r.Get("/ping", h.pingHandler)
	r.Get("/value/{type}/{name}", h.getMetricValueHandler)
	r.Post("/update/{type}/{name}/{value}", h.updateMetricHandler)

	return r
}

func (h *Handler) pingHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) updateMetricHandler(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	name := chi.URLParam(r, "name")
	value := chi.URLParam(r, "value")

	if name == "" {
		http.Error(w, "metric name not found", http.StatusNotFound)
		return
	}

	if metricType != "gauge" && metricType != "counter" {
		http.Error(w, "invalid metric type", http.StatusBadRequest)
		return
	}

	switch metricType {
	case "gauge":
		if _, err := strconv.ParseFloat(value, 64); err != nil {
			http.Error(w, "invalid gauge value", http.StatusBadRequest)
			return
		}
	case "counter":
		if _, err := strconv.ParseInt(value, 10, 64); err != nil {
			http.Error(w, "invalid counter value", http.StatusBadRequest)
			return
		}
	}

	if err := h.store.Update(metricType, name, value); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
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
			http.Error(w, "metric not found", http.StatusNotFound)
			return
		}
		result = strconv.FormatFloat(val, 'f', -1, 64)
	case "counter":
		val, err := h.store.GetCounter(name)
		if err != nil {
			http.Error(w, "metric not found", http.StatusNotFound)
			return
		}
		result = strconv.FormatInt(val, 10)
	default:
		http.Error(w, "invalid metric type", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(result))
}

func (h *Handler) listMetricsHandler(w http.ResponseWriter, r *http.Request) {
	type Metric struct {
		ID    string
		MType string
		Value string
	}

	all := h.store.GetAll()
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
		http.Error(w, "failed to render template", http.StatusInternalServerError)
	}
}
