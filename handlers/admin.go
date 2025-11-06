package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/b2util"
	"github.com/parts-pile/site/local"
	"github.com/parts-pile/site/metrics"
	"github.com/parts-pile/site/part"
	"github.com/parts-pile/site/spec"
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/vector"
	g "maragu.dev/gomponents"
)

// renderAdminSection renders an admin section with HX-Request handling
func renderAdminSection(c *fiber.Ctx, sectionID string, sectionContent g.Node) error {
	userID := local.GetUserID(c)
	userName := local.GetUserName(c)
	path := c.Path()
	if c.Get("HX-Request") != "" {
		return render(c, ui.AdminSectionPage(userID, userName, path, sectionID, sectionContent))
	}
	csrfToken := local.GetCSRFToken(c)
	return render(c, ui.Page(
		"Admin Dashboard",
		userID,
		userName,
		c.Path(),
		csrfToken,
		[]g.Node{ui.AdminSectionPage(userID, userName, path, sectionID, sectionContent)},
	))
}

func HandleAdminDashboard(c *fiber.Ctx) error {
	stats := b2util.GetCacheStats()
	return renderAdminSection(c, "b2-cache", ui.AdminB2CacheSection(stats))
}

// B2 Cache

func HandleAdminB2Cache(c *fiber.Ctx) error {
	stats := b2util.GetCacheStats()
	return renderAdminSection(c, "b2-cache", ui.AdminB2CacheSection(stats))
}

func HandleClearB2Cache(c *fiber.Ctx) error {
	stats := b2util.ClearCache()
	return render(c, ui.AdminB2CacheSection(stats))
}

func HandleRefreshB2Cache(c *fiber.Ctx) error {
	stats := b2util.GetCacheStats()
	return render(c, ui.AdminB2CacheSection(stats))
}

// Embedding Cache

func HandleAdminEmbeddingCache(c *fiber.Ctx) error {
	stats := vector.GetEmbeddingCacheStats()
	return renderAdminSection(c, "embedding-cache", ui.AdminEmbeddingCacheSection(stats))
}

func HandleRefreshEmbeddingCache(c *fiber.Ctx) error {
	stats := vector.GetEmbeddingCacheStats()
	return render(c, ui.AdminEmbeddingCacheSection(stats))
}

func HandleClearQueryEmbeddingCache(c *fiber.Ctx) error {
	vector.ClearQueryEmbeddingCache()
	stats := vector.GetEmbeddingCacheStats()
	return render(c, ui.AdminEmbeddingCacheSection(stats))
}

func HandleClearUserEmbeddingCache(c *fiber.Ctx) error {
	vector.ClearUserEmbeddingCache()
	stats := vector.GetEmbeddingCacheStats()
	return render(c, ui.AdminEmbeddingCacheSection(stats))
}

func HandleClearSiteEmbeddingCache(c *fiber.Ctx) error {
	vector.ClearSiteEmbeddingCache()
	stats := vector.GetEmbeddingCacheStats()
	return render(c, ui.AdminEmbeddingCacheSection(stats))
}

// Spec Cache

func HandleAdminSpecCache(c *fiber.Ctx) error {
	stats := spec.GetCacheStats()
	return renderAdminSection(c, "spec-cache", ui.AdminSpecCacheSection(stats))
}

func HandleClearSpecCache(c *fiber.Ctx) error {
	stats := spec.ClearCache()
	return render(c, ui.AdminSpecCacheSection(stats))
}

func HandleRefreshSpecCache(c *fiber.Ctx) error {
	stats := spec.GetCacheStats()
	return render(c, ui.AdminSpecCacheSection(stats))
}

// Part Cache

func HandleAdminPartCache(c *fiber.Ctx) error {
	stats := part.GetPartCacheStats()
	return renderAdminSection(c, "part-cache", ui.AdminPartCacheSection(stats))
}

func HandleClearPartCache(c *fiber.Ctx) error {
	stats := part.ClearPartCache()
	return render(c, ui.AdminPartCacheSection(stats))
}

func HandleRefreshPartCache(c *fiber.Ctx) error {
	stats := part.GetPartCacheStats()
	return render(c, ui.AdminPartCacheSection(stats))
}

// Monitor

// HandleMonitor displays the server metrics dashboard
func HandleMonitor(c *fiber.Ctx) error {
	// Get metrics from last 24 hours for initial display
	metricsData, err := metrics.GetRecentMetrics(24)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).
			SendString("Error fetching metrics: " + err.Error())
	}

	return renderAdminSection(c, "monitor", ui.AdminMonitorSection(metricsData))
}

// HandleMonitorRefresh returns updated metrics data for HTMX polling
func HandleMonitorRefresh(c *fiber.Ctx) error {
	// Get metrics from last 24 hours
	metricsData, err := metrics.GetRecentMetrics(24)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).
			SendString("Error fetching metrics: " + err.Error())
	}

	return render(c, ui.AdminMonitorCharts(metricsData))
}
