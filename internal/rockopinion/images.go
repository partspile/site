package rockopinion

import (
	"encoding/base64"
	"regexp"

	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/imagestore"
	"github.com/partspile/site/internal/logger"
	"github.com/partspile/site/internal/message"
	"github.com/partspile/site/internal/rock"
	"github.com/partspile/site/internal/service/grok"
)

var opinionImageStore imagestore.Store

var imageMentionPattern = regexp.MustCompile(
	`(?i)\b(image|images|photo|photos|picture|pictures|pic|pics|` +
		`screenshot|screenshots|img|jpeg|jpg|png|webp|gallery)\b`,
)

// SetImageStore sets the store used to attach ad images to opinions.
func SetImageStore(store imagestore.Store) {
	opinionImageStore = store
}

// shouldAttachImages decides when listing images go to the model.
// Attach when the conversation mentions images, or when the reason is
// policy and there are no messages yet. Otherwise text-only.
func shouldAttachImages(reason string, messages []message.Message,
	imageCount int) bool {
	if imageCount <= 0 || opinionImageStore == nil {
		return false
	}
	if messagesMentionImages(messages) {
		return true
	}
	if reason == rock.ReasonPolicy && len(messages) == 0 {
		return true
	}
	return false
}

func messagesMentionImages(messages []message.Message) bool {
	for _, m := range messages {
		if imageMentionPattern.MatchString(m.Content) {
			return true
		}
	}
	return false
}

func imageDataURIParts(adID, imageCount int) []grok.ContentPart {
	if opinionImageStore == nil || imageCount <= 0 {
		return nil
	}
	n := imageCount
	if n > config.MaxImagesPerAd {
		n = config.MaxImagesPerAd
	}
	var parts []grok.ContentPart
	for i := 1; i <= n; i++ {
		data, err := opinionImageStore.Get(adID, i, "480w")
		if err != nil || len(data) == 0 {
			logger.Debug("rock opinion: skip image",
				"adID", adID, "index", i, "error", err)
			continue
		}
		uri := "data:image/jpeg;base64," +
			base64.StdEncoding.EncodeToString(data)
		parts = append(parts, grok.ContentPart{
			Type: "image_url",
			ImageURL: &grok.ImageURL{
				URL:    uri,
				Detail: "low",
			},
		})
	}
	if len(parts) > 0 {
		logger.Debug("rock opinion: attached images",
			"adID", adID, "count", len(parts))
	}
	return parts
}

func buildGrokParts(userPrompt string, adID, imageCount int,
	attach bool) []grok.ContentPart {
	parts := []grok.ContentPart{
		{Type: "text", Text: userPrompt},
	}
	if attach {
		parts = append(parts, imageDataURIParts(adID, imageCount)...)
	}
	return parts
}
