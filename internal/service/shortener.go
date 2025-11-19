package service

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/vsevolod-ryzhov/urlshortner.git/internal/model"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/repository"
)

var (
	ErrNotFound = errors.New("URL not found")
	urlStorage  = make(map[string]model.ShortenedRecord)
	mutex       sync.RWMutex
	Repo        repository.Repository
)

func InitRepo(r repository.Repository) error {
	Repo = r

	if Repo != nil {
		urlStorage = Repo.GetData()
	}

	return nil
}

func generateUUID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func CreateShortURL(ctx context.Context, url string) (string, bool, error) {
	shortID := generateShortID(url)

	if Repo == nil {
		var existingRecord model.ShortenedRecord
		var exists bool
		mutex.RLock()
		existingRecord, exists = urlStorage[shortID]
		mutex.RUnlock()
		if exists {
			return existingRecord.ShortURL, true, nil
		}
	} else {
		var r *model.ShortenedRecord
		var e error
		r, e = Repo.GetByShortURL(ctx, shortID)
		if e == nil {
			return r.ShortURL, true, nil
		}
	}

	mutex.Lock()
	defer mutex.Unlock()

	record := model.ShortenedRecord{
		UUID:        generateUUID(),
		ShortURL:    shortID,
		OriginalURL: url,
	}

	urlStorage[shortID] = record

	if Repo != nil {
		Repo.Save(&record)
	}

	return shortID, false, nil
}

func GetURL(id string) (string, error) {
	mutex.RLock()
	record, exists := urlStorage[id]
	mutex.RUnlock()

	if !exists {
		return "", ErrNotFound
	}

	return record.OriginalURL, nil
}

func generateShortID(originalURL string) string {
	mutex.RLock()
	record, ok := urlStorage[originalURL]
	if ok {
		mutex.RUnlock()
		return record.ShortURL
	}
	mutex.RUnlock()

	hash := sha256.Sum256([]byte(originalURL))
	shortID := base64.URLEncoding.EncodeToString(hash[:8])
	return strings.TrimRight(shortID, "=")
}
