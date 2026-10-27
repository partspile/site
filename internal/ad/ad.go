package ad

import (
	"fmt"
	"strings"
	"time"

	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/db"
	"github.com/partspile/site/internal/facet"
)

type Ad struct {
	// Core database fields
	ID          int        `db:"id"`
	CategoryID  int        `db:"category_id"`
	Title       string     `db:"title"`
	Description string     `db:"description"`
	CreatedAt   time.Time  `db:"created_at"`
	ExpiresAt   time.Time  `db:"expires_at"`
	InactiveAt  *time.Time `db:"inactive_at"`
	DeletedAt   *time.Time `db:"deleted_at"`
	UserID      int        `db:"user_id"`
	ImageCount  int        `db:"image_count"`
	LocationID  *int       `db:"location_id"`

	// Location fields from join
	City        string `db:"city"`
	AdminArea   string `db:"admin_area"`
	Country     string `db:"country"`
	RawLocation string `db:"raw_location"`

	// Computed fields
	Bookmarked bool `db:"bookmarked"`
	RockCount  int  `db:"rock_count"`

	// Category-specific facet values
	Facets map[string]facet.Value

	// Tags selected for the ad; loaded lazily on ad detail (ads.tags JSON)
	Tags []Suggestion
}

func (a Ad) IsActive() bool {
	return a.InactiveAt == nil && a.DeletedAt == nil
}

func (a Ad) IsInactive() bool {
	return a.InactiveAt != nil && a.DeletedAt == nil
}

func (a Ad) IsDeleted() bool {
	return a.DeletedAt != nil
}

// PriceValue returns the ad's price amount and currency, if a price facet is set.
func (a Ad) PriceValue() (amount int, currency string, ok bool) {
	v, exists := a.Facets["price"]
	if !exists || v.Num == nil {
		return 0, "", false
	}
	if v.Text != nil {
		currency = *v.Text
	}
	return *v.Num, currency, true
}

func placeholdersFrom(start, n int) string {
	if n == 0 {
		return ""
	}
	ph := make([]string, n)
	for i := range ph {
		ph[i] = fmt.Sprintf("$%d", start+i)
	}
	return strings.Join(ph, ",")
}

func GetAds(userID int, ids []int, tz *time.Location) ([]Ad, error) {
	if len(ids) == 0 {
		return []Ad{}, nil
	}
	query := `
		SELECT
			a.id,
			a.category_id,
			a.title,
			a.description,
			a.created_at,
			a.expires_at,
			a.inactive_at,
			a.deleted_at,
			a.user_id,
			a.image_count,
			a.location_id,
			COALESCE(l.city, '') AS city,
			COALESCE(l.admin_area, '') AS admin_area,
			COALESCE(l.country, '') AS country,
			COALESCE(l.raw_text, '') AS raw_location,
			CASE WHEN b.user_id IS NOT NULL THEN 1 ELSE 0 END AS bookmarked,
			a.rock_count
		FROM ads a
		LEFT JOIN locations l ON a.location_id = l.id
		LEFT JOIN bookmarks b ON a.id = b.ad_id AND b.user_id = $1
		WHERE a.id IN (` + placeholdersFrom(2, len(ids)) + `)
	`
	args := make([]any, len(ids)+1)
	args[0] = userID
	for i, id := range ids {
		args[i+1] = id
	}
	var ads []Ad
	err := db.Select(&ads, query, args...)
	if err != nil {
		return nil, err
	}

	// Convert timestamps to local timezone
	for i := range ads {
		ads[i].CreatedAt = ads[i].CreatedAt.In(tz)
		ads[i].ExpiresAt = ads[i].ExpiresAt.In(tz)
		if ads[i].InactiveAt != nil {
			converted := (*ads[i].InactiveAt).In(tz)
			ads[i].InactiveAt = &converted
		}
		if ads[i].DeletedAt != nil {
			converted := (*ads[i].DeletedAt).In(tz)
			ads[i].DeletedAt = &converted
		}
	}

	if err := attachFacets(ads); err != nil {
		return nil, err
	}

	byID := make(map[int]Ad, len(ads))
	for _, a := range ads {
		byID[a.ID] = a
	}
	ordered := make([]Ad, 0, len(ids))
	for _, id := range ids {
		if a, ok := byID[id]; ok {
			ordered = append(ordered, a)
		}
	}

	return ordered, nil
}

type facetRow struct {
	AdID int     `db:"ad_id"`
	Key  string  `db:"key"`
	Num  *int    `db:"num"`
	Text *string `db:"text"`
}

func attachFacets(ads []Ad) error {
	if len(ads) == 0 {
		return nil
	}
	ids := make([]int, len(ads))
	index := make(map[int]int, len(ads))
	for i := range ads {
		ads[i].Facets = map[string]facet.Value{}
		ids[i] = ads[i].ID
		index[ads[i].ID] = i
	}

	query := `SELECT ad_id, "key", num, "text" FROM ad_facets WHERE ad_id IN (` +
		placeholdersFrom(1, len(ids)) + `)`
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}

	var rows []facetRow
	if err := db.Select(&rows, query, args...); err != nil {
		return err
	}
	for _, r := range rows {
		if i, ok := index[r.AdID]; ok {
			v := facet.Value{Num: r.Num, Text: r.Text}
			if d, ok := facet.Get(r.Key); ok && d.Kind == facet.MultiEnum {
				v = facet.EncodeMultiEnum(v.MultiEnumValues())
			}
			ads[i].Facets[r.Key] = v
		}
	}
	return nil
}

