// Package audit provides asynchronous audit event publishing to multiple observers.
// It includes support for HTTP-based observers.
package audit

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"
)

// HTTPObserver sends audit events to a remote HTTP endpoint via POST requests.
type HTTPObserver struct {
	client *http.Client // HTTP client with timeout
	url    string       // Target URL for sending events
}

// NewHTTPObserver creates a new HTTPObserver with the given URL.
// The observer uses a default HTTP client with a 5-second timeout.
func NewHTTPObserver(url string) *HTTPObserver {
	return &HTTPObserver{
		url: url,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// Notify sends a single audit event to the configured HTTP endpoint.
// The event is marshaled as JSON and sent with content-type application/json.
// Returns an error if marshaling fails, the request cannot be created, or the HTTP request fails.
func (o *HTTPObserver) Notify(event Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, o.url, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	return nil
}
