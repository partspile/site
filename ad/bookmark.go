package ad

import (
	"database/sql"

	"github.com/parts-pile/site/db"
)

// BookmarkAd bookmarks an ad for a user
func BookmarkAd(userID, adID int) error {
	_, err := db.Exec(`INSERT OR IGNORE INTO BookmarkedAd (user_id, ad_id) VALUES (?, ?)`, userID, adID)
	return err
}

// UnbookmarkAd removes a bookmark for an ad by a user
func UnbookmarkAd(userID, adID int) error {
	_, err := db.Exec(`DELETE FROM BookmarkedAd WHERE user_id = ? AND ad_id = ?`, userID, adID)
	return err
}

// IsAdBookmarkedByUser checks if a user has bookmarked an ad
func IsAdBookmarkedByUser(userID, adID int) (bool, error) {
	row := db.QueryRow(`SELECT 1 FROM BookmarkedAd WHERE user_id = ? AND ad_id = ?`, userID, adID)
	var exists int
	err := row.Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

// GetBookmarkedAdIDsByUser returns a list of ad IDs bookmarked by the user
func GetBookmarkedAdIDsByUser(userID int) ([]int, error) {
	rows, err := db.Query(`SELECT ad_id FROM BookmarkedAd WHERE user_id = ? ORDER BY bookmarked_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var adIDs []int
	for rows.Next() {
		var adID int
		if err := rows.Scan(&adID); err != nil {
			continue
		}
		adIDs = append(adIDs, adID)
	}
	return adIDs, nil
}

// GetAdsByIDs moved back to ad.go (core ad functionality)

// GetAdsByIDsOptimizedWithBookmarks moved back to ad.go (core ad functionality)
