package vector

import (
	"fmt"
	"strings"

	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/db"
)

// ActivityInspect holds one weighted activity row for admin display.
type ActivityInspect struct {
	AdID         int
	AdTitle      string
	ActivityType string
	Weight       float32
	Timestamp    string
}

func InspectUserActivities(userID, categoryID int) ([]ActivityInspect, error) {
	activities, err := getUserActivities(
		userID, categoryID, config.VectorUserEmbeddingLimit,
	)
	if err != nil {
		return nil, err
	}
	return enrichActivities(activities)
}

func InspectSiteActivities(categoryID int) ([]ActivityInspect, error) {
	activities, err := getSiteActivities(
		categoryID, "default", config.VectorSiteEmbeddingLimit,
	)
	if err != nil {
		return nil, err
	}
	if len(activities) > 0 {
		vectors, weights := calculateWeightedVectors(activities)
		if AggregateEmbeddings(vectors, weights) != nil {
			return enrichActivities(activities)
		}
	}
	return inspectRecentAdEmbeddingSource(categoryID)
}

func inspectRecentAdEmbeddingSource(categoryID int) ([]ActivityInspect, error) {
	ids, err := recentAdEmbeddingIDs(categoryID)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 && categoryID > 0 {
		ids, err = recentAdEmbeddingIDs(0)
		if err != nil {
			return nil, err
		}
	}
	if len(ids) == 0 {
		return nil, nil
	}
	titles, err := adTitlesByIDs(ids)
	if err != nil {
		return nil, err
	}
	out := make([]ActivityInspect, len(ids))
	for i, id := range ids {
		title := titles[id]
		if title == "" {
			title = fmt.Sprintf("ad #%d", id)
		}
		out[i] = ActivityInspect{
			AdID:         id,
			AdTitle:      title,
			ActivityType: "recent_ad",
			Weight:       1,
			Timestamp:    "—",
		}
	}
	return out, nil
}

func enrichActivities(activities []AdActivity) ([]ActivityInspect, error) {
	if len(activities) == 0 {
		return nil, nil
	}
	ids := make([]int, len(activities))
	for i, act := range activities {
		ids[i] = act.AdID
	}
	titles, err := adTitlesByIDs(ids)
	if err != nil {
		return nil, err
	}
	out := make([]ActivityInspect, len(activities))
	for i, act := range activities {
		title := titles[act.AdID]
		if title == "" {
			title = fmt.Sprintf("ad #%d", act.AdID)
		}
		out[i] = ActivityInspect{
			AdID:         act.AdID,
			AdTitle:      title,
			ActivityType: act.ActivityType,
			Weight:       act.Weight,
			Timestamp:    act.Timestamp,
		}
	}
	return out, nil
}

func adTitlesByIDs(ids []int) (map[int]string, error) {
	if len(ids) == 0 {
		return map[int]string{}, nil
	}
	type titleRow struct {
		ID    int    `json:"id"`
		Title string `json:"title"`
	}
	ph := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		ph[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}
	query := fmt.Sprintf(`
		SELECT COALESCE(
			json_agg(json_build_object('id', id, 'title', title)),
			'[]'::json
		)
		FROM ads
		WHERE inactive_at IS NULL AND deleted_at IS NULL AND id IN (%s)`,
		strings.Join(ph, ","),
	)
	var rows []titleRow
	if err := db.QueryJSON(&rows, query, args...); err != nil {
		return nil, err
	}
	titles := make(map[int]string, len(rows))
	for _, r := range rows {
		titles[r.ID] = r.Title
	}
	return titles, nil
}

func QueueDepth() int {
	return len(adQueue)
}

func GetEmbeddingCacheStats() map[string]map[string]interface{} {
	stats := make(map[string]map[string]interface{})
	if queryEmbeddingCache != nil {
		stats["query"] = queryEmbeddingCache.StatsCopy()
	}
	if userEmbeddingCache != nil {
		stats["user"] = userEmbeddingCache.StatsCopy()
	}
	if siteEmbeddingCache != nil {
		stats["site"] = siteEmbeddingCache.StatsCopy()
	}
	return stats
}

func ClearQueryEmbeddingCache() {
	if queryEmbeddingCache != nil {
		queryEmbeddingCache.Clear()
	}
}

func ClearUserEmbeddingCache() {
	if userEmbeddingCache != nil {
		userEmbeddingCache.Clear()
	}
}

func ClearSiteEmbeddingCache() {
	if siteEmbeddingCache != nil {
		siteEmbeddingCache.Clear()
	}
}

func ClearAllEmbeddingCaches() {
	ClearQueryEmbeddingCache()
	ClearUserEmbeddingCache()
	ClearSiteEmbeddingCache()
}

func TriggerBackfill() {
	ProcessAdsWithoutVectors()
}
