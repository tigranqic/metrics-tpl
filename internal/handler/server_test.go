package handler

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	_ "github.com/lib/pq"
	"github.com/tigranqic/metrics-tpl/internal/repository"
)

func TestHandler_Router(t *testing.T) {
	store := repository.NewMemStorage("", 0)
	db := setupTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Fatalf("failed to close test DB: %v", err)
		}
	}()
	h := NewHandler(store, db)
	router := h.Router()

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/Alloc/123.45", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200; got %d", w.Code)
	}
	if val, err := store.GetGauge("Alloc"); err != nil || val != 123.45 {
		t.Fatalf("expected gauge 123.45, got %v, err %v", val, err)
	}

	req = httptest.NewRequest(http.MethodPost, "/update/counter/PollCount/5", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200; got %d", w.Code)
	}
	if val, err := store.GetCounter("PollCount"); err != nil || val != 5 {
		t.Fatalf("expected counter 5, got %v, err %v", val, err)
	}

	req = httptest.NewRequest(http.MethodGet, "/update/gauge/Alloc/123.45", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405; got %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/update/gauge/Alloc", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404; got %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/update/unknown/Alloc/123", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400; got %d", w.Code)
	}

	// --- Counter increment behavior
	req = httptest.NewRequest(http.MethodPost, "/update/counter/MyCounter/5", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	req = httptest.NewRequest(http.MethodPost, "/update/counter/MyCounter/3", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if val, err := store.GetCounter("MyCounter"); err != nil || val != 8 {
		t.Fatalf("expected counter 8, got %v, err %v", val, err)
	}

	req = httptest.NewRequest(http.MethodGet, "/value/gauge/Alloc", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 for /value/gauge/Alloc; got %d", w.Code)
	}
	if w.Body.String() != "123.45" {
		t.Fatalf("expected body '123.45'; got '%s'", w.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/value/counter/PollCount", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 for /value/counter/PollCount; got %d", w.Code)
	}
	if w.Body.String() != "5" {
		t.Fatalf("expected body '5'; got '%s'", w.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/value/gauge/UnknownMetric", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404 for unknown metric; got %d", w.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 for /; got %d", w.Code)
	}
	body := w.Body.String()
	if !containsAll(body, "Alloc", "PollCount", "MyCounter", "123.45", "5", "8") {
		t.Fatalf("expected HTML to contain metric names and values; got:\n%s", body)
	}
}

func TestHandler_JSONEndpoints(t *testing.T) {
	store := repository.NewMemStorage("", 0)
	db := setupTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Fatalf("failed to close test DB: %v", err)
		}
	}()
	h := NewHandler(store, db)
	router := h.Router()

	gaugeBody := `{"id":"Alloc","type":"gauge","value":123.45}`
	req := httptest.NewRequest(http.MethodPost, "/update/", strings.NewReader(gaugeBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 for /update gauge; got %d", w.Code)
	}
	if val, err := store.GetGauge("Alloc"); err != nil || val != 123.45 {
		t.Fatalf("expected stored gauge 123.45; got %v, err %v", val, err)
	}

	counterBody := `{"id":"PollCount","type":"counter","delta":5}`
	req = httptest.NewRequest(http.MethodPost, "/update/", strings.NewReader(counterBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 for /update counter; got %d", w.Code)
	}
	if val, err := store.GetCounter("PollCount"); err != nil || val != 5 {
		t.Fatalf("expected stored counter 5; got %v, err %v", val, err)
	}

	valueReq := `{"id":"Alloc","type":"gauge"}`
	req = httptest.NewRequest(http.MethodPost, "/value/", strings.NewReader(valueReq))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 for /value gauge; got %d", w.Code)
	}
	body := w.Body.String()
	expectedGaugeParts := []string{`"id":"Alloc"`, `"type":"gauge"`, `"value":123.45`}
	for _, p := range expectedGaugeParts {
		if !strings.Contains(body, p) {
			t.Errorf("expected response body to contain %q, got %s", p, body)
		}
	}

	valueReq = `{"id":"PollCount","type":"counter"}`
	req = httptest.NewRequest(http.MethodPost, "/value/", strings.NewReader(valueReq))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 for /value counter; got %d", w.Code)
	}
	body = w.Body.String()
	expectedCounterParts := []string{`"id":"PollCount"`, `"type":"counter"`, `"delta":5`}
	for _, p := range expectedCounterParts {
		if !strings.Contains(body, p) {
			t.Errorf("expected response body to contain %q, got %s", p, body)
		}
	}

	valueReq = `{"id":"Unknown","type":"gauge"}`
	req = httptest.NewRequest(http.MethodPost, "/value/", strings.NewReader(valueReq))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404 for unknown metric; got %d", w.Code)
	}
}

func containsAll(s string, substrings ...string) bool {
	for _, sub := range substrings {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}

func setupTestDB(t *testing.T) *sql.DB {
	dsn := os.Getenv("DATABASE_DSN")

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("failed to connect to test DB: %v", err)
	}

	if err := db.Ping(); err != nil {
		defer func() {
			if err := db.Close(); err != nil {
				t.Fatalf("failed to close test DB: %v", err)
			}
		}()
	}

	return db
}
