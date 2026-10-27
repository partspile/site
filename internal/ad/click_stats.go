package ad

import (
	"time"

	"github.com/partspile/site/internal/db"
)

type ClickSummary struct {
	UsersWithClicks int
	AdsClicked      int
	AdDetailViews   int
	ImageNavClicks  int
	ActiveLast7Days int
}

type TopAdClick struct {
	AdID         int        `db:"ad_id"`
	Title        string     `db:"title"`
	CategoryName string     `db:"category_name"`
	UserCount    int        `db:"user_count"`
	AdViews      int        `db:"ad_views"`
	ImageClicks  int        `db:"image_clicks"`
	LastActivity *time.Time `db:"last_activity"`
}

type TopImageClick struct {
	AdID          int        `db:"ad_id"`
	Title         string     `db:"title"`
	ImageIndex    int        `db:"image_index"`
	UserCount     int        `db:"user_count"`
	Clicks        int        `db:"clicks"`
	LastClickedAt *time.Time `db:"last_clicked_at"`
}

type RecentClick struct {
	LastClickedAt time.Time `db:"last_clicked_at"`
	UserID        int       `db:"user_id"`
	AdID          int       `db:"ad_id"`
	Title         string    `db:"title"`
	ClickType     string    `db:"click_type"`
	ImageIndex    *int      `db:"image_index"`
	ClickCount    int       `db:"click_count"`
}

type TopUserClick struct {
	UserID      int        `db:"user_id"`
	AdClicks    int        `db:"ad_clicks"`
	ImageClicks int        `db:"image_clicks"`
	LastActive  *time.Time `db:"last_active"`
}

func GetClickSummary() (ClickSummary, error) {
	var s ClickSummary
	err := db.QueryRow(`
		SELECT
			(SELECT COUNT(DISTINCT user_id) FROM (
				SELECT user_id FROM user_ad_clicks
				UNION
				SELECT user_id FROM user_ad_image_clicks
			) u),
			(SELECT COUNT(DISTINCT ad_id) FROM (
				SELECT ad_id FROM user_ad_clicks
				UNION
				SELECT ad_id FROM user_ad_image_clicks
			) a),
			COALESCE((SELECT SUM(click_count) FROM user_ad_clicks), 0),
			COALESCE((SELECT SUM(click_count) FROM user_ad_image_clicks), 0),
			(SELECT COUNT(*) FROM (
				SELECT last_clicked_at FROM user_ad_clicks
				WHERE last_clicked_at > NOW() - INTERVAL '7 days'
				UNION ALL
				SELECT last_clicked_at FROM user_ad_image_clicks
				WHERE last_clicked_at > NOW() - INTERVAL '7 days'
			) recent)`,
	).Scan(
		&s.UsersWithClicks,
		&s.AdsClicked,
		&s.AdDetailViews,
		&s.ImageNavClicks,
		&s.ActiveLast7Days,
	)
	return s, err
}

func GetTopAdsByClicks(limit int) ([]TopAdClick, error) {
	if limit <= 0 {
		limit = 20
	}
	var rows []TopAdClick
	err := db.Select(&rows, `
		SELECT
			a.id AS ad_id,
			a.title,
			c.name AS category_name,
			COALESCE(uc.user_count, 0) AS user_count,
			COALESCE(ac.ad_views, 0) AS ad_views,
			COALESCE(ic.image_clicks, 0) AS image_clicks,
			CASE
				WHEN ac.last_at IS NULL THEN ic.last_at
				WHEN ic.last_at IS NULL THEN ac.last_at
				ELSE GREATEST(ac.last_at, ic.last_at)
			END AS last_activity
		FROM ads a
		JOIN categories c ON c.id = a.category_id
		LEFT JOIN (
			SELECT ad_id,
				COUNT(DISTINCT user_id) AS user_count
			FROM (
				SELECT ad_id, user_id FROM user_ad_clicks
				UNION
				SELECT ad_id, user_id FROM user_ad_image_clicks
			) x
			GROUP BY ad_id
		) uc ON uc.ad_id = a.id
		LEFT JOIN (
			SELECT ad_id,
				SUM(click_count) AS ad_views,
				MAX(last_clicked_at) AS last_at
			FROM user_ad_clicks
			GROUP BY ad_id
		) ac ON ac.ad_id = a.id
		LEFT JOIN (
			SELECT ad_id,
				SUM(click_count) AS image_clicks,
				MAX(last_clicked_at) AS last_at
			FROM user_ad_image_clicks
			GROUP BY ad_id
		) ic ON ic.ad_id = a.id
		WHERE a.inactive_at IS NULL AND a.deleted_at IS NULL
		  AND (ac.ad_id IS NOT NULL OR ic.ad_id IS NOT NULL)
		ORDER BY
			(COALESCE(ac.ad_views, 0) + COALESCE(ic.image_clicks, 0)) DESC,
			last_activity DESC NULLS LAST
		LIMIT $1`, limit)
	return rows, err
}

