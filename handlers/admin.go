package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/part"
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/user"
	"github.com/parts-pile/site/vehicle"
)

func HandleAdminDashboard(c *fiber.Ctx) error {
	currentUser := c.Locals("user").(*user.User)
	return render(c, ui.AdminDashboard(currentUser, c.Path()))
}

func HandleAdminUsers(c *fiber.Ctx) error {
	currentUser := c.Locals("user").(*user.User)

	status := c.Query("status")
	var users []user.User
	var err error

	if status == "dead" {
		users, err = user.GetAllDeadUsers()
	} else {
		users, err = user.GetAllUsers()
	}

	if err != nil {
		return fiber.ErrInternalServerError
	}
	return render(c, ui.AdminUsers(currentUser, c.Path(), users, status))
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

	return render(c, ui.AdminUserTable(users, "active"))
}

func HandleAdminAds(c *fiber.Ctx) error {
	currentUser := c.Locals("user").(*user.User)
	status := c.Query("status")
	var ads []ad.Ad
	var err error

	if status == "dead" {
		ads, err = ad.GetAllDeadAds()
	} else {
		ads, err = ad.GetAllAds()
	}

	if err != nil {
		return fiber.ErrInternalServerError
	}
	return render(c, ui.AdminAds(currentUser, c.Path(), ads, status))
}

func HandleAdminTransactions(c *fiber.Ctx) error {
	currentUser := c.Locals("user").(*user.User)
	transactions, err := user.GetAllTransactions()
	if err != nil {
		return fiber.ErrInternalServerError
	}
	return render(c, ui.AdminTransactions(currentUser, c.Path(), transactions))
}

func HandleAdminExportUsers(c *fiber.Ctx) error {
	status := c.Query("status")
	var users []user.User
	var err error

	if status == "dead" {
		users, err = user.GetAllDeadUsers()
	} else {
		users, err = user.GetAllUsers()
	}

	if err != nil {
		return fiber.ErrInternalServerError
	}

	filename := "users.json"
	if status == "dead" {
		filename = "dead_users.json"
	}
	c.Set("Content-Type", "application/json")
	c.Set("Content-Disposition", "attachment; filename="+filename)
	return c.JSON(users)
}

func HandleAdminExportAds(c *fiber.Ctx) error {
	status := c.Query("status")
	var ads []ad.Ad
	var err error

	if status == "dead" {
		ads, err = ad.GetAllDeadAds()
	} else {
		ads, err = ad.GetAllAds()
	}

	if err != nil {
		return fiber.ErrInternalServerError
	}

	filename := "ads.json"
	if status == "dead" {
		filename = "dead_ads.json"
	}
	c.Set("Content-Type", "application/json")
	c.Set("Content-Disposition", "attachment; filename="+filename)
	return c.JSON(ads)
}

func HandleAdminExportTransactions(c *fiber.Ctx) error {
	transactions, err := user.GetAllTransactions()
	if err != nil {
		return fiber.ErrInternalServerError
	}

	c.Set("Content-Type", "application/json")
	c.Set("Content-Disposition", "attachment; filename=transactions.json")
	return c.JSON(transactions)
}

func HandleAdminMakes(c *fiber.Ctx) error {
	currentUser := c.Locals("user").(*user.User)
	makes, err := vehicle.GetAllMakes()
	if err != nil {
		return fiber.ErrInternalServerError
	}
	return render(c, ui.AdminMakes(currentUser, c.Path(), makes))
}

func HandleAdminModels(c *fiber.Ctx) error {
	currentUser := c.Locals("user").(*user.User)
	models, err := vehicle.GetAllModelsWithID()
	if err != nil {
		return fiber.ErrInternalServerError
	}
	return render(c, ui.AdminModels(currentUser, c.Path(), models))
}

func HandleAdminYears(c *fiber.Ctx) error {
	currentUser := c.Locals("user").(*user.User)
	years, err := vehicle.GetAllYears()
	if err != nil {
		return fiber.ErrInternalServerError
	}
	return render(c, ui.AdminYears(currentUser, c.Path(), years))
}

func HandleAdminPartCategories(c *fiber.Ctx) error {
	currentUser := c.Locals("user").(*user.User)
	categories, err := part.GetAllCategories()
	if err != nil {
		return fiber.ErrInternalServerError
	}
	return render(c, ui.AdminPartCategories(currentUser, c.Path(), categories))
}

func HandleAdminPartSubCategories(c *fiber.Ctx) error {
	currentUser := c.Locals("user").(*user.User)
	subCategories, err := part.GetAllSubCategories()
	if err != nil {
		return fiber.ErrInternalServerError
	}
	return render(c, ui.AdminPartSubCategories(currentUser, c.Path(), subCategories))
}

func HandleKillUser(c *fiber.Ctx) error {
	userID, err := c.ParamsInt("id")
	if err != nil {
		return fiber.ErrBadRequest
	}

	if err := user.DeleteUser(userID); err != nil {
		return fiber.ErrInternalServerError
	}

	users, err := user.GetAllUsers()
	if err != nil {
		return fiber.ErrInternalServerError
	}

	return render(c, ui.AdminUserTable(users, "active"))
}

func HandleResurrectUser(c *fiber.Ctx) error {
	userID, err := c.ParamsInt("id")
	if err != nil {
		return fiber.ErrBadRequest
	}

	if err := user.ResurrectUser(userID); err != nil {
		return fiber.ErrInternalServerError
	}

	users, err := user.GetAllDeadUsers()
	if err != nil {
		return fiber.ErrInternalServerError
	}

	return render(c, ui.AdminUserTable(users, "dead"))
}

func HandleKillAd(c *fiber.Ctx) error {
	adID, err := c.ParamsInt("id")
	if err != nil {
		return fiber.ErrBadRequest
	}

	if err := ad.DeleteAd(adID); err != nil {
		return fiber.ErrInternalServerError
	}

	ads, err := ad.GetAllAds()
	if err != nil {
		return fiber.ErrInternalServerError
	}

	return render(c, ui.AdminAdTable(ads, "active"))
}

func HandleResurrectAd(c *fiber.Ctx) error {
	adID, err := c.ParamsInt("id")
	if err != nil {
		return fiber.ErrBadRequest
	}

	if err := ad.ResurrectAd(adID); err != nil {
		return fiber.ErrInternalServerError
	}

	ads, err := ad.GetAllDeadAds()
	if err != nil {
		return fiber.ErrInternalServerError
	}

	return render(c, ui.AdminAdTable(ads, "dead"))
}
