package vector

import (
	"log"
	"sync"
	"time"

	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/config"
)

// VectorProcessor handles background processing of ad embeddings via queue
type VectorProcessor struct {
	queue    chan ad.Ad
	stopChan chan struct{}
	wg       sync.WaitGroup
	started  bool
	mu       sync.Mutex
}

var (
	vectorProcessor *VectorProcessor
	processorOnce   sync.Once
)

// GetVectorProcessor returns the singleton vector processor
func GetVectorProcessor() *VectorProcessor {
	processorOnce.Do(func() {
		vectorProcessor = &VectorProcessor{
			queue:    make(chan ad.Ad, config.QdrantProcessingQueueSize), // Buffer for 100 ads
			stopChan: make(chan struct{}),
		}
	})
	return vectorProcessor
}

// StartBackgroundProcessor starts the background processor
func (p *VectorProcessor) StartBackgroundProcessor() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.started {
		return
	}

	p.started = true
	p.wg.Add(1)

	go func() {
		defer p.wg.Done()
		log.Printf("[vector] Background vector processor started (queue-based)")

		for {
			select {
			case adObj := <-p.queue:
				log.Printf("[vector] Processing ad from queue: %d - %s", adObj.ID, adObj.Title)

				err := BuildAdEmbedding(adObj)
				if err != nil {
					log.Printf("[vector] Error building embedding for ad %d: %v", adObj.ID, err)
				} else {
					log.Printf("[vector] Successfully processed ad %d", adObj.ID)
				}

				// Sleep to avoid rate limits
				time.Sleep(config.QdrantProcessingSleepInterval)

			case <-p.stopChan:
				log.Printf("[vector] Background vector processor stopped")
				return
			}
		}
	}()
}

// StopBackgroundProcessor stops the background processor
func (p *VectorProcessor) StopBackgroundProcessor() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.started {
		return
	}

	close(p.stopChan)
	p.wg.Wait()
	p.started = false
	log.Printf("[vector] Background vector processor stopped")
}

// QueueAd adds an ad to the processing queue
func (p *VectorProcessor) QueueAd(adObj ad.Ad) {
	select {
	case p.queue <- adObj:
		log.Printf("[vector] Queued ad %d for processing", adObj.ID)
	default:
		log.Printf("[vector] Queue full, dropping ad %d", adObj.ID)
	}
}

// QueueAdsWithoutVectors loads ads without vectors and queues them for processing
func (p *VectorProcessor) QueueAdsWithoutVectors() {
	ads, err := ad.GetAdsWithoutVectors()
	if err != nil {
		log.Printf("[vector] Error getting ads without vectors: %v", err)
		return
	}

	if len(ads) == 0 {
		return
	}

	log.Printf("[vector] Queuing %d ads for vector generation", len(ads))

	for _, adObj := range ads {
		p.QueueAd(adObj)
	}
}

// GetQueueSize returns the current number of ads in the queue
func (p *VectorProcessor) GetQueueSize() int {
	return len(p.queue)
}

// IsRunning returns true if the processor is currently running
func (p *VectorProcessor) IsRunning() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.started
}
