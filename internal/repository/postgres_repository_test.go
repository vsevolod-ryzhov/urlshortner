package repository

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	myErrors "github.com/vsevolod-ryzhov/urlshortner.git/internal/errors"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/model"
)

type PostgresRepositoryTestSuite struct {
	suite.Suite
	repo   *PostgresRepository
	db     *sql.DB
	mockDB sqlmock.Sqlmock
	ctx    context.Context
}

var (
	originalSQLOpen         = sqlOpen
	originalApplyMigrations = applyMigrationsFunc
)

func (suite *PostgresRepositoryTestSuite) SetupSuite() {
	suite.ctx = context.Background()
}

func (suite *PostgresRepositoryTestSuite) SetupTest() {
	var err error
	suite.db, suite.mockDB, err = sqlmock.New(
		sqlmock.MonitorPingsOption(true),
	)
	require.NoError(suite.T(), err)

	suite.repo = &PostgresRepository{db: suite.db}
}

func (suite *PostgresRepositoryTestSuite) TearDownTest() {
	suite.db.Close()
}

func TestPostgresRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(PostgresRepositoryTestSuite))
}

func (suite *PostgresRepositoryTestSuite) TestSave() {
	record := &model.ShortenedRecord{
		UUID:        "test-uuid",
		ShortURL:    "short",
		OriginalURL: "original",
		UserID:      "user123",
		IsDeleted:   false,
	}

	suite.mockDB.ExpectExec(`INSERT INTO links .*`).
		WithArgs(record.UUID, record.ShortURL, record.OriginalURL, record.UserID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := suite.repo.Save(record)

	assert.NoError(suite.T(), err)
	assert.NoError(suite.T(), suite.mockDB.ExpectationsWereMet())
}

func (suite *PostgresRepositoryTestSuite) TestSave_Error() {
	record := &model.ShortenedRecord{
		UUID:        "test-uuid",
		ShortURL:    "short",
		OriginalURL: "original",
		UserID:      "user123",
	}

	expectedError := fmt.Errorf("database error")
	suite.mockDB.ExpectExec(`INSERT INTO links .*`).
		WithArgs(record.UUID, record.ShortURL, record.OriginalURL, record.UserID).
		WillReturnError(expectedError)

	err := suite.repo.Save(record)

	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), expectedError, err)
	assert.NoError(suite.T(), suite.mockDB.ExpectationsWereMet())
}

func (suite *PostgresRepositoryTestSuite) TestGetByShortURL_Found() {
	shortURL := "abc123"
	expectedRecord := &model.ShortenedRecord{
		UUID:        "test-uuid",
		ShortURL:    shortURL,
		OriginalURL: "http://example.com",
		UserID:      "user123",
		IsDeleted:   false,
	}

	rows := sqlmock.NewRows([]string{"id", "short", "original", "user_id", "is_deleted"}).
		AddRow(expectedRecord.UUID, expectedRecord.ShortURL,
			expectedRecord.OriginalURL, expectedRecord.UserID, expectedRecord.IsDeleted)

	suite.mockDB.ExpectQuery(`SELECT .* FROM links WHERE short = \$1`).
		WithArgs(shortURL).
		WillReturnRows(rows)

	record, err := suite.repo.GetByShortURL(suite.ctx, shortURL)

	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), record)
	assert.Equal(suite.T(), expectedRecord.UUID, record.UUID)
	assert.Equal(suite.T(), expectedRecord.ShortURL, record.ShortURL)
	assert.Equal(suite.T(), expectedRecord.OriginalURL, record.OriginalURL)
	assert.Equal(suite.T(), expectedRecord.UserID, record.UserID)
	assert.Equal(suite.T(), expectedRecord.IsDeleted, record.IsDeleted)
	assert.NoError(suite.T(), suite.mockDB.ExpectationsWereMet())
}

func (suite *PostgresRepositoryTestSuite) TestGetByShortURL_NotFound() {
	shortURL := "notfound"

	suite.mockDB.ExpectQuery(`SELECT .* FROM links WHERE short = \$1`).
		WithArgs(shortURL).
		WillReturnError(sql.ErrNoRows)

	record, err := suite.repo.GetByShortURL(suite.ctx, shortURL)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), record)
	assert.NoError(suite.T(), suite.mockDB.ExpectationsWereMet())
}

func (suite *PostgresRepositoryTestSuite) TestGetByShortURL_Error() {
	shortURL := generateRandomString(32)
	expectedError := myErrors.ErrNotFound

	suite.mockDB.ExpectQuery(`SELECT .* FROM links WHERE short = \$1`).
		WithArgs(shortURL).
		WillReturnError(expectedError)

	record, err := suite.repo.GetByShortURL(suite.ctx, shortURL)

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), record)
	assert.Equal(suite.T(), expectedError, err)
	assert.NoError(suite.T(), suite.mockDB.ExpectationsWereMet())
}

