package handler

import (
	"bytes"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	_ "github.com/lib/pq"
	"github.com/tigranqic/metrics-tpl/internal/repository"
	"github.com/tigranqic/metrics-tpl/pkg/hashutil"
	"go.uber.org/zap"
)

func TestHandler_Router(t *testing.T) {
	store := repository.NewMemStorage("", 0)
	db := setupTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Fatalf("failed to close test DB: %v", err)
		}
	}()
	logger, _ := zap.NewDevelopment()
	h := NewHandler(store, db, logger, "", nil, "")
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
	logger, _ := zap.NewDevelopment()
	h := NewHandler(store, db, logger, "", nil, "")
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

func TestHandler_WithKey_InvalidHash(t *testing.T) {
	store := repository.NewMemStorage("", 0)
	db := setupTestDB(t)
	logger, _ := zap.NewDevelopment()

	h := NewHandler(store, db, logger, "secret", nil, "")
	router := h.Router()

	body := `{"id":"Alloc","type":"gauge","value":123.45}`

	req := httptest.NewRequest(http.MethodPost, "/update/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Hash", "invalidhash")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid hash, got %d", w.Code)
	}
}

func TestHandler_WithKey_ValidHash(t *testing.T) {
	store := repository.NewMemStorage("", 0)
	db := setupTestDB(t)
	logger, _ := zap.NewDevelopment()

	key := "secret"
	h := NewHandler(store, db, logger, key, nil, "")
	router := h.Router()

	body := []byte(`{"id":"Alloc","type":"gauge","value":123.45}`)
	hash := hashutil.CalcSHA256(body, key)

	req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Hash", hash)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for valid hash, got %d", w.Code)
	}

	respHash := w.Header().Get("Hash")
	if respHash == "" {
		t.Fatal("expected Hash in response")
	}
}

func TestHandler_WithKey_HealthNoHash(t *testing.T) {
	store := repository.NewMemStorage("", 0)
	db := setupTestDB(t)
	logger, _ := zap.NewDevelopment()

	h := NewHandler(store, db, logger, "secret", nil, "")
	router := h.Router()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for /health, got %d", w.Code)
	}
}

func TestHandler_Ping(t *testing.T) {
	store := repository.NewMemStorage("", 0)
	db := setupTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Fatalf("failed to close db: %v", err)
		}
	}()
	logger, _ := zap.NewDevelopment()
	h := NewHandler(store, db, logger, "secret", nil, "")
	router := h.Router()

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 for /ping, got %d", w.Code)
	}

	badDB, _ := sql.Open("postgres", "postgres://invalid:invalid@127.0.0.1:5432/bad_db?sslmode=disable")
	h2 := NewHandler(store, badDB, logger, "", nil, "")
	router2 := h2.Router()
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/ping", nil)
	router2.ServeHTTP(w2, req2)

	if w2.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for bad DB ping, got %d", w2.Code)
	}

}

func TestHandler_UpdateMetricsBatch_Empty(t *testing.T) {
	store := repository.NewMemStorage("", 0)
	db := setupTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Fatalf("failed to close db: %v", err)
		}
	}()
	logger, _ := zap.NewDevelopment()
	h := NewHandler(store, db, logger, "", nil, "")
	router := h.Router()

	req := httptest.NewRequest(http.MethodPost, "/updates/", strings.NewReader("[]"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty batch, got %d", w.Code)
	}
}

func TestHandler_UpdateMetricsBatch_InvalidJSON(t *testing.T) {
	store := repository.NewMemStorage("", 0)
	db := setupTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Fatalf("failed to close db: %v", err)
		}
	}()
	logger, _ := zap.NewDevelopment()
	h := NewHandler(store, db, logger, "", nil, "")
	router := h.Router()

	req := httptest.NewRequest(http.MethodPost, "/updates/", strings.NewReader("{invalid-json}"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid JSON batch, got %d", w.Code)
	}
}

