package audit_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tigranqic/metrics-tpl/internal/audit"
	"go.uber.org/zap"
)

type fakeObserver struct {
	mu     sync.Mutex
	events []audit.Event
	fail   bool
	done   chan struct{}
}

func (f *fakeObserver) Notify(event audit.Event) error {
	if f.fail {
		return errors.New("fail observer")
	}

	f.mu.Lock()
	f.events = append(f.events, event)
	if f.done != nil {
		select {
		case f.done <- struct{}{}:
		default:
		}
	}
	f.mu.Unlock()

	return nil
}

func (f *fakeObserver) Len() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.events)
}

func (f *fakeObserver) Events() []audit.Event {
	f.mu.Lock()
	defer f.mu.Unlock()
	cpy := make([]audit.Event, len(f.events))
	copy(cpy, f.events)
	return cpy
}

func TestPublisherWorkerPool(t *testing.T) {
	f1 := &fakeObserver{}
	f2 := &fakeObserver{}
	p := audit.NewPublisherWithPool(nil, []audit.Observer{f1, f2}, 2)

	event := audit.Event{
		Timestamp: time.Now().Unix(),
		Metrics:   []string{"metric1"},
		IPAddress: "127.0.0.1",
	}

	p.NotifyAllAsync(event)
	p.NotifyAllAsync(event)

	time.Sleep(100 * time.Millisecond)

	assert.Len(t, f1.Events(), 2)
	assert.Len(t, f2.Events(), 2)

	p.Shutdown()
}

func TestPublisherMultipleWorkers(t *testing.T) {
	f := &fakeObserver{done: make(chan struct{}, 20)}
	p := audit.NewPublisherWithPool(nil, []audit.Observer{f}, 5)

	for i := 0; i < 20; i++ {
		p.NotifyAllAsync(audit.Event{
			Timestamp: int64(i),
			Metrics:   []string{"m" + strconv.Itoa(i)},
			IPAddress: "127.0.0.1",
		})
	}

	for i := 0; i < 20; i++ {
		<-f.done
	}

	assert.Len(t, f.Events(), 20)
	p.Shutdown()
}

func TestPublisherObserverError(t *testing.T) {
	f := &fakeObserver{fail: true}
	log, _ := zap.NewDevelopment()
	p := audit.NewPublisherWithPool(log, []audit.Observer{f}, 1)

	p.NotifyAllAsync(audit.Event{Timestamp: time.Now().Unix()})

	time.Sleep(50 * time.Millisecond)
	assert.Len(t, f.Events(), 0)

	p.Shutdown()
}

func TestPublisherShutdown(t *testing.T) {
	f := &fakeObserver{}
	p := audit.NewPublisherWithPool(nil, []audit.Observer{f}, 2)

	for i := 0; i < 5; i++ {
		p.NotifyAllAsync(audit.Event{Timestamp: int64(i)})
	}

	p.Shutdown()

	assert.True(t, len(f.Events()) <= 5)
}

func TestPublisherMixedEvents(t *testing.T) {
	f := &fakeObserver{}
	p := audit.NewPublisherWithPool(nil, []audit.Observer{f}, 2)

	events := []audit.Event{
		{Metrics: []string{"gauge1"}, Timestamp: 1},
		{Metrics: []string{"counter1"}, Timestamp: 2},
		{Metrics: []string{"gauge2"}, Timestamp: 3},
	}

	for _, e := range events {
		p.NotifyAllAsync(e)
	}

	time.Sleep(100 * time.Millisecond)
	assert.Len(t, f.Events(), 3)

	p.Shutdown()
}

func TestQueueFullWarning(t *testing.T) {
	log, _ := zap.NewDevelopment()
	f := &fakeObserver{}

	p := audit.NewPublisherWithPool(log, []audit.Observer{f}, 1)

	p.NotifyAllAsync(audit.Event{})
	p.NotifyAllAsync(audit.Event{})

	time.Sleep(50 * time.Millisecond)

	assert.GreaterOrEqual(t, f.Len(), 1)
	p.Shutdown()
}

func TestNotifyAllAsyncNonBlocking(t *testing.T) {
	log, _ := zap.NewDevelopment()
	const bufferSize = 2

	f := &fakeObserver{done: make(chan struct{}, bufferSize)}
	p := audit.NewPublisherWithPool(log, []audit.Observer{f}, 1)

	for i := 0; i < 5; i++ {
		p.NotifyAllAsync(audit.Event{Timestamp: int64(i)})
	}

	for i := 0; i < bufferSize; i++ {
		<-f.done
	}

	assert.GreaterOrEqual(t, f.Len(), bufferSize)
	p.Shutdown()
}

