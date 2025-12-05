package service

import (
	"context"
	"sync"
	"time"

	"github.com/vsevolod-ryzhov/urlshortner.git/internal/repository"
)

type DeleteManager struct {
	tasks       chan deleteTask
	workerCount int
	repo        repository.Repository
	once        sync.Once
}

type deleteTask struct {
	userID  string
	shortID string
}

var (
	deleteMgr *DeleteManager
	initMutex sync.Mutex
)

func InitDeleteManager(repo repository.Repository, workerCount int) {
	initMutex.Lock()
	defer initMutex.Unlock()

	if deleteMgr == nil {
		deleteMgr = &DeleteManager{
			tasks:       make(chan deleteTask, 1000),
			workerCount: workerCount,
			repo:        repo,
		}
		deleteMgr.start()
	}
}

func SubmitDeleteTask(userID, shortID string) {
	if deleteMgr != nil {
		select {
		case deleteMgr.tasks <- deleteTask{userID: userID, shortID: shortID}:
		default:
		}
	}
}

func (dm *DeleteManager) start() {
	batchChan := make(chan []deleteTask, 10)

	for i := 0; i < dm.workerCount; i++ {
		go dm.worker(batchChan)
	}

	go dm.batchProcessor(batchChan)
}

func (dm *DeleteManager) worker(batchChan chan<- []deleteTask) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	var batch []deleteTask

	for {
		select {
		case task := <-dm.tasks:
			batch = append(batch, task)

			if len(batch) >= 50 {
				batchChan <- batch
				batch = nil
			}

		case <-ticker.C:
			if len(batch) > 0 {
				batchChan <- batch
				batch = nil
			}
		}
	}
}

func (dm *DeleteManager) batchProcessor(batchChan <-chan []deleteTask) {
	for batch := range batchChan {
		dm.processBatch(batch)
	}
}

func (dm *DeleteManager) processBatch(batch []deleteTask) {
	if len(batch) == 0 {
		return
	}

	userToIDs := make(map[string][]string)
	for _, task := range batch {
		userToIDs[task.userID] = append(userToIDs[task.userID], task.shortID)
	}

	for userID, shortIDs := range userToIDs {
		dm.repo.BatchDelete(context.Background(), userID, shortIDs)
	}
}
