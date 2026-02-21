package repository

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/vsevolod-ryzhov/urlshortner.git/internal/config"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/model"
)

func TestFileRepository_SaveAndGetByShortURL(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_urls.json")

	config.Options.StorageFilePath = filePath

	repo, err := NewFileRepository(filePath)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	testRecord := &model.ShortenedRecord{
		UUID:        "test-uuid-123",
		ShortURL:    "abc123",
		OriginalURL: "https://example.com",
		UserID:      "test-user",
		IsDeleted:   false,
	}

	err = repo.Save(testRecord)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	ctx := context.Background()
	retrievedRecord, err := repo.GetByShortURL(ctx, "abc123")
	if err != nil {
		t.Fatalf("GetByShortURL failed: %v", err)
	}

	if retrievedRecord.ShortURL != testRecord.ShortURL {
		t.Errorf("ShortURL mismatch: got %s, want %s", retrievedRecord.ShortURL, testRecord.ShortURL)
	}
	if retrievedRecord.OriginalURL != testRecord.OriginalURL {
		t.Errorf("OriginalURL mismatch: got %s, want %s", retrievedRecord.OriginalURL, testRecord.OriginalURL)
	}
	if retrievedRecord.UserID != testRecord.UserID {
		t.Errorf("UserID mismatch: got %s, want %s", retrievedRecord.UserID, testRecord.UserID)
	}
}

func TestFileRepository_GetByShortURL_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_urls.json")
	config.Options.StorageFilePath = filePath

	repo, err := NewFileRepository(filePath)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	ctx := context.Background()
	_, err = repo.GetByShortURL(ctx, "non-existent-short-url")
	if err == nil {
		t.Error("Expected error for non-existent short URL, got nil")
	}
}

func TestFileRepository_SaveMultipleAndGetAll(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_multiple.json")
	config.Options.StorageFilePath = filePath

	repo, err := NewFileRepository(filePath)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	records := []model.ShortenedRecord{
		{
			UUID:        "uuid-1",
			ShortURL:    "short-1",
			OriginalURL: "https://example1.com",
			UserID:      "user-1",
		},
		{
			UUID:        "uuid-2",
			ShortURL:    "short-2",
			OriginalURL: "https://example2.com",
			UserID:      "user-2",
		},
		{
			UUID:        "uuid-3",
			ShortURL:    "short-3",
			OriginalURL: "https://example3.com",
			UserID:      "user-3",
		},
	}

	for i := range records {
		err = repo.Save(&records[i])
		if err != nil {
			t.Fatalf("Failed to save record %d: %v", i, err)
		}
	}

	allRecords, err := repo.GetAll()
	if err != nil {
		t.Fatalf("GetAll failed: %v", err)
	}

	if len(allRecords) != len(records) {
		t.Errorf("Record count mismatch: got %d, want %d", len(allRecords), len(records))
	}

	ctx := context.Background()
	for _, expectedRecord := range records {
		retrievedRecord, err := repo.GetByShortURL(ctx, expectedRecord.ShortURL)
		if err != nil {
			t.Errorf("Failed to get record with short URL %s: %v", expectedRecord.ShortURL, err)
			continue
		}

		if retrievedRecord.OriginalURL != expectedRecord.OriginalURL {
			t.Errorf("OriginalURL mismatch for %s: got %s, want %s",
				expectedRecord.ShortURL, retrievedRecord.OriginalURL, expectedRecord.OriginalURL)
		}
	}
}

func TestFileRepository_GetAll(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_getall.json")
	config.Options.StorageFilePath = filePath

	repo, err := NewFileRepository(filePath)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	expectedRecords := []model.ShortenedRecord{
		{
			UUID:        "uuid-1",
			ShortURL:    "short-1",
			OriginalURL: "https://example1.com",
			UserID:      "user-1",
		},
		{
			UUID:        "uuid-2",
			ShortURL:    "short-2",
			OriginalURL: "https://example2.com",
			UserID:      "user-2",
		},
	}

	for _, record := range expectedRecords {
		r := record
		err = repo.Save(&r)
		if err != nil {
			t.Fatalf("Failed to save record: %v", err)
		}
	}

	allRecords, err := repo.GetAll()
	if err != nil {
		t.Fatalf("GetAll failed: %v", err)
	}

	if len(allRecords) != len(expectedRecords) {
		t.Errorf("Record count mismatch: got %d, want %d", len(allRecords), len(expectedRecords))
	}

	for _, expectedRecord := range expectedRecords {
		foundRecord, exists := allRecords[expectedRecord.UUID]
		if !exists {
			t.Errorf("Record with short URL %s not found in GetAll result", expectedRecord.ShortURL)
			continue
		}

		if foundRecord.OriginalURL != expectedRecord.OriginalURL {
			t.Errorf("OriginalURL mismatch for %s: got %s, want %s",
				expectedRecord.ShortURL, foundRecord.OriginalURL, expectedRecord.OriginalURL)
		}
	}
}

func TestFileRepository_GetAll_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_empty.json")
	config.Options.StorageFilePath = filePath

	repo, err := NewFileRepository(filePath)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	allRecords, err := repo.GetAll()
	if err != nil {
		t.Fatalf("GetAll failed: %v", err)
	}

	if len(allRecords) != 0 {
		t.Errorf("Expected empty map, got %d records", len(allRecords))
	}
}

