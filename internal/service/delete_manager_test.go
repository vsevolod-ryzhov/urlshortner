package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/vsevolod-ryzhov/urlshortner.git/internal/model"
)

type MockRepositoryForDelete struct {
	mock.Mock
}

func (m *MockRepositoryForDelete) BatchDelete(ctx context.Context, userID string, shortIDs []string) error {
	args := m.Called(ctx, userID, shortIDs)
	return args.Error(0)
}

func (m *MockRepositoryForDelete) Save(record *model.ShortenedRecord) error { return nil }
func (m *MockRepositoryForDelete) GetByShortURL(ctx context.Context, shortURL string) (*model.ShortenedRecord, error) {
	return nil, nil
}
func (m *MockRepositoryForDelete) GetAll() (map[string]model.ShortenedRecord, error) { return nil, nil }
func (m *MockRepositoryForDelete) Ping(ctx context.Context) error                    { return nil }
func (m *MockRepositoryForDelete) Close() error                                      { return nil }
func (m *MockRepositoryForDelete) GetData() map[string]model.ShortenedRecord         { return nil }
func (m *MockRepositoryForDelete) GetUserURLs(ctx context.Context, userID string) (map[string]model.ShortenedRecord, error) {
	return nil, nil
}
func (m *MockRepositoryForDelete) GetStats(ctx context.Context) (int, int, error) { return 0, 0, nil }

func setupDeleteManagerForTest(t *testing.T) func() {
	originalMgr := deleteMgr

	deleteMgr = nil

	return func() {
		deleteMgr = originalMgr
	}
}

func TestInitDeleteManager(t *testing.T) {
	cleanup := setupDeleteManagerForTest(t)
	defer cleanup()

	mockRepo := new(MockRepositoryForDelete)

	InitDeleteManager(mockRepo, 3)

	assert.NotNil(t, deleteMgr, "DeleteManager should be initialized")
	assert.Equal(t, 3, deleteMgr.workerCount)
	assert.Equal(t, mockRepo, deleteMgr.repo)
	assert.NotNil(t, deleteMgr.tasks)

	firstMgr := deleteMgr

	anotherRepo := new(MockRepositoryForDelete)
	InitDeleteManager(anotherRepo, 5)

	assert.Equal(t, firstMgr, deleteMgr, "InitDeleteManager should not recreate manager if already exists")
	assert.Equal(t, 3, deleteMgr.workerCount, "Worker count should not change")
	assert.Equal(t, mockRepo, deleteMgr.repo, "Repo should not change")
}

func TestSubmitDeleteTask_WithoutInitializedManager(t *testing.T) {
	cleanup := setupDeleteManagerForTest(t)
	defer cleanup()

	assert.Nil(t, deleteMgr)

	assert.NotPanics(t, func() {
		SubmitDeleteTask("user1", "short1")
	}, "SubmitDeleteTask should not panic when manager is nil")
}