func (suite *PostgresRepositoryTestSuite) TestGetAll() {
	expectedRecords := map[string]model.ShortenedRecord{
		"short1": {
			UUID:        "uuid1",
			ShortURL:    "short1",
			OriginalURL: "url1",
			UserID:      "user1",
			IsDeleted:   false,
		},
		"short2": {
			UUID:        "uuid2",
			ShortURL:    "short2",
			OriginalURL: "url2",
			UserID:      "user2",
			IsDeleted:   false,
		},
	}

	rows := sqlmock.NewRows([]string{"id", "short", "original", "user_id", "is_deleted"})
	for _, record := range expectedRecords {
		rows.AddRow(record.UUID, record.ShortURL, record.OriginalURL,
			record.UserID, record.IsDeleted)
	}

	suite.mockDB.ExpectQuery(`SELECT .* FROM links`).
		WillReturnRows(rows)

	result, err := suite.repo.GetAll()

	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), result, len(expectedRecords))
	for key, expected := range expectedRecords {
		actual, exists := result[key]
		assert.True(suite.T(), exists)
		assert.Equal(suite.T(), expected.UUID, actual.UUID)
		assert.Equal(suite.T(), expected.OriginalURL, actual.OriginalURL)
	}
	assert.NoError(suite.T(), suite.mockDB.ExpectationsWereMet())
}

func (suite *PostgresRepositoryTestSuite) TestGetAll_Error() {
	expectedError := fmt.Errorf("query error")
	suite.mockDB.ExpectQuery(`SELECT .* FROM links`).
		WillReturnError(expectedError)

	result, err := suite.repo.GetAll()

	assert.Error(suite.T(), err)
	assert.Nil(suite.T(), result)
	assert.Equal(suite.T(), expectedError, err)
	assert.NoError(suite.T(), suite.mockDB.ExpectationsWereMet())
}

func (suite *PostgresRepositoryTestSuite) TestGetUserURLs() {
	userID := "user123"
	expectedRecords := map[string]model.ShortenedRecord{
		"short1": {
			UUID:        "uuid1",
			ShortURL:    "short1",
			OriginalURL: "url1",
			UserID:      userID,
			IsDeleted:   false,
		},
	}

	rows := sqlmock.NewRows([]string{"id", "short", "original", "user_id", "is_deleted"}).
		AddRow("uuid1", "short1", "url1", userID, false)

	suite.mockDB.ExpectQuery(`SELECT .* FROM links WHERE user_id = \$1`).
		WithArgs(userID).
		WillReturnRows(rows)

	result, err := suite.repo.GetUserURLs(suite.ctx, userID)

	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), result, 1)
	assert.Equal(suite.T(), expectedRecords["short1"].UUID, result["short1"].UUID)
	assert.NoError(suite.T(), suite.mockDB.ExpectationsWereMet())
}

func (suite *PostgresRepositoryTestSuite) TestGetUserURLs_NoResults() {
	userID := "user123"

	rows := sqlmock.NewRows([]string{"id", "short", "original", "user_id", "is_deleted"})
	suite.mockDB.ExpectQuery(`SELECT .* FROM links WHERE user_id = \$1`).
		WithArgs(userID).
		WillReturnRows(rows)

	result, err := suite.repo.GetUserURLs(suite.ctx, userID)

	assert.NoError(suite.T(), err)
	assert.Empty(suite.T(), result)
	assert.NoError(suite.T(), suite.mockDB.ExpectationsWereMet())
}

func (suite *PostgresRepositoryTestSuite) TestPing_Success() {
	suite.mockDB.ExpectPing()

	err := suite.repo.Ping(suite.ctx)

	assert.NoError(suite.T(), err)
	assert.NoError(suite.T(), suite.mockDB.ExpectationsWereMet())
}

func (suite *PostgresRepositoryTestSuite) TestPing_Error() {
	expectedError := fmt.Errorf("ping error")
	suite.mockDB.ExpectPing().WillReturnError(expectedError)

	err := suite.repo.Ping(suite.ctx)

	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), expectedError, err)
	assert.NoError(suite.T(), suite.mockDB.ExpectationsWereMet())
}

func (suite *PostgresRepositoryTestSuite) TestClose() {
	suite.mockDB.ExpectClose()

	err := suite.repo.Close()

	assert.NoError(suite.T(), err)
	assert.NoError(suite.T(), suite.mockDB.ExpectationsWereMet())
}

func (suite *PostgresRepositoryTestSuite) TestGetData() {
	result := suite.repo.GetData()

	assert.NotNil(suite.T(), result)
	assert.Empty(suite.T(), result)
}

