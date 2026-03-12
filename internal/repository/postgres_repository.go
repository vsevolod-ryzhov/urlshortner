package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/golang-migrate/migrate/v4/source/github"
	_ "github.com/jackc/pgx/v5/stdlib"
	myErrors "github.com/vsevolod-ryzhov/urlshortner.git/internal/errors"
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

// NewPostgresRepository creates new and applies migrations.
func NewPostgresRepository(connectionString string) (*PostgresRepository, error) {
	db, err := sqlOpen("pgx", connectionString)
	if err != nil {
		return nil, err
	}

	if errPing := db.Ping(); errPing != nil {
		return nil, errPing
	}

	migrationDB, err := sql.Open("pgx", connectionString)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to open migration database: %w", err)
	}
	defer migrationDB.Close()

	if errMigrations := applyMigrations(migrationDB); errMigrations != nil {
		db.Close()
		return nil, fmt.Errorf("failed to apply migrations: %w", errMigrations)
	}

	return &PostgresRepository{db: db}, nil
}

// Save inserts new record to links table.
func (r *PostgresRepository) Save(record *model.ShortenedRecord) error {
	query := `INSERT INTO links (id, short, original, user_id) 
              VALUES ($1, $2, $3, $4) 
              ON CONFLICT (id) DO UPDATE SET short = $2, original = $3`

	_, err := r.db.Exec(query, record.UUID, record.ShortURL, record.OriginalURL, record.UserID)
	return err
}

// GetByShortURL looks for an existing record in links table
func (r *PostgresRepository) GetByShortURL(ctx context.Context, shortURL string) (*model.ShortenedRecord, error) {
	var record model.ShortenedRecord
	query := `SELECT id, short, original, user_id, is_deleted FROM links WHERE short = $1`

	err := r.db.QueryRowContext(ctx, query, shortURL).Scan(&record.UUID, &record.ShortURL, &record.OriginalURL, &record.UserID, &record.IsDeleted)
	if err != nil {
		return nil, myErrors.ErrNotFound
	}

	return &record, err
}

// GetAll returns all records from links table.
func (r *PostgresRepository) GetAll() (map[string]model.ShortenedRecord, error) {
	ret := make(map[string]model.ShortenedRecord)

	query := `SELECT id, short, original, user_id, is_deleted FROM links`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var record model.ShortenedRecord

		if err := rows.Scan(&record.UUID, &record.ShortURL, &record.OriginalURL, &record.UserID, &record.IsDeleted); err != nil {
			return nil, err
		}

		ret[record.ShortURL] = record
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return ret, nil
}

// Ping checks database connection.
func (r *PostgresRepository) Ping(ctx context.Context) error {
	err := r.db.PingContext(ctx)
	if err != nil {
		fmt.Println(err)
	}
	return err
}

// Close closes database connection
func (r *PostgresRepository) Close() error {
	r.db.Close()

	return nil
}

// GetData returns empty ShortenedRecord map.
func (r *PostgresRepository) GetData() map[string]model.ShortenedRecord {
	return make(map[string]model.ShortenedRecord)
}

// GetUserURLs returns all ShortenedRecord made by specified user.
func (r *PostgresRepository) GetUserURLs(ctx context.Context, userID string) (map[string]model.ShortenedRecord, error) {
	ret := make(map[string]model.ShortenedRecord)

	query := `SELECT id, short, original, user_id, is_deleted FROM links WHERE user_id = $1`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var record model.ShortenedRecord

		if err := rows.Scan(&record.UUID, &record.ShortURL, &record.OriginalURL, &record.UserID, &record.IsDeleted); err != nil {
			return nil, err
		}

		ret[record.ShortURL] = record
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return ret, nil
}

// BatchDelete removes multiple records by passed short ID map for specified user.
func (r *PostgresRepository) BatchDelete(ctx context.Context, userID string, shortIDs []string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
        UPDATE links 
        SET is_deleted = true
        WHERE user_id = $1 
          AND short = $2
          AND is_deleted = false`,
	)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, shortID := range shortIDs {
		if _, err := stmt.ExecContext(ctx, userID, shortID); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetStats returns total numbers of links and unique users in storage
func (r *PostgresRepository) GetStats(ctx context.Context) (int, int, error) {
	var linksCount, usersCount int
	query := `SELECT count(*) as links, count(distinct user_id) as users FROM links`
	err := r.db.QueryRowContext(ctx, query).Scan(&linksCount, &usersCount)
	if err != nil {
		return 0, 0, err
	}

	return linksCount, usersCount, nil
}
