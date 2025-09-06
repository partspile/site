package vector

import (
	"log"
	"sync"
	"time"
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
		log.Printf("[queue] User %d already queued, skipping duplicate", userID)
		return
	}

	// Mark user as queued
	queuedUsers[userID] = true
	log.Printf("[queue] Queued user %d for embedding update", userID)
}

// StartUserBackgroundProcessor starts the background processor that runs every 5 minutes
func StartUserBackgroundProcessor() {
	go func() {
		log.Printf("[queue] Background user embedding processor started (runs every 5 minutes)")

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
			log.Printf("[queue] No more users to process")
			return
		}

		log.Printf("[queue] Processing user %d", userID)

		_, err := GetUserPersonalizedEmbedding(userID, true)
		if err != nil {
			log.Printf("[queue] Error updating embedding for user %d: %v", userID, err)
		} else {
			log.Printf("[queue] Successfully updated embedding for user %d", userID)
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
