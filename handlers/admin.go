package handlers

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/b2util"
	"github.com/parts-pile/site/db"
	"github.com/parts-pile/site/part"
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/user"
	"github.com/parts-pile/site/vector"
	"github.com/parts-pile/site/vehicle"
	g "maragu.dev/gomponents"
)

// adminHandler is a generic function that handles admin section pages
// T is the entity type (e.g., user.User, ad.Ad, etc.)
// getActiveData and getArchivedData are functions that retrieve active and archived data respectively
// sectionComponent is a function that renders the UI section for the entity
// If getArchivedData is nil, the entity doesn't support status filtering
func adminHandler[T any](c *fiber.Ctx, sectionName string,
	getActiveData func() ([]T, error),
	getArchivedData func() ([]T, error), // can be nil for no-status entities
	sectionComponent func([]T, string) g.Node) error {
	currentUser, err := CurrentUser(c)
	if err != nil {
		return err
	}
	status := c.Query("status")

	var data []T
	var err2 error

	if getArchivedData != nil {
		// Entity supports status filtering
		if status == "archived" {
			data, err2 = getArchivedData()
		} else {
			data, err2 = getActiveData()
		}
	} else {
		// Entity doesn't support status filtering
		data, err2 = getActiveData()
	}

	if err2 != nil {
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

func adminMakesSectionWrapper(makes []vehicle.MakeWithParentCompany, status string) g.Node {
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

func adminParentCompaniesSectionWrapper(pcs []vehicle.ParentCompany, status string) g.Node {
	return ui.AdminParentCompaniesSection(pcs)
}

func HandleAdminDashboard(c *fiber.Ctx) error {
	currentUser, err := CurrentUser(c)
	if err != nil {
		return err
	}
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
	return adminHandler(c, "users", user.GetAllUsers, user.GetAllArchivedUsers, ui.AdminUsersSection)
}

func HandleSetAdmin(c *fiber.Ctx) error {
	userID, err := ParseFormInt(c, "user_id")
	if err != nil {
		return err
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
	return adminHandler(c, "ads", ad.GetAllAds, ad.GetAllArchivedAds, ui.AdminAdsSection)
}

func HandleAdminTransactions(c *fiber.Ctx) error {
	return adminHandler(c, "transactions", user.GetAllTransactions, nil, adminTransactionsSectionWrapper)
}

// Generic export handler for entities with status (e.g., users, ads)
func exportWithStatus[T any](c *fiber.Ctx, getActive func() ([]T, error), getArchived func() ([]T, error), baseFilename string) error {
	status := c.Query("status")
	var data []T
	var err error
	filename := baseFilename + ".json"
	if status == "archived" {
		data, err = getArchived()
		filename = "archived_" + baseFilename + ".json"
	} else {
		data, err = getActive()
	}
	if err != nil {
		return fiber.ErrInternalServerError
	}
	c.Set("Content-Type", "application/json")
	c.Set("Content-Disposition", "attachment; filename="+filename)
	return c.JSON(data)
}

// Generic export handler for entities without status (e.g., transactions)
func exportSimple[T any](c *fiber.Ctx, getData func() ([]T, error), filename string) error {
	data, err := getData()
	if err != nil {
		return fiber.ErrInternalServerError
	}
	c.Set("Content-Type", "application/json")
	c.Set("Content-Disposition", "attachment; filename="+filename)
	return c.JSON(data)
}

func HandleAdminExportUsers(c *fiber.Ctx) error {
	return exportWithStatus(c, user.GetAllUsers, user.GetAllArchivedUsers, "users")
}

func HandleAdminExportAds(c *fiber.Ctx) error {
	return exportWithStatus(c, ad.GetAllAds, ad.GetAllArchivedAds, "ads")
}

func HandleAdminExportTransactions(c *fiber.Ctx) error {
	return exportSimple(c, user.GetAllTransactions, "transactions.json")
}

func HandleAdminMakes(c *fiber.Ctx) error {
	return adminHandler(c, "makes", vehicle.GetAllMakesWithParentCompany, nil, adminMakesSectionWrapper)
}

func HandleAdminModels(c *fiber.Ctx) error {
	return adminHandler(c, "models", vehicle.GetAllModelsWithID, nil, adminModelsSectionWrapper)
}

func HandleAdminYears(c *fiber.Ctx) error {
	return adminHandler(c, "years", vehicle.GetAllYears, nil, adminYearsSectionWrapper)
}

func HandleAdminPartCategories(c *fiber.Ctx) error {
	return adminHandler(c, "part-categories", part.GetAllCategories, nil, adminPartCategoriesSectionWrapper)
}

func HandleAdminPartSubCategories(c *fiber.Ctx) error {
	return adminHandler(c, "part-sub-categories", part.GetAllSubCategories, nil, adminPartSubCategoriesSectionWrapper)
}

// Archive/Restore handlers
func HandleArchiveUser(c *fiber.Ctx) error {
	userID, err := ParseIntParam(c, "id")
	if err != nil {
		return err
	}

	if err := user.ArchiveUser(userID); err != nil {
		return fiber.ErrInternalServerError
	}

	users, err := user.GetAllUsers()
	if err != nil {
		return fiber.ErrInternalServerError
	}

	return render(c, ui.AdminUserTable(users, "active"))
}

func HandleRestoreUser(c *fiber.Ctx) error {
	userID, err := ParseIntParam(c, "id")
	if err != nil {
		return err
	}

	if err := user.RestoreUser(userID); err != nil {
		return fiber.ErrInternalServerError
	}

	users, err := user.GetAllArchivedUsers()
	if err != nil {
		return fiber.ErrInternalServerError
	}

	return render(c, ui.AdminUserTable(users, "archived"))
}

func HandleRestoreAd(c *fiber.Ctx) error {
	adID, err := ParseIntParam(c, "id")
	if err != nil {
		return err
	}

	if err := ad.RestoreAd(adID); err != nil {
		log.Printf("Error restoring ad %d: %v", adID, err)
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("Failed to restore ad: %v", err))
	}

	// After successful restore, re-add the ad to the vector database
	restoredAd, ok := ad.GetAd(adID, nil)
	if ok {
		// Queue the ad for vector processing (async)
		go func() {
			log.Printf("[restore] Queuing restored ad %d for vector processing", adID)
			vector.GetVectorProcessor().QueueAd(restoredAd)
		}()
	}

	ads, err := ad.GetAllArchivedAds()
	if err != nil {
		log.Printf("Error getting archived ads: %v", err)
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("Failed to get archived ads: %v", err))
	}

	log.Printf("Successfully restored ad %d, returning %d archived ads", adID, len(ads))
	return render(c, ui.AdminAdTable(ads, "archived"))
}

func HandleAdminParentCompanies(c *fiber.Ctx) error {
	return adminHandler(c, "parent-companies", vehicle.GetAllParentCompanies, nil, adminParentCompaniesSectionWrapper)
}

func HandleAdminMakeParentCompanies(c *fiber.Ctx) error {
	currentUser, err := CurrentUser(c)
	if err != nil {
		return err
	}
	rows, err := db.Query(`
		SELECT Make.name, ParentCompany.name
		FROM Make
		LEFT JOIN ParentCompany ON Make.parent_company_id = ParentCompany.id
		ORDER BY Make.name
	`)
	if err != nil {
		return c.Status(500).SendString("DB error")
	}
	defer rows.Close()
	var data []struct{ Make, ParentCompany string }
	for rows.Next() {
		var make string
		var parent sql.NullString
		if err := rows.Scan(&make, &parent); err != nil {
			return c.Status(500).SendString("Scan error")
		}
		parentName := "Independent"
		if parent.Valid {
			parentName = parent.String
		}
		data = append(data, struct{ Make, ParentCompany string }{make, parentName})
	}
	c.Type("html")
	return ui.AdminSectionPage(currentUser, c.Path(), "make-parent-companies", ui.AdminMakeParentCompaniesSection(data)).Render(c.Context().Response.BodyWriter())
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
