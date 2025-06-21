package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/user"
)

func HandleAdminDashboard(c *fiber.Ctx) error {
	currentUser := c.Locals("user").(*user.User)
	return render(c, ui.AdminDashboard(currentUser, c.Path()))
}

func HandleAdminUsers(c *fiber.Ctx) error {
	currentUser := c.Locals("user").(*user.User)
	users, err := user.GetAllUsers()
	if err != nil {
		return fiber.ErrInternalServerError
	}
	return render(c, ui.AdminUsers(currentUser, c.Path(), users))
}

func HandleSetAdmin(c *fiber.Ctx) error {
	userID, err := strconv.Atoi(c.FormValue("user_id"))
	if err != nil {
		return fiber.ErrBadRequest
	}
	isAdmin := c.FormValue("is_admin") == "true"

	if err := user.SetAdmin(userID, isAdmin); err != nil {
		return fiber.ErrInternalServerError
	}

	users, err := user.GetAllUsers()
	if err != nil {
		return fiber.ErrInternalServerError
	}

	return render(c, ui.AdminUserTable(users))
}

func HandleAdminAds(c *fiber.Ctx) error {
	return c.SendString("Admin Ads")
}

func HandleAdminTransactions(c *fiber.Ctx) error {
	return c.SendString("Admin Transactions")
}

func HandleAdminExport(c *fiber.Ctx) error {
	return c.SendString("Admin Export")
}
