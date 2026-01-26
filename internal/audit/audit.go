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
	Data any `json:"data"`
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

func NewAuditMessenger() *AuditMessenger {
	return &AuditMessenger{
		observers: make([]Observer, 0),
		mu:        sync.RWMutex{},
	}
}

func (a *AuditMessenger) RegisterObserver(o Observer) {
	if a == nil {
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	a.observers = append(a.observers, o)
}

func (a *AuditMessenger) RemoveObserver(o Observer) {
	if a == nil {
		return
	}

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
	if a == nil || len(a.observers) == 0 {
		return
	}

	a.mu.RLock()
	message := a.message

	var wg sync.WaitGroup
	wg.Add(len(a.observers))

	for _, observer := range a.observers {
		go func(o Observer) {
			defer wg.Done()
			o.Update(message)
		}(observer)
	}

	a.mu.RUnlock()
	wg.Wait()
}

func (a *AuditMessenger) Audit(message AuditMessage) {
	if a == nil {
		return
	}

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
