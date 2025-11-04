package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/config"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/logger"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/model"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/service"
	"go.uber.org/zap"
)

func handleCreateLink(res http.ResponseWriter, req *http.Request) {
	var body []byte
	var err error

	if req.Header.Get("Content-Type") == "application/json" {
		var requestModel model.JSONRequest
		dec := json.NewDecoder(req.Body)
		if err := dec.Decode(&requestModel); err != nil {
			logger.Log.Debug("cannot decode request JSON body", zap.Error(err))
			res.WriteHeader(http.StatusInternalServerError)
			return
		}
		body = []byte(requestModel.URL)
	} else {
		body, err = io.ReadAll(req.Body)
		if err != nil {
			http.Error(res, "Bad request", http.StatusBadRequest)
			return
		}
	}

	url := string(body)

	shortened := service.CreateShortURL(url)
	baseURL := config.Options.ShortenedBaseURL
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "http://" + baseURL
	}
	shortenedURL := fmt.Sprintf("%s/%s", strings.TrimSuffix(baseURL, "/"), shortened)

	if req.Header.Get("Content-Type") == "application/json" {
		resp := model.JSONResponse{
			Result: shortenedURL,
		}
		res.Header().Set("Content-Type", "application/json")
		res.WriteHeader(http.StatusCreated)

		enc := json.NewEncoder(res)
		if err := enc.Encode(resp); err != nil {
			logger.Log.Debug("error encoding response", zap.Error(err))
			return
		}
	} else {
		res.Header().Set("Content-Type", "text/plain")
		res.WriteHeader(http.StatusCreated)
		_, err := res.Write([]byte(shortenedURL))
		if err != nil {
			return
		}
	}
}

func handleGetLink(res http.ResponseWriter, req *http.Request) {
	id := strings.TrimPrefix(req.URL.Path, "/")

	if id == "" {
		http.Error(res, "ID is not specified", http.StatusBadRequest)
		return
	}

	url, err := service.GetURL(id)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			http.Error(res, "URL not found", http.StatusNotFound)
		} else {
			http.Error(res, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	http.Redirect(res, req, url, http.StatusTemporaryRedirect)
}

func MakeHandler() *chi.Mux {
	r := chi.NewRouter()
	r.Get("/{link}", handleGetLink)
	r.Post("/", handleCreateLink)
	r.Post("/api/shorten", handleCreateLink)

	return r
}
