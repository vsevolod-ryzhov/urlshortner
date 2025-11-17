package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/config"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/logger"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/model"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/service"
	"go.uber.org/zap"
)

func readCreateLinkRequestBody(req *http.Request) ([]byte, error) {
	var body []byte
	var err error

	if req.Header.Get("Content-Type") == "application/json" {
		var requestModel model.JSONRequest
		dec := json.NewDecoder(req.Body)
		if err := dec.Decode(&requestModel); err != nil {
			logger.Log.Debug("cannot decode request JSON body", zap.Error(err))
			return nil, err
		}
		return []byte(requestModel.URL), nil
	}

	body, err = io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func sendCreateLinkResponse(res http.ResponseWriter, req *http.Request, shortenedURL string, alreadyExists bool) {
	if req.Header.Get("Content-Type") == "application/json" {
		resp := model.JSONResponse{
			Result: shortenedURL,
		}
		res.Header().Set("Content-Type", "application/json")
		if !alreadyExists {
			res.WriteHeader(http.StatusCreated)
		} else {
			res.WriteHeader(http.StatusConflict)
		}

		enc := json.NewEncoder(res)
		if err := enc.Encode(resp); err != nil {
			logger.Log.Debug("error encoding response", zap.Error(err))
		}

		return
	}

	res.Header().Set("Content-Type", "text/plain")
	if !alreadyExists {
		res.WriteHeader(http.StatusCreated)
	} else {
		res.WriteHeader(http.StatusConflict)
	}
	_, err := res.Write([]byte(shortenedURL))

	if err != nil {
		logger.Log.Debug("error writing response", zap.Error(err))
	}
}

func formatShortenedURL(shortenedURL string) string {
	baseURL := config.Options.ShortenedBaseURL
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "http://" + baseURL
	}
	return fmt.Sprintf("%s/%s", strings.TrimSuffix(baseURL, "/"), shortenedURL)
}

func handleCreateLink(res http.ResponseWriter, req *http.Request) {
	var body []byte
	var err error

	body, err = readCreateLinkRequestBody(req)
	if err != nil {
		http.Error(res, "Bad request", http.StatusBadRequest)
	}

	url := string(body)

	shortened, alreadyExists, errCreation := service.CreateShortURL(url)
	if errCreation != nil {
		logger.Log.Debug("Shortened result was not saved to file", zap.Error(errCreation))
	}

	sendCreateLinkResponse(res, req, formatShortenedURL(shortened), alreadyExists)
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

func handlePing(res http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if err := service.Repo.Ping(ctx); err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	res.WriteHeader(http.StatusOK)
}

func handleBatch(res http.ResponseWriter, req *http.Request) {
	if req.Header.Get("Content-Type") != "application/json" {
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	var requestModel model.JSONBatchRequest
	dec := json.NewDecoder(req.Body)
	if err := dec.Decode(&requestModel); err != nil {
		logger.Log.Debug("cannot decode request JSON body", zap.Error(err))
		fmt.Println(err)
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	var responseModel model.JSONBatchResponse
	for _, request := range requestModel {
		shortened, _, errCreation := service.CreateShortURL(request.OriginalURL)
		if errCreation != nil {
			logger.Log.Debug("Shortened result was not saved to file", zap.Error(errCreation))
		}

		responseModel = append(responseModel, model.JSONBatchResponseItem{
			CorrelationID: request.CorrelationID,
			ShortURL:      formatShortenedURL(shortened),
		})
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(res).Encode(responseModel); err != nil {
		logger.Log.Debug("Failed to encode response", zap.Error(err))
		res.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func MakeHandler() *chi.Mux {
	r := chi.NewRouter()
	r.Get("/{link}", handleGetLink)
	r.Get("/ping", handlePing)
	r.Post("/", handleCreateLink)
	r.Post("/api/shorten", handleCreateLink)
	r.Post("/api/shorten/batch", handleBatch)

	return r
}
