// Package audit provides asynchronous audit event publishing to multiple observers.
package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// FileObserver writes audit events to a local file in JSON format.
// Each event is written as a single line of JSON.
type FileObserver struct {
	File *os.File   // File handle used for writing events
	mu   sync.Mutex // Mutex to synchronize writes to the file
}

// NewFileObserver creates a new FileObserver that writes to the given file path.
// If the file does not exist, it will be created. Events are appended to the file.
// Returns an error if the file cannot be opened.
func NewFileObserver(path string) (*FileObserver, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	return &FileObserver{File: f}, nil
}

// Notify writes a single audit event to the file in JSON format followed by a newline.
// Returns an error if marshaling or writing fails.
func (o *FileObserver) Notify(event Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal audit event: %w", err)
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	if _, err := o.File.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("write audit event to file: %w", err)
	}

	return nil
}
