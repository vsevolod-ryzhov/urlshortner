package service

import (
	"encoding/json"
	"os"
	"strconv"

	"github.com/vsevolod-ryzhov/urlshortner.git/internal/config"
)

func LoadFromFile() error {
	mutex.Lock()
	defer mutex.Unlock()

	data, err := os.ReadFile(config.Options.StorageFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var records []ShortenedRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return err
	}

	for _, record := range records {
		urlStorage[record.ShortURL] = record.OriginalURL
	}

	return nil
}

func SaveToFile() error {
	mutex.RLock()
	defer mutex.RUnlock()

	records := make([]ShortenedRecord, 0, len(urlStorage))

	shortToUUID := make(map[string]string)
	uuidCounter := 1

	for shortURL, originalURL := range urlStorage {
		uuid := strconv.Itoa(uuidCounter)
		records = append(records, ShortenedRecord{
			UUID:        uuid,
			ShortURL:    shortURL,
			OriginalURL: originalURL,
		})
		shortToUUID[shortURL] = uuid
		uuidCounter++
	}

	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(config.Options.StorageFilePath, data, 0644)
}
