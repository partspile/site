package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/b2util"
	"github.com/parts-pile/site/part"
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/vector"
	"github.com/parts-pile/site/vehicle"
	g "maragu.dev/gomponents"
)

// renderAdminSection renders an admin section with HX-Request handling
func renderAdminSection(c *fiber.Ctx, sectionID string, sectionContent g.Node) error {
	currentUser, _ := CurrentUser(c)
	if c.Get("HX-Request") != "" {
		return render(c, ui.AdminSectionPage(currentUser, c.Path(), sectionID, sectionContent))
	}
	return render(c, ui.Page(
		"Admin Dashboard",
		currentUser,
		c.Path(),
		[]g.Node{ui.AdminSectionPage(currentUser, c.Path(), sectionID, sectionContent)},
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

// Vehicle Cache

func HandleAdminVehicleCache(c *fiber.Ctx) error {
	stats := vehicle.GetCacheStats()
	return renderAdminSection(c, "vehicle-cache", ui.AdminVehicleCacheSection(stats))
}

func HandleClearVehicleCache(c *fiber.Ctx) error {
	stats := vehicle.ClearCache()
	return render(c, ui.AdminVehicleCacheSection(stats))
}

func HandleRefreshVehicleCache(c *fiber.Ctx) error {
	stats := vehicle.GetCacheStats()
	return render(c, ui.AdminVehicleCacheSection(stats))
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
