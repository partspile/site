package ad

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/db"
	"github.com/partspile/site/internal/facet"
)

type UpdateInput struct {
	AdID         int
	UserID       int
	Title        string
	Description  string
	LocationText string
	Facets       map[string]facet.Value
	Suggestions  []Suggestion
	ImagesAdded  int
	Tz           *time.Location
	Now          time.Time
}

func UpdateAd(input UpdateInput) error {
	a, err := GetAd(input.UserID, input.AdID, input.Tz)
	if err != nil {
		return err
	}
	if a.UserID != input.UserID {
		return fmt.Errorf("you are not the owner of this ad")
	}
	if !a.IsActive() {
		return fmt.Errorf("cannot edit a deleted or inactive ad")
	}

	category := GetCategory(a.CategoryID)

	title := strings.TrimSpace(SanitizeAdText(input.Title))
	if title == "" {
		return fmt.Errorf("title is required")
	}
	if utf8.RuneCountInString(title) > config.MaxAdTitleLength {
		return fmt.Errorf(
			"title must be at most %d characters",
			config.MaxAdTitleLength,
		)
	}
	if TitleContainsEmoji(title) {
		return fmt.Errorf("title cannot contain emoji")
	}

	if err := LoadTags(&a); err != nil {
		return err
	}

	newSuggestions := dedupeSuggestions(input.Suggestions)
	defs := category.Facets()
	values := make(map[string]facet.Value, len(defs))
	for _, d := range defs {
		v := input.Facets[d.Key]
		if err := d.Validate(v); err != nil {
			return err
		}
		if v.Present() {
			values[d.Key] = v
		}
	}

	now := input.Now
	if now.IsZero() {
		now = time.Now()
	}

	desc, err := applyDescriptionEdit(
		a.Description, input.Description, now, input.Tz,
	)
	if err != nil {
		return err
	}
	if input.ImagesAdded > 0 {
		if a.ImageCount+input.ImagesAdded > config.MaxImagesPerAd {
			return fmt.Errorf(
				"too many images. Maximum %d images allowed per ad",
				config.MaxImagesPerAd,
			)
		}
		body := formatImageAdditionBody(a.ImageCount+1, input.ImagesAdded)
		desc = AppendHistoryEntry(
			desc, imagesAddedLabel, body, now, input.Tz,
		)
	}
	locText := input.LocationText
	if HasLocationFacet(category) {
		locText = LocationTextFromFacets(category, values)
	}
	for _, e := range BuildFieldChangeEntries(
		a, title, locText, values, category,
	) {
		desc = AppendHistoryEntry(desc, e.label, e.body, now, input.Tz)
	}
	if body := FormatTagUpdates(a.Tags, newSuggestions); body != "" {
		desc = AppendHistoryEntry(
			desc, "Description Tags", body, now, input.Tz,
		)
	}

	desc, err = EnsureDescriptionFits(desc, now, input.Tz)
	if err != nil {
		return err
	}

	locationID, latitude, longitude, err := resolveLocationFields(locText)
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("update ad: %w", err)
	}
	defer tx.Rollback()

	newImageCount := a.ImageCount + input.ImagesAdded

	expiresAt := a.ExpiresAt.UTC()
	if _, ok := SaleEndDateString(values); ok {
		expiresAt = ComputeExpiresAt(values, now.UTC())
	}

	_, err = tx.Exec(
		`UPDATE ads SET title = $1, description = $2, location_id = $3,
		 latitude = $4, longitude = $5, tags = $6, image_count = $7,
		 expires_at = $8
		 WHERE id = $9 AND user_id = $10
		   AND inactive_at IS NULL AND deleted_at IS NULL`,
		title, desc, locationID, latitude, longitude,
		tagsJSON(newSuggestions), newImageCount,
		expiresAt, input.AdID, input.UserID,
	)
	if err != nil {
		return fmt.Errorf("update ad: %w", err)
	}

	if _, err := tx.Exec(`DELETE FROM ad_facets WHERE ad_id = $1`, input.AdID); err != nil {
		return fmt.Errorf("update ad facets: %w", err)
	}
	for key, v := range values {
		if _, err := tx.Exec(
			`INSERT INTO ad_facets (ad_id, "key", num, "text") VALUES ($1, $2, $3, $4)`,
			input.AdID, key, v.Num, v.Text,
		); err != nil {
			return fmt.Errorf("update ad facet %s: %w", key, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("update ad commit: %w", err)
	}
	return nil
}
