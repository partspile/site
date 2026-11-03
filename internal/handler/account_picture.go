package handler

import (
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/partspile/site/internal/ad"
	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/cookie"
	"github.com/partspile/site/internal/flag"
	"github.com/partspile/site/internal/local"
	"github.com/partspile/site/internal/logger"
	"github.com/partspile/site/internal/ui"
	"github.com/partspile/site/internal/user"
)

func ownProfilePictureData(c *fiber.Ctx) (ui.UserProfileData, error) {
	userID := local.GetUserID(c)
	u, err := user.GetByID(userID)
	if err != nil {
		return ui.UserProfileData{},
			fiber.NewError(fiber.StatusNotFound, "User not found")
	}
	activeAdCount, _ := ad.CountActiveAdsByUser(userID)
	userFlagCount, _ := flag.GetFlagCountForUser(userID)
	tz := cookie.GetTimezone(c)
	if tz == nil {
		tz = time.UTC
	}
	d := userProfileData(u, activeAdCount, userFlagCount, tz,
		memberSinceLayoutPage)
	d.IsOwn = true
	d.ShowShare = true
	return d, nil
}

// ProfilePictureDisplayHandler returns the owner display block (Edit button).
func ProfilePictureDisplayHandler(c *fiber.Ctx) error {
	d, err := ownProfilePictureData(c)
	if err != nil {
		return err
	}
	return render(c, ui.AccountPictureDisplaySection(d))
}

// ProfilePictureEditHandler returns the owner edit form.
func ProfilePictureEditHandler(c *fiber.Ctx) error {
	d, err := ownProfilePictureData(c)
	if err != nil {
		return err
	}
	return render(c, ui.AccountPictureEditSection(d, ""))
}

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
	raw := strings.TrimSpace(c.FormValue("account_picture_url"))
	d, err := ownProfilePictureData(c)
	if err != nil {
		return err
	}
	if raw != "" {
		if err := validateAccountPictureURL(raw); err != nil {
			d.AccountPictureURL = raw
			return render(c, ui.AccountPictureEditSection(d, err.Error()))
		}
	}
	if err := user.SetAccountPictureURL(d.ID, raw); err != nil {
		logger.Error("Failed to save account picture URL",
			"error", err, "userID", d.ID)
		return fiber.NewError(fiber.StatusInternalServerError,
			"Failed to save link")
	}
	d, err = ownProfilePictureData(c)
	if err != nil {
		return err
	}
	return render(c, ui.AccountPictureDisplaySection(d))
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
	d, err := ownProfilePictureData(c)
	if err != nil {
		return err
	}
	return render(c, ui.AccountPictureDisplaySection(d))
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
