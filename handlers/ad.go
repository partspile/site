package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ad"
)

func AdCategory(c *fiber.Ctx) string {
	return c.Query("ad_category", ad.CarPart)
}

func AdID(c *fiber.Ctx) (int, error) {
	return c.ParamsInt("id")
}
