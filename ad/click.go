package ad

import (
	"time"

	"github.com/parts-pile/site/db"
	"github.com/parts-pile/site/logger"
)

// IncrementAdClick increments the global click count for an ad
func IncrementAdClick(adID int) error {
	_, err := db.Exec(`INSERT INTO AdClick (ad_id, click_count, last_clicked_at) VALUES ($1, 1, $2)
		ON CONFLICT(ad_id) DO UPDATE SET click_count = AdClick.click_count + 1, last_clicked_at = $3`,
		adID, time.Now().UTC(), time.Now().UTC())
	if err != nil {
		logger.Error("Failed to increment ad click",
			"error", err, "adID", adID)
		return err
	}
	return nil
}

// IncrementAdClickForUser increments the click count for an ad for a specific user
func IncrementAdClickForUser(adID int, userID int) error {
	now := time.Now().UTC()
	_, err := db.Exec(`INSERT INTO UserAdClick (ad_id, user_id, click_count, last_clicked_at) VALUES ($1, $2, 1, $3)
		ON CONFLICT(ad_id, user_id) DO UPDATE SET click_count = UserAdClick.click_count + 1, last_clicked_at = $4`,
		adID, userID, now, now)
	if err != nil {
		logger.Error("Failed to increment user ad click",
			"error", err, "adID", adID, "userID", userID)
		return err
	}
	return nil
}

// GetRecentlyClickedAdIDsByUser returns ad IDs the user has clicked, most recent first.
func GetRecentlyClickedAdIDsByUser(userID, limit int) ([]int, error) {
	logger.Debug("GetRecentlyClickedAdIDsByUser called",
		"component", "ad", "userID", userID, "limit", limit)

	query := `SELECT ad_id FROM UserAdClick WHERE user_id = $1 ORDER BY last_clicked_at DESC LIMIT $2`
	var adIDs []int
	err := db.Select(&adIDs, query, userID, limit)

	logger.Debug("GetRecentlyClickedAdIDsByUser returning",
		"component", "ad", "userID", userID, "count", len(adIDs), "adIDs", adIDs)
	return adIDs, err
}
