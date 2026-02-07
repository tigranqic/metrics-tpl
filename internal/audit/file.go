package audit

import (
	"encoding/json"
	"os"
)

type FileObserver struct {
	File *os.File
}

func NewFileObserver(path string) (*FileObserver, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	return &FileObserver{File: f}, nil
}

func (o *FileObserver) Notify(event Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	_, err = o.File.Write(append(data, '\n'))
	return err
}
