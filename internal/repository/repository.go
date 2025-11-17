package repository

import (
	"context"

	"github.com/vsevolod-ryzhov/urlshortner.git/internal/model"
)

type Repository interface {
	Save(record *model.ShortenedRecord) error
	GetByUUID(uuid string) (*model.ShortenedRecord, error)
	GetByShortURL(shortURL string) (*model.ShortenedRecord, error)
	GetAll() (map[string]model.ShortenedRecord, error)
	Ping(ctx context.Context) error
	Close() error
	GetData() map[string]model.ShortenedRecord
}
