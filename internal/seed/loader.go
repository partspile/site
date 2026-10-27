package seed

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/nyaruka/phonenumbers"
	adp "github.com/partspile/site/internal/ad"
	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/currency"
	"github.com/partspile/site/internal/db"
	"github.com/partspile/site/internal/encryption"
	"github.com/partspile/site/internal/facet"
	"github.com/partspile/site/internal/logger"
	"github.com/partspile/site/internal/password"
)

// categoryJSON represents a category from ad-category.json
type categoryJSON struct {
	Name       string   `json:"name"`
	ImageFile  string   `json:"image_file"`
	SeedAdFile string   `json:"seed_ad_file"`
	Facets     []string `json:"facets"`
}

// CategoryFiles stores file information for a category
type CategoryFiles struct {
	ID         int
	SeedAdFile string
	Facets     []string
}

var categoryFiles = make(map[string]CategoryFiles)

// LoadAll loads all seed data into the database.
func LoadAll() error {
	startTime := time.Now()
	if err := LoadUsers(); err != nil {
		return fmt.Errorf("loading users: %w", err)
	}
	logger.Info("LoadUsers completed", "duration", time.Since(startTime))

	startTime = time.Now()
	if err := LoadCategories(); err != nil {
		return fmt.Errorf("loading categories: %w", err)
	}
	logger.Info("LoadCategories completed", "duration", time.Since(startTime))

	startTime = time.Now()
	if err := LoadAds(); err != nil {
		return fmt.Errorf("loading ads: %w", err)
	}
	logger.Info("LoadAds completed", "duration", time.Since(startTime))

	if err := syncIdentitySequences(); err != nil {
		return fmt.Errorf("syncing identity sequences: %w", err)
	}
	return nil
}

func syncIdentitySequences() error {
	var maxID int
	err := db.QueryRow(`SELECT COALESCE(MAX(id), 0) FROM ads`).Scan(&maxID)
	if err != nil {
		return err
	}
	if maxID == 0 {
		return nil
	}
	_, err = db.Exec(`
		SELECT setval(pg_get_serial_sequence('ads', 'id'), $1)`,
		maxID,
	)
	return err
}

// userJSON represents a user from user.json
type userJSON struct {
	Name     string `json:"name"`
	Password string `json:"password"`
	Phone    string `json:"phone"`
	IsAdmin  bool   `json:"is_admin,omitempty"`
}

