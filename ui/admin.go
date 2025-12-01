package ui

import (
	"fmt"
	"time"

	"github.com/parts-pile/site/ad"
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
		{"embedding-cache", "Embedding Cache", "/admin/embedding-cache"},
		{"monitor", "Monitor", "/admin/metrics"},
		{"clicks", "Clicks", "/admin/clicks"},
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

// Clicks

func AdminClicksSection(adStats []ad.AdClickStats, imageStats []ad.ImageClickStats) g.Node {
	return Div(
		Class("space-y-6"),
		Div(
			Class("flex justify-between items-center"),
			Div(
				Div(Class("text-lg font-medium text-gray-900"), g.Text("Click Statistics")),
				Div(Class("text-gray-600 text-sm"), g.Text("Ad and image click statistics (updates every 10 seconds).")),
			),
		),
		Div(
			ID("clicks-content"),
			hx.Get("/api/admin/clicks/refresh"),
			hx.Trigger("every 10s"),
			hx.Swap("innerHTML"),
			AdminClicksContent(adStats, imageStats),
		),
	)
}

func AdminClicksContent(adStats []ad.AdClickStats, imageStats []ad.ImageClickStats) g.Node {
	return Div(
		Class("space-y-6"),
		adminAdClickStats(adStats),
		adminImageClickStats(imageStats),
	)
}

func adminAdClickStats(stats []ad.AdClickStats) g.Node {
	if len(stats) == 0 {
		return Div(
			Class("bg-white p-4 rounded-lg border"),
			H3(Class("text-md font-semibold mb-2"), g.Text("Top Ads by Clicks")),
			Div(Class("text-gray-500"), g.Text("No ad click data available yet.")),
		)
	}

	var rows []g.Node
	for i, s := range stats {
		lastClicked := "never"
		if s.LastClickedAt != nil {
			lastClicked = formatAdAge(*s.LastClickedAt)
		}

		rows = append(rows, Tr(
			Class(func() string {
				if i%2 == 0 {
					return "bg-gray-50"
				}
				return "bg-white"
			}()),
			Td(
				Class("px-4 py-2 text-sm"),
				A(
					Href(fmt.Sprintf("/ad/%d", s.AdID)),
					Class("text-blue-600 hover:text-blue-800 hover:underline"),
					g.Textf("%d", s.AdID),
				),
			),
			Td(Class("px-4 py-2 text-sm"), g.Text(s.Title)),
			Td(Class("px-4 py-2 text-sm text-right"), g.Textf("%d", s.ClickCount)),
			Td(Class("px-4 py-2 text-sm text-right"), g.Textf("%d", s.ImageClickCount)),
			Td(Class("px-4 py-2 text-sm text-gray-600"), g.Text(lastClicked)),
		))
	}

	return Div(
		Class("bg-white p-4 rounded-lg border"),
		H3(Class("text-md font-semibold mb-4"), g.Text("Top Ads by Clicks")),
		Div(
			Class("overflow-x-auto"),
			Table(
				Class("min-w-full divide-y divide-gray-200"),
				THead(
					Class("bg-gray-100"),
					Tr(
						Th(Class("px-4 py-2 text-left text-xs font-medium text-gray-700 uppercase"), g.Text("Ad ID")),
						Th(Class("px-4 py-2 text-left text-xs font-medium text-gray-700 uppercase"), g.Text("Title")),
						Th(Class("px-4 py-2 text-right text-xs font-medium text-gray-700 uppercase"), g.Text("Ad Clicks")),
						Th(Class("px-4 py-2 text-right text-xs font-medium text-gray-700 uppercase"), g.Text("Image Clicks")),
						Th(Class("px-4 py-2 text-left text-xs font-medium text-gray-700 uppercase"), g.Text("Last Clicked")),
					),
				),
				TBody(
					Class("divide-y divide-gray-200"),
					g.Group(rows),
				),
			),
		),
	)
}

func adminImageClickStats(stats []ad.ImageClickStats) g.Node {
	if len(stats) == 0 {
		return Div(
			Class("bg-white p-4 rounded-lg border"),
			H3(Class("text-md font-semibold mb-2"), g.Text("Top Images by Clicks")),
			Div(Class("text-gray-500"), g.Text("No image click data available yet.")),
		)
	}

	var rows []g.Node
	for i, s := range stats {
		lastClicked := "never"
		if s.LastClickedAt != nil {
			lastClicked = formatAdAge(*s.LastClickedAt)
		}

		rows = append(rows, Tr(
			Class(func() string {
				if i%2 == 0 {
					return "bg-gray-50"
				}
				return "bg-white"
			}()),
			Td(
				Class("px-4 py-2 text-sm"),
				A(
					Href(fmt.Sprintf("/ad/%d", s.AdID)),
					Class("text-blue-600 hover:text-blue-800 hover:underline"),
					g.Textf("%d", s.AdID),
				),
			),
			Td(Class("px-4 py-2 text-sm"), g.Text(s.Title)),
			Td(
				Class("px-4 py-2 text-sm"),
				A(
					Href(fmt.Sprintf("/image/%d/%d/1200w", s.AdID, s.ImageIdx)),
					Class("text-blue-600 hover:text-blue-800 hover:underline"),
					g.Textf("%d", s.ImageIdx),
				),
			),
			Td(Class("px-4 py-2 text-sm text-right"), g.Textf("%d", s.ClickCount)),
			Td(Class("px-4 py-2 text-sm text-gray-600"), g.Text(lastClicked)),
		))
	}

	return Div(
		Class("bg-white p-4 rounded-lg border"),
		H3(Class("text-md font-semibold mb-4"), g.Text("Top Images by Clicks")),
		Div(
			Class("overflow-x-auto"),
			Table(
				Class("min-w-full divide-y divide-gray-200"),
				THead(
					Class("bg-gray-100"),
					Tr(
						Th(Class("px-4 py-2 text-left text-xs font-medium text-gray-700 uppercase"), g.Text("Ad ID")),
						Th(Class("px-4 py-2 text-left text-xs font-medium text-gray-700 uppercase"), g.Text("Ad Title")),
						Th(Class("px-4 py-2 text-left text-xs font-medium text-gray-700 uppercase"), g.Text("Image Index")),
						Th(Class("px-4 py-2 text-right text-xs font-medium text-gray-700 uppercase"), g.Text("Clicks")),
						Th(Class("px-4 py-2 text-left text-xs font-medium text-gray-700 uppercase"), g.Text("Last Clicked")),
					),
				),
				TBody(
					Class("divide-y divide-gray-200"),
					g.Group(rows),
				),
			),
		),
	)
}
