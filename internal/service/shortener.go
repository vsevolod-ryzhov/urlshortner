package service

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"sync"
)

var (
	ErrNotFound = errors.New("URL not found")
	urlStorage  = make(map[string]string)
	mutex       sync.RWMutex
)

type ShortenedRecord struct {
	UUID        string `json:"uuid,omitempty"`
	ShortURL    string `json:"short_url,omitempty"`
	OriginalURL string `json:"original_url,omitempty"`
}

func CreateShortURL(url string) string {
	shortID := generateShortID(url)

	mutex.Lock()
	urlStorage[shortID] = url
	mutex.Unlock()

	err := SaveToFile()
	if err != nil {
		fmt.Println("Error saving to file")
	}

	return shortID
}

func GetURL(id string) (string, error) {
	mutex.RLock()
	url, exists := urlStorage[id]
	mutex.RUnlock()

	if !exists {
		return "", ErrNotFound
	}

	return url, nil
}

func generateShortID(originalURL string) string {
	mutex.RLock()
	id, ok := urlStorage[originalURL]
	if ok {
		mutex.RUnlock()
		return id
	}
	mutex.RUnlock()

	hash := sha256.Sum256([]byte(originalURL))
	shortID := base64.URLEncoding.EncodeToString(hash[:8])
	return strings.TrimRight(shortID, "=")
}
