package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/part"
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/user"
	"github.com/parts-pile/site/vehicle"
	g "maragu.dev/gomponents"
)

// adminHandler is a generic function that handles admin section pages
// T is the entity type (e.g., user.User, ad.Ad, etc.)
// getActiveData and getDeadData are functions that retrieve active and dead data respectively
// sectionComponent is a function that renders the UI section for the entity
// If getDeadData is nil, the entity doesn't support status filtering
func adminHandler[T any](c *fiber.Ctx, sectionName string,
	getActiveData func() ([]T, error),
	getDeadData func() ([]T, error), // can be nil for no-status entities
	sectionComponent func([]T, string) g.Node) error {
	currentUser := c.Locals("user").(*user.User)
	status := c.Query("status")

	var data []T
	var err error

	if getDeadData != nil {
		// Entity supports status filtering
		if status == "dead" {
			data, err = getDeadData()
		} else {
			data, err = getActiveData()
		}
	} else {
		// Entity doesn't support status filtering
		data, err = getActiveData()
	}

	if err != nil {
		return fiber.ErrInternalServerError
	}

	if c.Get("HX-Request") != "" {
		return render(c, ui.AdminSectionPage(currentUser, c.Path(), sectionName, sectionComponent(data, status)))
	}
	return render(c, ui.Page(
		"Admin Dashboard",
		currentUser,
		c.Path(),
		[]g.Node{ui.AdminSectionPage(currentUser, c.Path(), sectionName, sectionComponent(data, status))},
	))
}

// Wrapper functions for UI components that don't take a status parameter
func adminTransactionsSectionWrapper(transactions []user.Transaction, status string) g.Node {
	return ui.AdminTransactionsSection(transactions)
}

func adminMakesSectionWrapper(makes []vehicle.Make, status string) g.Node {
	return ui.AdminMakesSection(makes)
}

func adminModelsSectionWrapper(models []vehicle.Model, status string) g.Node {
	return ui.AdminModelsSection(models)
}

func adminYearsSectionWrapper(years []vehicle.Year, status string) g.Node {
	return ui.AdminYearsSection(years)
}

func adminPartCategoriesSectionWrapper(categories []part.Category, status string) g.Node {
	return ui.AdminPartCategoriesSection(categories)
}

func adminPartSubCategoriesSectionWrapper(subCategories []part.SubCategory, status string) g.Node {
	return ui.AdminPartSubCategoriesSection(subCategories)
}

func HandleAdminDashboard(c *fiber.Ctx) error {
	currentUser := c.Locals("user").(*user.User)
	users, err := user.GetAllUsers()
	if err != nil {
		return fiber.ErrInternalServerError
	}
	if c.Get("HX-Request") != "" {
		return render(c, ui.AdminSectionPage(currentUser, c.Path(), "users", ui.AdminUsersSection(users, "")))
	}
	return render(c, ui.Page(
		"Admin Dashboard",
		currentUser,
		c.Path(),
		[]g.Node{ui.AdminSectionPage(currentUser, c.Path(), "users", ui.AdminUsersSection(users, ""))},
	))
}

func HandleAdminUsers(c *fiber.Ctx) error {
	return adminHandler[user.User](c, "users", user.GetAllUsers, user.GetAllDeadUsers, ui.AdminUsersSection)
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
	return adminHandler[ad.Ad](c, "ads", ad.GetAllAds, ad.GetAllDeadAds, ui.AdminAdsSection)
}

func HandleAdminTransactions(c *fiber.Ctx) error {
	return adminHandler[user.Transaction](c, "transactions", user.GetAllTransactions, nil, adminTransactionsSectionWrapper)
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
	return adminHandler[vehicle.Make](c, "makes", vehicle.GetAllMakes, nil, adminMakesSectionWrapper)
}

func HandleAdminModels(c *fiber.Ctx) error {
	return adminHandler[vehicle.Model](c, "models", vehicle.GetAllModelsWithID, nil, adminModelsSectionWrapper)
}

func HandleAdminYears(c *fiber.Ctx) error {
	return adminHandler[vehicle.Year](c, "years", vehicle.GetAllYears, nil, adminYearsSectionWrapper)
}

func HandleAdminPartCategories(c *fiber.Ctx) error {
	return adminHandler[part.Category](c, "part-categories", part.GetAllCategories, nil, adminPartCategoriesSectionWrapper)
}

func HandleAdminPartSubCategories(c *fiber.Ctx) error {
	return adminHandler[part.SubCategory](c, "part-sub-categories", part.GetAllSubCategories, nil, adminPartSubCategoriesSectionWrapper)
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