// LocationData represents location data from ad JSON files
type LocationData struct {
	City      string  `json:"city"`
	AdminArea string  `json:"admin_area"`
	Country   string  `json:"country"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// Ad represents an ad for loading purposes
type Ad struct {
	ID          int
	CategoryID  int
	Title       string
	Description string
	Price       float64
	CreatedAt   string
	UserID      int
	ImageCount  int
	Location    LocationData
}

// LoadUsers loads users from user.json into the users table
func LoadUsers() error {
	data, err := os.ReadFile("internal/seed/user.json")
	if err != nil {
		return err
	}

	var users []userJSON
	if err := json.Unmarshal(data, &users); err != nil {
		return err
	}

	for _, u := range users {
		passwordHash, passwordSalt, err := password.HashPassword(u.Password)
		if err != nil {
			return fmt.Errorf("hashing password for user %s: %w", u.Name, err)
		}

		var phoneE64 string
		num, err := phonenumbers.Parse(u.Phone, "")
		if err != nil {
			if !strings.HasPrefix(u.Phone, "+") {
				num, err = phonenumbers.Parse(u.Phone, "US")
				if err != nil {
					return fmt.Errorf("parsing phone number for user %s: %w", u.Name, err)
				}
			} else {
				return fmt.Errorf("parsing phone number for user %s: %w", u.Name, err)
			}
		}

		if !phonenumbers.IsPossibleNumber(num) {
			return fmt.Errorf("invalid phone number format for user %s: %s", u.Name, u.Phone)
		}

		phoneE64 = phonenumbers.Format(num, phonenumbers.E164)

		isAdmin := 0
		if u.IsAdmin {
			isAdmin = 1
		}
		var userID int
		err = db.QueryRow(
			`INSERT INTO users (encrypted_name, name_nonce, name_hash, password_hash, password_salt, password_algo, encrypted_phone, phone_nonce, phone_hash, is_admin)
			 VALUES ($1, $2, $3, $4, $5, 'argon2id', $6, $7, $8, $9)
			 RETURNING id`,
			[]byte{}, []byte{}, db.HashString(u.Name), passwordHash, passwordSalt,
			[]byte{}, []byte{}, db.HashString(phoneE64),
			isAdmin,
		).Scan(&userID)
		if err != nil {
			return fmt.Errorf("inserting user %s: %w", u.Name, err)
		}

		encryptedName, nameNonce, err := encryption.Encrypt(int(userID), u.Name, config.DBEncryptionKey)
		if err != nil {
			return fmt.Errorf("encrypting name for user %s: %w", u.Name, err)
		}
		nameHash := db.HashString(u.Name)

		encryptedPhone, phoneNonce, err := encryption.Encrypt(int(userID), phoneE64, config.DBEncryptionKey)
		if err != nil {
			return fmt.Errorf("encrypting phone for user %s: %w", u.Name, err)
		}
		phoneHash := db.HashString(phoneE64)

		encryptedNameBytes, _ := base64.StdEncoding.DecodeString(encryptedName)
		nameNonceBytes, _ := base64.StdEncoding.DecodeString(nameNonce)
		encryptedPhoneBytes, _ := base64.StdEncoding.DecodeString(encryptedPhone)
		phoneNonceBytes, _ := base64.StdEncoding.DecodeString(phoneNonce)

		_, err = db.Exec(
			`UPDATE users SET 
				encrypted_name = $1, name_nonce = $2, name_hash = $3,
				encrypted_phone = $4, phone_nonce = $5, phone_hash = $6,
				phone_verified = 1
			WHERE id = $7`,
			encryptedNameBytes, nameNonceBytes, nameHash,
			encryptedPhoneBytes, phoneNonceBytes, phoneHash,
			userID,
		)
		if err != nil {
			return fmt.Errorf("updating encrypted fields for user %s: %w", u.Name, err)
		}
	}

	return nil
}

func readCategoryJSON() ([]categoryJSON, error) {
	data, err := os.ReadFile("internal/seed/ad-category.json")
	if err != nil {
		return nil, err
	}
	var categories []categoryJSON
	if err := json.Unmarshal(data, &categories); err != nil {
		return nil, err
	}
	return categories, nil
}

// LoadCategories loads ad-category.json into categories
func LoadCategories() error {
	categories, err := readCategoryJSON()
	if err != nil {
		return err
	}

	sort.Slice(categories, func(i, j int) bool {
		return categories[i].Name < categories[j].Name
	})

	for _, cat := range categories {
		facetKeys := cat.Facets
		if facetKeys == nil {
			facetKeys = []string{}
		}
		facetsJSON, err := json.Marshal(facetKeys)
		if err != nil {
			return fmt.Errorf("marshaling facets for category %s: %w", cat.Name, err)
		}

		var categoryID int
		err = db.QueryRow(
			`INSERT INTO categories (name, seed_ad_file, image_file, facets) VALUES ($1, $2, $3, $4)
			 RETURNING id`,
			cat.Name, cat.SeedAdFile, cat.ImageFile, string(facetsJSON),
		).Scan(&categoryID)
		if err != nil {
			return fmt.Errorf("inserting category %s: %w", cat.Name, err)
		}
		categoryFiles[cat.Name] = CategoryFiles{
			ID:         int(categoryID),
			SeedAdFile: cat.SeedAdFile,
			Facets:     facetKeys,
		}
	}

	return nil
}

// LoadAds loads ad files into ads table
func LoadAds() error {
	usedIDs := make(map[int]string)

	categoryNames := make([]string, 0, len(categoryFiles))
	for name := range categoryFiles {
		categoryNames = append(categoryNames, name)
	}
	sort.Strings(categoryNames)

	for _, categoryName := range categoryNames {
		files := categoryFiles[categoryName]
		if err := loadAdsFromFile(files.ID, files.SeedAdFile, files.Facets, usedIDs); err != nil {
			return fmt.Errorf("loading ads from %s for category %s: %w", files.SeedAdFile, categoryName, err)
		}
	}

	return nil
}

// HistoryEntryJSON is one description edit-history block in seed ad files.
type HistoryEntryJSON struct {
	Label string `json:"label"`
	Body  string `json:"body"`
	At    string `json:"at"`
}

// SuggestionJSON is one description tag in seed ad files.
type SuggestionJSON struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// adJSON represents the flat JSON structure from ad-*.json files
type adJSON struct {
	ID                 int                  `json:"id"`
	Title              string               `json:"title"`
	Description        string               `json:"description,omitempty"`
	DescriptionHistory []HistoryEntryJSON   `json:"description_history,omitempty"`
	Suggestions        []SuggestionJSON     `json:"suggestions,omitempty"`
	Price              float64              `json:"price"`
	CreatedAt          string               `json:"created_at"`
	UserID             int                  `json:"user_id"`
	ImageCount         int                  `json:"image_count"`
	Location           LocationData         `json:"location"`
	Facets             map[string]facetJSON `json:"facets,omitempty"`
}

// facetJSON is a single facet value in a seed ad file (maps to ad_facets).
type facetJSON struct {
	Num    *int     `json:"num,omitempty"`
	Text   *string  `json:"text,omitempty"`
	Values []string `json:"values,omitempty"`
}

var seedHistoryLocation *time.Location

func init() {
	loc, err := time.LoadLocation("America/Detroit")
	if err != nil {
		seedHistoryLocation = time.UTC
		return
	}
	seedHistoryLocation = loc
}

// AssembleDescription applies seed history entries to the original body.
func AssembleDescription(original string, history []HistoryEntryJSON,
	createdAt time.Time) (string, error) {
	desc := adp.WrapDescription(original, createdAt, seedHistoryLocation)
	for _, h := range history {
		at, err := time.Parse(time.RFC3339, h.At)
		if err != nil {
			return "", fmt.Errorf("parsing history at %q: %w", h.At, err)
		}
		desc = adp.AppendHistoryEntry(
			desc, h.Label, h.Body, at, seedHistoryLocation,
		)
	}
	return desc, nil
}

func seedSuggestions(aj adJSON) []adp.Suggestion {
	if len(aj.Suggestions) == 0 {
		return nil
	}
	out := make([]adp.Suggestion, 0, len(aj.Suggestions))
	for _, s := range aj.Suggestions {
		out = append(out, adp.Suggestion{Label: s.Label, Value: s.Value})
	}
	return out
}

func convertAdJSON(aj adJSON) Ad {
	return Ad{
		ID:          aj.ID,
		Title:       aj.Title,
		Description: aj.Description,
		Price:       aj.Price,
		CreatedAt:   aj.CreatedAt,
		UserID:      aj.UserID,
		ImageCount:  aj.ImageCount,
		Location:    aj.Location,
	}
}

func loadAdsFromFile(categoryID int, filename string, categoryFacets []string,
	usedIDs map[int]string) error {
	data, err := os.ReadFile("internal/seed/" + filename)
	if err != nil {
		return err
	}

	var adsJSON []adJSON
	if err := json.Unmarshal(data, &adsJSON); err != nil {
		return err
	}

	var testUserID int
	err = db.QueryRow("SELECT id FROM users WHERE name_hash = $1", db.HashString("test")).Scan(&testUserID)
	if err != nil {
		return fmt.Errorf("finding test user: %w", err)
	}

	for i, aj := range adsJSON {
		if aj.ID == 0 {
			return fmt.Errorf("ad at index %d in %s is missing required 'id' field", i, filename)
		}

		adID := aj.ID
		if existingFile, exists := usedIDs[adID]; exists {
			return fmt.Errorf("duplicate ad ID %d: found in both %s and %s", adID, filename, existingFile)
		}
		usedIDs[adID] = filename

		ad := convertAdJSON(aj)

		locationID, err := getOrCreateLocationForAd(aj)
		if err != nil {
			return fmt.Errorf("getting/creating location: %w", err)
		}

		price := int(ad.Price + 0.5)
		priceCurrency := currency.DefaultFromRegion(aj.Location.Country)

		createdAt, err := time.Parse(time.RFC3339, ad.CreatedAt)
		if err != nil {
			return fmt.Errorf("parsing created_at: %w", err)
		}

		description, err := AssembleDescription(
			ad.Description, aj.DescriptionHistory, createdAt,
		)
		if err != nil {
			return fmt.Errorf("assembling description for ad %d: %w", adID, err)
		}
		tagsJSON := adp.TagsJSON(seedSuggestions(aj))

		expiresAt := createdAt.AddDate(0, config.AdExpireMonths, 0)
		if v, ok := aj.Facets["sale_end_date"]; ok && v.Text != nil {
			if t, err := adp.ExpiresAtFromSaleEnd(*v.Text); err == nil {
				expiresAt = t
			}
		} else {
			// Seed created_at values are often years old (for listing-age
			// variety). Give them a fresh expire window so init doesn't
			// immediately pause every ad.
			expiresAt = time.Now().UTC().AddDate(0, config.AdExpireMonths, 0)
		}

		_, err = db.Exec(
			`INSERT INTO ads (id, category_id, title, description, created_at,
			 user_id, image_count, location_id, tags, expires_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
			adID, categoryID, ad.Title, description, createdAt,
			testUserID, ad.ImageCount, locationID, tagsJSON, expiresAt,
		)
		if err != nil {
			return fmt.Errorf("inserting ad with ID %d: %w", adID, err)
		}

		if err := insertAdFacets(adID, price, priceCurrency, categoryFacets, aj.Facets); err != nil {
			return fmt.Errorf("inserting facets for ad %d: %w", adID, err)
		}
	}

	return nil
}

