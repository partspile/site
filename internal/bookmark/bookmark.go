package bookmark

import "github.com/partspile/site/internal/db"

func Add(userID, adID int) error {
	_, err := db.Exec(
		`INSERT INTO bookmarks (user_id, ad_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id, ad_id)
		DO UPDATE SET bookmarked_at = CURRENT_TIMESTAMP`,
		userID, adID,
	)
	return err
}

func Remove(userID, adID int) error {
	_, err := db.Exec(
		"DELETE FROM bookmarks WHERE user_id = $1 AND ad_id = $2",
		userID, adID,
	)
	return err
}
