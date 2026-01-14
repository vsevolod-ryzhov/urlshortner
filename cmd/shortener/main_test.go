package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/audit"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/config"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/handler"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/repository"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/service"
)

func testRequest(t *testing.T, ts *httptest.Server, method string, path string, body io.Reader) (*http.Response, string) {
	req, err := http.NewRequest(method, ts.URL+path, body)
	require.NoError(t, err)

	resp, err := ts.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return resp, string(respBody)
}

func TestRouter(t *testing.T) {
	config.ParseFlags()
	os.Remove(config.Options.StorageFilePath)
	auditObserver := audit.NewAuditMessenger()
	repo, repoErr := repository.NewRepository()
	if repoErr != nil {
		panic(repoErr)
	}
	service.InitRepo(repo)
	ts := httptest.NewServer(handler.MakeHandler(auditObserver))
	defer ts.Close()
	originalURL := "https://ya.ru/"

	resp, get := testRequest(t, ts, "POST", "/", strings.NewReader(originalURL))
	defer resp.Body.Close()
	code := strings.TrimPrefix(get, "http://"+config.Options.ShortenedBaseURL+"/")
	shortURL, _, _ := service.CreateShortURL(t.Context(), originalURL, "1")
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.Equal(t, shortURL, code)

	shortenedCode := strings.TrimPrefix(get, "http://"+config.Options.ShortenedBaseURL+"/")
	getResp, _ := testRequest(t, ts, "GET", "/"+shortenedCode, nil)
	getResp.Body.Close()
	assert.Equal(t, http.StatusOK, getResp.StatusCode) // status code from destination URL on which script is redirected
}
