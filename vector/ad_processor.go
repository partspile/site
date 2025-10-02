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

		const chunkSize = 50

		for {
			// Collect ads up to chunk size
			var ads []ad.Ad

			// Get the first ad (blocking)
			adObj := <-adQueue
			ads = append(ads, adObj)

			// Collect additional ads up to chunk size
			for i := 1; i < chunkSize; i++ {
				select {
				case adObj := <-adQueue:
					ads = append(ads, adObj)
				default:
					// No more ads available, break out of the inner loop
					goto processChunk
				}
			}

		processChunk:
			log.Printf("[vector] Processing chunk of %d ads from queue", len(ads))

			err := BuildAdEmbeddings(ads)
			if err != nil {
				log.Printf("[vector] Error building embeddings for chunk: %v", err)
			} else {
				log.Printf("[vector] Successfully processed chunk of %d ads", len(ads))
			}

			// Sleep to avoid rate limits
			time.Sleep(config.QdrantProcessingSleepInterval)
		}
	}()
}

// QueueAd adds an ad to the processing queue
func QueueAd(adObj ad.Ad) {
	adQueue <- adObj
}

// ProcessAdsWithoutVectors loads ads without vectors and queues them for processing
func ProcessAdsWithoutVectors() {
	go func() {
		ads, err := ad.GetAdsWithoutVectors()
		if err != nil {
			log.Printf("[vector] Error getting ads without vectors: %v", err)
			return
		}

		if len(ads) == 0 {
			log.Printf("[vector] No ads without vectors found")
			return
		}

		log.Printf("[vector] Queueing %d ads without vectors for processing", len(ads))

		// Queue all ads for background processing
		for _, adObj := range ads {
			QueueAd(adObj)
		}
	}()
}
