package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vsevolod-ryzhov/urlshortner.git/internal/model"
)

func TestHandlerPOSTSuccess(t *testing.T) {
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
	requestModel := model.JSONRequest{URL: "https://ya.ru"}

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
