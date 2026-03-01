package models

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMetricsJSON(t *testing.T) {
	t.Run("gauge", func(t *testing.T) {
		val := 123.45
		m := Metrics{
			ID:    "gauge1",
			MType: Gauge,
			Value: &val,
		}
		data, err := json.Marshal(m)
		assert.NoError(t, err)
		assert.Contains(t, string(data), `"value":123.45`)

		var decoded Metrics
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)
		assert.Equal(t, m.ID, decoded.ID)
		assert.Equal(t, *m.Value, *decoded.Value)
	})

	t.Run("counter", func(t *testing.T) {
		delta := int64(10)
		m := Metrics{
			ID:    "counter1",
			MType: Counter,
			Delta: &delta,
		}
		data, err := json.Marshal(m)
		assert.NoError(t, err)
		assert.Contains(t, string(data), `"delta":10`)

		var decoded Metrics
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)
		assert.Equal(t, m.ID, decoded.ID)
		assert.Equal(t, *m.Delta, *decoded.Delta)
	})
}
