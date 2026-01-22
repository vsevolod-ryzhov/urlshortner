package audit

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

type MockObserver struct {
	id          int
	updated     bool
	lastMessage AuditMessage
	updateCount int
	mu          sync.Mutex
}

func (m *MockObserver) Update(message AuditMessage) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updated = true
	m.lastMessage = message
	m.updateCount++
}

func (m *MockObserver) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updated = false
	m.lastMessage = AuditMessage{}
	m.updateCount = 0
}

func TestAuditMessenger_NewAuditMessenger(t *testing.T) {
	am := NewAuditMessenger()
	if am == nil {
		t.Fatal("NewAuditMessenger() returned nil")
	}
	if am.observers == nil {
		t.Error("observers slice should be initialized")
	}
	if len(am.observers) != 0 {
		t.Errorf("expected 0 observers, got %d", len(am.observers))
	}
}

func TestAuditMessenger_RegisterObserver(t *testing.T) {
	am := NewAuditMessenger()

	observer := &MockObserver{}
	am.RegisterObserver(observer)

	if len(am.observers) != 1 {
		t.Errorf("expected 1 observer, got %d", len(am.observers))
	}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			obs := &MockObserver{id: idx}
			am.RegisterObserver(obs)
		}(i)
	}
	wg.Wait()

	if len(am.observers) != 101 {
		t.Errorf("expected 101 observers after concurrent registration, got %d", len(am.observers))
	}
}

func TestAuditMessenger_RemoveObserver(t *testing.T) {
	am := NewAuditMessenger()

	observer1 := &MockObserver{id: 1}
	observer2 := &MockObserver{id: 2}

	am.RegisterObserver(observer1)
	am.RegisterObserver(observer2)

	if len(am.observers) != 2 {
		t.Errorf("expected 2 observers, got %d", len(am.observers))
	}

	am.RemoveObserver(observer1)

	if len(am.observers) != 1 {
		t.Errorf("expected 1 observer after removal, got %d", len(am.observers))
	}

	am.RemoveObserver(&MockObserver{id: 999})
	if len(am.observers) != 1 {
		t.Errorf("observer count should not change when removing non-existent observer")
	}
}

func TestAuditMessenger_Audit(t *testing.T) {
	am := NewAuditMessenger()

	mockObserver := &MockObserver{}
	am.RegisterObserver(mockObserver)

	testMessage := AuditMessage{
		Data: map[string]interface{}{
			"action": "user_login",
			"user":   "testuser",
			"time":   time.Now().Unix(),
		},
	}

	am.Audit(testMessage)

	if !mockObserver.updated {
		t.Error("observer should have been updated")
	}

	if mockObserver.lastMessage.Data == nil {
		t.Error("message data should not be nil")
	}
}

func TestAuditMessenger_NotifyObservers(t *testing.T) {
	tests := []struct {
		name      string
		observers []Observer
		message   AuditMessage
		wantCalls int
	}{
		{
			name:      "no observers",
			observers: []Observer{},
			message:   AuditMessage{Data: "test"},
			wantCalls: 0,
		},
		{
			name: "single observer",
			observers: []Observer{
				&MockObserver{id: 1},
			},
			message:   AuditMessage{Data: "test"},
			wantCalls: 1,
		},
		{
			name: "multiple observers",
			observers: []Observer{
				&MockObserver{id: 1},
				&MockObserver{id: 2},
				&MockObserver{id: 3},
			},
			message:   AuditMessage{Data: "test"},
			wantCalls: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			am := NewAuditMessenger()

			mocks := make([]*MockObserver, len(tt.observers))
			for i := range tt.observers {
				mock := &MockObserver{id: i}
				mocks[i] = mock
				am.RegisterObserver(mock)
			}

			am.message = tt.message
			am.NotifyObservers()

			for i, mock := range mocks {
				if !mock.updated && tt.wantCalls > 0 {
					t.Errorf("observer %d was not updated", i)
				}
				if mock.lastMessage.Data != tt.message.Data {
					t.Errorf("observer %d received wrong message", i)
				}
			}
		})
	}
}

func TestFileObserver_Update(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "audit.log")

	fo := &FileObserver{FilePath: filePath}

	testMessage := AuditMessage{
		Data: map[string]interface{}{
			"event":   "test_event",
			"success": true,
		},
	}

	fo.Update(testMessage)

	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	var readMessage AuditMessage
	if err := json.Unmarshal(content, &readMessage); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	dataMap, ok := readMessage.Data.(map[string]interface{})
	if !ok {
		t.Fatal("data should be a map")
	}

	if dataMap["event"] != "test_event" {
		t.Errorf("expected event 'test_event', got %v", dataMap["event"])
	}

	for i := 0; i < 5; i++ {
		msg := AuditMessage{Data: fmt.Sprintf("message_%d", i)}
		fo.Update(msg)
	}

	file, err := os.Open(filePath)
	if err != nil {
		t.Fatalf("failed to open file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineCount := 0
	for scanner.Scan() {
		lineCount++
	}

	if lineCount != 6 {
		t.Errorf("expected 6 lines in file, got %d", lineCount)
	}
}

func TestHTTPObserver_Update(t *testing.T) {
	var receivedRequests []AuditMessage
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST method, got %s", r.Method)
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		var msg AuditMessage
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("failed to read body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if err := json.Unmarshal(body, &msg); err != nil {
			t.Errorf("failed to unmarshal JSON: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		mu.Lock()
		receivedRequests = append(receivedRequests, msg)
		mu.Unlock()

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ho := &HTTPObserver{URL: server.URL}

	testMessage := AuditMessage{
		Data: map[string]interface{}{
			"type":   "http_test",
			"value":  42,
			"nested": map[string]string{"key": "value"},
		},
	}

	ho.Update(testMessage)

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if len(receivedRequests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(receivedRequests))
	}

	reqData, ok := receivedRequests[0].Data.(map[string]interface{})
	if !ok {
		t.Fatal("data should be a map")
	}

	if reqData["type"] != "http_test" {
		t.Errorf("expected type 'http_test', got %v", reqData["type"])
	}
}

func TestNilSafety(t *testing.T) {
	var nilAuditMessenger *AuditMessenger

	nilAuditMessenger.RegisterObserver(&MockObserver{})
	nilAuditMessenger.RemoveObserver(&MockObserver{})
	nilAuditMessenger.NotifyObservers()
	nilAuditMessenger.Audit(AuditMessage{Data: "test"})

	am := NewAuditMessenger()
	am.observers = nil
	am.NotifyObservers()
}

func TestConcurrentAccess(t *testing.T) {
	am := NewAuditMessenger()

	var wg sync.WaitGroup
	iterations := 100

	for i := 0; i < iterations; i++ {
		wg.Add(3)

		go func(id int) {
			defer wg.Done()
			obs := &MockObserver{id: id}
			am.RegisterObserver(obs)
		}(i)

		go func(id int) {
			defer wg.Done()
			obs := &MockObserver{id: id}
			am.RemoveObserver(obs)
		}(i)

		go func() {
			defer wg.Done()
			am.Audit(AuditMessage{Data: fmt.Sprintf("message_%d", i)})
		}()
	}

	wg.Wait()

	am.Audit(AuditMessage{Data: "final_message"})
}
