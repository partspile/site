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

// Update AdminDashboard to default to b2-cache section
func AdminDashboard(currentUser *user.User, path string) g.Node {
	return Page(
		"Admin Dashboard",
		currentUser,
		path,
		[]g.Node{
			AdminSectionPage(currentUser, path, "b2-cache", nil),
		},
	)
}

func AdminB2CacheSection(stats map[string]interface{}) g.Node {
	return Div(
		H1(g.Text("B2 Cache Management")),
		Div(
			Class("bg-gray-100 p-4 rounded-lg mb-4"),
			H2(Class("text-lg font-semibold mb-2"), g.Text("Cache Statistics")),
			Div(
				Class("grid grid-cols-2 md:grid-cols-4 gap-4"),
				Div(
					Class("bg-white p-3 rounded border"),
					Strong(g.Text("Hits: ")),
					g.Textf("%d", stats["hits"]),
				),
				Div(
					Class("bg-white p-3 rounded border"),
					Strong(g.Text("Misses: ")),
					g.Textf("%d", stats["misses"]),
				),
				Div(
					Class("bg-white p-3 rounded border"),
					Strong(g.Text("Hit Rate: ")),
					g.Textf("%.1f%%", stats["hit_rate"]),
				),
				Div(
					Class("bg-white p-3 rounded border"),
					Strong(g.Text("Sets: ")),
					g.Textf("%d", stats["sets"]),
				),
			),
		),
		Div(
			Class("bg-gray-100 p-4 rounded-lg mb-4"),
			H2(Class("text-lg font-semibold mb-2"), g.Text("Memory Usage")),
			Div(
				Class("grid grid-cols-2 md:grid-cols-4 gap-4"),
				Div(
					Class("bg-white p-3 rounded border"),
					Strong(g.Text("Memory Used: ")),
					g.Textf("%.2f MB", stats["memory_used_mb"]),
				),
				Div(
					Class("bg-white p-3 rounded border"),
					Strong(g.Text("Total Added: ")),
					g.Textf("%.2f MB", stats["total_added_mb"]),
				),
				Div(
					Class("bg-white p-3 rounded border"),
					Strong(g.Text("Total Evicted: ")),
					g.Textf("%.2f MB", stats["total_evicted_mb"]),
				),
				Div(
					Class("bg-white p-3 rounded border"),
					Strong(g.Text("Memory Used (bytes): ")),
					g.Textf("%d", stats["memory_used"]),
				),
			),
		),
		Div(
			Class("bg-gray-100 p-4 rounded-lg mb-4"),
			H2(Class("text-lg font-semibold mb-2"), g.Text("Cache Metrics")),
			Div(
				Class("grid grid-cols-2 md:grid-cols-4 gap-4"),
				Div(
					Class("bg-white p-3 rounded border"),
					Strong(g.Text("Cost Added: ")),
					g.Textf("%d", stats["cost_added"]),
				),
				Div(
					Class("bg-white p-3 rounded border"),
					Strong(g.Text("Cost Evicted: ")),
					g.Textf("%d", stats["cost_evicted"]),
				),
				Div(
					Class("bg-white p-3 rounded border"),
					Strong(g.Text("Gets Dropped: ")),
					g.Textf("%d", stats["gets_dropped"]),
				),
				Div(
					Class("bg-white p-3 rounded border"),
					Strong(g.Text("Gets Kept: ")),
					g.Textf("%d", stats["gets_kept"]),
				),
				Div(
					Class("bg-white p-3 rounded border"),
					Strong(g.Text("Sets Dropped: ")),
					g.Textf("%d", stats["sets_dropped"]),
				),
				Div(
					Class("bg-white p-3 rounded border"),
					Strong(g.Text("Sets Rejected: ")),
					g.Textf("%d", stats["sets_rejected"]),
				),
			),
		),
		Div(
			Class("bg-gray-100 p-4 rounded-lg mb-4"),
			H2(Class("text-lg font-semibold mb-2"), g.Text("TTL Statistics")),
			Div(
				Class("grid grid-cols-2 md:grid-cols-4 gap-4"),
				Div(
					Class("bg-white p-3 rounded border"),
					Strong(g.Text("B2 Token TTL: ")),
					g.Textf("%s", stats["b2_token_expiry_formatted"]),
				),
				Div(
					Class("bg-white p-3 rounded border"),
					Strong(g.Text("Cache TTL: ")),
					g.Textf("%s", stats["b2_cache_ttl_formatted"]),
				),
				Div(
					Class("bg-white p-3 rounded border"),
					Strong(g.Text("Life Expectancy Count: ")),
					g.Textf("%d", stats["life_expectancy_count"]),
				),
				Div(
					Class("bg-white p-3 rounded border"),
					Strong(g.Text("Life Expectancy Mean: ")),
					g.Textf("%.1fs", stats["life_expectancy_mean"]),
				),
				Div(
					Class("bg-white p-3 rounded border"),
					Strong(g.Text("Current Items: ")),
					g.Textf("%d", stats["current_items"]),
				),
			),
			Div(
				Class("grid grid-cols-3 gap-4 mt-4"),
				Div(
					Class("bg-white p-3 rounded border"),
					Strong(g.Text("Life Expectancy P50: ")),
					g.Textf("%.1fs", stats["life_expectancy_p50"]),
				),
				Div(
					Class("bg-white p-3 rounded border"),
					Strong(g.Text("Life Expectancy P95: ")),
					g.Textf("%.1fs", stats["life_expectancy_p95"]),
				),
				Div(
					Class("bg-white p-3 rounded border"),
					Strong(g.Text("Life Expectancy P99: ")),
					g.Textf("%.1fs", stats["life_expectancy_p99"]),
				),
			),
		),
		Div(
			Class("mb-4"),
			H2(Class("text-lg font-semibold mb-2"), g.Text("Cache Information")),
			Div(
				Class("bg-white border border-gray-300 rounded-lg p-4"),
				P(Class("text-gray-600"),
					g.Text("The cache doesn't expose individual items for security reasons. "),
					g.Text("The cache automatically manages memory usage and eviction based on cost."),
				),
			),
		),
		Div(
			Class("mb-4"),
			H2(Class("text-lg font-semibold mb-2"), g.Text("Cache Actions")),
			Div(
				Class("bg-white border border-gray-300 rounded-lg p-4"),
				Div(
					Class("flex gap-4 mb-4"),
					Button(
						Class("px-4 py-2 bg-red-500 text-white rounded hover:bg-red-600"),
						hx.Post("/api/admin/b2-cache/clear"),
						hx.Target("#admin-section-content"),
						hx.Swap("innerHTML"),
						g.Text("Clear All Cache"),
					),
				),
				Div(
					Class("border-t pt-4"),
					H3(Class("text-md font-semibold mb-2"), g.Text("Refresh Specific Token")),
					P(Class("text-gray-600 mb-3"),
						g.Text("Enter an ad directory prefix (e.g., '22/') to refresh its B2 download token:"),
					),
					Form(
						Class("flex gap-2"),
						hx.Post("/api/admin/b2-cache/refresh"),
						hx.Target("#admin-section-content"),
						hx.Swap("innerHTML"),
						Input(
							Type("text"),
							Name("prefix"),
							Placeholder("e.g., 22/"),
							Class("flex-1 px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"),
							Required(),
						),
						Button(
							Type("submit"),
							Class("px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600"),
							g.Text("Refresh Token"),
						),
					),
				),
			),
		),
	)
}