func GetAd(userID int, id int, tz *time.Location) (Ad, error) {
	ads, err := GetAds(userID, []int{id}, tz)
	if err != nil {
		return Ad{}, err
	}
	if len(ads) == 0 {
		return Ad{}, fmt.Errorf("ad not found")
	}
	return ads[0], nil
}

// GetUserAdIDs returns ad IDs for a user based on filter type
// filterType: "bookmarked", "active", "inactive", or "deleted"
func GetUserAdIDs(userID int, filterType string) ([]int, error) {
	var query string
	var args []any

	switch filterType {
	case "bookmarked":
		query = `
			SELECT COALESCE(json_agg(a.id ORDER BY a.created_at DESC), '[]'::json)
			FROM ads a
			JOIN bookmarks b ON a.id = b.ad_id
			WHERE b.user_id = $1
			  AND a.inactive_at IS NULL AND a.deleted_at IS NULL
		`
		args = []any{userID}
	case "active":
		query = `
			SELECT COALESCE(json_agg(id ORDER BY created_at DESC), '[]'::json)
			FROM ads
			WHERE user_id = $1
			  AND inactive_at IS NULL AND deleted_at IS NULL
		`
		args = []any{userID}
	case "inactive":
		query = `
			SELECT COALESCE(json_agg(id ORDER BY inactive_at DESC), '[]'::json)
			FROM ads
			WHERE user_id = $1 AND inactive_at IS NOT NULL AND deleted_at IS NULL
		`
		args = []any{userID}
	case "deleted":
		query = `
			SELECT COALESCE(json_agg(id ORDER BY deleted_at DESC), '[]'::json)
			FROM ads
			WHERE user_id = $1 AND deleted_at IS NOT NULL
		`
		args = []any{userID}
	default:
		return nil, fmt.Errorf("invalid filter type: %s", filterType)
	}

	var adIDs []int
	err := db.QueryJSON(&adIDs, query, args...)
	return adIDs, err
}

// CountActiveAdsByUser returns the number of live ads for a user
func CountActiveAdsByUser(userID int) (int, error) {
	var n int
	err := db.QueryRow(
		"SELECT COUNT(*) FROM ads WHERE user_id = $1 AND inactive_at IS NULL AND deleted_at IS NULL",
		userID,
	).Scan(&n)
	return n, err
}

// Pause hides an ad from listings; the owner can activate it again.
func Pause(id int) error {
	_, err := db.Exec(`
		UPDATE ads
		SET inactive_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND deleted_at IS NULL AND inactive_at IS NULL
	`, id)
	return err
}

// Activate restores an inactive ad to live listings and renews expires_at.
func Activate(id int) error {
	a, err := GetAd(0, id, time.UTC)
	if err != nil {
		return err
	}
	if !a.IsInactive() {
		return fmt.Errorf("ad is not inactive")
	}
	expiresAt := ComputeExpiresAt(a.Facets, time.Now().UTC())
	_, err = db.Exec(`
		UPDATE ads
		SET inactive_at = NULL, expires_at = $1
		WHERE id = $2 AND deleted_at IS NULL AND inactive_at IS NOT NULL
	`, expiresAt, id)
	return err
}

// Renew sets an active ad's expires_at to now + AdExpireMonths.
func Renew(id int) error {
	a, err := GetAd(0, id, time.UTC)
	if err != nil {
		return err
	}
	if !a.IsActive() {
		return fmt.Errorf("ad is not active")
	}
	now := time.Now().UTC()
	if !RenewEligible(a.ExpiresAt.UTC(), now) {
		return fmt.Errorf("ad is not eligible to renew")
	}
	expiresAt := now.AddDate(0, config.AdExpireMonths, 0)
	_, err = db.Exec(`
		UPDATE ads
		SET expires_at = $1
		WHERE id = $2 AND deleted_at IS NULL AND inactive_at IS NULL
	`, expiresAt, id)
	return err
}

// DueExpire holds an active ad past its expires_at.
type DueExpire struct {
	ID     int `db:"id"`
	UserID int `db:"user_id"`
}

// ListAdsDueToExpire returns active ads whose expires_at is in the past.
func ListAdsDueToExpire() ([]DueExpire, error) {
	var rows []DueExpire
	err := db.Select(&rows, `
		SELECT id, user_id
		FROM ads
		WHERE inactive_at IS NULL
		  AND deleted_at IS NULL
		  AND expires_at < CURRENT_TIMESTAMP
		ORDER BY expires_at ASC, id ASC
	`)
	return rows, err
}

// PermanentlyDelete soft-deletes an ad as a zombie that cannot be restored.
func PermanentlyDelete(id int) error {
	_, err := db.Exec(`
		UPDATE ads
		SET deleted_at = CURRENT_TIMESTAMP, inactive_at = NULL
		WHERE id = $1 AND deleted_at IS NULL
	`, id)
	return err
}

// PermanentlyDeleteByUser soft-deletes all non-zombie ads owned by userID.
func PermanentlyDeleteByUser(userID int) ([]int, error) {
	var ids []int
	err := db.Select(&ids, `
		UPDATE ads
		SET deleted_at = CURRENT_TIMESTAMP, inactive_at = NULL
		WHERE user_id = $1 AND deleted_at IS NULL
		RETURNING id
	`, userID)
	return ids, err
}
