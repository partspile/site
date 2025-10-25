package handlers

import (
	"github.com/gofiber/fiber/v2"
)

func AdID(c *fiber.Ctx) (int, error) {
	return c.ParamsInt("id")
}
