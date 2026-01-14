package audit

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"sync"
)

type AuditMessage struct {
	Data interface{} `json:"data"`
}

type Subject interface {
	RegisterObserver(o Observer)
	RemoveObserver(o Observer)
	NotifyObservers()
}

type Observer interface {
	Update(message AuditMessage)
}
type AuditMessenger struct {
	observers []Observer
	message   AuditMessage
	mu        sync.RWMutex
}

func (a *AuditMessenger) RegisterObserver(o Observer) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.observers = append(a.observers, o)
}

func (a *AuditMessenger) RemoveObserver(o Observer) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for i, observer := range a.observers {
		if observer == o {
			a.observers = append(a.observers[:i], a.observers[i+1:]...)
			break
		}
	}
}

func (a *AuditMessenger) NotifyObservers() {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, observer := range a.observers {
		go observer.Update(a.message)
	}
}

func (a *AuditMessenger) Audit(message AuditMessage) {
	a.message = message
	a.NotifyObservers()
}

type FileObserver struct {
	FilePath string
}

func (fo *FileObserver) Update(message AuditMessage) {
	file, err := os.OpenFile(fo.FilePath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	encoder := json.NewEncoder(writer)

	if err := encoder.Encode(message); err != nil {
		return
	}
	writer.Flush()
}

type HTTPObserver struct {
	URL string
}

func (h *HTTPObserver) Update(message AuditMessage) {
	jsonData, err := json.Marshal(message)
	if err != nil {
		return
	}

	req, err := http.NewRequest("POST", h.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
}
