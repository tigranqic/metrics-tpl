package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"strconv"
	"testing"

	models "github.com/tigranqic/metrics-tpl/internal/model"
)

func BenchmarkJSON_ReadAll_Unmarshal(b *testing.B) {
	data := []byte(`{"id":"test","type":"gauge","value":123.45}`)

	for i := 0; i < b.N; i++ {
		var m models.Metrics
		body := io.NopCloser(bytes.NewReader(data))
		buf, _ := io.ReadAll(body)
		_ = json.Unmarshal(buf, &m)
	}
}

func BenchmarkJSON_Decoder(b *testing.B) {
	data := []byte(`{"id":"test","type":"gauge","value":123.45}`)

	for i := 0; i < b.N; i++ {
		var m models.Metrics
		body := io.NopCloser(bytes.NewReader(data))
		_ = json.NewDecoder(body).Decode(&m)
	}
}

func BenchmarkBatchDecoder(b *testing.B) {
	type M = models.Metrics
	batch := make([]M, 100)
	for i := range batch {
		v := float64(i)
		batch[i] = M{ID: "id", MType: "gauge", Value: &v}
	}

	data, _ := json.Marshal(batch)

	for i := 0; i < b.N; i++ {
		var out []M
		body := io.NopCloser(bytes.NewReader(data))
		_ = json.NewDecoder(body).Decode(&out)
	}
}

func BenchmarkJSONEncode(b *testing.B) {
	v := 123.45
	m := models.Metrics{
		ID:    "test",
		MType: "gauge",
		Value: &v,
	}

	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_ = json.NewEncoder(&buf).Encode(m)
	}
}

func BenchmarkParseFloat(b *testing.B) {
	value := "123.456789"

	for i := 0; i < b.N; i++ {
		_, _ = strconv.ParseFloat(value, 64)
	}
}

func BenchmarkParseInt(b *testing.B) {
	value := "12345678"

	for i := 0; i < b.N; i++ {
		_, _ = strconv.ParseInt(value, 10, 64)
	}
}

func BenchmarkFormatFloat(b *testing.B) {
	value := 123.456789

	for i := 0; i < b.N; i++ {
		_ = strconv.FormatFloat(value, 'f', -1, 64)
	}
}

func BenchmarkFormatInt(b *testing.B) {
	value := int64(12345678)

	for i := 0; i < b.N; i++ {
		_ = strconv.FormatInt(value, 10)
	}
}

func BenchmarkBatchDecoder_LargeBatch(b *testing.B) {
	type M = models.Metrics
	batch := make([]M, 1000)
	for i := range batch {
		v := float64(i)
		delta := int64(i)
		if i%2 == 0 {
			batch[i] = M{ID: "gauge_" + strconv.Itoa(i), MType: "gauge", Value: &v}
		} else {
			batch[i] = M{ID: "counter_" + strconv.Itoa(i), MType: "counter", Delta: &delta}
		}
	}

	data, _ := json.Marshal(batch)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var out []M
		body := io.NopCloser(bytes.NewReader(data))
		_ = json.NewDecoder(body).Decode(&out)
	}
}

func BenchmarkCreateGaugeMetric(b *testing.B) {
	for i := 0; i < b.N; i++ {
		v := float64(123.45)
		_ = models.Metrics{
			ID:    "test_gauge",
			MType: "gauge",
			Value: &v,
		}
	}
}

func BenchmarkCreateCounterMetric(b *testing.B) {
	for i := 0; i < b.N; i++ {
		d := int64(100)
		_ = models.Metrics{
			ID:    "test_counter",
			MType: "counter",
			Delta: &d,
		}
	}
}

func BenchmarkMetricsMapCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		metricsMap := make(map[string]*models.Metrics)

		for j := 0; j < 100; j++ {
			v := float64(j)
			d := int64(j)

			if j%2 == 0 {
				metricsMap["gauge_"+strconv.Itoa(j)] = &models.Metrics{
					ID:    "gauge_" + strconv.Itoa(j),
					MType: "gauge",
					Value: &v,
				}
			} else {
				metricsMap["counter_"+strconv.Itoa(j)] = &models.Metrics{
					ID:    "counter_" + strconv.Itoa(j),
					MType: "counter",
					Delta: &d,
				}
			}
		}
	}
}

func BenchmarkMetricsMapIteration(b *testing.B) {
	metricsMap := make(map[string]*models.Metrics)
	for j := 0; j < 100; j++ {
		v := float64(j)
		d := int64(j)

		if j%2 == 0 {
			metricsMap["gauge_"+strconv.Itoa(j)] = &models.Metrics{
				ID:    "gauge_" + strconv.Itoa(j),
				MType: "gauge",
				Value: &v,
			}
		} else {
			metricsMap["counter_"+strconv.Itoa(j)] = &models.Metrics{
				ID:    "counter_" + strconv.Itoa(j),
				MType: "counter",
				Delta: &d,
			}
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		type Metric struct {
			ID    string
			MType string
			Value string
		}
		var metrics []Metric

		for _, m := range metricsMap {
			switch m.MType {
			case "gauge":
				if m.Value != nil {
					_ = append(metrics, Metric{
						ID:    m.ID,
						MType: m.MType,
						Value: strconv.FormatFloat(*m.Value, 'f', -1, 64),
					})
				}
			case "counter":
				if m.Delta != nil {
					_ = append(metrics, Metric{
						ID:    m.ID,
						MType: m.MType,
						Value: strconv.FormatInt(*m.Delta, 10),
					})
				}
			}
		}
	}
}
