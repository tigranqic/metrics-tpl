package repository

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	models "github.com/tigranqic/metrics-tpl/internal/model"
)

func TestMemStorage_UpdateAndGetGauge(t *testing.T) {
	store := NewMemStorage("", 0)
	err := store.Update(models.Gauge, "g1", "42.5")
	assert.NoError(t, err)

	val, err := store.GetGauge("g1")
	assert.NoError(t, err)
	assert.Equal(t, 42.5, val)
}

func TestMemStorage_UpdateAndGetCounter(t *testing.T) {
	store := NewMemStorage("", 0)
	err := store.Update(models.Counter, "c1", "10")
	assert.NoError(t, err)

	val, err := store.GetCounter("c1")
	assert.NoError(t, err)
	assert.Equal(t, int64(10), val)

	err = store.Update(models.Counter, "c1", "5")
	assert.NoError(t, err)

	val, err = store.GetCounter("c1")
	assert.Equal(t, int64(15), val)
}

func TestMemStorage_UpdateInvalidCounter(t *testing.T) {
	store := NewMemStorage("", 0)
	err := store.Update(models.Counter, "c1", "abc")
	assert.Error(t, err)
}

func TestMemStorage_UpdateUnsupportedType(t *testing.T) {
	store := NewMemStorage("", 0)
	err := store.Update("unknown", "m1", "123")
	assert.Error(t, err)
	assert.Equal(t, "unsupported metric type", err.Error())
}

func TestMemStorage_UpdateBatch(t *testing.T) {
	store := NewMemStorage("", 0)

	batch := []models.Metrics{
		{ID: "g1", MType: models.Gauge, Value: ptrFloat64(1.1)},
		{ID: "c1", MType: models.Counter, Delta: ptrInt64(5)},
		{ID: "c1", MType: models.Counter, Delta: ptrInt64(3)},
		{ID: "invalid", MType: "unknown"},
	}

	err := store.UpdateBatch(batch)
	assert.NoError(t, err)

	g, _ := store.GetGauge("g1")
	assert.Equal(t, 1.1, g)

	c, _ := store.GetCounter("c1")
	assert.Equal(t, int64(8), c)

	_, err = store.GetGauge("invalid")
	assert.Error(t, err)
}

func TestMemStorage_SaveAndLoadFile(t *testing.T) {
	tmpFile := "test_metrics.json"
	defer os.Remove(tmpFile)

	store := NewMemStorage(tmpFile, 0)
	_ = store.Update(models.Gauge, "g1", "1.1")
	_ = store.Update(models.Counter, "c1", "5")

	err := store.SaveToFile(tmpFile)
	assert.NoError(t, err)

	newStore := NewMemStorage(tmpFile, 0)
	err = newStore.LoadFromFile(tmpFile)
	assert.NoError(t, err)

	g, _ := newStore.GetGauge("g1")
	c, _ := newStore.GetCounter("c1")
	assert.Equal(t, 1.1, g)
	assert.Equal(t, int64(5), c)
}

func TestMemStorage_LoadFileNotExist(t *testing.T) {
	store := NewMemStorage("nonexistent.json", 0)
	assert.NoError(t, store.LoadFromFile("nonexistent.json"))
}

func TestMemStorage_StartAutoSaveImmediate(t *testing.T) {
	tmpFile := "test_auto.json"
	defer os.Remove(tmpFile)

	store := NewMemStorage(tmpFile, 0)
	_ = store.Update(models.Gauge, "g1", "1.1")

	stop := make(chan struct{})
	store.StartAutoSave(tmpFile, 0, stop)

	newStore := NewMemStorage(tmpFile, 0)
	err := newStore.LoadFromFile(tmpFile)
	assert.NoError(t, err)
	g, _ := newStore.GetGauge("g1")
	assert.Equal(t, 1.1, g)
}
