package handlers

import (
	"fmt"

	"mime/multipart"

	"log"
	"path/filepath"

	"net/http"
	"strings"
	"time"

	"bytes"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"database/sql"

	"github.com/chai2010/webp"
	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/b2util"
	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/db"
	"github.com/parts-pile/site/part"
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/vector"
	"github.com/parts-pile/site/vehicle"
	"golang.org/x/image/draw"
	"gopkg.in/kothar/go-backblaze.v0"
)

func HandleNewAd(c *fiber.Ctx) error {
	currentUser, _ := CurrentUser(c)
	makes := vehicle.GetMakes()
	categories := part.GetCategories()

	return render(c, ui.NewAdPage(currentUser, c.Path(), makes, categories))
}

func HandleDuplicateAd(c *fiber.Ctx) error {
	currentUser, _ := CurrentUser(c)
	adID, err := c.ParamsInt("id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}

	// Fetch the original ad
	adObj, ok := ad.GetAd(adID, currentUser)
	if !ok {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}

	// Populate vehicle data (Make, Years, Models, Engines)
	adObj.Make, adObj.Years, adObj.Models, adObj.Engines =
		ad.GetVehicleData(adObj.ID)

	// Fetch subcategory name
	var subcategoryName string
	if adObj.SubCategory.Valid {
		subcategoryName = adObj.SubCategory.String
	}

	// Get category name for fetching subcategories
	var categoryName string
	if adObj.Category.Valid {
		categoryName = adObj.Category.String
	}

	// Pre-fetch all the dropdown/checkbox data needed for the form
	makes := vehicle.GetMakes()
	categories := part.GetCategories()
	years := vehicle.GetYears(adObj.Make)
	models := vehicle.GetModels(adObj.Make, adObj.Years)
	engines := vehicle.GetEngines(adObj.Make, adObj.Years, adObj.Models)
	var subcategories []part.SubCategory
	if categoryName != "" {
		subcategories, _ = part.GetSubCategoriesForCategory(categoryName)
	}

	return render(c, ui.DuplicateAdPage(
		currentUser, c.Path(), makes, categories,
		adObj, years, models, engines, subcategories, subcategoryName))
}

// Helper to resolve location using Grok and upsert into Location table
func resolveAndStoreLocation(raw string) (int, error) {
	if raw == "" {
		return 0, nil
	}

	// Check if location already exists first to avoid expensive Grok API call
	var id int
	err := db.QueryRow("SELECT id FROM Location WHERE raw_text = ?", raw).Scan(&id)
	if err == nil {
		// Location already exists, return the ID
		return id, nil
	} else if err != sql.ErrNoRows {
		// Database error
		return 0, err
	}

	// Location doesn't exist, resolve using Grok API
	loc, err := ad.ResolveLocation(raw)
	if err != nil {
		return 0, err
	}

	// Insert new location into database
	res, err := db.Exec("INSERT INTO Location (raw_text, city, admin_area, country, latitude, longitude) VALUES (?, ?, ?, ?, ?, ?)",
		raw, loc.City, loc.AdminArea, loc.Country, loc.Latitude, loc.Longitude)
	if err != nil {
		return 0, err
	}
	lastID, _ := res.LastInsertId()
	return int(lastID), nil
}

func HandleNewAdSubmission(c *fiber.Ctx) error {
	_, userID := CurrentUser(c)

	// Resolve and store location first
	locationRaw := c.FormValue("location")
	locID, err := resolveAndStoreLocation(locationRaw)
	if err != nil {
		return ValidationErrorResponse(c, "Could not resolve location.")
	}

	newAd, imageFiles, _, err := BuildAdFromForm(c, userID, locID)
	if err != nil {
		return ValidationErrorResponse(c, err.Error())
	}
	adID := ad.AddAd(newAd)
	fmt.Printf("[DEBUG] Created ad ID=%d with ImageCount=%d\n", adID, newAd.ImageCount)
	fmt.Printf("[DEBUG] Image files count: %d\n", len(imageFiles))
	if len(imageFiles) > 0 {
		for i, file := range imageFiles {
			fmt.Printf("[DEBUG] Image file %d: %s (size: %d bytes)\n", i+1, file.Filename, file.Size)
		}
	}
	uploadAdImagesToB2(adID, imageFiles)

	// Update the ad with the correct ID for embedding processing
	newAd.ID = adID

	// Attempt inline vector processing, fallback to queue if it fails
	log.Printf("[embedding] Attempting inline vector processing for ad %d", adID)
	err = vector.BuildAdEmbedding(newAd)
	if err != nil {
		log.Printf("[embedding] Inline processing failed for ad %d: %v, queuing for background processing", adID, err)
		vector.QueueAd(newAd)
	} else {
		log.Printf("[embedding] Successfully processed ad %d inline", adID)
	}

	return render(c, ui.SuccessMessage("Ad created successfully", "/"))
}

func HandleAdPage(c *fiber.Ctx) error {
	adID, err := c.ParamsInt("id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}

	currentUser, userID := CurrentUser(c)

	adObj, ok := ad.GetAd(adID, currentUser)
	if !ok {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}

	// If ad is deleted and user is not the owner, show deleted message
	if adObj.IsArchived() && adObj.UserID != userID {
		return render(c, ui.AdDeletedPage(currentUser, c.Path()))
	}

	// Owner can see their deleted ads, or anyone can see active ads
	return render(c, ui.AdPage(adObj, currentUser, userID, c.Path(), getLocation(c), getView(c)))
}

// HandleUpdateAdPrice updates only the price of an ad
func HandleUpdateAdPrice(c *fiber.Ctx) error {
	currentUser, _ := CurrentUser(c)
	adID, err := ParseIntParam(c, "id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}

	existingAd, ok := ad.GetAd(adID, currentUser)
	if !ok {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}

	_, err = RequireOwnership(c, existingAd.UserID)
	if err != nil {
		return err
	}

	// Validate and parse price
	price, err := ValidateAndParsePrice(c)
	if err != nil {
		return ValidationErrorResponse(c, err.Error())
	}

	// Update only the price
	_, err = db.Exec("UPDATE Ad SET price = ? WHERE id = ?", price, adID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError,
			"Failed to update price")
	}

	// Fetch updated ad for display
	updatedAd, ok := ad.GetAd(adID, currentUser)
	if !ok {
		return fiber.NewError(fiber.StatusInternalServerError,
			"Failed to fetch updated ad")
	}

	// Queue for vector update since price affects search
	vector.QueueAd(updatedAd)

	return render(c, ui.AdDetail(updatedAd, getLocation(c),
		currentUser.ID, getView(c)))
}

// HandleUpdateAdLocation updates only the location of an ad
func HandleUpdateAdLocation(c *fiber.Ctx) error {
	currentUser, _ := CurrentUser(c)
	adID, err := ParseIntParam(c, "id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}

	existingAd, ok := ad.GetAd(adID, currentUser)
	if !ok {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}

	_, err = RequireOwnership(c, existingAd.UserID)
	if err != nil {
		return err
	}

	// Resolve and store location
	locationRaw := c.FormValue("location")
	locID, err := resolveAndStoreLocation(locationRaw)
	if err != nil {
		return ValidationErrorResponse(c, "Could not resolve location.")
	}

	// Update only the location
	_, err = db.Exec("UPDATE Ad SET location_id = ? WHERE id = ?", locID, adID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError,
			"Failed to update location")
	}

	// Fetch updated ad for display
	updatedAd, ok := ad.GetAd(adID, currentUser)
	if !ok {
		return fiber.NewError(fiber.StatusInternalServerError,
			"Failed to fetch updated ad")
	}

	// Queue for vector update since location affects search
	vector.QueueAd(updatedAd)

	return render(c, ui.AdDetail(updatedAd, getLocation(c),
		currentUser.ID, getView(c)))
}

// HandleUpdateAdDescription appends to the description of an ad
func HandleUpdateAdDescription(c *fiber.Ctx) error {
	currentUser, _ := CurrentUser(c)
	adID, err := ParseIntParam(c, "id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}

	existingAd, ok := ad.GetAd(adID, currentUser)
	if !ok {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}

	_, err = RequireOwnership(c, existingAd.UserID)
	if err != nil {
		return err
	}

	// Handle description addition
	descriptionAddition := c.FormValue("description_addition")
	updatedDescription := existingAd.Description

	if descriptionAddition != "" {
		// Clean the addition text
		descriptionAddition = strings.TrimSpace(descriptionAddition)
		if descriptionAddition != "" {
			// Create timestamp
			timestamp := time.Now().Format("2006-01-02 15:04")
			// Append with timestamp
			addition := fmt.Sprintf("\n\n[%s] %s",
				timestamp, descriptionAddition)
			updatedDescription = existingAd.Description + addition

			// Validate total length
			if len(updatedDescription) > 500 {
				return ValidationErrorResponse(c, fmt.Sprintf(
					"Total description would be %d characters (max 500). "+
						"Please shorten your addition.",
					len(updatedDescription)))
			}
		}
	}

	// Update only the description
	_, err = db.Exec("UPDATE Ad SET description = ? WHERE id = ?",
		updatedDescription, adID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError,
			"Failed to update description")
	}

	// Fetch updated ad for display
	updatedAd, ok := ad.GetAd(adID, currentUser)
	if !ok {
		return fiber.NewError(fiber.StatusInternalServerError,
			"Failed to fetch updated ad")
	}

	// Queue for vector update since description affects search
	vector.QueueAd(updatedAd)

	return render(c, ui.AdDetail(updatedAd, getLocation(c),
		currentUser.ID, getView(c)))
}

// Handler for ad card (collapse)
func HandleAdCard(c *fiber.Ctx) error {
	adID, err := c.ParamsInt("id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}
	currentUser, userID := CurrentUser(c)
	adObj, ok := ad.GetAd(adID, currentUser)
	if !ok {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}
	loc := getLocation(c)
	view := getView(c)
	switch view {
	case "list", "map":
		return render(c, ui.AdListNode(adObj, loc, userID))
	case "grid":
		return render(c, ui.AdGridNode(adObj, loc, userID))
	}
	return nil
}

// Handler for ad detail (expanded view)
func HandleAdDetail(c *fiber.Ctx) error {
	adID, err := c.ParamsInt("id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}

	currentUser, userID := CurrentUser(c)
	adObj, ok := ad.GetAd(adID, currentUser)
	if !ok {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}

	// Only increment click counts for non-deleted ads
	if !adObj.IsArchived() {
		_ = ad.IncrementAdClick(adID)
		if userID != 0 {
			_ = ad.IncrementAdClickForUser(adID, userID)
			// Queue user for background embedding update
			vector.QueueUserForUpdate(userID)
		}
	}

	loc := getLocation(c)
	view := getView(c)
	return render(c, ui.AdDetail(adObj, loc, userID, view))
}

// Add this handler for deleting an ad
func HandleDeleteAd(c *fiber.Ctx) error {
	adID, err := ParseIntParam(c, "id")
	if err != nil {
		return err
	}
	currentUser, _ := CurrentUser(c)
	adObj, ok := ad.GetAd(adID, currentUser)
	if !ok {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}
	if adObj.UserID != currentUser.ID {
		return fiber.NewError(fiber.StatusForbidden, "You do not own this ad")
	}
	if err := ad.ArchiveAd(adID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to delete ad")
	}

	// Delete from vector database (Qdrant) - this is done after DB commit
	if err := vector.DeleteAdEmbedding(adID); err != nil {
		// Log the error but don't fail the entire operation since DB is already committed
		log.Printf("Warning: Failed to delete ad %d from vector database: %v", adID, err)
	}

	log.Printf("Delete ad %d: HX-Request header = '%s'", adID, c.Get("HX-Request"))
	if c.Get("HX-Request") != "" {
		log.Printf("Returning 200 with empty body for HTMX request to delete ad %d", adID)
		// Return 200 with empty body instead of 204 because HTMX only swaps on 200/300 responses
		// See: https://github.com/bigskysoftware/htmx/issues/1130
		return render(c, ui.EmptyResponse())
	}
	log.Printf("Returning success page for non-HTMX request to delete ad %d", adID)
	return render(c, ui.SuccessMessage("Ad deleted successfully", "/"))
}

// Handler for restoring a deleted ad
func HandleRestoreAd(c *fiber.Ctx) error {
	adID, err := ParseIntParam(c, "id")
	if err != nil {
		return err
	}
	currentUser, userID := CurrentUser(c)
	adObj, ok := ad.GetAd(adID, currentUser)
	if !ok {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}
	if adObj.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "You do not own this ad")
	}
	if !adObj.IsArchived() {
		return fiber.NewError(fiber.StatusBadRequest, "Ad is not deleted")
	}
	if err := ad.RestoreAd(adID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to restore ad")
	}

	// Queue ad for background re-addition to vector database (Qdrant)
	vector.QueueAd(adObj)

	// Get the restored ad with updated data
	restoredAd, ok := ad.GetAd(adID, currentUser)
	if !ok {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to retrieve restored ad")
	}

	// If viewing in collapsed list/grid view (from deleted ads page), remove the ad from the list
	// Otherwise, show the updated ad detail
	view := getView(c)
	if view == "list" || view == "grid" || view == "map" {
		// Return empty response to remove the ad from the deleted ads list
		log.Printf("Restore ad %d from list view, removing from DOM", adID)
		return render(c, ui.EmptyResponse())
	}

	// For expanded detail view, return the updated ad without deleted styling
	loc := getLocation(c)
	return render(c, ui.AdDetail(restoredAd, loc, userID, view))
}

// uploadAdImagesToB2 uploads user-uploaded images to B2 with multiple sizes
func uploadAdImagesToB2(adID int, files []*multipart.FileHeader) {
	log.Printf("[B2] Starting upload for ad %d with %d images", adID, len(files))

	accountID := config.B2MasterKeyID
	keyID := config.B2KeyID
	appKey := config.B2AppKey

	log.Printf("[B2] B2 config check for ad %d: accountID=%s, keyID=%s, appKey=%s", adID,
		func() string {
			if accountID == "" {
				return "EMPTY"
			} else {
				return "SET"
			}
		}(),
		func() string {
			if keyID == "" {
				return "EMPTY"
			} else {
				return "SET"
			}
		}(),
		func() string {
			if appKey == "" {
				return "EMPTY"
			} else {
				return "SET"
			}
		}())

	if accountID == "" || appKey == "" || keyID == "" {
		log.Printf("[B2] ERROR: B2 credentials not set in env vars for ad %d", adID)
		return
	}

	b2, err := backblaze.NewB2(backblaze.Credentials{
		AccountID:      accountID,
		ApplicationKey: appKey,
		KeyID:          keyID,
	})
	if err != nil {
		log.Printf("[B2] ERROR: B2 auth error for ad %d: %v", adID, err)
		return
	}

	log.Printf("[B2] Using bucket name: %s for ad %d", config.B2BucketName, adID)
	bucket, err := b2.Bucket(config.B2BucketName)
	if err != nil {
		log.Printf("[B2] ERROR: B2 bucket error for ad %d: %v", adID, err)
		return
	}
	log.Printf("[B2] Successfully connected to bucket for ad %d", adID)

	sizes := []struct {
		Width   int
		Suffix  string
		Quality float32
	}{
		{160, "160w", 60},
		{480, "480w", 70},
		{1200, "1200w", 80},
	}

	successCount := 0
	totalExpected := len(files) * len(sizes)

	for i, fileHeader := range files {
		log.Printf("[B2] Processing image %d/%d for ad %d", i+1, len(files), adID)

		file, err := fileHeader.Open()
		if err != nil {
			log.Printf("[B2] ERROR: Failed to open file %d for ad %d: %v", i+1, adID, err)
			continue
		}

		var buf bytes.Buffer
		if _, err := buf.ReadFrom(file); err != nil {
			log.Printf("[B2] ERROR: Failed to read file %d for ad %d: %v", i+1, adID, err)
			file.Close()
			continue
		}
		file.Close()

		img, _, err := image.Decode(bytes.NewReader(buf.Bytes()))
		if err != nil {
			log.Printf("[B2] ERROR: Failed to decode image %d for ad %d: %v", i+1, adID, err)
			continue
		}

		bounds := img.Bounds()
		log.Printf("[B2] Image %d for ad %d: %dx%d pixels", i+1, adID, bounds.Dx(), bounds.Dy())

		for _, sz := range sizes {
			w := sz.Width
			h := bounds.Dy() * w / bounds.Dx()
			dst := image.NewRGBA(image.Rect(0, 0, w, h))
			draw.CatmullRom.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)

			var webpBuf bytes.Buffer
			opt := &webp.Options{Lossless: false, Quality: sz.Quality}
			if err := webp.Encode(&webpBuf, dst, opt); err != nil {
				log.Printf("[B2] ERROR: WebP encode error for image %d size %s ad %d: %v", i+1, sz.Suffix, adID, err)
				continue
			}

			b2Path := filepath.Join(
				fmt.Sprintf("%d", adID),
				fmt.Sprintf("%d-%s.webp", i+1, sz.Suffix),
			)

			log.Printf("[B2] Uploading %s to %s for ad %d", sz.Suffix, b2Path, adID)
			_, err = bucket.UploadTypedFile(b2Path, "image/webp", nil, bytes.NewReader(webpBuf.Bytes()))
			if err != nil {
				log.Printf("[B2] ERROR: Upload failed for %s to %s ad %d: %v", sz.Suffix, b2Path, adID, err)
			} else {
				log.Printf("[B2] SUCCESS: Uploaded %s to %s for ad %d", sz.Suffix, b2Path, adID)
				successCount++
			}
		}
	}

	log.Printf("[B2] Upload complete for ad %d: %d/%d files uploaded successfully", adID, successCount, totalExpected)
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
		"expires": time.Now().Unix() + config.B2DownloadTokenExpiry,
	})
}

// Handler for HTMX image carousel
func HandleAdImage(c *fiber.Ctx) error {
	adID, err := c.ParamsInt("adID")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}
	idx, err := c.ParamsInt("idx")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid image index")
	}
	return render(c, ui.AdCarouselImage(adID, idx))
}