// insertAdFacets writes the price facet (from the ad's top-level price) plus any
// generic facets declared in the seed file into ad_facets.
func insertAdFacets(adID, price int, priceCurrency string,
	categoryFacets []string, facets map[string]facetJSON) error {
	hasPrice := false
	for _, key := range categoryFacets {
		if key == "price" {
			hasPrice = true
			break
		}
	}
	if hasPrice {
		if _, err := db.Exec(
			`INSERT INTO ad_facets (ad_id, "key", num, "text") VALUES ($1, 'price', $2, $3)`,
			adID, price, priceCurrency,
		); err != nil {
			return err
		}
	}
	for key, v := range facets {
		if key == "price" {
			continue
		}
		text := v.Text
		if len(v.Values) > 0 {
			encoded := facet.EncodeMultiEnum(v.Values)
			text = encoded.Text
		}
		if _, err := db.Exec(
			`INSERT INTO ad_facets (ad_id, "key", num, "text") VALUES ($1, $2, $3, $4)`,
			adID, key, v.Num, text,
		); err != nil {
			return err
		}
	}
	return nil
}

func getOrCreateLocationForAd(aj adJSON) (int, error) {
	if v, ok := aj.Facets["address"]; ok && v.Text != nil {
		if raw := strings.TrimSpace(*v.Text); raw != "" {
			return getOrCreateLocationRaw(raw, aj.Location)
		}
	}
	return getOrCreateLocation(aj.Location)
}

