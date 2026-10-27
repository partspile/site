package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/partspile/site/internal/cookie"
	"github.com/partspile/site/internal/ui"
)

func IntroBannerDismissHandler(c *fiber.Ctx) error {
	cookie.SetIntroBannerDismissed(c)

	redirect := c.Query("redirect")
	if redirect == "/about" {
		if c.Get("HX-Request") != "" {
			c.Set("HX-Redirect", redirect)
			return c.SendStatus(fiber.StatusOK)
		}
		return c.Redirect(redirect, fiber.StatusFound)
	}

	return render(c, ui.RemoveIntroBanner())
}
