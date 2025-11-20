package ui

import (
	"fmt"
	"time"

	"github.com/parts-pile/site/metrics"
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
		{"spec-cache", "Spec Cache", "/admin/spec-cache"},
		{"part-cache", "Part Cache", "/admin/part-cache"},
		{"field-values-cache", "Field Values Cache", "/admin/field-values-cache"},
		{"monitor", "Monitor", "/admin/metrics"},
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

func AdminSpecCacheSection(stats map[string]interface{}) g.Node {
	return Div(
		Class("space-y-4"),
		Div(Class("text-lg font-medium text-gray-900"), g.Text("Spec Cache Management")),
		Div(Class("text-gray-600 text-sm"), g.Text("Manage spec data cache performance and statistics.")),
		CacheStatsPanel("Cache Statistics", stats, "/api/admin/spec-cache/clear", "/api/admin/spec-cache/refresh"),
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

func AdminFieldValuesCacheSection(stats map[string]interface{}) g.Node {
	return Div(
		Class("space-y-4"),
		Div(Class("text-lg font-medium text-gray-900"), g.Text("Field Values Cache Management")),
		Div(Class("text-gray-600 text-sm"), g.Text("Manage field values cache performance and statistics.")),
		CacheStatsPanel("Cache Statistics", stats, "/api/admin/field-values-cache/clear", "/api/admin/field-values-cache/refresh"),
	)
}

// Monitor

func AdminMonitorSection(metricsData []metrics.Metric) g.Node {
	return Div(
		Class("space-y-6"),
		Div(
			Class("flex justify-between items-center"),
			Div(
				Div(Class("text-lg font-medium text-gray-900"), g.Text("Server Metrics")),
				Div(Class("text-gray-600 text-sm"), g.Text("Real-time server performance metrics (updates every 5 seconds).")),
			),
		),
		Div(
			ID("monitor-charts"),
			hx.Get("/api/admin/metrics/refresh"),
			hx.Trigger("every 5s"),
			hx.Swap("innerHTML"),
			AdminMonitorCharts(metricsData),
		),
	)
}

func AdminMonitorCharts(metricsData []metrics.Metric) g.Node {
	if len(metricsData) == 0 {
		return Div(
			Class("text-gray-500 text-center py-8"),
			g.Text("No metrics data available yet. Metrics are collected every 5 seconds."),
		)
	}

	return Div(
		Class("space-y-6"),
		renderMetricChart("CPU Usage (%)", metricsData, func(m metrics.Metric) float64 { return m.PIDCPU }, "blue"),
		renderMetricChart("Memory Usage (MB)", metricsData, func(m metrics.Metric) float64 { return float64(m.PIDRAM) / 1024 / 1024 }, "green"),
		renderMetricChart("Connections", metricsData, func(m metrics.Metric) float64 { return float64(m.PIDConns) }, "purple"),
	)
}

func renderMetricChart(title string, metricsData []metrics.Metric, valueFn func(metrics.Metric) float64, color string) g.Node {
	if len(metricsData) == 0 {
		return Div()
	}

	// Calculate chart dimensions
	const width = 800
	const height = 200
	const padding = 40
	const chartWidth = width - padding*2
	const chartHeight = height - padding*2

	// Find min/max values for scaling
	minVal := valueFn(metricsData[0])
	maxVal := minVal
	for _, m := range metricsData {
		val := valueFn(m)
		if val < minVal {
			minVal = val
		}
		if val > maxVal {
			maxVal = val
		}
	}

	// Add 10% padding to range
	rangeVal := maxVal - minVal
	if rangeVal == 0 {
		rangeVal = 1
	}
	minVal -= rangeVal * 0.1
	maxVal += rangeVal * 0.1
	rangeVal = maxVal - minVal

	// Generate path for line chart
	var pathData string
	divisor := len(metricsData) - 1
	if divisor == 0 {
		divisor = 1 // Handle single data point
	}
	for i, m := range metricsData {
		x := float64(i) * chartWidth / float64(divisor)
		val := valueFn(m)
		y := chartHeight - ((val-minVal)/rangeVal)*chartHeight
		if i == 0 {
			pathData += fmt.Sprintf("M %d %d", padding+int(x), padding+int(y))
		} else {
			pathData += fmt.Sprintf(" L %d %d", padding+int(x), padding+int(y))
		}
	}

	// Generate grid lines and labels as SVG strings
	var gridLinesHTML string
	var labelsHTML string

	// Y-axis labels (5 ticks)
	for i := 0; i <= 4; i++ {
		yVal := minVal + (rangeVal * float64(i) / 4)
		yPos := padding + chartHeight - (chartHeight * i / 4)
		gridLinesHTML += fmt.Sprintf(`<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="#e5e7eb" stroke-width="1" />`,
			padding, yPos, width-padding, yPos)
		labelsHTML += fmt.Sprintf(`<text x="%d" y="%d" text-anchor="end" font-size="12" fill="#6b7280">%.1f</text>`,
			padding-10, yPos+4, yVal)
	}

	// X-axis time labels (show first, middle, last)
	if len(metricsData) > 0 {
		divisor := len(metricsData) - 1
		if divisor == 0 {
			divisor = 1 // Handle single data point
		}

		var timeLabels []struct {
			idx int
			t   time.Time
		}

		// Always show first and last
		timeLabels = append(timeLabels, struct {
			idx int
			t   time.Time
		}{0, metricsData[0].RecordedAt})

		// Show middle only if we have more than 2 points
		if len(metricsData) > 2 {
			midIdx := len(metricsData) / 2
			timeLabels = append(timeLabels, struct {
				idx int
				t   time.Time
			}{midIdx, metricsData[midIdx].RecordedAt})
		}

		// Show last only if we have more than 1 point
		if len(metricsData) > 1 {
			timeLabels = append(timeLabels, struct {
				idx int
				t   time.Time
			}{len(metricsData) - 1, metricsData[len(metricsData)-1].RecordedAt})
		}

		for _, tl := range timeLabels {
			xPos := padding + (chartWidth * tl.idx / divisor)
			labelsHTML += fmt.Sprintf(`<text x="%d" y="%d" text-anchor="middle" font-size="12" fill="#6b7280">%s</text>`,
				xPos, height-padding+20, tl.t.Format("15:04"))
		}
	}

	// Get color hex
	colorMap := map[string]string{
		"blue":   "#3b82f6",
		"green":  "#10b981",
		"purple": "#8b5cf6",
		"orange": "#f59e0b",
		"red":    "#ef4444",
		"teal":   "#14b8a6",
	}
	strokeColor := colorMap[color]
	if strokeColor == "" {
		strokeColor = "#3b82f6"
	}

	// Get current value
	currentValue := valueFn(metricsData[len(metricsData)-1])

	return Div(
		Class("bg-white p-4 rounded-lg border"),
		H3(Class("text-md font-semibold mb-2"), g.Text(title)),
		Div(
			Class("text-2xl font-bold mb-2"),
			Style(fmt.Sprintf("color: %s", strokeColor)),
			g.Textf("%.2f", currentValue),
		),
		Div(
			Class("w-full overflow-x-auto"),
			g.Raw(fmt.Sprintf(`<svg width="%d" height="%d" viewBox="0 0 %d %d">
				%s
				<path d="%s" fill="none" stroke="%s" stroke-width="2" />
				%s
			</svg>`, width, height, width, height, gridLinesHTML, pathData, strokeColor, labelsHTML)),
		),
	)
}
