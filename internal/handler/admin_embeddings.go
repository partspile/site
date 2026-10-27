package handler

import (
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/partspile/site/internal/ad"
	"github.com/partspile/site/internal/cookie"
	"github.com/partspile/site/internal/local"
	"github.com/partspile/site/internal/logger"
	"github.com/partspile/site/internal/ui"
	"github.com/partspile/site/internal/user"
	"github.com/partspile/site/internal/vector"
)

func AdminEmbeddingsHandler(c *fiber.Ctx) error {
	currentUserID := local.GetUserID(c)
	categoryID := embeddingCategoryID(c)
	siteCategoryID := embeddingSiteCategoryID(c, categoryID)
	inspectUserID := embeddingUserID(c, currentUserID)
	data, err := embeddingAdminData(
		inspectUserID, categoryID, siteCategoryID,
		cookie.GetTimezone(c),
	)
	if err != nil {
		logger.Error("Failed to load embedding admin data", "error", err)
		return showError(c, "Failed to load embedding data")
	}
	return render(c, ui.AdminDashboardContainerWithEmbeddings(
		"embeddings", data,
	))
}

func embeddingCategoryID(c *fiber.Ctx) int {
	if q := embeddingParam(c, "category"); q != "" {
		return ad.ParseCategory(q)
	}
	return cookie.GetCategoryID(c)
}

func embeddingSiteCategoryID(c *fiber.Ctx, fallback int) int {
	if q := embeddingParam(c, "site_category"); q != "" {
		return ad.ParseCategory(q)
	}
	return fallback
}

func embeddingUserID(c *fiber.Ctx, fallback int) int {
	raw := embeddingParam(c, "user")
	if raw == "" {
		return fallback
	}
	id, err := strconv.Atoi(raw)
	if err != nil || id <= 0 {
		return fallback
	}
	return id
}

func embeddingParam(c *fiber.Ctx, key string) string {
	if v := strings.TrimSpace(c.Query(key)); v != "" {
		return v
	}
	return strings.TrimSpace(c.FormValue(key))
}

func AdminEmbeddingsUserActivitiesHandler(c *fiber.Ctx) error {
	userID := embeddingUserID(c, local.GetUserID(c))
	categoryID := embeddingCategoryID(c)
	acts, err := vector.InspectUserActivities(userID, categoryID)
	if err != nil {
		logger.Error("Failed to load user embedding activities",
			"error", err)
		return showError(c, "Failed to load embedding data")
	}
	rows := embeddingActivityRows(acts, cookie.GetTimezone(c))
	return render(c, ui.EmbeddingActivityRows(
		"embedding-user-activity-rows", rows,
	))
}

func AdminEmbeddingsSiteActivitiesHandler(c *fiber.Ctx) error {
	categoryID := embeddingSiteCategoryID(c, embeddingCategoryID(c))
	acts, err := vector.InspectSiteActivities(categoryID)
	if err != nil {
		logger.Error("Failed to load site embedding activities",
			"error", err)
		return showError(c, "Failed to load embedding data")
	}
	rows := embeddingActivityRows(acts, cookie.GetTimezone(c))
	return render(c, ui.EmbeddingActivityRows(
		"embedding-site-activity-rows", rows,
	))
}

func AdminEmbeddingsClearQueryCacheHandler(c *fiber.Ctx) error {
	vector.ClearQueryEmbeddingCache()
	return AdminEmbeddingsHandler(c)
}

func AdminEmbeddingsClearUserCacheHandler(c *fiber.Ctx) error {
	vector.ClearUserEmbeddingCache()
	return AdminEmbeddingsHandler(c)
}

func AdminEmbeddingsClearSiteCacheHandler(c *fiber.Ctx) error {
	vector.ClearSiteEmbeddingCache()
	return AdminEmbeddingsHandler(c)
}

func AdminEmbeddingsClearAllCachesHandler(c *fiber.Ctx) error {
	vector.ClearAllEmbeddingCaches()
	return AdminEmbeddingsHandler(c)
}