func GetTopImagesByClicks(limit int) ([]TopImageClick, error) {
	if limit <= 0 {
		limit = 20
	}
	var rows []TopImageClick
	err := db.Select(&rows, `
		SELECT
			uaic.ad_id,
			a.title,
			uaic.image_index,
			COUNT(DISTINCT uaic.user_id) AS user_count,
			SUM(uaic.click_count) AS clicks,
			MAX(uaic.last_clicked_at) AS last_clicked_at
		FROM user_ad_image_clicks uaic
		JOIN ads a ON a.id = uaic.ad_id AND a.inactive_at IS NULL AND a.deleted_at IS NULL
		GROUP BY uaic.ad_id, a.title, uaic.image_index
		ORDER BY clicks DESC, last_clicked_at DESC NULLS LAST
		LIMIT $1`, limit)
	return rows, err
}

func GetRecentClickActivity(limit int) ([]RecentClick, error) {
	if limit <= 0 {
		limit = 25
	}
	var rows []RecentClick
	err := db.Select(&rows, `
		SELECT * FROM (
			SELECT
				uac.last_clicked_at,
				uac.user_id,
				uac.ad_id,
				a.title,
				'ad' AS click_type,
				NULL::int AS image_index,
				uac.click_count
			FROM user_ad_clicks uac
			JOIN ads a ON a.id = uac.ad_id
			WHERE a.inactive_at IS NULL AND a.deleted_at IS NULL

			UNION ALL

			SELECT
				uaic.last_clicked_at,
				uaic.user_id,
				uaic.ad_id,
				a.title,
				'image' AS click_type,
				uaic.image_index,
				uaic.click_count
			FROM user_ad_image_clicks uaic
			JOIN ads a ON a.id = uaic.ad_id
			WHERE a.inactive_at IS NULL AND a.deleted_at IS NULL
		) activity
		ORDER BY last_clicked_at DESC
		LIMIT $1`, limit)
	return rows, err
}

func GetTopUsersByClicks(limit int) ([]TopUserClick, error) {
	if limit <= 0 {
		limit = 15
	}
	var rows []TopUserClick
	err := db.Select(&rows, `
		SELECT
			u.user_id,
			COALESCE(a.ad_clicks, 0) AS ad_clicks,
			COALESCE(i.image_clicks, 0) AS image_clicks,
			CASE
				WHEN a.last_at IS NULL THEN i.last_at
				WHEN i.last_at IS NULL THEN a.last_at
				ELSE GREATEST(a.last_at, i.last_at)
			END AS last_active
		FROM (
			SELECT user_id FROM user_ad_clicks
			UNION
			SELECT user_id FROM user_ad_image_clicks
		) u
		LEFT JOIN (
			SELECT user_id,
				SUM(click_count) AS ad_clicks,
				MAX(last_clicked_at) AS last_at
			FROM user_ad_clicks
			GROUP BY user_id
		) a ON a.user_id = u.user_id
		LEFT JOIN (
			SELECT user_id,
				SUM(click_count) AS image_clicks,
				MAX(last_clicked_at) AS last_at
			FROM user_ad_image_clicks
			GROUP BY user_id
		) i ON i.user_id = u.user_id
		ORDER BY
			(COALESCE(a.ad_clicks, 0) + COALESCE(i.image_clicks, 0)) DESC,
			last_active DESC NULLS LAST
		LIMIT $1`, limit)
	return rows, err
}