func TestObserverErrorLogged(t *testing.T) {
	log, _ := zap.NewDevelopment()
	f := &fakeObserver{fail: true}
	p := audit.NewPublisherWithPool(log, []audit.Observer{f}, 1)

	p.NotifyAllAsync(audit.Event{Timestamp: 1})
	time.Sleep(50 * time.Millisecond)

	assert.Len(t, f.Events(), 0)
	p.Shutdown()
}

func TestPublisherZeroWorkers(t *testing.T) {
	f := &fakeObserver{}
	p := audit.NewPublisherWithPool(nil, []audit.Observer{f}, 0)

	p.NotifyAllAsync(audit.Event{Timestamp: 1})
	time.Sleep(50 * time.Millisecond)

	assert.Len(t, f.Events(), 0)
	p.Shutdown()
}

func TestPublisherNilLogger(t *testing.T) {
	f := &fakeObserver{}
	p := audit.NewPublisherWithPool(nil, []audit.Observer{f}, 1)

	p.NotifyAllAsync(audit.Event{Timestamp: 1})
	time.Sleep(50 * time.Millisecond)

	assert.Len(t, f.Events(), 1)
	p.Shutdown()
}

func TestPublisherMultipleObserversOneFails(t *testing.T) {
	f1 := &fakeObserver{fail: true}
	f2 := &fakeObserver{}
	log, _ := zap.NewDevelopment()
	p := audit.NewPublisherWithPool(log, []audit.Observer{f1, f2}, 1)

	p.NotifyAllAsync(audit.Event{Timestamp: 1})
	time.Sleep(50 * time.Millisecond)

	assert.Len(t, f1.Events(), 0)
	assert.Len(t, f2.Events(), 1)
	p.Shutdown()
}

func TestFileObserverNotifyWritesToFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "fileobserver_test")
	assert.NoError(t, err)
	defer func() {
		if err := os.Remove(tmpFile.Name()); err != nil {
			t.Fatalf("failed to remove temp file: %v", err)
		}
	}()
	o := &audit.FileObserver{File: tmpFile}

	event := audit.Event{Timestamp: 123, Metrics: []string{"m1"}}
	err = o.Notify(event)
	assert.NoError(t, err)

	data, err := os.ReadFile(tmpFile.Name())
	assert.NoError(t, err)
	var readEvent audit.Event
	if err := json.Unmarshal(data, &readEvent); err != nil {
		t.Fatalf("failed to unmarshal event: %v", err)
	}
	assert.NoError(t, err)
	assert.Equal(t, event.Timestamp, readEvent.Timestamp)
	assert.Equal(t, event.Metrics, readEvent.Metrics)
}

func TestHTTPObserverNotifySuccess(t *testing.T) {
	var gotEvent audit.Event

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		if err := json.NewDecoder(r.Body).Decode(&gotEvent); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	o := audit.NewHTTPObserver(server.URL)
	event := audit.Event{Timestamp: 1, Metrics: []string{"m1"}}
	err := o.Notify(event)
	assert.NoError(t, err)
	assert.Equal(t, event.Timestamp, gotEvent.Timestamp)
	assert.Equal(t, event.Metrics, gotEvent.Metrics)
}

type mockObserver struct {
	mu     sync.Mutex
	called bool
}

func (m *mockObserver) Notify(e audit.Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.called = true
	return nil
}

func (m *mockObserver) IsCalled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.called
}

func (m *mockObserver) SetCalled(v bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.called = v
}

func TestPublisher_AddRemoveObserver(t *testing.T) {
	log := zap.NewNop()
	p := audit.NewPublisherWithPool(log, nil, 1)
	defer p.Shutdown()

	mock := &mockObserver{}

	p.AddObserver(mock)
	p.NotifyAllAsync(audit.Event{Timestamp: 1})

	// Wait a bit for async processing
	time.Sleep(100 * time.Millisecond)
	assert.True(t, mock.IsCalled())

	p.RemoveObserver(mock)
	mock.SetCalled(false)
	p.NotifyAllAsync(audit.Event{Timestamp: 2})

	time.Sleep(100 * time.Millisecond)
	assert.False(t, mock.IsCalled())
}
