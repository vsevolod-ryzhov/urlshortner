package storage

import (
	"encoding/json"
	"os"
	"sync"

	"github.com/vsevolod-ryzhov/urlshortner.git/internal/config"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/model"
)

var mutex sync.RWMutex

func initEmptyFile() error {
	var emptyArray []model.ShortenedRecord
	data, err := json.Marshal(emptyArray)
	if err != nil {
		return err
	}

	return os.WriteFile(config.Options.StorageFilePath, data, 0644)
}

func LoadFromFile() ([]model.ShortenedRecord, error) {
	mutex.Lock()
	defer mutex.Unlock()

	data, err := os.ReadFile(config.Options.StorageFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, initEmptyFile()
		}
		return nil, err
	}

	if len(data) == 0 {
		return nil, initEmptyFile()
	}

	var records []model.ShortenedRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, err
	}

	return records, nil
}

func SaveToFile(record model.ShortenedRecord) error {
	data, err := os.ReadFile(config.Options.StorageFilePath)
	if err != nil {
		return err
	}

	var records []model.ShortenedRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return err
	}

	records = append(records, record)

	newData, err := json.Marshal(records)
	if err != nil {
		return err
	}

	return os.WriteFile(config.Options.StorageFilePath, newData, 0644)
}