func (suite *PostgresRepositoryTestSuite) TestBatchDelete() {
	userID := "user123"
	shortIDs := []string{"short1", "short2", "short3"}

	suite.mockDB.ExpectBegin()

	prep := suite.mockDB.ExpectPrepare(`UPDATE links .*`)

	for _, shortID := range shortIDs {
		prep.ExpectExec().
			WithArgs(userID, shortID).
			WillReturnResult(sqlmock.NewResult(0, 1))
	}

	suite.mockDB.ExpectCommit()

	err := suite.repo.BatchDelete(suite.ctx, userID, shortIDs)

	assert.NoError(suite.T(), err)
	assert.NoError(suite.T(), suite.mockDB.ExpectationsWereMet())
}

func (suite *PostgresRepositoryTestSuite) TestBatchDelete_ErrorOnPrepare() {
	userID := "user123"
	shortIDs := []string{"short1"}

	expectedError := fmt.Errorf("prepare error")
	suite.mockDB.ExpectBegin()
	suite.mockDB.ExpectPrepare(`UPDATE links .*`).WillReturnError(expectedError)
	suite.mockDB.ExpectRollback()

	err := suite.repo.BatchDelete(suite.ctx, userID, shortIDs)

	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), expectedError, err)
	assert.NoError(suite.T(), suite.mockDB.ExpectationsWereMet())
}

func (suite *PostgresRepositoryTestSuite) TestBatchDelete_ErrorOnExec() {
	userID := "user123"
	shortIDs := []string{"short1"}

	expectedError := fmt.Errorf("exec error")
	suite.mockDB.ExpectBegin()

	prep := suite.mockDB.ExpectPrepare(`UPDATE links .*`)
	prep.ExpectExec().
		WithArgs(userID, shortIDs[0]).
		WillReturnError(expectedError)

	suite.mockDB.ExpectRollback()

	err := suite.repo.BatchDelete(suite.ctx, userID, shortIDs)

	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), expectedError, err)
	assert.NoError(suite.T(), suite.mockDB.ExpectationsWereMet())
}

func generateRandomString(length int) string {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}

func (suite *PostgresRepositoryTestSuite) TestGetStats_Success() {
	expectedLinks := 10
	expectedUsers := 5

	rows := sqlmock.NewRows([]string{"links", "users"}).AddRow(expectedLinks, expectedUsers)

	suite.mockDB.ExpectQuery(`SELECT count\(\*\) as links, count\(distinct user_id\) as users FROM links`).
		WillReturnRows(rows)

	links, users, err := suite.repo.GetStats(suite.ctx)

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), expectedLinks, links)
	assert.Equal(suite.T(), expectedUsers, users)
	assert.NoError(suite.T(), suite.mockDB.ExpectationsWereMet())
}

func (suite *PostgresRepositoryTestSuite) TestGetStats_DBError() {
	expectedError := fmt.Errorf("database connection lost")

	suite.mockDB.ExpectQuery(`SELECT count\(\*\) as links, count\(distinct user_id\) as users FROM links`).
		WillReturnError(expectedError)

	links, users, err := suite.repo.GetStats(suite.ctx)

	assert.Error(suite.T(), err)
	assert.Equal(suite.T(), expectedError, err)
	assert.Equal(suite.T(), 0, links)
	assert.Equal(suite.T(), 0, users)
	assert.NoError(suite.T(), suite.mockDB.ExpectationsWereMet())
}

func TestNewPostgresRepository_SQLOpenError(t *testing.T) {
	defer func() {
		sqlOpen = originalSQLOpen
		applyMigrationsFunc = originalApplyMigrations
	}()

	expectedErr := errors.New("failed to open db")
	sqlOpen = func(driverName, dataSourceName string) (*sql.DB, error) {
		return nil, expectedErr
	}

	repo, err := NewPostgresRepository("any-dsn")

	assert.Error(t, err)
	assert.Nil(t, repo)
	assert.ErrorIs(t, err, expectedErr)
}

func TestNewPostgresRepository_PingError(t *testing.T) {
	defer func() {
		sqlOpen = originalSQLOpen
		applyMigrationsFunc = originalApplyMigrations
	}()

	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)
	defer db.Close()

	expectedErr := errors.New("ping failed")
	mock.ExpectPing().WillReturnError(expectedErr)

	sqlOpen = func(driverName, dataSourceName string) (*sql.DB, error) {
		return db, nil
	}

	applyMigrationsFunc = func(db *sql.DB) error {
		t.Error("applyMigrations should not be called")
		return nil
	}

	repo, err := NewPostgresRepository("any-dsn")

	assert.Error(t, err)
	assert.Nil(t, repo)
	assert.ErrorIs(t, err, expectedErr)
	assert.NoError(t, mock.ExpectationsWereMet())
}
