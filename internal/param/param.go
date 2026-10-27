package param

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/partspile/site/internal/ad"
)

func GetCategoryID(c *fiber.Ctx) int {
	category := c.Params("category")
	return ad.ParseCategory(category)
}

func GetAdID(c *fiber.Ctx) (int, error) {
	idStr := c.Params("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return 0, err
	}
	return id, nil
}
