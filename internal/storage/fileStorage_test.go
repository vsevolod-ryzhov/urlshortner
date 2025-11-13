package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/vsevolod-ryzhov/urlshortner.git/internal/config"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/model"
)

func TestLoadFromFile(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		fileContent string
		wantRecords []model.ShortenedRecord
		wantErr     bool
	}{
		{
			name:        "empty file",
			fileContent: "",
			wantRecords: []model.ShortenedRecord{},
			wantErr:     false,
		},
		{
			name: "valid json lines",
			fileContent: `{"uuid":"1","short_url":"abc","original_url":"https://example.com"}` + "\n" +
				`{"uuid":"2","short_url":"def","original_url":"https://google.com"}`,
			wantRecords: []model.ShortenedRecord{
				{UUID: "1", ShortURL: "abc", OriginalURL: "https://example.com"},
				{UUID: "2", ShortURL: "def", OriginalURL: "https://google.com"},
			},
			wantErr: false,
		},
		{
			name: "with empty lines",
			fileContent: `{"uuid":"1","short_url":"abc","original_url":"https://example.com"}` + "\n" +
				"\n" +
				`{"uuid":"2","short_url":"def","original_url":"https://google.com"}`,
			wantRecords: []model.ShortenedRecord{
				{UUID: "1", ShortURL: "abc", OriginalURL: "https://example.com"},
				{UUID: "2", ShortURL: "def", OriginalURL: "https://google.com"},
			},
			wantErr: false,
		},
		{
			name: "invalid json",
			fileContent: `{"uuid":"1","short_url":"abc","original_url":"https://example.com"}` + "\n" +
				`invalid json`,
			wantRecords: nil,
			wantErr:     true,
		},
		{
			name: "partial records",
			fileContent: `{"uuid":"1","short_url":"abc"}` + "\n" +
				`{"uuid":"2","original_url":"https://google.com"}`,
			wantRecords: []model.ShortenedRecord{
				{UUID: "1", ShortURL: "abc"},
				{UUID: "2", OriginalURL: "https://google.com"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile := filepath.Join(tmpDir, "test.jsonl")
			err := os.WriteFile(tmpFile, []byte(tt.fileContent), 0644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			originalPath := config.Options.StorageFilePath
			config.Options.StorageFilePath = tmpFile
			defer func() {
				config.Options.StorageFilePath = originalPath
			}()

			gotRecords, err := LoadFromFile()

			if (err != nil) != tt.wantErr {
				t.Errorf("LoadFromFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(gotRecords) != len(tt.wantRecords) {
				t.Errorf("LoadFromFile() returned %d records, want %d", len(gotRecords), len(tt.wantRecords))
				return
			}

			for i, wantRecord := range tt.wantRecords {
				if i >= len(gotRecords) {
					break
				}
				if gotRecords[i].UUID != wantRecord.UUID ||
					gotRecords[i].ShortURL != wantRecord.ShortURL ||
					gotRecords[i].OriginalURL != wantRecord.OriginalURL {
					t.Errorf("LoadFromFile() record[%d] = %+v, want %+v", i, gotRecords[i], wantRecord)
				}
			}
		})
	}
}

func TestSaveToFile(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "save_test.jsonl")

	originalPath := config.Options.StorageFilePath
	config.Options.StorageFilePath = tmpFile
	defer func() {
		config.Options.StorageFilePath = originalPath
	}()

	tests := []struct {
		name    string
		records []model.ShortenedRecord
		wantErr bool
	}{
		{
			name: "single record",
			records: []model.ShortenedRecord{
				{UUID: "1", ShortURL: "abc", OriginalURL: "https://example.com"},
			},
			wantErr: false,
		},
		{
			name: "multiple records",
			records: []model.ShortenedRecord{
				{UUID: "1", ShortURL: "abc", OriginalURL: "https://example.com"},
				{UUID: "2", ShortURL: "def", OriginalURL: "https://google.com"},
				{UUID: "3", ShortURL: "ghi", OriginalURL: "https://github.com"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Remove(tmpFile)

			for _, record := range tt.records {
				err := SaveToFile(record)
				if (err != nil) != tt.wantErr {
					t.Errorf("SaveToFile() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
			}

			if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
				t.Error("SaveToFile() failed to create file")
				return
			}

			content, err := os.ReadFile(tmpFile)
			if err != nil {
				t.Errorf("Failed to read test file: %v", err)
				return
			}

			lines := splitLines(string(content))
			if len(lines) != len(tt.records) {
				t.Errorf("Expected %d lines in file, got %d", len(tt.records), len(lines))
				return
			}

			for i, line := range lines {
				if line == "" {
					continue
				}

				var record model.ShortenedRecord
				if err := json.Unmarshal([]byte(line), &record); err != nil {
					t.Errorf("Failed to unmarshal line %d: %v", i, err)
					continue
				}

				wantRecord := tt.records[i]
				if record.UUID != wantRecord.UUID ||
					record.ShortURL != wantRecord.ShortURL ||
					record.OriginalURL != wantRecord.OriginalURL {
					t.Errorf("Line %d: got %+v, want %+v", i, record, wantRecord)
				}
			}
		})
	}
}

func TestSaveAndLoadIntegration(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "integration_test.jsonl")

	originalPath := config.Options.StorageFilePath
	config.Options.StorageFilePath = tmpFile
	defer func() {
		config.Options.StorageFilePath = originalPath
	}()

	recordsToSave := []model.ShortenedRecord{
		{UUID: "1", ShortURL: "abc", OriginalURL: "https://example.com"},
		{UUID: "2", ShortURL: "def", OriginalURL: "https://google.com"},
		{UUID: "3", ShortURL: "ghi", OriginalURL: "https://github.com"},
	}

	// Сохраняем записи
	for _, record := range recordsToSave {
		if err := SaveToFile(record); err != nil {
			t.Fatalf("Failed to save record: %v", err)
		}
	}

	loadedRecords, err := LoadFromFile()
	if err != nil {
		t.Fatalf("Failed to load records: %v", err)
	}

	if len(loadedRecords) != len(recordsToSave) {
		t.Errorf("Expected %d records, got %d", len(recordsToSave), len(loadedRecords))
	}

	for i, loaded := range loadedRecords {
		expected := recordsToSave[i]
		if loaded.UUID != expected.UUID ||
			loaded.ShortURL != expected.ShortURL ||
			loaded.OriginalURL != expected.OriginalURL {
			t.Errorf("Record %d mismatch: got %+v, want %+v", i, loaded, expected)
		}
	}
}

func splitLines(s string) []string {
	var lines []string
	var line []byte
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, string(line))
			line = nil
		} else {
			line = append(line, s[i])
		}
	}
	if len(line) > 0 {
		lines = append(lines, string(line))
	}
	return lines
}
