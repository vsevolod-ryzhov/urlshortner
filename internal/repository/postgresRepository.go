package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/golang-migrate/migrate/v4/source/github"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/model"
)

type PostgresRepository struct {
	db *sql.DB
}

func applyMigrations(db *sql.DB) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres",
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}
	return nil
}

func NewPostgresRepository(connectionString string) (*PostgresRepository, error) {
	db, err := sql.Open("pgx", connectionString)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	migrationDB, err := sql.Open("pgx", connectionString)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to open migration database: %w", err)
	}
	defer migrationDB.Close()

	if err := applyMigrations(migrationDB); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to apply migrations: %w", err)
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

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return ret, nil
}

func (r *PostgresRepository) Ping(ctx context.Context) error {
	err := r.db.PingContext(ctx)
	if err != nil {
		fmt.Println(err)
	}
	return err
}

func (r *PostgresRepository) Close() error {
	r.db.Close()

	return nil
}
