package vector

import (
	"log"
	"sync"
	"time"
)

// UserEmbeddingQueue manages background processing of user embedding updates
type UserEmbeddingQueue struct {
	queue    map[int]bool // userID -> needs update
	mu       sync.RWMutex
	stopChan chan struct{}
	ticker   *time.Ticker
}

var (
	embeddingQueue *UserEmbeddingQueue
	queueOnce      sync.Once
)

// GetEmbeddingQueue returns the singleton embedding queue
func GetEmbeddingQueue() *UserEmbeddingQueue {
	queueOnce.Do(func() {
		embeddingQueue = &UserEmbeddingQueue{
			queue:    make(map[int]bool),
			stopChan: make(chan struct{}),
		}
	})
	return embeddingQueue
}

// QueueUserForUpdate adds a user to the update queue
func (q *UserEmbeddingQueue) QueueUserForUpdate(userID int) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.queue[userID] = true
	log.Printf("[queue] Queued user %d for embedding update", userID)
}

// StartBackgroundProcessor starts the background processor that runs periodically
func (q *UserEmbeddingQueue) StartBackgroundProcessor() {
	q.ticker = time.NewTicker(5 * time.Minute) // Process every 5 minutes
	go func() {
		for {
			select {
			case <-q.ticker.C:
				q.processQueue()
			case <-q.stopChan:
				q.ticker.Stop()
				return
			}
		}
	}()
	log.Printf("[queue] Background user embedding processor started (5-minute intervals)")
}

// StopBackgroundProcessor stops the background processor
func (q *UserEmbeddingQueue) StopBackgroundProcessor() {
	if q.ticker != nil {
		q.stopChan <- struct{}{}
	}
}

// processQueue processes all queued users
func (q *UserEmbeddingQueue) processQueue() {
	q.mu.Lock()
	usersToProcess := make([]int, 0, len(q.queue))
	for userID := range q.queue {
		usersToProcess = append(usersToProcess, userID)
	}
	// Clear the queue
	q.queue = make(map[int]bool)
	q.mu.Unlock()

	if len(usersToProcess) == 0 {
		return
	}

	log.Printf("[queue] Processing %d users for embedding updates", len(usersToProcess))

	for _, userID := range usersToProcess {
		log.Printf("[queue] Updating embedding for user %d", userID)
		_, err := GetUserPersonalizedEmbedding(userID, true)
		if err != nil {
			log.Printf("[queue] Error updating embedding for user %d: %v", userID, err)
			// Re-queue the user for later retry
			q.QueueUserForUpdate(userID)
		} else {
			log.Printf("[queue] Successfully updated embedding for user %d", userID)
		}
	}
}

// GetQueueSize returns the current number of users in the queue
func (q *UserEmbeddingQueue) GetQueueSize() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.queue)
}

// GetQueuedUsers returns a list of user IDs currently in the queue
func (q *UserEmbeddingQueue) GetQueuedUsers() []int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	users := make([]int, 0, len(q.queue))
	for userID := range q.queue {
		users = append(users, userID)
	}
	return users
}
