package repository

import (
	"context"
	"database/sql"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/model"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(connectionString string) (*PostgresRepository, error) {
	db, err := sql.Open("pgx", connectionString)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &PostgresRepository{db: db}, nil
}

func (r *PostgresRepository) Save(record *model.ShortenedRecord) error {
	query := `INSERT INTO links (id, short, original) 
              VALUES ($1, $2, $3) 
              ON CONFLICT (id) DO UPDATE SET short = $2, original = $3`

	_, err := r.db.Exec(query, record.UUID, record.ShortURL, record.OriginalURL)
	return err
}

func (r *PostgresRepository) GetByUUID(uuid string) (*model.ShortenedRecord, error) {
	var record model.ShortenedRecord
	query := `SELECT id, short, original FROM links WHERE id = $1`

	err := r.db.QueryRow(query, uuid).Scan(&record.UUID, &record.ShortURL, &record.OriginalURL)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}

	return &record, err
}

func (r *PostgresRepository) GetByShortURL(shortURL string) (*model.ShortenedRecord, error) {
	var record model.ShortenedRecord
	query := `SELECT id, short, original FROM links WHERE short = $1`

	err := r.db.QueryRow(query, shortURL).Scan(&record.UUID, &record.ShortURL, &record.OriginalURL)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}

	return &record, err
}

func (r *PostgresRepository) GetAll() (map[string]model.ShortenedRecord, error) {
	ret := make(map[string]model.ShortenedRecord)

	query := `SELECT id, short, original FROM links`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var record model.ShortenedRecord

		if err := rows.Scan(&record.UUID, &record.ShortURL, &record.OriginalURL); err != nil {
			return nil, err
		}

		ret[record.UUID] = record
	}

	return ret, nil
}

func (r *PostgresRepository) Ping(ctx context.Context) error {
	return r.db.PingContext(ctx)
}

func (r *PostgresRepository) Close() error {
	r.db.Close()

	return nil
}
