package ad

import (
	"fmt"
	"log"
	"time"

	"github.com/parts-pile/site/db"
)

// IncrementAdClick increments the global click count for an ad
func IncrementAdClick(adID int) error {
	_, err := db.Exec("UPDATE Ad SET click_count = click_count + 1, last_clicked_at = ? WHERE id = ?",
		time.Now().UTC(), adID)
	if err != nil {
		fmt.Println("DEBUG IncrementAdClick error:", err)
		return err
	}
	return nil
}

// IncrementAdClickForUser increments the click count for an ad for a specific user
func IncrementAdClickForUser(adID int, userID int) error {
	now := time.Now().UTC()
	_, err := db.Exec(`INSERT INTO UserAdClick (ad_id, user_id, click_count, last_clicked_at) VALUES (?, ?, 1, ?)
		ON CONFLICT(ad_id, user_id) DO UPDATE SET click_count = click_count + 1, last_clicked_at = ?`,
		adID, userID, now, now)
	return err
}

// GetRecentlyClickedAdIDsByUser returns ad IDs the user has clicked, most recent first.
func GetRecentlyClickedAdIDsByUser(userID, limit int) ([]int, error) {
	log.Printf("[DEBUG] GetRecentlyClickedAdIDsByUser called with userID=%d, limit=%d", userID, limit)

	query := `SELECT ad_id FROM UserAdClick WHERE user_id = ? ORDER BY last_clicked_at DESC LIMIT ?`
	var adIDs []int
	err := db.Select(&adIDs, query, userID, limit)

	log.Printf("[DEBUG] GetRecentlyClickedAdIDsByUser returning %d adIDs: %v", len(adIDs), adIDs)
	return adIDs, err
}
