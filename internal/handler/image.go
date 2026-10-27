package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/partspile/site/internal/param"
	"github.com/partspile/site/internal/ui"
)

func ImageNavigationHandler(c *fiber.Ctx) error {
	adID, err := param.GetAdID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	index, err := strconv.Atoi(c.Query("index", "1"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid index")
	}

	count, err := strconv.Atoi(c.Query("count", "1"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid count")
	}

	size := c.Query("size", "480w")
	heightClass := c.Query("heightClass", "aspect-[4/3]")
	clickable := c.Query("clickable", "false") == "true"

	if size == "1200w" && clickable {
		return render(c, ui.ImageNodeWithThumbnails(adID, count, index, size,
			heightClass, clickable))
	}

	return render(c, ui.ImageNode(adID, count, index, size, heightClass,
		clickable))
}

func ImageFullScreenHandler(c *fiber.Ctx) error {
	adID, err := param.GetAdID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	index, err := strconv.Atoi(c.Query("index", "1"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid index")
	}

	count, err := strconv.Atoi(c.Query("count", "1"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid count")
	}

	size := c.Query("size", "1200w")

	if c.Get("HX-Request") != "" && c.Query("update") == "true" {
		return render(c, ui.ImageFullScreenUpdate(adID, index, count, size))
	}

	return render(c, ui.ImageFullScreen(adID, index, count, size))
}
