package ad

import (
	"fmt"
	"time"

	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/db"
)

// ConfirmImages sets image_count after client uploads succeed.
// newCount must be >= current count and <= MaxImagesPerAd.
func ConfirmImages(userID, adID, newCount int, tz *time.Location) error {
	if tz == nil {
		tz = time.UTC
	}
	a, err := GetAd(userID, adID, tz)
	if err != nil {
		return err
	}
	if a.UserID != userID {
		return fmt.Errorf("you are not the owner of this ad")
	}
	if !a.IsActive() {
		return fmt.Errorf("cannot edit a deleted or inactive ad")
	}
	if newCount < a.ImageCount {
		return fmt.Errorf("image count cannot decrease")
	}
	if newCount > config.MaxImagesPerAd {
		return fmt.Errorf(
			"too many images. Maximum %d images allowed per ad",
			config.MaxImagesPerAd,
		)
	}
	if newCount == a.ImageCount {
		return nil
	}

	added := newCount - a.ImageCount
	desc := a.Description
	if a.ImageCount > 0 {
		body := formatImageAdditionBody(a.ImageCount+1, added)
		desc = AppendHistoryEntry(
			desc, imagesAddedLabel, body, time.Now(), tz,
		)
	}

	_, err = db.Exec(
		`UPDATE ads SET description = $1, image_count = $2
		 WHERE id = $3 AND user_id = $4
		   AND inactive_at IS NULL AND deleted_at IS NULL`,
		desc, newCount, adID, userID,
	)
	if err != nil {
		return fmt.Errorf("confirm images: %w", err)
	}
	return nil
}
