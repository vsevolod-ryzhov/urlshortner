package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/model"
)

func TestCreateShortURLAndGet(t *testing.T) {
	type args struct {
		url string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "First URL",
			args: args{url: "https://ya.com"},
			want: "",
		},
		{
			name: "Second URL",
			args: args{url: "https://yandex.ru"},
			want: "",
		},
		{
			name: "URL with special characters",
			args: args{url: "https://someurl.com/somepath?query=param&other=value"},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shortened, _, _ := CreateShortURL(t.Context(), tt.args.url, "1")

			if shortened == "" {
				t.Errorf("generateShortID(%q) returns empty string", tt.args.url)
			}

			restored, err := GetURL(shortened)

			if err != nil {
				t.Errorf("GetURL(%q) returns error: %v", shortened, err)
			}

			if tt.args.url != restored {
				t.Errorf("GetURL(%q) returns %q, want %q", shortened, restored, tt.args.url)
			}
		})
	}
}

func Test_generateShortID(t *testing.T) {
	type args struct {
		originalURL string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "First URL",
			args: args{originalURL: "https://ya.com"},
			want: "",
		},
		{
			name: "Second URL",
			args: args{originalURL: "https://yandex.ru"},
			want: "",
		},
		{
			name: "empty string",
			args: args{originalURL: ""},
			want: "",
		},
		{
			name: "URL with special characters",
			args: args{originalURL: "https://someurl.com/somepath?query=param&other=value"},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateShortID(tt.args.originalURL)

			if result == "" {
				t.Errorf("generateShortID(%q) returns empty string", tt.args.originalURL)
			}

			if strings.HasSuffix(result, "=") {
				t.Errorf("generateShortID(%q) contains = suffix", tt.args.originalURL)
			}

			if !base64Save(result) {
				t.Errorf("generateShortID(%q) contains unsefe for base64 symbols", tt.args.originalURL)
			}
		})
	}
}

func base64Save(str string) bool {
	allowedChars := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"

	for _, char := range str {
		if !strings.ContainsRune(allowedChars, char) {
			return false
		}
	}
	return true
}

func ExampleCreateShortURL() {
	shortened1, _, _ := CreateShortURL(context.Background(), "https://ya.ru", "1")
	fmt.Println(shortened1)

	shortened2, _, _ := CreateShortURL(context.Background(), "https://ya.com", "1")
	fmt.Println(shortened2)

	shortened3, _, _ := CreateShortURL(context.Background(), "https://ya.net", "1")
	fmt.Println(shortened3)

	// Output:
	// fpCk-cMLTn4
	// ZOiLcK9R_6I
	// gUbMZ5KfCnQ
}

func ExampleGetURL() {
	shortened1, _, _ := CreateShortURL(context.Background(), "https://ya.ru", "1")
	originalURL1, _ := GetURL(shortened1)
	fmt.Println(originalURL1)

	shortened2, _, _ := CreateShortURL(context.Background(), "https://ya.com", "1")
	originalURL2, _ := GetURL(shortened2)
	fmt.Println(originalURL2)

	shortened3, _, _ := CreateShortURL(context.Background(), "https://ya.net", "1")
	originalURL3, _ := GetURL(shortened3)
	fmt.Println(originalURL3)

	// Output:
	// https://ya.ru
	// https://ya.com
	// https://ya.net
}

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Save(record *model.ShortenedRecord) error {
	args := m.Called(record)
	return args.Error(0)
}

func (m *MockRepository) GetByShortURL(ctx context.Context, shortURL string) (*model.ShortenedRecord, error) {
	args := m.Called(ctx, shortURL)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ShortenedRecord), args.Error(1)
}

func (m *MockRepository) GetAll() (map[string]model.ShortenedRecord, error) {
	args := m.Called()
	return args.Get(0).(map[string]model.ShortenedRecord), args.Error(1)
}

func (m *MockRepository) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockRepository) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockRepository) GetData() map[string]model.ShortenedRecord {
	args := m.Called()
	return args.Get(0).(map[string]model.ShortenedRecord)
}

func (m *MockRepository) GetUserURLs(ctx context.Context, userID string) (map[string]model.ShortenedRecord, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(map[string]model.ShortenedRecord), args.Error(1)
}

func (m *MockRepository) BatchDelete(ctx context.Context, userID string, shortIDs []string) error {
	args := m.Called(ctx, userID, shortIDs)
	return args.Error(0)
}

func (m *MockRepository) GetStats(ctx context.Context) (int, int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Int(1), args.Error(2)
}

func setupTest(t *testing.T, mockRepo *MockRepository) func() {
	originalRepo := Repo
	Repo = mockRepo
	return func() {
		Repo = originalRepo
	}
}