func getOrCreateLocationRaw(rawText string, loc LocationData) (int, error) {
	var locationID int
	err := db.QueryRow("SELECT id FROM locations WHERE raw_text = $1", rawText).Scan(&locationID)
	if err == nil {
		return locationID, nil
	}

	err = db.QueryRow(
		`INSERT INTO locations (raw_text, city, admin_area, country, latitude, longitude)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (raw_text) DO NOTHING
		 RETURNING id`,
		rawText, loc.City, loc.AdminArea, loc.Country, loc.Latitude, loc.Longitude,
	).Scan(&locationID)
	if err == nil {
		return locationID, nil
	}

	err = db.QueryRow("SELECT id FROM locations WHERE raw_text = $1", rawText).Scan(&locationID)
	if err != nil {
		return 0, fmt.Errorf("inserting location: %w", err)
	}
	return locationID, nil
}

func getOrCreateLocation(loc LocationData) (int, error) {
	rawText := fmt.Sprintf("%s, %s, %s", loc.City, loc.AdminArea, loc.Country)

	var locationID int
	err := db.QueryRow("SELECT id FROM locations WHERE raw_text = $1", rawText).Scan(&locationID)
	if err == nil {
		return locationID, nil
	}

	err = db.QueryRow(
		`INSERT INTO locations (raw_text, city, admin_area, country, latitude, longitude)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (raw_text) DO NOTHING
		 RETURNING id`,
		rawText, loc.City, loc.AdminArea, loc.Country, loc.Latitude, loc.Longitude,
	).Scan(&locationID)
	if err == nil {
		return locationID, nil
	}

	err = db.QueryRow("SELECT id FROM locations WHERE raw_text = $1", rawText).Scan(&locationID)
	if err != nil {
		return 0, fmt.Errorf("inserting location: %w", err)
	}
	return locationID, nil
}
