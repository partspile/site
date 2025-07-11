package handlers

import (
	"fmt"

	"mime/multipart"

	"log"
	"os"
	"path/filepath"

	"net/http"
	"time"

	"bytes"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"database/sql"
	"encoding/json"

	"github.com/chai2010/webp"
	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/b2util"
	"github.com/parts-pile/site/grok"
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/vehicle"
	"golang.org/x/image/draw"
	"gopkg.in/kothar/go-backblaze.v0"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

func HandleNewAd(c *fiber.Ctx) error {
	currentUser, err := CurrentUser(c)
	if err != nil {
		return err
	}
	makes := vehicle.GetMakes()
	return render(c, ui.NewAdPage(currentUser, c.Path(), makes))
}

// Helper to resolve location using Grok and upsert into Location table
func resolveAndStoreLocation(raw string) (int, error) {
	if raw == "" {
		return 0, nil
	}
	// Update Grok prompt
	systemPrompt := `You are a location resolver for an auto parts website.
Given a user input (which may be a address, city, zip code, or country),
return a JSON object with the best guess for city, admin_area (state,
province, or region), and country. The country field must be a 2-letter
ISO country code (e.g., "US" for United States, "CA" for Canada, "GB"
for United Kingdom). For US and Canada, the admin_area field must be the
official 2-letter code (e.g., "OR" for Oregon, "NY" for New York, "BC"
for British Columbia, "ON" for Ontario). For all other countries, use
the full name for admin_area. If a field is unknown, leave it blank.
Example input: "97333" -> {"city": "Corvallis", "admin_area": "OR",
"country": "US"}`
	resp, err := grok.CallGrok(systemPrompt, raw)
	if err != nil {
		return 0, err
	}
	var loc struct {
		City      string `json:"city"`
		AdminArea string `json:"admin_area"`
		Country   string `json:"country"`
	}
	err = json.Unmarshal([]byte(resp), &loc)
	if err != nil {
		return 0, err
	}
	// Upsert into Location table
	db := ad.DB
	var id int
	err = db.QueryRow("SELECT id FROM Location WHERE raw_text = ?", raw).Scan(&id)
	if err == sql.ErrNoRows {
		res, err := db.Exec("INSERT INTO Location (raw_text, city, admin_area, country) VALUES (?, ?, ?, ?)", raw, loc.City, loc.AdminArea, loc.Country)
		if err != nil {
			return 0, err
		}
		lastID, _ := res.LastInsertId()
		id = int(lastID)
	} else if err != nil {
		return 0, err
	}
	return id, nil
}

func HandleNewAdSubmission(c *fiber.Ctx) error {
	currentUser, err := CurrentUser(c)
	if err != nil {
		return err
	}

	// Resolve and store location first
	locationRaw := c.FormValue("location")
	locID, err := resolveAndStoreLocation(locationRaw)
	if err != nil {
		return ValidationErrorResponse(c, "Could not resolve location.")
	}

	newAd, imageFiles, _, err := BuildAdFromForm(c, currentUser.ID, locID)
	if err != nil {
		return ValidationErrorResponse(c, err.Error())
	}
	ad.AddAd(newAd)
	fmt.Printf("[DEBUG] Created ad ID=%d with ImageOrder=%v\n", newAd.ID, newAd.ImageOrder)
	uploadAdImagesToB2(newAd.ID, imageFiles)
	return render(c, ui.SuccessMessage("Ad created successfully", "/"))
}

func HandleViewAd(c *fiber.Ctx) error {
	adID, err := c.ParamsInt("id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}

	currentUser, _ := GetCurrentUser(c)

	// Get ad from either active or archived tables
	adObj, _, ok := ad.GetAdByID(adID)
	if !ok {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}

	bookmarked := false
	if currentUser != nil {
		bookmarked, _ = ad.IsAdBookmarkedByUser(currentUser.ID, adID)
	}

	return render(c, ui.ViewAdPage(currentUser, c.Path(), adObj, bookmarked))
}

func HandleEditAd(c *fiber.Ctx) error {
	currentUser, err := CurrentUser(c)
	if err != nil {
		return err
	}

	adID, err := ParseIntParam(c, "id")
	if err != nil {
		return err
	}

	adObj, ok := ad.GetAd(adID)
	if !ok {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}

	_, err = RequireOwnership(c, adObj.UserID)
	if err != nil {
		return err
	}

	// Prepare make options
	makes := vehicle.GetMakes()
	// Prepare year checkboxes
	years := vehicle.GetYears(adObj.Make)
	// Prepare model checkboxes
	modelAvailability := vehicle.GetModelsWithAvailability(adObj.Make, adObj.Years)
	// Prepare engine checkboxes
	engineAvailability := vehicle.GetEnginesWithAvailability(adObj.Make, adObj.Years, adObj.Models)

	return render(c, ui.EditAdPage(currentUser, c.Path(), adObj, makes, years, modelAvailability, engineAvailability))
}

func HandleUpdateAdSubmission(c *fiber.Ctx) error {
	println("HandleUpdateAdSubmission")
	currentUser, err := CurrentUser(c)
	if err != nil {
		return err
	}

	adID, err := ParseIntParam(c, "id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}

	existingAd, ok := ad.GetAd(adID)
	if !ok {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}

	_, err = RequireOwnership(c, existingAd.UserID)
	if err != nil {
		return err
	}

	// Resolve and store location first
	locationRaw := c.FormValue("location")
	locID, err := resolveAndStoreLocation(locationRaw)
	if err != nil {
		return ValidationErrorResponse(c, "Could not resolve location.")
	}

	updatedAd, imageFiles, deletedImages, err := BuildAdFromForm(c, currentUser.ID, locID, adID)
	if err != nil {
		return ValidationErrorResponse(c, err.Error())
	}
	ad.UpdateAd(updatedAd)
	// Delete images from B2 if needed
	if len(deletedImages) > 0 {
		deleteAdImagesFromB2(updatedAd.ID, deletedImages)
	}
	uploadAdImagesToB2(updatedAd.ID, imageFiles)

	if c.Get("HX-Request") != "" {
		// For htmx, return the updated detail partial
		bookmarked := false
		if currentUser != nil {
			bookmarked, _ = ad.IsAdBookmarkedByUser(currentUser.ID, adID)
		}
		view := c.Query("view", "list")
		return render(c, ui.AdDetailPartial(updatedAd, bookmarked, currentUser.ID, view))
	}
	return render(c, ui.SuccessMessage("Ad updated successfully", fmt.Sprintf("/ad/%d", adID)))
}

// Handler to bookmark an ad
func HandleBookmarkAd(c *fiber.Ctx) error {
	currentUser, err := CurrentUser(c)
	if err != nil {
		return err
	}
	adID, err := ParseIntParam(c, "id")
	if err != nil {
		return err
	}
	if err := ad.BookmarkAd(currentUser.ID, adID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to bookmark ad")
	}
	// Return the bookmarked button HTML for HTMX swap
	return render(c, ui.BookmarkButton(true, adID))
}

// Handler to unbookmark an ad
func HandleUnbookmarkAd(c *fiber.Ctx) error {
	currentUser, err := CurrentUser(c)
	if err != nil {
		return err
	}
	adID, err := ParseIntParam(c, "id")
	if err != nil {
		return err
	}
	if err := ad.UnbookmarkAd(currentUser.ID, adID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to unbookmark ad")
	}
	// Return the unbookmarked button HTML for HTMX swap
	return render(c, ui.BookmarkButton(false, adID))
}

// Handler to get bookmarked ads for the current user (for settings page)
func HandleBookmarkedAds(c *fiber.Ctx) error {
	currentUser, err := CurrentUser(c)
	if err != nil {
		return err
	}
	adIDs, err := ad.GetBookmarkedAdIDsByUser(currentUser.ID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to get bookmarked ad IDs")
	}
	ads, err := ad.GetAdsByIDs(adIDs)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to get bookmarked ads")
	}
	return render(c, ui.BookmarkedAdsSection(currentUser, ads))
}

func HandleArchiveAd(c *fiber.Ctx) error {
	adID, err := ParseIntParam(c, "id")
	if err != nil {
		return err
	}
	if err := ad.ArchiveAd(adID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to archive ad")
	}
	return render(c, ui.SuccessMessage("Ad archived successfully", "/"))
}

// Handler for ad card partial (collapse)
func HandleAdCardPartial(c *fiber.Ctx) error {
	adID, err := c.ParamsInt("id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}
	adObj, _, ok := ad.GetAdByID(adID)
	if !ok {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}
	currentUser, _ := CurrentUser(c)
	bookmarked := false
	userID := 0
	if currentUser != nil {
		bookmarked, _ = ad.IsAdBookmarkedByUser(currentUser.ID, adID)
		userID = currentUser.ID
	}
	loc := c.Context().Time().Location()
	view := c.Query("view", "list")
	return render(c, ui.AdCardExpandable(adObj, loc, bookmarked, userID, view))
}

// Handler for ad detail partial (expand)
func HandleAdDetailPartial(c *fiber.Ctx) error {
	adID, err := c.ParamsInt("id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}

	// Increment global click count
	_ = ad.IncrementAdClick(adID)

	currentUser, _ := GetCurrentUser(c)
	if currentUser != nil {
		_ = ad.IncrementAdClickForUser(adID, currentUser.ID)
	}

	adObj, _, ok := ad.GetAdByID(adID)
	if !ok {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}
	bookmarked := false
	userID := 0
	if currentUser != nil {
		bookmarked, _ = ad.IsAdBookmarkedByUser(currentUser.ID, adID)
		userID = currentUser.ID
	}
	view := c.Query("view", "list")
	return render(c, ui.AdDetailPartial(adObj, bookmarked, userID, view))
}

// Add this handler for deleting an ad
func HandleDeleteAd(c *fiber.Ctx) error {
	adID, err := ParseIntParam(c, "id")
	if err != nil {
		return err
	}
	currentUser, err := CurrentUser(c)
	if err != nil {
		return err
	}
	adObj, ok := ad.GetAd(adID)
	if !ok {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}
	if adObj.UserID != currentUser.ID {
		return fiber.NewError(fiber.StatusForbidden, "You do not own this ad")
	}
	if err := ad.ArchiveAd(adID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to delete ad")
	}
	if c.Get("HX-Request") != "" {
		return c.SendStatus(fiber.StatusNoContent) // 204, so htmx removes the card
	}
	return render(c, ui.SuccessMessage("Ad deleted successfully", "/"))
}

// Handler for ad edit partial (inline edit)
func HandleEditAdPartial(c *fiber.Ctx) error {
	adID, err := c.ParamsInt("id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}
	adObj, ok := ad.GetAd(adID)
	if !ok {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}
	currentUser, err := CurrentUser(c)
	if err != nil {
		return err
	}
	if adObj.UserID != currentUser.ID {
		return fiber.NewError(fiber.StatusForbidden, "You do not own this ad")
	}
	makes := vehicle.GetMakes()
	years := vehicle.GetYears(adObj.Make)
	modelAvailability := vehicle.GetModelsWithAvailability(adObj.Make, adObj.Years)
	engineAvailability := vehicle.GetEnginesWithAvailability(adObj.Make, adObj.Years, adObj.Models)
	view := c.Query("view", "list")
	cancelTarget := fmt.Sprintf("/ad/detail/%d?view=%s", adObj.ID, view)
	htmxTarget := fmt.Sprintf("#ad-%d", adObj.ID)
	if view == "grid" {
		htmxTarget = fmt.Sprintf("#ad-grid-wrap-%d", adObj.ID)
	}
	return render(c, ui.AdEditPartial(adObj, makes, years, modelAvailability, engineAvailability, cancelTarget, htmxTarget, view))
}

// uploadAdImagesToB2 is a stub for now
func uploadAdImagesToB2(adID int, files []*multipart.FileHeader) {
	accountID := os.Getenv("BACKBLAZE_MASTER_KEY_ID")
	keyID := os.Getenv("BACKBLAZE_KEY_ID")
	appKey := os.Getenv("BACKBLAZE_APP_KEY")
	if accountID == "" || appKey == "" || keyID == "" {
		log.Println("B2 credentials not set in env vars")
		return
	}
	b2, err := backblaze.NewB2(backblaze.Credentials{
		AccountID:      accountID,
		ApplicationKey: appKey,
		KeyID:          keyID,
	})
	if err != nil {
		log.Println("B2 auth error:", err)
		return
	}
	bucket, err := b2.Bucket("parts-pile")
	if err != nil {
		log.Println("B2 bucket error:", err)
		return
	}

	sizes := []struct {
		Width   int
		Suffix  string
		Quality float32
	}{
		{160, "160w", 60},
		{480, "480w", 70},
		{1200, "1200w", 80},
	}

	for i, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			log.Println("B2 open file error:", err)
			continue
		}
		var buf bytes.Buffer
		if _, err := buf.ReadFrom(file); err != nil {
			log.Println("Read file error:", err)
			file.Close()
			continue
		}
		file.Close()
		img, _, err := image.Decode(bytes.NewReader(buf.Bytes()))
		if err != nil {
			log.Println("Decode image error:", err)
			continue
		}
		bounds := img.Bounds()
		for _, sz := range sizes {
			w := sz.Width
			h := bounds.Dy() * w / bounds.Dx()
			dst := image.NewRGBA(image.Rect(0, 0, w, h))
			draw.CatmullRom.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)
			var webpBuf bytes.Buffer
			opt := &webp.Options{Lossless: false, Quality: sz.Quality}
			if err := webp.Encode(&webpBuf, dst, opt); err != nil {
				log.Println("WebP encode error:", err)
				continue
			}
			b2Path := filepath.Join(
				fmt.Sprintf("%d", adID),
				fmt.Sprintf("%d-%s.webp", i+1, sz.Suffix),
			)
			_, err = bucket.UploadTypedFile(b2Path, "image/webp", nil, bytes.NewReader(webpBuf.Bytes()))
			if err != nil {
				log.Println("B2 upload error for", b2Path, ":", err)
			}
		}
	}
}

// Handler to get a signed B2 download URL for all images under an ad (prefix)
func HandleAdImageSignedURL(c *fiber.Ctx) error {
	adID := c.Params("adID")
	if adID == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "adID required"})
	}
	token, err := b2util.GetB2DownloadTokenForPrefixCached(adID + "/")
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{
		"prefix":  "/" + adID + "/",
		"token":   token,
		"expires": time.Now().Unix() + 3600,
	})
}

