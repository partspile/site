package ad

import (
	"encoding/json"
	"fmt"

	"github.com/partspile/site/internal/db"
	"github.com/partspile/site/internal/facet"
)

type EmbeddingInput struct {
	ID          int
	CategoryID  int
	Title       string
	Description string
	TagLabels   []string
	City        string
	AdminArea   string
	Country     string
	Latitude    float64
	Longitude   float64
	HasLocation bool
	RockCount   int
	Facets      map[string]facet.Value
	FacetDefs   []facet.Def
}

func (in EmbeddingInput) PriceValue() (amount int, currency string, ok bool) {
	v, exists := in.Facets["price"]
	if !exists || v.Num == nil {
		return 0, "", false
	}
	if v.Text != nil {
		currency = *v.Text
	}
	return *v.Num, currency, true
}

func GetForEmbedding(adID int) (EmbeddingInput, error) {
	query := `
		SELECT
			a.id,
			a.category_id,
			a.title,
			a.description,
			a.tags,
			COALESCE(l.city, '') AS city,
			COALESCE(l.admin_area, '') AS admin_area,
			COALESCE(l.country, '') AS country,
			COALESCE(l.latitude, 0) AS latitude,
			COALESCE(l.longitude, 0) AS longitude,
			CASE WHEN l.id IS NOT NULL THEN 1 ELSE 0 END AS has_location,
			a.rock_count
		FROM ads a
		LEFT JOIN locations l ON a.location_id = l.id
		WHERE a.id = $1 AND a.inactive_at IS NULL AND a.deleted_at IS NULL`
	var row struct {
		ID          int     `db:"id"`
		CategoryID  int     `db:"category_id"`
		Title       string  `db:"title"`
		Description string  `db:"description"`
		Tags        string  `db:"tags"`
		City        string  `db:"city"`
		AdminArea   string  `db:"admin_area"`
		Country     string  `db:"country"`
		Latitude    float64 `db:"latitude"`
		Longitude   float64 `db:"longitude"`
		HasLocation int     `db:"has_location"`
		RockCount   int     `db:"rock_count"`
	}
	if err := db.QueryRow(query, adID).Scan(
		&row.ID,
		&row.CategoryID,
		&row.Title,
		&row.Description,
		&row.Tags,
		&row.City,
		&row.AdminArea,
		&row.Country,
		&row.Latitude,
		&row.Longitude,
		&row.HasLocation,
		&row.RockCount,
	); err != nil {
		return EmbeddingInput{}, fmt.Errorf("embedding ad %d: %w", adID, err)
	}
	category := GetCategory(row.CategoryID)
	facets, err := loadFacetsForAd(adID)
	if err != nil {
		return EmbeddingInput{}, err
	}
	var tags []Suggestion
	if row.Tags != "" && row.Tags != "[]" {
		_ = json.Unmarshal([]byte(row.Tags), &tags)
	}
	tagLabels := make([]string, 0, len(tags))
	for _, t := range tags {
		if t.Label != "" {
			tagLabels = append(tagLabels, t.Label)
		}
	}
	return EmbeddingInput{
		ID:          row.ID,
		CategoryID:  row.CategoryID,
		Title:       row.Title,
		Description: row.Description,
		TagLabels:   tagLabels,
		City:        row.City,
		AdminArea:   row.AdminArea,
		Country:     row.Country,
		Latitude:    row.Latitude,
		Longitude:   row.Longitude,
		HasLocation: row.HasLocation != 0,
		RockCount:   row.RockCount,
		Facets:      facets,
		FacetDefs:   category.Facets(),
	}, nil
}

func GetAdsWithoutVectors() ([]int, error) {
	var ids []int
	err := db.QueryJSON(&ids, `
		SELECT COALESCE(json_agg(id), '[]'::json)
		FROM ads
		WHERE inactive_at IS NULL AND deleted_at IS NULL AND embedding IS NULL`)
	return ids, err
}

func loadFacetsForAd(adID int) (map[string]facet.Value, error) {
	var rows []struct {
		Key  string  `db:"key"`
		Num  *int    `db:"num"`
		Text *string `db:"text"`
	}
	if err := db.Select(&rows,
		`SELECT "key", num, "text" FROM ad_facets WHERE ad_id = $1`, adID,
	); err != nil {
		return nil, err
	}
	out := make(map[string]facet.Value, len(rows))
	for _, r := range rows {
		v := facet.Value{Num: r.Num, Text: r.Text}
		if d, ok := facet.Get(r.Key); ok && d.Kind == facet.MultiEnum {
			v = facet.EncodeMultiEnum(v.MultiEnumValues())
		}
		out[r.Key] = v
	}
	return out, nil
}
