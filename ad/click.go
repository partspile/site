package ad

import (
	"database/sql"
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

// GetAdClickCount returns the global click count for an ad
func GetAdClickCount(adID int) (int, error) {
	var count int
	err := db.QueryRow("SELECT click_count FROM Ad WHERE id = ?", adID).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// GetAdClickCountForUser returns the click count for an ad for a specific user
func GetAdClickCountForUser(adID int, userID int) (int, error) {
	var count int
	err := db.QueryRow("SELECT click_count FROM UserAdClick WHERE ad_id = ? AND user_id = ?", adID, userID).Scan(&count)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return count, nil
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
