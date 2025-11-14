package service

import (
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

	var err error
	urlStorage, err = r.GetAll()

	if err != nil {
		return err
	}

	return nil
}

func generateUUID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func CreateShortURL(url string) (string, error) {
	shortID := generateShortID(url)

	mutex.RLock()
	existingRecord, exists := urlStorage[shortID]
	mutex.RUnlock()

	if exists {
		return existingRecord.ShortURL, nil
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
	//if err := storage.SaveToFile(record); err != nil {
	//	return shortID, err
	//}

	return shortID, nil
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