func TestFileRepository_Persistence(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_persistence.json")
	config.Options.StorageFilePath = filePath

	repo1, err := NewFileRepository(filePath)
	if err != nil {
		t.Fatalf("Failed to create first repository: %v", err)
	}

	testRecord := &model.ShortenedRecord{
		UUID:        "persistent-uuid",
		ShortURL:    "persistent-short",
		OriginalURL: "https://persistent.com",
		UserID:      "persistent-user",
	}

	err = repo1.Save(testRecord)
	if err != nil {
		t.Fatalf("Save to first repo failed: %v", err)
	}
	repo1.Close()

	repo2, err := NewFileRepository(filePath)
	if err != nil {
		t.Fatalf("Failed to create second repository: %v", err)
	}
	defer repo2.Close()

	ctx := context.Background()
	retrievedRecord, err := repo2.GetByShortURL(ctx, "persistent-short")
	if err != nil {
		t.Fatalf("Failed to retrieve persisted record: %v", err)
	}

	if retrievedRecord.OriginalURL != "https://persistent.com" {
		t.Errorf("Data not persisted: got %s, want %s",
			retrievedRecord.OriginalURL, "https://persistent.com")
	}
}

func TestFileRepository_GetUserURLs(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_user_urls.json")
	config.Options.StorageFilePath = filePath

	repo, err := NewFileRepository(filePath)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	user1Records := []model.ShortenedRecord{
		{
			UUID:        "user1-uuid-1",
			ShortURL:    "user1-short-1",
			OriginalURL: "https://user1-site1.com",
			UserID:      "user-1",
		},
		{
			UUID:        "user1-uuid-2",
			ShortURL:    "user1-short-2",
			OriginalURL: "https://user1-site2.com",
			UserID:      "user-1",
		},
	}

	user2Records := []model.ShortenedRecord{
		{
			UUID:        "user2-uuid-1",
			ShortURL:    "user2-short-1",
			OriginalURL: "https://user2-site1.com",
			UserID:      "user-2",
		},
	}

	for _, record := range append(user1Records, user2Records...) {
		r := record
		err = repo.Save(&r)
		if err != nil {
			t.Fatalf("Failed to save record: %v", err)
		}
	}

	ctx := context.Background()
	user1URLs, err := repo.GetUserURLs(ctx, "user-1")
	if err != nil {
		t.Fatalf("GetUserURLs failed for user-1: %v", err)
	}

	if len(user1URLs) != 2 {
		t.Errorf("Expected 2 URLs for user-1, got %d", len(user1URLs))
	}

	for _, expectedRecord := range user1Records {
		foundRecord, exists := user1URLs[expectedRecord.ShortURL]
		if !exists {
			t.Errorf("Record %s not found for user-1", expectedRecord.ShortURL)
			continue
		}
		if foundRecord.OriginalURL != expectedRecord.OriginalURL {
			t.Errorf("OriginalURL mismatch: got %s, want %s",
				foundRecord.OriginalURL, expectedRecord.OriginalURL)
		}
	}

	user2URLs, err := repo.GetUserURLs(ctx, "user-2")
	if err != nil {
		t.Fatalf("GetUserURLs failed for user-2: %v", err)
	}

	if len(user2URLs) != 1 {
		t.Errorf("Expected 1 URL for user-2, got %d", len(user2URLs))
	}

	nonExistentURLs, err := repo.GetUserURLs(ctx, "non-existent-user")
	if err != nil {
		t.Fatalf("GetUserURLs failed for non-existent user: %v", err)
	}

	if len(nonExistentURLs) != 0 {
		t.Errorf("Expected 0 URLs for non-existent user, got %d", len(nonExistentURLs))
	}
}

func TestFileRepository_Ping(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_ping.json")
	config.Options.StorageFilePath = filePath

	repo, err := NewFileRepository(filePath)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	ctx := context.Background()
	err = repo.Ping(ctx)

	if err == nil {
		t.Error("Expected error from FileRepository.Ping, got nil")
	}
}

func TestFileRepository_ConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_concurrent.json")
	config.Options.StorageFilePath = filePath

	repo, err := NewFileRepository(filePath)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	done := make(chan bool)
	errors := make(chan error, 10)

	for i := 0; i < 5; i++ {
		go func(id int) {
			record := &model.ShortenedRecord{
				UUID:        string(rune('A' + id)),
				ShortURL:    string(rune('a' + id)),
				OriginalURL: "https://example.com",
				UserID:      "test-user",
			}

			if errSave := repo.Save(record); errSave != nil {
				errors <- errSave
				return
			}

			ctx := context.Background()
			_, errGet := repo.GetByShortURL(ctx, string(rune('a'+id)))
			if errGet != nil {
				errors <- errGet
				return
			}

			done <- true
		}(i)
	}

	for i := 0; i < 5; i++ {
		select {
		case <-done:
		case errCh := <-errors:
			t.Errorf("Concurrent operation failed: %v", errCh)
		}
	}

	allRecords, err := repo.GetAll()
	if err != nil {
		t.Fatalf("GetAll failed: %v", err)
	}

	if len(allRecords) != 5 {
		t.Errorf("Expected 5 records after concurrent access, got %d", len(allRecords))
	}
}

func TestFileRepository_Close(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_close.json")
	config.Options.StorageFilePath = filePath

	repo, err := NewFileRepository(filePath)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	testRecord := &model.ShortenedRecord{
		UUID:        "close-test",
		ShortURL:    "close-short",
		OriginalURL: "https://close-test.com",
		UserID:      "test-user",
	}

	err = repo.Save(testRecord)
	if err != nil {
		t.Fatalf("Save failed before close: %v", err)
	}

	err = repo.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	err = repo.Close()
	if err != nil {
		t.Logf("Second close returned error (might be expected): %v", err)
	}
}
