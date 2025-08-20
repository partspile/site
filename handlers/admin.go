package handlers

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/b2util"
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/vector"
	g "maragu.dev/gomponents"
)

func HandleAdminDashboard(c *fiber.Ctx) error {
	currentUser, err := CurrentUser(c)
	if err != nil {
		return err
	}

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
	currentUser, err := CurrentUser(c)
	if err != nil {
		return err
	}

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
	currentUser, err := CurrentUser(c)
	if err != nil {
		return err
	}

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

func HandleClearEmbeddingCache(c *fiber.Ctx) error {
	vector.ClearEmbeddingCache()

	stats := vector.GetEmbeddingCacheStats()
	return render(c, ui.AdminEmbeddingCacheSection(stats))
}
