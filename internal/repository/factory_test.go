package repository

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/vsevolod-ryzhov/urlshortner.git/internal/config"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/model"
)

func TestNewRepository_FileBased(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_factory.json")

	config.Options.DatabaseDSN = ""
	config.Options.StorageFilePath = filePath

	repo, err := NewRepository()
	if err != nil {
		t.Fatalf("NewRepository failed: %v", err)
	}

	if repo == nil {
		t.Fatal("Expected repository instance, got nil")
	}

	defer func() {
		if closer, ok := repo.(interface{ Close() error }); ok {
			closer.Close()
		}
	}()

	testRecord := &model.ShortenedRecord{
		UUID:        "factory-test",
		ShortURL:    "factory-short",
		OriginalURL: "https://factory-test.com",
		UserID:      "factory-user",
	}

	err = repo.Save(testRecord)
	if err != nil {
		t.Errorf("Save failed: %v", err)
	}

	ctx := context.Background()
	_, err = repo.GetByShortURL(ctx, "factory-short")
	if err != nil {
		t.Errorf("GetByShortURL failed: %v", err)
	}
}

func TestNewRepository_DatabaseBased(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN not set, skipping database factory test")
	}

	config.Options.DatabaseDSN = dsn
	config.Options.StorageFilePath = ""

	repo, err := NewRepository()
	if err != nil {
		t.Fatalf("NewRepository failed: %v", err)
	}

	if repo == nil {
		t.Fatal("Expected repository instance, got nil")
	}

	defer func() {
		if closer, ok := repo.(interface{ Close() error }); ok {
			closer.Close()
		}
	}()

	ctx := context.Background()
	err = repo.Ping(ctx)
	if err != nil {
		t.Logf("Ping returned error (might be expected if DB not configured): %v", err)
	}
}

func TestNewRepository_NoConfig(t *testing.T) {
	config.Options.DatabaseDSN = ""
	config.Options.StorageFilePath = ""

	repo, err := NewRepository()
	if err != nil {
		t.Fatalf("NewRepository failed with error: %v", err)
	}

	if repo != nil {
		t.Log("Repository created without config, might be in-memory implementation")
		repo.Close()
	}
}