func TestGetUserURLs(t *testing.T) {
	ctx := context.Background()
	userID := "test-user-123"

	tests := []struct {
		name          string
		setupMock     func(*MockRepository)
		expectedError bool
		expectedCount int
	}{
		{
			name: "success - returns user URLs",
			setupMock: func(m *MockRepository) {
				expectedData := map[string]model.ShortenedRecord{
					"short1": {
						UUID:        "uuid1",
						ShortURL:    "short1",
						OriginalURL: "https://example1.com",
						UserID:      userID,
						IsDeleted:   false,
					},
					"short2": {
						UUID:        "uuid2",
						ShortURL:    "short2",
						OriginalURL: "https://example2.com",
						UserID:      userID,
						IsDeleted:   false,
					},
				}
				m.On("GetUserURLs", ctx, userID).Return(expectedData, nil)
			},
			expectedError: false,
			expectedCount: 2,
		},
		{
			name: "empty result - returns empty map",
			setupMock: func(m *MockRepository) {
				m.On("GetUserURLs", ctx, userID).Return(map[string]model.ShortenedRecord{}, nil)
			},
			expectedError: false,
			expectedCount: 0,
		},
		{
			name: "repository error - returns error",
			setupMock: func(m *MockRepository) {
				m.On("GetUserURLs", ctx, userID).Return(map[string]model.ShortenedRecord{}, errors.New("database error"))
			},
			expectedError: true,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			tt.setupMock(mockRepo)

			cleanup := setupTest(t, mockRepo)
			defer cleanup()

			result, err := GetUserURLs(ctx, userID)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Len(t, result, tt.expectedCount)
				if tt.expectedCount > 0 {
					for _, record := range result {
						assert.Equal(t, userID, record.UserID)
					}
				}
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestGetUserURLs_NilRepo(t *testing.T) {
	originalRepo := Repo
	Repo = nil
	defer func() { Repo = originalRepo }()

	ctx := context.Background()
	result, err := GetUserURLs(ctx, "any-user")

	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestGetStats(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		setupMock     func(*MockRepository)
		expectedLinks int
		expectedUsers int
		expectedError bool
	}{
		{
			name: "success - returns stats",
			setupMock: func(m *MockRepository) {
				m.On("GetStats", ctx).Return(10, 5, nil)
			},
			expectedLinks: 10,
			expectedUsers: 5,
			expectedError: false,
		},
		{
			name: "repository error - returns error",
			setupMock: func(m *MockRepository) {
				m.On("GetStats", ctx).Return(0, 0, errors.New("database error"))
			},
			expectedLinks: 0,
			expectedUsers: 0,
			expectedError: true,
		},
		{
			name: "zero stats - returns zeros",
			setupMock: func(m *MockRepository) {
				m.On("GetStats", ctx).Return(0, 0, nil)
			},
			expectedLinks: 0,
			expectedUsers: 0,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			tt.setupMock(mockRepo)

			cleanup := setupTest(t, mockRepo)
			defer cleanup()

			links, users, err := GetStats(ctx)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Equal(t, 0, links)
				assert.Equal(t, 0, users)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedLinks, links)
				assert.Equal(t, tt.expectedUsers, users)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestGetStats_NilRepo(t *testing.T) {
	originalRepo := Repo
	Repo = nil
	defer func() { Repo = originalRepo }()

	ctx := context.Background()

	assert.Panics(t, func() {
		GetStats(ctx)
	}, "GetStats should panic when Repo is nil")
}

func TestCreateShortURL_WithRepo(t *testing.T) {
	ctx := context.Background()
	userID := "test-user"
	url := "https://example.com"

	mockRepo := new(MockRepository)

	mockRepo.On("GetByShortURL", ctx, mock.AnythingOfType("string")).
		Return((*model.ShortenedRecord)(nil), ErrNotFound)

	mockRepo.On("Save", mock.AnythingOfType("*model.ShortenedRecord")).
		Return(nil)

	cleanup := setupTest(t, mockRepo)
	defer cleanup()

	shortID, exists, err := CreateShortURL(ctx, url, userID)

	assert.NoError(t, err)
	assert.False(t, exists)
	assert.NotEmpty(t, shortID)

	mockRepo.AssertExpectations(t)
}

func TestCreateShortURL_WithRepo_AlreadyExists(t *testing.T) {
	ctx := context.Background()
	userID := "test-user"
	url := "https://example.com"
	expectedShortID := "abc123"

	mockRepo := new(MockRepository)

	existingRecord := &model.ShortenedRecord{
		ShortURL:    expectedShortID,
		OriginalURL: url,
		UserID:      userID,
	}
	mockRepo.On("GetByShortURL", ctx, mock.AnythingOfType("string")).
		Return(existingRecord, nil)

	cleanup := setupTest(t, mockRepo)
	defer cleanup()

	shortID, exists, err := CreateShortURL(ctx, url, userID)

	assert.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, expectedShortID, shortID)

	mockRepo.AssertExpectations(t)
}