func AdminEmbeddingCacheSection(stats map[string]interface{}) g.Node {
	return Div(
		H1(g.Text("Embedding Cache Management")),
		Div(
			Class("bg-gray-100 p-4 rounded-lg mb-4"),
			H2(Class("text-lg font-semibold mb-2"), g.Text("Cache Statistics")),
			Div(
				Class("grid grid-cols-2 md:grid-cols-4 gap-4"),
				Div(
					Class("bg-white p-3 rounded border"),
					Strong(g.Text("Hits: ")),
					g.Textf("%d", stats["hits"]),
				),
				Div(
					Class("bg-white p-3 rounded border"),
					Strong(g.Text("Misses: ")),
					g.Textf("%d", stats["misses"]),
				),
				Div(
					Class("bg-white p-3 rounded border"),
					Strong(g.Text("Hit Rate: ")),
					g.Textf("%.1f%%", stats["hit_rate"]),
				),
				Div(
					Class("bg-white p-3 rounded border"),
					Strong(g.Text("Sets: ")),
					g.Textf("%d", stats["sets"]),
				),
			),
		),
		Div(
			Class("bg-gray-100 p-4 rounded-lg mb-4"),
			H2(Class("text-lg font-semibold mb-2"), g.Text("Memory Usage")),
			Div(
				Class("grid grid-cols-2 md:grid-cols-4 gap-4"),
				Div(
					Class("bg-white p-3 rounded border"),
					Strong(g.Text("Memory Used: ")),
					g.Textf("%.2f MB", stats["memory_used_mb"]),
				),
				Div(
					Class("bg-white p-3 rounded border"),
					Strong(g.Text("Total Added: ")),
					g.Textf("%.2f MB", stats["total_added_mb"]),
				),
				Div(
					Class("bg-white p-3 rounded border"),
					Strong(g.Text("Total Evicted: ")),
					g.Textf("%.2f MB", stats["total_evicted_mb"]),
				),
				Div(
					Class("bg-white p-3 rounded border"),
					Strong(g.Text("Memory Used (bytes): ")),
					g.Textf("%d", stats["memory_used"]),
				),
			),
		),
		Div(
			Class("bg-gray-100 p-4 rounded-lg mb-4"),
			H2(Class("text-lg font-semibold mb-2"), g.Text("Cache Metrics")),
			Div(
				Class("grid grid-cols-2 md:grid-cols-4 gap-4"),
				Div(
					Class("bg-white p-3 rounded border"),
					Strong(g.Text("Cost Added: ")),
					g.Textf("%d", stats["cost_added"]),
				),
				Div(
					Class("bg-white p-3 rounded border"),
					Strong(g.Text("Cost Evicted: ")),
					g.Textf("%d", stats["cost_evicted"]),
				),
				Div(
					Class("bg-white p-3 rounded border"),
					Strong(g.Text("Gets Dropped: ")),
					g.Textf("%d", stats["gets_dropped"]),
				),
				Div(
					Class("bg-white p-3 rounded border"),
					Strong(g.Text("Gets Kept: ")),
					g.Textf("%d", stats["gets_kept"]),
				),
				Div(
					Class("bg-white p-3 rounded border"),
					Strong(g.Text("Sets Dropped: ")),
					g.Textf("%d", stats["sets_dropped"]),
				),
				Div(
					Class("bg-white p-3 rounded border"),
					Strong(g.Text("Sets Rejected: ")),
					g.Textf("%d", stats["sets_rejected"]),
				),
			),
		),
		Div(
			Class("mb-4"),
			H2(Class("text-lg font-semibold mb-2"), g.Text("Cache Information")),
			Div(
				Class("bg-white border border-gray-300 rounded-lg p-4"),
				P(Class("text-gray-600"),
					g.Text("The cache doesn't expose individual items for security reasons. "),
					g.Text("The cache automatically manages memory usage and eviction based on cost. "),
					g.Text("This cache stores embedding vectors for user search queries to improve performance."),
				),
			),
		),
		Div(
			Class("flex gap-4"),
			Button(
				Class("px-4 py-2 bg-red-500 text-white rounded hover:bg-red-600"),
				hx.Post("/api/admin/embedding-cache/clear"),
				hx.Target("#admin-section-content"),
				hx.Swap("innerHTML"),
				g.Text("Clear Cache"),
			),
		),
	)
}
