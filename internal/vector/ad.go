package vector

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/partspile/site/internal/ad"
	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/currency"
	"github.com/partspile/site/internal/logger"
)

var (
	ErrNoUserActivity = errors.New("no user activity to aggregate")
	ErrNoEmbeddedAds  = errors.New("no embedded ads")
	adQueue           = make(chan int, config.VectorProcessingQueueSize)
)

// IsEmbeddingUnavailable reports whether search should degrade gracefully
// instead of returning HTTP 500 (e.g. backfill still in progress).
func IsEmbeddingUnavailable(err error) bool {
	return errors.Is(err, ErrNoEmbeddedAds)
}

func BuildAdEmbedding(input ad.EmbeddingInput) error {
	return BuildAdEmbeddings([]ad.EmbeddingInput{input})
}

func BuildAdEmbeddings(inputs []ad.EmbeddingInput) error {
	if len(inputs) == 0 {
		return nil
	}
	prompts := make([]string, len(inputs))
	for i, in := range inputs {
		expansion := ""
		if !skipEmbeddingExpand {
			expansion = ad.ExpandForEmbedding(in)
		}
		prompts[i] = buildAdEmbeddingPrompt(in, expansion)
	}
	embeddings, err := embedDocuments(prompts)
	if err != nil {
		return err
	}
	adIDs := make([]int, len(inputs))
	metas := make([]map[string]any, len(inputs))
	for i, in := range inputs {
		adIDs[i] = in.ID
		metas[i] = buildAdEmbeddingMetadata(in)
	}
	return UpsertAdEmbeddings(adIDs, embeddings, metas)
}

func buildAdEmbeddingPrompt(in ad.EmbeddingInput, expansion string) string {
	var parts []string
	parts = append(parts, `Encode the following ad for semantic search.
Focus on title, description, tags, location, price, flag disputes, and field values.

Title: `+in.Title)
	parts = append(parts, "Description: "+in.Description)
	if len(in.TagLabels) > 0 {
		parts = append(parts, "Tags: "+strings.Join(in.TagLabels, ", "))
	}
	for _, d := range in.FacetDefs {
		if v, ok := in.Facets[d.Key]; ok && v.Present() {
			if s := d.EmbeddingSnippet(v); s != "" {
				parts = append(parts, s)
			}
		}
	}
	if in.City != "" || in.AdminArea != "" {
		parts = append(parts, fmt.Sprintf(
			"Location: %s, %s, %s",
			in.City, in.AdminArea, in.Country,
		))
	}
	if amount, code, ok := in.PriceValue(); ok {
		parts = append(parts, fmt.Sprintf(
			"Price: %s", currency.Format(amount, code),
		))
	}
	parts = append(parts, flagContext(in.FlagCount))
	if strings.TrimSpace(expansion) != "" {
		parts = append(parts, "Search background:\n"+
			strings.TrimSpace(expansion))
	}
	return strings.Join(parts, "\n")
}

func flagContext(count int) string {
	switch count {
	case 0:
		return "This ad has no reported disputes (0 flags raised)."
	case 1:
		return "This ad has 1 reported dispute (1 flag raised)."
	default:
		return fmt.Sprintf(
			"This ad has %d reported disputes (%d flags raised).",
			count, count,
		)
	}
}

func buildAdEmbeddingMetadata(in ad.EmbeddingInput) map[string]any {
	meta := map[string]any{
		"category_id": in.CategoryID,
	}
	if amount, _, ok := in.PriceValue(); ok {
		meta["price"] = amount
	}
	for _, d := range in.FacetDefs {
		v, ok := in.Facets[d.Key]
		if !ok || !v.Present() {
			continue
		}
		if val, ok := d.VectorMetadataValue(v); ok {
			meta[d.Key] = val
		}
	}
	return meta
}

func StartBackgroundProcessor() {
	go func() {
		const chunkSize = 50
		for {
			adID := <-adQueue
			adIDs := []int{adID}
			for i := 1; i < chunkSize; i++ {
				select {
				case id := <-adQueue:
					adIDs = append(adIDs, id)
				default:
					goto processChunk
				}
			}
		processChunk:
			var inputs []ad.EmbeddingInput
			for _, id := range adIDs {
				in, err := ad.GetForEmbedding(id)
				if err != nil {
					logger.Warn("embedding load ad", "adID", id, "error", err)
					continue
				}
				inputs = append(inputs, in)
			}
			if len(inputs) == 0 {
				continue
			}
			if err := BuildAdEmbeddings(inputs); err != nil {
				logger.Error("embedding batch failed", "error", err)
			}
		}
	}()
}

func QueueAd(adID int) {
	adQueue <- adID
}

func ProcessAdsWithoutVectors() {
	go func() {
		delay := 5 * time.Second
		for {
			remaining, err := BackfillAllAdsSync()
			if remaining == 0 {
				return
			}
			if err != nil {
				logger.Error("backfill ad embeddings",
					"error", err, "remaining", remaining)
			}
			time.Sleep(delay)
			if delay < time.Minute {
				delay *= 2
			}
		}
	}()
}

func BackfillAllAdsSync() (remaining int, err error) {
	if err := ad.LoadCategories(); err != nil {
		return 0, err
	}
	ids, err := ad.GetAdsWithoutVectors()
	if err != nil {
		return 0, err
	}
	if len(ids) == 0 {
		return 0, nil
	}
	logger.Info("backfilling ad embeddings", "count", len(ids))
	if err := encodeAdIDs(ids); err != nil {
		remainingIDs, countErr := ad.GetAdsWithoutVectors()
		if countErr != nil {
			return 0, countErr
		}
		return len(remainingIDs), fmt.Errorf("backfill incomplete: %w", err)
	}
	remainingIDs, countErr := ad.GetAdsWithoutVectors()
	if countErr != nil {
		return 0, countErr
	}
	return len(remainingIDs), nil
}

func RebuildAllAdsSync() (int, error) {
	if err := ad.LoadCategories(); err != nil {
		return 0, err
	}
	ids, err := ad.GetActiveAdIDs()
	if err != nil {
		return 0, err
	}
	if len(ids) == 0 {
		return 0, nil
	}
	logger.Info("rebuilding ad embeddings", "count", len(ids))
	if err := encodeAdIDs(ids); err != nil {
		return 0, err
	}
	return len(ids), nil
}

func encodeAdIDs(ids []int) error {
	const chunkSize = 10
	var firstErr error
	for i := 0; i < len(ids); i += chunkSize {
		end := i + chunkSize
		if end > len(ids) {
			end = len(ids)
		}
		var inputs []ad.EmbeddingInput
		for _, id := range ids[i:end] {
			in, loadErr := ad.GetForEmbedding(id)
			if loadErr != nil {
				return loadErr
			}
			inputs = append(inputs, in)
		}
		if chunkErr := BuildAdEmbeddings(inputs); chunkErr != nil {
			logger.Error("embedding chunk failed",
				"error", chunkErr, "offset", i, "size", len(inputs))
			if firstErr == nil {
				firstErr = chunkErr
			}
		}
	}
	return firstErr
}
