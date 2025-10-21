package handlers

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/b2util"
	"github.com/parts-pile/site/part"
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/vector"
	"github.com/parts-pile/site/vehicle"
	g "maragu.dev/gomponents"
)

func HandleAdminDashboard(c *fiber.Ctx) error {
	currentUser, _ := CurrentUser(c)

	// Default to b2-cache section
	if c.Get("HX-Request") != "" {
		return render(c, ui.AdminSectionPage(currentUser, c.Path(), "b2-cache", ui.AdminB2CacheSection(b2util.GetCacheStats())))
	}
	return render(c, ui.Page(
		"Admin Dashboard",
		currentUser,
		c.Path(),
		[]g.Node{ui.AdminSectionPage(currentUser, c.Path(), "b2-cache", ui.AdminB2CacheSection(b2util.GetCacheStats()))},
	))
}

func HandleAdminB2Cache(c *fiber.Ctx) error {
	currentUser, _ := CurrentUser(c)

	stats := b2util.GetCacheStats()

	if c.Get("HX-Request") != "" {
		return render(c, ui.AdminSectionPage(currentUser, c.Path(), "b2-cache", ui.AdminB2CacheSection(stats)))
	}
	return render(c, ui.Page(
		"Admin Dashboard",
		currentUser,
		c.Path(),
		[]g.Node{ui.AdminSectionPage(currentUser, c.Path(), "b2-cache", ui.AdminB2CacheSection(stats))},
	))
}

func HandleClearB2Cache(c *fiber.Ctx) error {
	b2util.ClearCache()

	stats := b2util.GetCacheStats()
	return render(c, ui.AdminB2CacheSection(stats))
}

func HandleRefreshB2Cache(c *fiber.Ctx) error {
	stats := b2util.GetCacheStats()
	return render(c, ui.AdminB2CacheSection(stats))
}

func HandleRefreshB2Token(c *fiber.Ctx) error {
	prefix := c.FormValue("prefix")
	if prefix == "" {
		return fiber.NewError(fiber.StatusBadRequest, "prefix parameter is required")
	}

	_, err := b2util.ForceRefreshToken(prefix)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("Failed to refresh token: %v", err))
	}

	stats := b2util.GetCacheStats()
	return render(c, ui.AdminB2CacheSection(stats))
}

func HandleAdminEmbeddingCache(c *fiber.Ctx) error {
	currentUser, _ := CurrentUser(c)

	stats := vector.GetEmbeddingCacheStats()

	if c.Get("HX-Request") != "" {
		return render(c, ui.AdminSectionPage(currentUser, c.Path(), "embedding-cache", ui.AdminEmbeddingCacheSection(stats)))
	}
	return render(c, ui.Page(
		"Admin Dashboard",
		currentUser,
		c.Path(),
		[]g.Node{ui.AdminSectionPage(currentUser, c.Path(), "embedding-cache", ui.AdminEmbeddingCacheSection(stats))},
	))
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

func HandleAdminVehicleCache(c *fiber.Ctx) error {
	currentUser, _ := CurrentUser(c)

	stats := vehicle.GetVehicleCacheStats()

	if c.Get("HX-Request") != "" {
		return render(c, ui.AdminSectionPage(currentUser, c.Path(), "vehicle-cache", ui.AdminVehicleCacheSection(stats)))
	}
	return render(c, ui.Page(
		"Admin Dashboard",
		currentUser,
		c.Path(),
		[]g.Node{ui.AdminSectionPage(currentUser, c.Path(), "vehicle-cache", ui.AdminVehicleCacheSection(stats))},
	))
}

func HandleClearVehicleCache(c *fiber.Ctx) error {
	vehicle.ClearVehicleCache()

	stats := vehicle.GetVehicleCacheStats()
	return render(c, ui.AdminVehicleCacheSection(stats))
}

func HandleRefreshVehicleCache(c *fiber.Ctx) error {
	stats := vehicle.GetVehicleCacheStats()
	return render(c, ui.AdminVehicleCacheSection(stats))
}

func HandleAdminPartCache(c *fiber.Ctx) error {
	currentUser, _ := CurrentUser(c)

	stats := part.GetPartCacheStats()

	if c.Get("HX-Request") != "" {
		return render(c, ui.AdminSectionPage(currentUser, c.Path(), "part-cache", ui.AdminPartCacheSection(stats)))
	}
	return render(c, ui.Page(
		"Admin Dashboard",
		currentUser,
		c.Path(),
		[]g.Node{ui.AdminSectionPage(currentUser, c.Path(), "part-cache", ui.AdminPartCacheSection(stats))},
	))
}

func HandleClearPartCache(c *fiber.Ctx) error {
	part.ClearPartCache()

	stats := part.GetPartCacheStats()
	return render(c, ui.AdminPartCacheSection(stats))
}

func HandleRefreshPartCache(c *fiber.Ctx) error {
	stats := part.GetPartCacheStats()
	return render(c, ui.AdminPartCacheSection(stats))
}
