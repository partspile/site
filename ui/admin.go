package ui

import (
	"github.com/parts-pile/site/user"
	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"
)

// AdminSectionPage renders the admin section navigation and the current section content.
func AdminSectionPage(currentUser *user.User, path, activeSection string, content g.Node) g.Node {
	sections := []struct {
		Name  string
		Label string
	}{
		{"b2-cache", "B2 Cache"},
		{"embedding-cache", "Embedding Cache"},
		{"vehicle-cache", "Vehicle Cache"},
	}
	return Div(
		ID("admin-section"),
		Class("my-8"),
		H1(g.Text("Admin Dashboard")),
		Div(
			Class("flex flex-wrap gap-2 mb-6"),
			g.Group(g.Map(sections, func(s struct{ Name, Label string }) g.Node {
				colorClass := "bg-gray-200 text-gray-800 hover:bg-gray-300"
				if s.Name == activeSection {
					colorClass = "bg-blue-500 text-white"
				}
				return Button(
					Class("px-4 py-1 rounded "+colorClass),
					ID("btn-"+s.Name),
					hx.Get("/admin/"+s.Name),
					hx.Target("#admin-section"),
					hx.Swap("outerHTML"),
					g.Text(s.Label),
				)
			})),
		),
		Div(
			ID("admin-section-content"),
			content,
		),
	)
}

// Generic cache stats panel component
func CacheStatsPanel(title string, stats map[string]interface{}, clearEndpoint, refreshEndpoint string) g.Node {
	return Div(
		Class("bg-gray-100 p-4 rounded-lg mb-4"),
		H2(Class("text-lg font-semibold mb-2"), g.Text(title)),
		Div(
			Class("grid grid-cols-2 md:grid-cols-4 gap-4 mb-4"),
			statCard("Hits", "%d", stats["hits"]),
			statCard("Misses", "%d", stats["misses"]),
			statCard("Hit Rate", "%.1f%%", stats["hit_rate"]),
			statCard("Sets", "%d", stats["sets"]),
			statCard("Memory Used", "%.0f KB", stats["memory_used_kb"]),
			statCard("Total Added", "%.0f KB", stats["total_added_kb"]),
			statCard("Total Evicted", "%.0f KB", stats["total_evicted_kb"]),
			statCard("Current Items", "%d", stats["current_items"]),
		),
		Div(
			Class("flex gap-4"),
			Button(
				Class("px-4 py-2 bg-red-500 text-white rounded hover:bg-red-600"),
				hx.Post(clearEndpoint),
				hx.Target("#admin-section-content"),
				hx.Swap("innerHTML"),
				g.Text("Clear Cache"),
			),
			Button(
				Class("px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600"),
				hx.Get(refreshEndpoint),
				hx.Target("#admin-section-content"),
				hx.Swap("innerHTML"),
				g.Text("Refresh Stats"),
			),
		),
	)
}

func statCard(label, format string, value interface{}) g.Node {
	return Div(
		Class("bg-white p-3 rounded border"),
		Strong(g.Text(label+": ")),
		g.Textf(format, value),
	)
}

func AdminB2CacheSection(stats map[string]interface{}) g.Node {
	return Div(
		H1(g.Text("B2 Cache Management")),
		CacheStatsPanel("Cache Statistics", stats, "/api/admin/b2-cache/clear", "/api/admin/b2-cache/refresh"),
	)
}

func AdminEmbeddingCacheSection(stats map[string]interface{}) g.Node {
	// Get stats for each cache type
	queryStats := getCacheStats(stats, "query", "Query Embedding Cache")
	userStats := getCacheStats(stats, "user", "User Embedding Cache")
	siteStats := getCacheStats(stats, "site", "Site Embedding Cache")

	return Div(
		H1(g.Text("Embedding Cache Management")),
		CacheStatsPanel("Query Cache Statistics", queryStats, "/api/admin/embedding-cache/query/clear", "/api/admin/embedding-cache/refresh"),
		CacheStatsPanel("User Cache Statistics", userStats, "/api/admin/embedding-cache/user/clear", "/api/admin/embedding-cache/refresh"),
		CacheStatsPanel("Site Cache Statistics", siteStats, "/api/admin/embedding-cache/site/clear", "/api/admin/embedding-cache/refresh"),
	)
}

func getCacheStats(allStats map[string]interface{}, cacheType, cacheName string) map[string]interface{} {
	if cacheStats, exists := allStats[cacheType]; exists {
		if stats, ok := cacheStats.(map[string]interface{}); ok {
			return stats
		}
	}
	return map[string]interface{}{
		"cache_type": cacheName,
		"error":      "Cache not initialized",
	}
}

func AdminVehicleCacheSection(stats map[string]interface{}) g.Node {
	return Div(
		H1(g.Text("Vehicle Cache Management")),
		CacheStatsPanel("Cache Statistics", stats, "/api/admin/vehicle-cache/clear", "/api/admin/vehicle-cache/refresh"),
	)
}
