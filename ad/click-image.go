package ad

import (
	"time"

	"github.com/parts-pile/site/db"
	"github.com/parts-pile/site/logger"
)

// IncrementAdImageClick increments the global click count for an ad image
func IncrementAdImageClick(adID int, imageIdx int) error {
	_, err := db.Exec(`INSERT INTO AdImageClick (ad_id, image_idx, click_count, last_clicked_at) VALUES ($1, $2, 1, $3)
		ON CONFLICT(ad_id, image_idx) DO UPDATE SET click_count = AdImageClick.click_count + 1, last_clicked_at = $4`,
		adID, imageIdx, time.Now().UTC(), time.Now().UTC())
	if err != nil {
		logger.Error("Failed to increment ad image click",
			"error", err, "adID", adID, "imageIdx", imageIdx)
		return err
	}
	return nil
}

// IncrementAdImageClickForUser increments the click count for an ad image for a specific user
func IncrementAdImageClickForUser(adID int, imageIdx int, userID int) error {
	now := time.Now().UTC()
	_, err := db.Exec(`INSERT INTO UserAdImageClick (ad_id, image_idx, user_id, click_count, last_clicked_at) VALUES ($1, $2, $3, 1, $4)
		ON CONFLICT(ad_id, image_idx, user_id) DO UPDATE SET click_count = UserAdImageClick.click_count + 1, last_clicked_at = $5`,
		adID, imageIdx, userID, now, now)
	if err != nil {
		logger.Error("Failed to increment user ad image click",
			"error", err, "adID", adID, "imageIdx", imageIdx, "userID", userID)
		return err
	}
	return nil
}

// ImageClickStats represents click statistics for an ad image
type ImageClickStats struct {
	AdID          int        `db:"ad_id"`
	ImageIdx      int        `db:"image_idx"`
	ClickCount    int        `db:"click_count"`
	LastClickedAt *time.Time `db:"last_clicked_at"`
}

// AdClickStats represents click statistics for an ad
type AdClickStats struct {
	AdID            int        `db:"ad_id"`
	Title           string     `db:"title"`
	ClickCount      int        `db:"click_count"`
	LastClickedAt   *time.Time `db:"last_clicked_at"`
	ImageClickCount int        `db:"image_click_count"`
}

// GetTopAdClickStats returns the top N ads by click count
func GetTopAdClickStats(limit int) ([]AdClickStats, error) {
	query := `
		SELECT 
			a.id AS ad_id,
			a.title,
			COALESCE(ac.click_count, 0) AS click_count,
			ac.last_clicked_at,
			COALESCE(SUM(aic.click_count), 0) AS image_click_count
		FROM Ad a
		LEFT JOIN AdClick ac ON a.id = ac.ad_id
		LEFT JOIN AdImageClick aic ON a.id = aic.ad_id
		WHERE a.deleted_at IS NULL
		GROUP BY a.id, a.title, ac.click_count, ac.last_clicked_at
		ORDER BY click_count DESC, image_click_count DESC
		LIMIT $1
	`
	var stats []AdClickStats
	err := db.Select(&stats, query, limit)
	return stats, err
}

// GetTopImageClickStats returns the top N images by click count
func GetTopImageClickStats(limit int) ([]ImageClickStats, error) {
	query := `
		SELECT 
			ad_id,
			image_idx,
			click_count,
			last_clicked_at
		FROM AdImageClick
		ORDER BY click_count DESC, last_clicked_at DESC
		LIMIT $1
	`
	var stats []ImageClickStats
	err := db.Select(&stats, query, limit)
	return stats, err
}

// GetAdImageClickStats returns click stats for all images of an ad
func GetAdImageClickStats(adID int) ([]ImageClickStats, error) {
	query := `
		SELECT 
			ad_id,
			image_idx,
			click_count,
			last_clicked_at
		FROM AdImageClick
		WHERE ad_id = $1
		ORDER BY image_idx ASC
	`
	var stats []ImageClickStats
	err := db.Select(&stats, query, adID)
	return stats, err
}
