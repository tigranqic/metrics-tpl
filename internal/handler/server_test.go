package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tigranqic/metrics-tpl/internal/repository"
)

func TestHandler_ServeHTTP(t *testing.T) {
	store := repository.NewMemStorage()
	h := NewHandler(store)

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/Alloc/123.45", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200; got %d", w.Code)
	}
	if val, err := store.GetGauge("Alloc"); err != nil || val != 123.45 {
		t.Fatalf("expected gauge 123.45, got %v, err %v", val, err)
	}

	req = httptest.NewRequest(http.MethodPost, "/update/counter/PollCount/5", nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200; got %d", w.Code)
	}
	if val, err := store.GetCounter("PollCount"); err != nil || val != 5 {
		t.Fatalf("expected counter 5, got %v, err %v", val, err)
	}

	req = httptest.NewRequest(http.MethodGet, "/update/gauge/Alloc/123.45", nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405; got %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/update/gauge/Alloc", nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404; got %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/update/unknown/Alloc/123", nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400; got %d", w.Code)
	}

	// Check counter increment behavior
	req = httptest.NewRequest(http.MethodPost, "/update/counter/MyCounter/5", nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)

	req = httptest.NewRequest(http.MethodPost, "/update/counter/MyCounter/3", nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if val, err := store.GetCounter("MyCounter"); err != nil || val != 8 {
		t.Fatalf("expected counter 8, got %v, err %v", val, err)
	}

}
