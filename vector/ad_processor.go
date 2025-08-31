package vector

import (
	"log"
	"time"

	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/config"
)

// Static queue for ad processing
var adQueue = make(chan ad.Ad, config.QdrantProcessingQueueSize)

// StartBackgroundProcessor starts the background processor
func StartBackgroundProcessor() {
	go func() {
		log.Printf("[vector] Background vector processor started (queue-based)")

		for {
			adObj := <-adQueue
			log.Printf("[vector] Processing ad from queue: %d - %s", adObj.ID, adObj.Title)

			err := BuildAdEmbedding(adObj)
			if err != nil {
				log.Printf("[vector] Error building embedding for ad %d: %v", adObj.ID, err)
			} else {
				log.Printf("[vector] Successfully processed ad %d", adObj.ID)
			}

			// Sleep to avoid rate limits
			time.Sleep(config.QdrantProcessingSleepInterval)
		}
	}()
}

// QueueAd adds an ad to the processing queue
func QueueAd(adObj ad.Ad) {
	adQueue <- adObj
	log.Printf("[vector] Queued ad %d for processing", adObj.ID)
}

// QueueAdsWithoutVectors loads ads without vectors and processes them in batch
func QueueAdsWithoutVectors() {
	ads, err := ad.GetAdsWithoutVectors()
	if err != nil {
		log.Printf("[vector] Error getting ads without vectors: %v", err)
		return
	}

	if len(ads) == 0 {
		return
	}

	log.Printf("[vector] Processing %d ads without vectors in batch", len(ads))

	// Process all ads in a single batch
	err = BuildAdEmbeddings(ads)
	if err != nil {
		log.Printf("[vector] Error building embeddings for batch: %v", err)
	} else {
		log.Printf("[vector] Successfully processed batch of %d ads", len(ads))
	}
}
