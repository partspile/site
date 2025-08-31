package vector

import (
	"log"
)

// Static queue for user processing
var userQueue = make(chan int, 1000) // Buffer size of 1000

// QueueUserForUpdate adds a user to the update queue
func QueueUserForUpdate(userID int) {
	userQueue <- userID
	log.Printf("[queue] Queued user %d for embedding update", userID)
}

// StartBackgroundProcessor starts the background processor
func StartUserBackgroundProcessor() {
	go func() {
		log.Printf("[queue] Background user embedding processor started (on-demand)")

		for {
			// Wait for a user to be queued
			userID := <-userQueue
			log.Printf("[queue] Processing user %d from queue", userID)

			// Process the user immediately
			_, err := GetUserPersonalizedEmbedding(userID, true)
			if err != nil {
				log.Printf("[queue] Error updating embedding for user %d: %v", userID, err)
				// Re-queue the user for later retry
				QueueUserForUpdate(userID)
			} else {
				log.Printf("[queue] Successfully updated embedding for user %d", userID)
			}
		}
	}()
}
