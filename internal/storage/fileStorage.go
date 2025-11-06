package storage

import (
	"bufio"
	"encoding/json"
	"os"
	"sync"

	"github.com/vsevolod-ryzhov/urlshortner.git/internal/config"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/model"
)

var mutex sync.RWMutex

func LoadFromFile() ([]model.ShortenedRecord, error) {
	mutex.Lock()
	defer mutex.Unlock()

	file, err := os.OpenFile(config.Options.StorageFilePath, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var records []model.ShortenedRecord
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var record model.ShortenedRecord
		if err := json.Unmarshal(line, &record); err != nil {
			return nil, err
		}
		records = append(records, record)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return records, nil
}

func SaveToFile(record model.ShortenedRecord) error {
	file, err := os.OpenFile(config.Options.StorageFilePath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	encoder := json.NewEncoder(writer)

	if err := encoder.Encode(record); err != nil {
		return err
	}

	return writer.Flush()
}
