package handler

import (
	"errors"
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/local"
	"github.com/partspile/site/internal/logger"
	"github.com/partspile/site/internal/ui"
	"github.com/partspile/site/internal/user"
)

// PresignAccountPictureHandler returns a short-lived PUT URL for account.jpg.
func PresignAccountPictureHandler(c *fiber.Ctx) error {
	userID := local.GetUserID(c)
	if adImageStore == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "image store unavailable",
		})
	}
	putURL, err := adImageStore.PresignPutUserAccount(userID,
		config.MinIOPresignedPutExpiry)
	if err != nil {
		logger.Error("Failed to presign account picture",
			"error", err, "userID", userID)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to prepare upload",
		})
	}
	return c.JSON(fiber.Map{"putUrl": putURL})
}

// ConfirmAccountPictureHandler sets has_account_picture after upload.
func ConfirmAccountPictureHandler(c *fiber.Ctx) error {
	userID := local.GetUserID(c)
	if err := user.ConfirmAccountPicture(userID); err != nil {
		logger.Error("Failed to confirm account picture",
			"error", err, "userID", userID)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to confirm upload",
		})
	}
	if adImageURLCache != nil {
		adImageURLCache.InvalidateUserAccountURL(userID)
	}
	return c.JSON(fiber.Map{"ok": true})
}

// SaveAccountPictureURLHandler saves or clears the click-through URL.
func SaveAccountPictureURLHandler(c *fiber.Ctx) error {
	userID := local.GetUserID(c)
	raw := strings.TrimSpace(c.FormValue("account_picture_url"))
	u, err := user.GetByID(userID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "User not found")
	}
	if raw != "" {
		if err := validateAccountPictureURL(raw); err != nil {
			return render(c, ui.AccountPictureSection(
				u.ID, u.HasAccountPicture, raw, err.Error(),
			))
		}
	}
	if err := user.SetAccountPictureURL(userID, raw); err != nil {
		logger.Error("Failed to save account picture URL",
			"error", err, "userID", userID)
		return fiber.NewError(fiber.StatusInternalServerError,
			"Failed to save link")
	}
	u, err = user.GetByID(userID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "User not found")
	}
	return render(c, ui.AccountPictureSection(
		u.ID, u.HasAccountPicture, u.AccountPictureURL, "",
	))
}

// RemoveAccountPictureHandler clears the picture flag and deletes MinIO object.
func RemoveAccountPictureHandler(c *fiber.Ctx) error {
	userID := local.GetUserID(c)
	if err := user.ClearAccountPicture(userID); err != nil {
		logger.Error("Failed to clear account picture",
			"error", err, "userID", userID)
		return fiber.NewError(fiber.StatusInternalServerError,
			"Failed to remove picture")
	}
	deleteUserAccountPicture(userID)
	u, err := user.GetByID(userID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "User not found")
	}
	return render(c, ui.AccountPictureSection(
		u.ID, u.HasAccountPicture, u.AccountPictureURL, "",
	))
}

func validateAccountPictureURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return errors.New("Enter a valid http or https URL")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return errors.New("Link must start with http:// or https://")
	}
	return nil
}