func deleteAdImagesFromB2(adID int, indices []int) {
	accountID := os.Getenv("BACKBLAZE_MASTER_KEY_ID")
	keyID := os.Getenv("BACKBLAZE_KEY_ID")
	appKey := os.Getenv("BACKBLAZE_APP_KEY")
	if accountID == "" || appKey == "" || keyID == "" {
		log.Println("B2 credentials not set in env vars")
		return
	}
	b2, err := backblaze.NewB2(backblaze.Credentials{
		AccountID:      accountID,
		ApplicationKey: appKey,
		KeyID:          keyID,
	})
	if err != nil {
		log.Println("B2 auth error:", err)
		return
	}
	bucket, err := b2.Bucket("parts-pile")
	if err != nil {
		log.Println("B2 bucket error:", err)
		return
	}
	for _, idx := range indices {
		b2Path := filepath.Join(
			fmt.Sprintf("%d", adID),
			fmt.Sprintf("%d.webp", idx),
		)
		// List file versions for this file name
		resp, err := bucket.ListFileVersions(b2Path, "", 10)
		if err != nil {
			log.Println("B2 list file versions error for", b2Path, ":", err)
			continue
		}
		for _, file := range resp.Files {
			_, err := bucket.DeleteFileVersion(file.Name, file.ID)
			if err != nil {
				log.Println("B2 delete error for", b2Path, file.ID, ":", err)
			}
		}
	}
}

// Handler for HTMX image carousel partial (single image)
func HandleAdImagePartial(c *fiber.Ctx) error {
	adID, err := c.ParamsInt("adID")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}
	idx, err := c.ParamsInt("idx")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid image index")
	}
	adObj, _, ok := ad.GetAdByID(adID)
	if !ok {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}
	// Render only the image and price badge (no container)
	mainImage := ui.AdImageWithFallbackSrcSet(adObj.ID, idx, adObj.Title)
	priceBadge := Div(
		Class("absolute top-0 left-0 bg-white text-green-600 text-base font-normal px-2 rounded-br-md"),
		g.Textf("$%.0f", adObj.Price),
	)
	return render(c, g.Group([]g.Node{mainImage, priceBadge}))
}
