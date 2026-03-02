package repository

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/vsevolod-ryzhov/urlshortner.git/internal/config"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/errors"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/model"
)

type FileRepository struct {
	filePath string
	mu       sync.RWMutex
	data     map[string]model.ShortenedRecord
}

// NewFileRepository creates new FileRepository object and reads all stored data from file located in filePath.
func NewFileRepository(filePath string) (*FileRepository, error) {
	repo := &FileRepository{
		filePath: filePath,
		data:     make(map[string]model.ShortenedRecord),
	}

	if err := repo.loadFromFile(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	return repo, nil
}

// Save func append new record to file.
func (r *FileRepository) Save(record *model.ShortenedRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.data[record.UUID] = *record
	return r.saveToFile(*record)
}

// GetByShortURL looks for an record by short URL, returns pointer to record or error ErrNotFound if record is not found.
func (r *FileRepository) GetByShortURL(ctx context.Context, shortURL string) (*model.ShortenedRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, record := range r.data {
		if record.ShortURL == shortURL {
			return &record, nil
		}
	}
	return nil, errors.ErrNotFound
}

// GetAll returns all store records.
func (r *FileRepository) GetAll() (map[string]model.ShortenedRecord, error) {
	return r.data, nil
}

func (r *FileRepository) loadFromFile() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	file, err := os.OpenFile(config.Options.StorageFilePath, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var record model.ShortenedRecord
		if err := json.Unmarshal(line, &record); err != nil {
			return err
		}
		r.data[record.ShortURL] = record
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

func (r *FileRepository) saveToFile(record model.ShortenedRecord) error {
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

func (r *FileRepository) Ping(ctx context.Context) error {
	return fmt.Errorf("not applicable")
}

func (r *FileRepository) Close() error {
	return nil
}

func (r *FileRepository) GetData() map[string]model.ShortenedRecord {
	return r.data
}

// GetUserURLs returns all records made by specified user.
func (r *FileRepository) GetUserURLs(ctx context.Context, userID string) (map[string]model.ShortenedRecord, error) {
	ret := make(map[string]model.ShortenedRecord)

	for _, record := range r.data {
		if record.UserID == userID {
			ret[record.ShortURL] = record
		}
	}

	return ret, nil
}

// BatchDelete removes multiple records by passed short ID map for specified user.
func (r *FileRepository) BatchDelete(ctx context.Context, userID string, shortIDs []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(shortIDs) == 0 {
		return nil
	}

	var recordsToSave []model.ShortenedRecord

	for _, shortID := range shortIDs {
		record, exists := r.data[shortID]
		if !exists {
			continue
		}

		if record.UserID == userID && !record.IsDeleted {
			record.IsDeleted = true
			r.data[shortID] = record

			recordsToSave = append(recordsToSave, record)
		}
	}

	for _, record := range recordsToSave {
		if err := r.saveToFile(record); err != nil {
			return err
		}
	}

	return nil
}

// GetStats returns total numbers of links and unique users in storage
func (r *FileRepository) GetStats(ctx context.Context) (int, int, error) {
	uniqueUsers := make(map[string]struct{})

	for _, record := range r.data {
		uniqueUsers[record.UserID] = struct{}{}
	}

	return len(r.data), len(uniqueUsers), nil
}