func TestHandler_UpdateMetricJSON_InvalidType(t *testing.T) {
	store := repository.NewMemStorage("", 0)
	db := setupTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Fatalf("failed to close db: %v", err)
		}
	}()
	logger, _ := zap.NewDevelopment()
	h := NewHandler(store, db, logger, "", nil, "")
	router := h.Router()

	body := `{"id":"Alloc","type":"unknown","value":123.45}`
	req := httptest.NewRequest(http.MethodPost, "/update/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid metric type, got %d", w.Code)
	}
}

func TestHandler_ConcurrentCounterUpdates(t *testing.T) {
	store := repository.NewMemStorage("", 0)
	db := setupTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Fatalf("failed to close db: %v", err)
		}
	}()
	logger, _ := zap.NewDevelopment()
	h := NewHandler(store, db, logger, "", nil, "")
	router := h.Router()

	const goroutines = 10
	const increment = 3
	done := make(chan struct{}, goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			req := httptest.NewRequest(http.MethodPost, "/update/counter/Concurrent/3", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}()
	}

	for i := 0; i < goroutines; i++ {
		<-done
	}

	val, err := store.GetCounter("Concurrent")
	if err != nil || val != goroutines*increment {
		t.Fatalf("expected counter %d, got %v, err %v", goroutines*increment, val, err)
	}
}

func TestHandler_BatchUpdate_ValidMetrics(t *testing.T) {
	store := repository.NewMemStorage("", 0)
	db := setupTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Fatalf("failed to close db: %v", err)
		}
	}()
	logger, _ := zap.NewDevelopment()
	h := NewHandler(store, db, logger, "", nil, "")
	router := h.Router()

	batch := `[{"id":"G1","type":"gauge","value":12.3},{"id":"C1","type":"counter","delta":7}]`
	req := httptest.NewRequest(http.MethodPost, "/updates/", strings.NewReader(batch))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for valid batch, got %d", w.Code)
	}

	if v, _ := store.GetGauge("G1"); v != 12.3 {
		t.Fatalf("expected G1=12.3, got %v", v)
	}
	if c, _ := store.GetCounter("C1"); c != 7 {
		t.Fatalf("expected C1=7, got %v", c)
	}
}

func TestHandler_Ping_ClosedDB(t *testing.T) {
	store := repository.NewMemStorage("", 0)
	db := setupTestDB(t)
	if err := db.Close(); err != nil {
		t.Fatalf("failed to close test db: %v", err)
	}
	logger, _ := zap.NewDevelopment()
	h := NewHandler(store, db, logger, "", nil, "")
	router := h.Router()

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for closed DB ping, got %d", w.Code)
	}
}

func TestHandler_Index_EmptyStore(t *testing.T) {
	store := repository.NewMemStorage("", 0)
	db := setupTestDB(t)
	defer func() {
		if err := db.Close(); err != nil {
			t.Fatalf("failed to close db: %v", err)
		}
	}()
	logger, _ := zap.NewDevelopment()
	h := NewHandler(store, db, logger, "", nil, "")
	router := h.Router()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for / index with empty store, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "<html>") {
		t.Fatalf("expected HTML content, got %s", w.Body.String())
	}
}

func TestHandler_SubnetMiddleware(t *testing.T) {
	store := repository.NewMemStorage("", 0)
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()
	logger, _ := zap.NewDevelopment()

	t.Run("allowed IP", func(t *testing.T) {
		h := NewHandler(store, db, logger, "", nil, "192.168.1.0/24")
		router := h.Router()
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		req.Header.Set("X-Real-IP", "192.168.1.10")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})

	t.Run("forbidden IP", func(t *testing.T) {
		h := NewHandler(store, db, logger, "", nil, "192.168.1.0/24")
		router := h.Router()
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		req.Header.Set("X-Real-IP", "10.0.0.1")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d", w.Code)
		}
	})

	t.Run("missing header", func(t *testing.T) {
		h := NewHandler(store, db, logger, "", nil, "192.168.1.0/24")
		router := h.Router()
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d", w.Code)
		}
	})

	t.Run("empty subnet allows all", func(t *testing.T) {
		h := NewHandler(store, db, logger, "", nil, "")
		router := h.Router()
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})
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
	if dsn == "" {
		t.Fatal("DATABASE_DSN is not set")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("failed to connect to test DB: %v", err)
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		t.Fatalf("failed to ping test DB: %v", err)
	}

	return db
}
