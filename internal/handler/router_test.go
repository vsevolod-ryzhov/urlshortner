package handler

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/vsevolod-ryzhov/urlshortner.git/internal/config"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/model"
)

func TestHandlerPOSTSuccess(t *testing.T) {
	os.Remove(config.Options.StorageFilePath)
	requestBody := "https://ya.ru"
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(requestBody))

	recorder := httptest.NewRecorder()

	handleCreateLink(recorder, req)

	if recorder.Code != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v", recorder.Code, http.StatusCreated)
	}

	contentType := recorder.Header().Get("Content-Type")
	if contentType != "text/plain" {
		t.Errorf("handler returned wrong content type: got %v want %v", contentType, "text/plain")
	}
}

func TestHandlerPOSTJsonSuccess(t *testing.T) {
	os.Remove(config.Options.StorageFilePath)
	requestModel := model.JSONRequest{URL: "https://google.ru"}

	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(requestModel)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/shorten", &buf)
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()

	handleCreateLink(recorder, req)

	if recorder.Code != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v", recorder.Code, http.StatusCreated)
	}

	contentType := recorder.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("handler returned wrong content type: got %v want %v", contentType, "application/json")
	}
}

func TestHandlerGETNotFound(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/123", nil)

	recorder := httptest.NewRecorder()

	handleGetLink(recorder, req)

	if recorder.Code != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v", recorder.Code, http.StatusNotFound)
	}
}
func BenchmarkMultiplePostSuccess(b *testing.B) {
	os.Remove(config.Options.StorageFilePath)
	for i := 0; i < b.N; i++ {
		requestBody := "https://" + generateRandomString(32) + ".ru"
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(requestBody))

		recorder := httptest.NewRecorder()

		handleCreateLink(recorder, req)

		if recorder.Code != http.StatusCreated {
			b.Errorf("handler returned wrong status code: got %v want %v", recorder.Code, http.StatusCreated)
		}

		contentType := recorder.Header().Get("Content-Type")
		if contentType != "text/plain" {
			b.Errorf("handler returned wrong content type: got %v want %v", contentType, "text/plain")
		}
	}
}

func BenchmarkMultiplePostNotFound(b *testing.B) {
	os.Remove(config.Options.StorageFilePath)
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/"+generateRandomString(128), nil)

		recorder := httptest.NewRecorder()

		handleGetLink(recorder, req)

		if recorder.Code != http.StatusNotFound {
			b.Errorf("handler returned wrong status code: got %v want %v", recorder.Code, http.StatusNotFound)
		}
	}
}

func generateRandomString(length int) string {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}
