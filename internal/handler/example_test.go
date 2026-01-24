package handler_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vsevolod-ryzhov/urlshortner.git/internal/audit"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/config"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/handler"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/model"
)

func getTempFile() string {
	return filepath.Join(os.TempDir(), fmt.Sprintf("test_example_%d.json", time.Now().UnixNano()))
}

func setupExampleConfig() (cleanup func()) {
	originalDSN := config.Options.DatabaseDSN
	originalFilePath := config.Options.StorageFilePath
	originalBaseURL := config.Options.ShortenedBaseURL

	config.Options.DatabaseDSN = ""
	config.Options.StorageFilePath = getTempFile()
	config.Options.ShortenedBaseURL = "http://localhost:8080"

	return func() {
		os.Remove(config.Options.StorageFilePath)
		config.Options.DatabaseDSN = originalDSN
		config.Options.StorageFilePath = originalFilePath
		config.Options.ShortenedBaseURL = originalBaseURL
	}
}

func Example_createShortURL_text() {
	cleanup := setupExampleConfig()
	defer cleanup()

	publisher := &audit.AuditMessenger{}
	router := handler.MakeHandler(publisher)

	reqBody := "https://example.com"
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(reqBody))

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	fmt.Printf("Status: %d\n", rr.Code)
	fmt.Printf("Content-Type: %s\n", rr.Header().Get("Content-Type"))

	body, _ := io.ReadAll(rr.Body)
	response := string(body)
	if strings.HasPrefix(response, "http://localhost:8080/") {
		fmt.Println("Response: OK")
	}

	// Output:
	// Status: 201
	// Content-Type: text/plain
	// Response: OK
}

func Example_createShortURL_json() {
	cleanup := setupExampleConfig()
	defer cleanup()

	publisher := &audit.AuditMessenger{}
	router := handler.MakeHandler(publisher)

	request := model.JSONRequest{URL: "https://example.org"}
	reqBody, _ := json.Marshal(request)

	req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	fmt.Printf("Status: %d\n", rr.Code)
	fmt.Printf("Content-Type: %s\n", rr.Header().Get("Content-Type"))

	// Output:
	// Status: 201
	// Content-Type: application/json
}

func Example_getOriginalURL() {
	cleanup := setupExampleConfig()
	defer cleanup()

	publisher := &audit.AuditMessenger{}
	router := handler.MakeHandler(publisher)

	reqCreate := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://example.org"))
	rrCreate := httptest.NewRecorder()
	router.ServeHTTP(rrCreate, reqCreate)

	response := rrCreate.Body.String()
	shortID := strings.TrimPrefix(response, "http://localhost:8080/")

	reqGet := httptest.NewRequest(http.MethodGet, "/"+shortID, nil)
	rrGet := httptest.NewRecorder()
	router.ServeHTTP(rrGet, reqGet)

	fmt.Printf("Redirect status: %d\n", rrGet.Code)

	location := rrGet.Header().Get("Location")
	if location != "" {
		fmt.Println("Location header: present")
	}

	// Output:
	// Redirect status: 307
	// Location header: present
}

func Example_batchCreateURLs() {
	cleanup := setupExampleConfig()
	defer cleanup()

	publisher := &audit.AuditMessenger{}
	router := handler.MakeHandler(publisher)

	batchRequest := model.JSONBatchRequest{
		{CorrelationID: "1", OriginalURL: "https://github.com"},
		{CorrelationID: "2", OriginalURL: "https://gitlab.com"},
	}

	reqBody, _ := json.Marshal(batchRequest)
	req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	fmt.Printf("Status: %d\n", rr.Code)
	fmt.Printf("Content-Type: %s\n", rr.Header().Get("Content-Type"))

	// Output:
	// Status: 201
	// Content-Type: application/json
}

func Example_getNonExistentURL() {
	cleanup := setupExampleConfig()
	defer cleanup()

	publisher := &audit.AuditMessenger{}
	router := handler.MakeHandler(publisher)

	req := httptest.NewRequest(http.MethodGet, "/nonexistent123", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	fmt.Printf("Status for non-existent URL: %d\n", rr.Code)

	// Output:
	// Status for non-existent URL: 404
}

func Example_createDuplicateURL() {
	cleanup := setupExampleConfig()
	defer cleanup()

	publisher := &audit.AuditMessenger{}
	router := handler.MakeHandler(publisher)

	url := "https://duplicate-example.com"
	req1 := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(url))
	rr1 := httptest.NewRecorder()
	router.ServeHTTP(rr1, req1)

	fmt.Printf("First request status: %d\n", rr1.Code)

	req2 := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(url))
	rr2 := httptest.NewRecorder()
	router.ServeHTTP(rr2, req2)

	fmt.Printf("Second request status: %d\n", rr2.Code)

	// Output:
	// First request status: 201
	// Second request status: 409
}

func Example_wrongContentType() {
	cleanup := setupExampleConfig()
	defer cleanup()

	publisher := &audit.AuditMessenger{}
	router := handler.MakeHandler(publisher)

	request := model.JSONRequest{URL: "https://vk.com"}
	reqBody, _ := json.Marshal(request)

	req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewReader(reqBody))

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	fmt.Printf("Status with wrong Content-Type: %d\n", rr.Code)

	// Output:
	// Status with wrong Content-Type: 201
}

func Example_fullRequest() {
	cleanup := setupExampleConfig()
	defer cleanup()

	publisher := &audit.AuditMessenger{}
	router := handler.MakeHandler(publisher)

	originalURL := "https://yandex.ru"
	reqCreate := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(originalURL))
	rrCreate := httptest.NewRecorder()
	router.ServeHTTP(rrCreate, reqCreate)

	fmt.Printf("Create status: %d\n", rrCreate.Code)

	shortURL := rrCreate.Body.String()
	if strings.HasPrefix(shortURL, "http://localhost:8080/") {
		fmt.Println("Short URL created successfully")
	}

	shortID := strings.TrimPrefix(shortURL, "http://localhost:8080/")
	reqGet := httptest.NewRequest(http.MethodGet, "/"+shortID, nil)
	rrGet := httptest.NewRecorder()
	router.ServeHTTP(rrGet, reqGet)

	fmt.Printf("Get status: %d\n", rrGet.Code)

	// Output:
	// Create status: 201
	// Short URL created successfully
	// Get status: 307
}
