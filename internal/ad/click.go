package ad

import (
	"time"

	"github.com/partspile/site/internal/db"
)

func IncrementAdClickForUser(adID, userID int) error {
	now := time.Now().UTC()
	_, err := db.Exec(`
		INSERT INTO user_ad_clicks (ad_id, user_id, click_count, last_clicked_at)
		VALUES ($1, $2, 1, $3)
		ON CONFLICT (ad_id, user_id) DO UPDATE SET
			click_count = user_ad_clicks.click_count + 1,
			last_clicked_at = $3`,
		adID, userID, now,
	)
	return err
}

func IncrementAdImageClickForUser(adID, userID, imageIndex int) error {
	now := time.Now().UTC()
	_, err := db.Exec(`
		INSERT INTO user_ad_image_clicks
			(ad_id, user_id, image_index, click_count, last_clicked_at)
		VALUES ($1, $2, $3, 1, $4)
		ON CONFLICT (ad_id, user_id, image_index) DO UPDATE SET
			click_count = user_ad_image_clicks.click_count + 1,
			last_clicked_at = $4`,
		adID, userID, imageIndex, now,
	)
	return err
}