func embeddingAdminData(userID, categoryID, siteCategoryID int,
	loc *time.Location) (ui.EmbeddingAdminData, error) {
	stats, err := ad.GetEmbeddingStats()
	if err != nil {
		return ui.EmbeddingAdminData{}, err
	}
	missing, err := ad.ListMissingEmbeddings(25)
	if err != nil {
		return ui.EmbeddingAdminData{}, err
	}
	categories, err := ad.GetCategories()
	if err != nil {
		return ui.EmbeddingAdminData{}, err
	}
	users, err := user.GetAllUsers("name", "ASC")
	if err != nil {
		return ui.EmbeddingAdminData{}, err
	}
	userActs, err := vector.InspectUserActivities(userID, categoryID)
	if err != nil {
		return ui.EmbeddingAdminData{}, err
	}
	siteActs, err := vector.InspectSiteActivities(siteCategoryID)
	if err != nil {
		return ui.EmbeddingAdminData{}, err
	}
	cacheStats := vector.GetEmbeddingCacheStats()
	provider, model := vector.EmbedderInfo()
	return ui.EmbeddingAdminData{
		EmbedderProvider: provider,
		EmbedderModel:    model,
		EmbeddedCount:    stats.Embedded,
		MissingCount:     stats.Missing,
		QueueDepth:       vector.QueueDepth(),
		CategoryID:       categoryID,
		SiteCategoryID:   siteCategoryID,
		Categories:       categoryOptions(categories),
		UserID:           userID,
		Users:            userOptions(users),
		UserActivities:   embeddingActivityRows(userActs, loc),
		SiteActivities:   embeddingActivityRows(siteActs, loc),
		Caches: []ui.EmbeddingCachePanel{
			cachePanelFromStats("Query", cacheStats["query"]),
			cachePanelFromStats("User", cacheStats["user"]),
			cachePanelFromStats("Site", cacheStats["site"]),
		},
		MissingAds: missingEmbeddingRows(missing),
	}, nil
}

func userOptions(users []user.User) []ui.UserOption {
	out := make([]ui.UserOption, 0, len(users))
	for _, u := range users {
		if u.DeletedAt != nil {
			continue
		}
		name := u.Name
		if name == "" {
			name = "User " + strconv.Itoa(u.ID)
		}
		out = append(out, ui.UserOption{ID: u.ID, Name: name})
	}
	return out
}

func embeddingActivityRows(rows []vector.ActivityInspect,
	loc *time.Location) []ui.EmbeddingActivityRow {
	out := make([]ui.EmbeddingActivityRow, len(rows))
	for i, r := range rows {
		out[i] = ui.EmbeddingActivityRow{
			AdID:         r.AdID,
			AdTitle:      r.AdTitle,
			ActivityType: r.ActivityType,
			Weight:       r.Weight,
			Timestamp:    formatActivityTime(r.Timestamp, loc),
		}
	}
	return out
}

func formatActivityTime(raw string, loc *time.Location) string {
	if raw == "" || raw == "—" {
		return "—"
	}
	if loc == nil {
		loc = time.UTC
	}
	t, ok := parseActivityTime(raw)
	if !ok {
		return raw
	}
	return t.In(loc).Format("Jan 2, 3:04 PM")
}

func parseActivityTime(raw string) (time.Time, bool) {
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.999999999",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, raw); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func cachePanelFromStats(name string,
	stats map[string]interface{}) ui.EmbeddingCachePanel {
	if stats == nil {
		return ui.EmbeddingCachePanel{Name: name}
	}
	return ui.EmbeddingCachePanel{
		Name:       name,
		Hits:       statsInt64(stats["hits"]),
		Misses:     statsInt64(stats["misses"]),
		HitRatePct: statsFloat64(stats["hit_rate"]),
		ItemCount:  statsInt64(stats["current_items"]),
		MemoryKB:   statsFloat64(stats["memory_used_kb"]),
	}
}

func missingEmbeddingRows(rows []ad.MissingEmbedding) []ui.MissingEmbeddingRow {
	out := make([]ui.MissingEmbeddingRow, len(rows))
	for i, r := range rows {
		out[i] = ui.MissingEmbeddingRow{
			AdID:         r.ID,
			Title:        r.Title,
			CategoryName: r.CategoryName,
		}
	}
	return out
}

func statsInt64(v any) int64 {
	switch n := v.(type) {
	case int:
		return int64(n)
	case int64:
		return n
	case uint64:
		return int64(n)
	case float64:
		return int64(n)
	default:
		return 0
	}
}

func statsFloat64(v any) float64 {
	switch n := v.(type) {
	case float64:
		if math.IsNaN(n) {
			return 0
		}
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case uint64:
		return float64(n)
	default:
		return 0
	}
}
