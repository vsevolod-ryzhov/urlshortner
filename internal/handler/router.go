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

func getStatusCode(alreadyExists bool) int {
	if !alreadyExists {
		return http.StatusCreated
	}
	return http.StatusConflict
}

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
		res.WriteHeader(getStatusCode(alreadyExists))

		enc := json.NewEncoder(res)
		if err := enc.Encode(resp); err != nil {
			logger.Log.Debug("error encoding response", zap.Error(err))
		}

		return
	}

	res.Header().Set("Content-Type", "text/plain")
	res.WriteHeader(getStatusCode(alreadyExists))
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
	userID, ok := service.GetUserIDFromContext(req.Context())
	if !ok {
		userID = "unknown"
	}

	var body []byte
	var err error

	body, err = readCreateLinkRequestBody(req)
	if err != nil {
		http.Error(res, "Bad request", http.StatusBadRequest)
	}

	url := string(body)

	shortened, alreadyExists, errCreation := service.CreateShortURL(req.Context(), url, userID)
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

	url, isDeleted, err := service.GetURL(id)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			http.Error(res, "URL not found", http.StatusNotFound)
		} else {
			http.Error(res, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	if isDeleted {
		res.WriteHeader(http.StatusGone)
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
	userID, ok := service.GetUserIDFromContext(req.Context())
	if !ok {
		userID = "unknown"
	}

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
		shortened, _, errCreation := service.CreateShortURL(req.Context(), request.OriginalURL, userID)
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

func handleUserListURL(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "application/json")

	userID, ok := service.GetUserIDFromContext(req.Context())
	if !ok || userID == "" {
		http.Error(res, "Unauthorized", http.StatusUnauthorized)
		return
	}

	urls, err := service.GetUserURLs(req.Context(), userID)
	if err != nil {
		logger.Log.Error("Failed to get user URLs", zap.Error(err))
		http.Error(res, "Internal server error", http.StatusInternalServerError)
		return
	}

	if len(urls) == 0 {
		res.WriteHeader(http.StatusNoContent)
		return
	}

	response := make([]model.UserURLResponse, 0, len(urls))
	for _, url := range urls {
		response = append(response, model.UserURLResponse{
			ShortURL:    formatShortenedURL(url.ShortURL),
			OriginalURL: url.OriginalURL,
		})
	}

	res.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(res).Encode(response); err != nil {
		logger.Log.Error("Failed to encode user URLs", zap.Error(err))
		http.Error(res, "Internal server error", http.StatusInternalServerError)
		return
	}
}

func handleDeleteURLs(res http.ResponseWriter, req *http.Request) {
	userID, ok := service.GetUserIDFromContext(req.Context())
	if !ok || userID == "" {
		http.Error(res, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var requestModel model.BatchDeleteRequest
	dec := json.NewDecoder(req.Body)
	if err := dec.Decode(&requestModel); err != nil {
		logger.Log.Debug("cannot decode request JSON body", zap.Error(err))
		fmt.Println(err)
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	service.BatchDeleteURLs(userID, requestModel)

	res.WriteHeader(http.StatusAccepted)
}

func MakeHandler() *chi.Mux {
	r := chi.NewRouter()
	r.Get("/{link}", handleGetLink)
	r.Get("/ping", handlePing)
	r.Get("/api/user/urls", handleUserListURL)
	r.Post("/", handleCreateLink)
	r.Post("/api/shorten", handleCreateLink)
	r.Post("/api/shorten/batch", handleBatch)
	r.Delete("/api/user/urls", handleDeleteURLs)

	return r
}
