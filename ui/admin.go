package ui

import (
	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"
)

// AdminSectionPage renders the admin section navigation and the current section content.
func AdminSectionPage(userID int, userName string, path, activeSection string, content g.Node) g.Node {
	return Div(
		ID("admin-section"),
		H1(Class("text-4xl font-bold mb-8"), g.Text("Admin Dashboard")),
		Div(Class("text-gray-600 text-sm mb-6"), g.Text("Manage system caches and configurations.")),
		adminNavigation(activeSection),
		Div(
			ID("admin-section-content"),
			Class("mt-6"),
			content,
		),
	)
}

// adminNavigation renders the tab navigation for the admin page
func adminNavigation(activeSection string) g.Node {
	sections := []struct {
		name  string
		label string
		href  string
	}{
		{"b2-cache", "B2 Cache", "/admin/b2-cache"},
		{"embedding-cache", "Embedding Cache", "/admin/embedding-cache"},
		{"vehicle-cache", "Vehicle Cache", "/admin/vehicle-cache"},
		{"part-cache", "Part Cache", "/admin/part-cache"},
	}

	var tabNodes []g.Node
	for _, section := range sections {
		var classes string
		if activeSection == section.name {
			classes = "px-4 py-2 text-sm font-medium text-blue-600 border-b-2 border-blue-600"
		} else {
			classes = "px-4 py-2 text-sm font-medium text-gray-500 hover:text-gray-700 hover:border-gray-300 border-b-2 border-transparent"
		}

		tabNodes = append(tabNodes,
			A(
				Href(section.href),
				Class(classes),
				hx.Get(section.href),
				hx.Target("#admin-section"),
				hx.Swap("outerHTML"),
				g.Text(section.label),
			),
		)
	}

	return Div(
		Class("border-b border-gray-200 mb-6"),
		Nav(
			Class("flex space-x-8"),
			g.Group(tabNodes),
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
			buttonDanger("Clear Cache",
				withClass("px-4 py-2"),
				withAttributes(
					hx.Post(clearEndpoint),
					hx.Target("#admin-section-content"),
					hx.Swap("innerHTML"),
				),
			),
			button("Refresh Stats",
				withClass("px-4 py-2"),
				withAttributes(
					hx.Get(refreshEndpoint),
					hx.Target("#admin-section-content"),
					hx.Swap("innerHTML"),
				),
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
		Class("space-y-4"),
		Div(Class("text-lg font-medium text-gray-900"), g.Text("B2 Cache Management")),
		Div(Class("text-gray-600 text-sm"), g.Text("Manage B2 file storage cache performance and statistics.")),
		CacheStatsPanel("Cache Statistics", stats, "/api/admin/b2-cache/clear", "/api/admin/b2-cache/refresh"),
	)
}

func AdminEmbeddingCacheSection(stats map[string]interface{}) g.Node {
	// Get stats for each cache type
	queryStats := getCacheStats(stats, "query", "Query Embedding Cache")
	userStats := getCacheStats(stats, "user", "User Embedding Cache")
	siteStats := getCacheStats(stats, "site", "Site Embedding Cache")

	return Div(
		Class("space-y-4"),
		Div(Class("text-lg font-medium text-gray-900"), g.Text("Embedding Cache Management")),
		Div(Class("text-gray-600 text-sm"), g.Text("Manage AI embedding cache performance and statistics.")),
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
		Class("space-y-4"),
		Div(Class("text-lg font-medium text-gray-900"), g.Text("Vehicle Cache Management")),
		Div(Class("text-gray-600 text-sm"), g.Text("Manage vehicle data cache performance and statistics.")),
		CacheStatsPanel("Cache Statistics", stats, "/api/admin/vehicle-cache/clear", "/api/admin/vehicle-cache/refresh"),
	)
}

func AdminPartCacheSection(stats map[string]interface{}) g.Node {
	return Div(
		Class("space-y-4"),
		Div(Class("text-lg font-medium text-gray-900"), g.Text("Part Cache Management")),
		Div(Class("text-gray-600 text-sm"), g.Text("Manage part data cache performance and statistics.")),
		CacheStatsPanel("Cache Statistics", stats, "/api/admin/part-cache/clear", "/api/admin/part-cache/refresh"),
	)
}
