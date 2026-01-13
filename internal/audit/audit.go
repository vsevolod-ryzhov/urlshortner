package audit

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
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
}

func (a *AuditMessenger) RegisterObserver(o Observer) {
	a.observers = append(a.observers, o)
}

func (a *AuditMessenger) RemoveObserver(o Observer) {
	for i, observer := range a.observers {
		if observer == o {
			a.observers = append(a.observers[:i], a.observers[i+1:]...)
			break
		}
	}
}

func (a *AuditMessenger) NotifyObservers() {
	for _, observer := range a.observers {
		observer.Update(a.message)
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
	/// TODO: implement HTTP request here
	fmt.Println("HTTP observer output", message.Data)
}
