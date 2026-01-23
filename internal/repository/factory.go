package repository

import "github.com/vsevolod-ryzhov/urlshortner.git/internal/config"

// NewRepository create an instance of Repository interface based on startup options
func NewRepository() (Repository, error) {
	if config.Options.DatabaseDSN != "" {
		return NewPostgresRepository(config.Options.DatabaseDSN)
	}

	if config.Options.StorageFilePath != "" {
		return NewFileRepository(config.Options.StorageFilePath)
	}

	return nil, nil
}
