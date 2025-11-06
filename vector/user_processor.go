package vector

import (
	"sync"
	"time"

	"github.com/parts-pile/site/logger"
)

// Track which users are queued for processing
var queuedUsers = make(map[int]bool)
var queueMutex sync.RWMutex

// QueueUserForUpdate adds a user to the processing queue if not already queued
func QueueUserForUpdate(userID int) {
	queueMutex.Lock()
	defer queueMutex.Unlock()

	// Check if user is already queued
	if queuedUsers[userID] {
		logger.Debug("User already queued, skipping duplicate",
			"component", "queue", "userID", userID)
		return
	}

	// Mark user as queued
	queuedUsers[userID] = true
	logger.Debug("Queued user for embedding update",
		"component", "queue", "userID", userID)
}

// StartUserBackgroundProcessor starts the background processor that runs every 5 minutes
func StartUserBackgroundProcessor() {
	go func() {
		logger.Info("Background user embedding processor started",
			"component", "queue", "interval", "5 minutes")

		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			processAllQueuedUsers()
		}
	}()
}

// processAllQueuedUsers processes all users in the queue
func processAllQueuedUsers() {
	for {
		userID := getAndRemoveNextUser()
		if userID == 0 {
			return
		}

		logger.Debug("Processing user",
			"component", "queue", "userID", userID)

		_, err := GetUserPersonalizedEmbedding(userID, true)
		if err != nil {
			logger.Error("Error updating embedding for user",
				"error", err, "component", "queue", "userID", userID)
		} else {
			logger.Debug("Successfully updated embedding for user",
				"component", "queue", "userID", userID)
		}
	}
}

// getAndRemoveNextUser gets and removes one user from the queue with minimal lock time
func getAndRemoveNextUser() int {
	queueMutex.Lock()
	defer queueMutex.Unlock()

	// Get first user from map
	for userID := range queuedUsers {
		delete(queuedUsers, userID)
		return userID
	}
	return 0 // No users to process
}
