package handlers

import (
	"fmt"

	"mime/multipart"

	"log"
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
	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/db"
	"github.com/parts-pile/site/grok"
	"github.com/parts-pile/site/part"
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/vector"
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
	categories, err := part.GetAllCategories()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to get categories")
	}

	// Convert categories to string slice
	categoryNames := make([]string, len(categories))
	for i, cat := range categories {
		categoryNames[i] = cat.Name
	}

	return render(c, ui.NewAdPage(currentUser, c.Path(), makes, categoryNames))
}

// Helper to resolve location using Grok and upsert into Location table
func resolveAndStoreLocation(raw string) (int, error) {
	if raw == "" {
		return 0, nil
	}
	// Update Grok prompt to include coordinates
	systemPrompt := `You are a location resolver for an auto parts website.
Given a user input (which may be a address, city, zip code, or country),
return a JSON object with the best guess for city, admin_area (state,
province, or region), country, latitude, and longitude. The country field 
must be a 2-letter ISO country code (e.g., "US" for United States, "CA" 
for Canada, "GB" for United Kingdom). For US and Canada, the admin_area 
field must be the official 2-letter code (e.g., "OR" for Oregon, "NY" 
for New York, "BC" for British Columbia, "ON" for Ontario). For all 
other countries, use the full name for admin_area. Latitude and longitude 
should be decimal degrees (positive for North/East, negative for South/West).
If a field is unknown, leave it blank or null.
Example input: "97333" -> {"city": "Corvallis", "admin_area": "OR",
"country": "US", "latitude": 44.5646, "longitude": -123.2620}`
	resp, err := grok.CallGrok(systemPrompt, raw)
	if err != nil {
		return 0, err
	}
	var loc struct {
		City      string   `json:"city"`
		AdminArea string   `json:"admin_area"`
		Country   string   `json:"country"`
		Latitude  *float64 `json:"latitude"`
		Longitude *float64 `json:"longitude"`
	}
	err = json.Unmarshal([]byte(resp), &loc)
	if err != nil {
		return 0, err
	}
	// Upsert into Location table
	var id int
	err = db.QueryRow("SELECT id FROM Location WHERE raw_text = ?", raw).Scan(&id)
	if err == sql.ErrNoRows {
		res, err := db.Exec("INSERT INTO Location (raw_text, city, admin_area, country, latitude, longitude) VALUES (?, ?, ?, ?, ?, ?)",
			raw, loc.City, loc.AdminArea, loc.Country, loc.Latitude, loc.Longitude)
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
	adID := ad.AddAd(newAd)
	fmt.Printf("[DEBUG] Created ad ID=%d with ImageOrder=%v\n", adID, newAd.ImageOrder)
	fmt.Printf("[DEBUG] Image files count: %d\n", len(imageFiles))
	if len(imageFiles) > 0 {
		for i, file := range imageFiles {
			fmt.Printf("[DEBUG] Image file %d: %s (size: %d bytes)\n", i+1, file.Filename, file.Size)
		}
	}
	uploadAdImagesToB2(adID, imageFiles)

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

func HandleViewAd(c *fiber.Ctx) error {
	adID, err := c.ParamsInt("id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}

	currentUser, _ := GetCurrentUser(c)

	// Get ad from either active or archived tables with bookmark status
	adObj, ok := ad.GetAdWithVehicle(adID, currentUser)
	if !ok {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}

	return render(c, ui.ViewAdPage(currentUser, c.Path(), adObj))
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

	adObj, ok := ad.GetAdWithVehicle(adID, currentUser)
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

	// Get categories
	categories, err := part.GetAllCategories()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to get categories")
	}

	// Convert categories to string slice
	categoryNames := make([]string, len(categories))
	for i, cat := range categories {
		categoryNames[i] = cat.Name
	}

	// Get subcategories for the current category if it exists
	var subcategoryNames []string
	if adObj.Category != "" {
		subCategories, err := part.GetSubCategoriesForCategory(adObj.Category)
		if err == nil {
			subcategoryNames = make([]string, len(subCategories))
			for i, subCat := range subCategories {
				subcategoryNames[i] = subCat.Name
			}
		}
	}

	return render(c, ui.EditAdPage(currentUser, c.Path(), adObj, makes, years, modelAvailability, engineAvailability, categoryNames, subcategoryNames))
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

	existingAd, ok := ad.GetAd(adID, currentUser)
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

	// Attempt inline vector processing, fallback to queue if it fails
	log.Printf("[embedding] Attempting inline vector processing for updated ad %d", adID)
	err = vector.BuildAdEmbedding(updatedAd)
	if err != nil {
		log.Printf("[embedding] Inline processing failed for ad %d: %v, queuing for background processing", adID, err)
		vector.QueueAd(updatedAd)
	} else {
		log.Printf("[embedding] Successfully processed updated ad %d inline", adID)
	}

	if c.Get("HX-Request") != "" {
		// For htmx, return the updated detail partial
		view := c.Query("view", "list")
		return render(c, ui.AdDetailPartial(updatedAd, currentUser.ID, view))
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
	// Queue user for background embedding update
	vector.QueueUserForUpdate(currentUser.ID)
	// Get the updated ad with bookmark status
	adObj, ok := ad.GetAd(adID, currentUser)
	if !ok {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}
	// Return the bookmarked button HTML for HTMX swap
	return render(c, ui.BookmarkButton(adObj))
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
	// Queue user for background embedding update
	vector.QueueUserForUpdate(currentUser.ID)
	// Get the updated ad with bookmark status
	adObj, ok := ad.GetAd(adID, currentUser)
	if !ok {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}
	// Return the unbookmarked button HTML for HTMX swap
	return render(c, ui.BookmarkButton(adObj))
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
	ads, err := ad.GetAdsByIDs(adIDs, currentUser)
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
	currentUser, _ := CurrentUser(c)
	adObj, ok := ad.GetAd(adID, currentUser)
	if !ok {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}
	loc := c.Context().Time().Location()
	view := c.Query("view", "list")
	if view == "list" {
		return render(c, ui.AdCardCompactList(adObj, loc, currentUser))
	} else if view == "tree" {
		return render(c, ui.AdCardCompactTree(adObj, loc, currentUser))
	}
	return render(c, ui.AdCardExpandable(adObj, loc, currentUser, view))
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
		// Queue user for background embedding update
		vector.QueueUserForUpdate(currentUser.ID)
	}

	adObj, ok := ad.GetAd(adID, currentUser)
	if !ok {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}
	_, userID := getUser(c)
	view := c.Query("view", "list")
	return render(c, ui.AdDetailPartial(adObj, userID, view))
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

// Handler for ad edit partial (inline edit)
func HandleEditAdPartial(c *fiber.Ctx) error {
	adID, err := c.ParamsInt("id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}
	currentUser, err := CurrentUser(c)
	if err != nil {
		return err
	}
	adObj, ok := ad.GetAd(adID, currentUser)
	if !ok {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}
	if adObj.UserID != currentUser.ID {
		return fiber.NewError(fiber.StatusForbidden, "You do not own this ad")
	}
	makes := vehicle.GetMakes()
	years := vehicle.GetYears(adObj.Make)
	modelAvailability := vehicle.GetModelsWithAvailability(adObj.Make, adObj.Years)
	engineAvailability := vehicle.GetEnginesWithAvailability(adObj.Make, adObj.Years, adObj.Models)

	// Get categories
	categories, err := part.GetAllCategories()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to get categories")
	}

	// Convert categories to string slice
	categoryNames := make([]string, len(categories))
	for i, cat := range categories {
		categoryNames[i] = cat.Name
	}

	// Get subcategories for the current category if it exists
	var subcategoryNames []string
	if adObj.Category != "" {
		subCategories, err := part.GetSubCategoriesForCategory(adObj.Category)
		if err == nil {
			subcategoryNames = make([]string, len(subCategories))
			for i, subCat := range subCategories {
				subcategoryNames[i] = subCat.Name
			}
		}
	}

	view := c.Query("view", "list")
	cancelTarget := fmt.Sprintf("/ad/detail/%d?view=%s", adObj.ID, view)
	htmxTarget := fmt.Sprintf("#ad-%d", adObj.ID)
	if view == "grid" {
		htmxTarget = fmt.Sprintf("#ad-grid-wrap-%d", adObj.ID)
	}
	return render(c, ui.AdEditPartial(adObj, makes, years, modelAvailability, engineAvailability, categoryNames, subcategoryNames, cancelTarget, htmxTarget, view))
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

func deleteAdImagesFromB2(adID int, indices []int) {
	accountID := config.B2MasterKeyID
	keyID := config.B2KeyID
	appKey := config.B2AppKey
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
	bucket, err := b2.Bucket(config.B2BucketName)
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
	adObj, ok := ad.GetAd(adID, nil)
	if !ok {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}
	// Render only the image and price badge (no container)
	mainImage := ui.AdImageWithFallbackSrcSet(adObj.ID, idx, adObj.Title, "carousel")
	priceBadge := Div(
		Class("absolute top-0 left-0 bg-white text-green-600 text-base font-normal px-2 rounded-br-md"),
		g.Textf("$%.0f", adObj.Price),
	)
	return render(c, g.Group([]g.Node{mainImage, priceBadge}))
}

// --- VECTOR EMBEDDING HELPERS ---
func buildAdEmbeddingPrompt(adObj ad.Ad) string {
	// Get parent company information for the make
	var parentCompanyStr, parentCompanyCountry string
	if adObj.Make != "" {
		if pcInfo, err := vehicle.GetParentCompanyInfoForMake(adObj.Make); err == nil && pcInfo != nil {
			parentCompanyStr = pcInfo.Name
			parentCompanyCountry = pcInfo.Country
		}
	}

	return fmt.Sprintf(`Encode the following ad for semantic search. Focus on what the part is, what vehicles it fits, and any relevant details for a buyer. Return only the embedding vector.\n\nTitle: %s\nDescription: %s\nMake: %s\nParent Company: %s\nParent Company Country: %s\nYears: %s\nModels: %s\nEngines: %s\nCategory: %s\nLocation: %s, %s, %s`,
		adObj.Title,
		adObj.Description,
		adObj.Make,
		parentCompanyStr,
		parentCompanyCountry,
		joinStrings(adObj.Years),
		joinStrings(adObj.Models),
		joinStrings(adObj.Engines),
		adObj.Category,
		adObj.City,
		adObj.AdminArea,
		adObj.Country,
	)
}

func buildAdEmbeddingMetadata(adObj ad.Ad) map[string]interface{} {
	metadata := map[string]interface{}{
		"created_at":  adObj.CreatedAt.Format(time.RFC3339),
		"click_count": adObj.ClickCount,
	}

	// Add location coordinates if available
	if adObj.LocationID > 0 {
		_, _, _, _, lat, lon, err := ad.GetLocation(adObj.LocationID)
		if err == nil {
			// Add coordinates if available
			if lat != 0 && lon != 0 {
				metadata["latitude"] = lat
				metadata["longitude"] = lon
			}
		}
	}

	return metadata
}

// Helper functions for embedding generation
func interfaceSlice(ss []string) []interface{} {
	out := make([]interface{}, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}

func joinStrings(ss []string) string {
	if len(ss) == 0 {
		return ""
	}
	return fmt.Sprintf("%s", ss)
}

// --- END VECTOR EMBEDDING HELPERS ---

// HandleExpandAdTree expands an ad in tree view from compact to detailed view
func HandleExpandAdTree(c *fiber.Ctx) error {
	adID, err := c.ParamsInt("id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}

	currentUser, _ := GetCurrentUser(c)

	// Get ad from either active or archived tables with bookmark status
	adObj, ok := ad.GetAd(adID, currentUser)
	if !ok {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}

	loc, _ := time.LoadLocation(c.Get("X-Timezone"))
	return render(c, ui.AdCardExpandedTree(adObj, loc, currentUser))
}

// HandleCollapseAdTree collapses an ad in tree view from detailed to compact view
func HandleCollapseAdTree(c *fiber.Ctx) error {
	adID, err := c.ParamsInt("id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}

	currentUser, _ := GetCurrentUser(c)

	// Get ad from either active or archived tables with bookmark status
	adObj, ok := ad.GetAd(adID, currentUser)
	if !ok {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}

	loc, _ := time.LoadLocation(c.Get("X-Timezone"))
	return render(c, ui.AdCardCompactTree(adObj, loc, currentUser))
}
