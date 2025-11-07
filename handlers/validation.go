package handlers

import (
	"fmt"
	"mime/multipart"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/cookie"
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/spec"
)

// ValidateRequired validates that a required form field is not empty
func ValidateRequired(c *fiber.Ctx, fieldName, displayName string) (string, error) {
	value := c.FormValue(fieldName)
	if value == "" {
		return "", fmt.Errorf("%s is required", displayName)
	}
	return value, nil
}

// ValidateCleanText validates that text contains only clean characters and is within length limits
func ValidateCleanText(c *fiber.Ctx, fieldName, displayName string, maxLength int) (string, error) {
	value := c.FormValue(fieldName)
	if value == "" {
		return "", fmt.Errorf("%s is required", displayName)
	}

	// Trim whitespace
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("%s cannot be empty", displayName)
	}

	// Check length
	if len(value) > maxLength {
		return "", fmt.Errorf("%s must be %d characters or less", displayName, maxLength)
	}

	// Check for printable ASCII characters only (0x20-0x7E)
	for _, char := range value {
		if char < 0x20 || char > 0x7E {
			return "", fmt.Errorf("%s contains invalid characters. Only printable ASCII characters are allowed", displayName)
		}
	}

	return value, nil
}

// ValidateUsername validates that a username follows conventions
// Rules:
// - 3 to 20 characters
// - Only letters (a-z, A-Z) and digits (0-9)
// - First character must be a letter (a-z, A-Z)
func ValidateUsername(username string) error {
	if username == "" {
		return fmt.Errorf("username is required")
	}

	// Check length (3 to 20 characters)
	if len(username) > 20 {
		return fmt.Errorf("username must be 20 characters or less")
	}
	if len(username) < 3 {
		return fmt.Errorf("username must be at least 3 characters")
	}

	// Check that first character is a letter
	firstChar := username[0]
	if !((firstChar >= 'a' && firstChar <= 'z') || (firstChar >= 'A' && firstChar <= 'Z')) {
		return fmt.Errorf("username must start with a letter (a-z, A-Z)")
	}

	// Check that all characters are letters or digits
	validPattern := regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	if !validPattern.MatchString(username) {
		return fmt.Errorf("username can only contain letters and digits")
	}

	return nil
}

// ValidateEmail validates that a string is a valid email address
func ValidateEmail(email string) error {
	if email == "" {
		return fmt.Errorf("email address is required")
	}

	// Basic email validation using regex-like string operations
	// Check for @ symbol and basic structure
	if !strings.Contains(email, "@") {
		return fmt.Errorf("email address must contain @ symbol")
	}

	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return fmt.Errorf("email address must have exactly one @ symbol")
	}

	localPart := parts[0]
	domainPart := parts[1]

	// Check local part (before @)
	if len(localPart) == 0 {
		return fmt.Errorf("email address local part cannot be empty")
	}

	// Check domain part (after @)
	if len(domainPart) == 0 {
		return fmt.Errorf("email address domain cannot be empty")
	}

	// Check for at least one dot in domain
	if !strings.Contains(domainPart, ".") {
		return fmt.Errorf("email address domain must contain at least one dot")
	}

	// Check domain parts
	domainParts := strings.Split(domainPart, ".")
	if len(domainParts) < 2 {
		return fmt.Errorf("email address domain must have at least two parts")
	}

	// Check that TLD is at least 2 characters
	tld := domainParts[len(domainParts)-1]
	if len(tld) < 2 {
		return fmt.Errorf("email address TLD must be at least 2 characters")
	}

	return nil
}

// ValidateRequiredMultipart validates that a required multipart form field has at least one value
func ValidateRequiredMultipart(form *multipart.Form, fieldName, displayName string) ([]string, error) {
	values := form.Value[fieldName]
	if len(values) == 0 {
		return nil, fmt.Errorf("please select at least one %s", displayName)
	}
	return values, nil
}

// ParseIntParam parses an integer parameter from the URL with consistent error handling
func ParseIntParam(c *fiber.Ctx, paramName string) (int, error) {
	value, err := c.ParamsInt(paramName)
	if err != nil {
		return 0, fiber.NewError(fiber.StatusBadRequest, "Invalid parameter: "+paramName)
	}
	return value, nil
}

// ValidationErrorResponse returns a validation error response
func ValidationErrorResponse(c *fiber.Ctx, message string) error {
	return render(c, ui.ValidationError(message))
}

// ValidationErrorResponseWithStatus returns a validation error response with custom status code
func ValidationErrorResponseWithStatus(c *fiber.Ctx, message string, statusCode int) error {
	c.Response().SetStatusCode(statusCode)
	return render(c, ui.ValidationError(message))
}

// ValidateAdForm validates the common ad form fields dynamically based on category configuration
func ValidateAdForm(form *multipart.Form, adCat int) error {
	// Get category fields to determine which fields to validate
	fieldConfigs, err := spec.GetCategoryFields(adCat)
	if err != nil {
		return fmt.Errorf("failed to get category fields: %w", err)
	}

	// Validate multi-value fields (keyword_array type)
	for _, fieldConfig := range fieldConfigs {
		if fieldConfig.QdrantType == "keyword_array" && fieldConfig.IsRequired {
			fieldName := fieldConfig.Name + "s" // Form uses plural
			values := form.Value[fieldName]
			if len(values) == 0 {
				return fmt.Errorf("please select at least one %s", fieldConfig.DisplayName)
			}
		}
	}

	return nil
}

// validateAdFormMultipart validates ad form and returns the values dynamically
func validateAdFormMultipart(form *multipart.Form, adCat int) (years, models, engines []string, err error) {
	// Get category fields to determine which fields to validate
	fieldConfigs, err := spec.GetCategoryFields(adCat)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get category fields: %w", err)
	}

	// Validate multi-value fields (keyword_array type)
	for _, fieldConfig := range fieldConfigs {
		if fieldConfig.QdrantType == "keyword_array" && fieldConfig.IsRequired {
			fieldName := fieldConfig.Name + "s" // Form fields use plural (years, models, engines)
			values := form.Value[fieldName]
			if len(values) == 0 {
				return nil, nil, nil, fmt.Errorf("please select at least one %s", fieldConfig.DisplayName)
			}
			// Store values generically - no need for hard-coded switch
			// The caller will extract what it needs
		}
	}

	return years, models, engines, nil
}

// ValidateAndParsePrice validates and parses a price field
func ValidateAndParsePrice(c *fiber.Ctx) (float64, error) {
	priceStr := c.FormValue("price")
	if priceStr == "" {
		return 0, fmt.Errorf("price is required")
	}

	price, err := strconv.ParseFloat(priceStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid price format")
	}

	if price < 0 {
		return 0, fmt.Errorf("price cannot be negative")
	}

	return price, nil
}

// Allowed image MIME types
var allowedImageMIMETypes = map[string]bool{
	"image/jpeg": true,
	"image/jpg":  true,
	"image/png":  true,
	"image/gif":  true,
}

// Allowed image file extensions
var allowedImageExtensions = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".gif":  true,
}

// ValidateImageFile validates a single image file
// Checks file extension, MIME type, and size
func ValidateImageFile(fileHeader *multipart.FileHeader) error {
	// Check file extension
	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if !allowedImageExtensions[ext] {
		return fmt.Errorf("file %s has invalid extension. Only .jpg, .jpeg, .png, and .gif are allowed", fileHeader.Filename)
	}

	// Check MIME type
	mimeType := fileHeader.Header.Get("Content-Type")
	if mimeType == "" {
		// If MIME type is not provided, infer from extension
		switch ext {
		case ".jpg", ".jpeg":
			mimeType = "image/jpeg"
		case ".png":
			mimeType = "image/png"
		case ".gif":
			mimeType = "image/gif"
		default:
			return fmt.Errorf("unable to determine MIME type for file %s", fileHeader.Filename)
		}
	}
	if !allowedImageMIMETypes[mimeType] {
		return fmt.Errorf("file %s has invalid MIME type %s. Only image/jpeg, image/png, and image/gif are allowed", fileHeader.Filename, mimeType)
	}

	// Check file size
	if fileHeader.Size > config.ServerUploadLimit {
		return fmt.Errorf("file %s exceeds maximum size of %d MB", fileHeader.Filename, config.ServerUploadLimit/(1024*1024))
	}

	// Check minimum file size (reject empty or suspiciously small files)
	minSize := int64(100) // 100 bytes minimum
	if fileHeader.Size < minSize {
		return fmt.Errorf("file %s is too small and may be invalid", fileHeader.Filename)
	}

	return nil
}

// ValidateImageFiles validates a collection of image files
// Checks file count, and validates each file
func ValidateImageFiles(files []*multipart.FileHeader) error {
	// Check file count
	maxFiles := config.MaxImagesPerAd
	if len(files) > maxFiles {
		return fmt.Errorf("too many images. Maximum %d images allowed per ad", maxFiles)
	}

	// Validate each file
	for _, fileHeader := range files {
		if err := ValidateImageFile(fileHeader); err != nil {
			return err
		}
	}

	return nil
}

// BuildAdFromForm builds an Ad struct from form data dynamically based on category field configuration
func BuildAdFromForm(c *fiber.Ctx, userID int, locationID int, adID ...int) (ad.AdDetail, []*multipart.FileHeader, []int, error) {
	adCat := cookie.GetAdCategory(c)

	title, err := ValidateCleanText(c, "title", "Title", 35)
	if err != nil {
		return ad.AdDetail{}, nil, nil, err
	}

	// Get category fields to determine which fields to validate
	fieldConfigs, err := spec.GetCategoryFields(adCat)
	if err != nil {
		return ad.AdDetail{}, nil, nil, fmt.Errorf("failed to get category fields: %w", err)
	}

	form, err := c.MultipartForm()
	if err != nil {
		return ad.AdDetail{}, nil, nil, err
	}

	// Validate and extract spec fields dynamically
	// Build a map to store field values generically
	fieldValues := make(map[string]interface{})

	for _, fieldConfig := range fieldConfigs {
		if fieldConfig.IsRequired {
			if fieldConfig.QdrantType == "keyword_array" {
				// Multi-value fields (years, models, engines, etc.)
				fieldName := fieldConfig.Name + "s" // Form uses plural
				values := form.Value[fieldName]
				if len(values) == 0 {
					return ad.AdDetail{}, nil, nil, fmt.Errorf("please select at least one %s", fieldConfig.DisplayName)
				}
				fieldValues[fieldConfig.Name] = values
			} else {
				// Single-value fields (make, breed, part_subcategory, etc.)
				value, err := ValidateRequired(c, fieldConfig.Name, fieldConfig.DisplayName)
				if err != nil {
					return ad.AdDetail{}, nil, nil, err
				}
				fieldValues[fieldConfig.Name] = value
			}
		}
	}

	// Extract values for AdDetail (temporary compatibility layer)
	var make string
	var years, models, engines []string
	if val, ok := fieldValues["make"]; ok {
		make = val.(string)
	}
	if val, ok := fieldValues["year"]; ok {
		years = val.([]string)
	}
	if val, ok := fieldValues["model"]; ok {
		models = val.([]string)
	}
	if val, ok := fieldValues["engine"]; ok {
		engines = val.([]string)
	}

	// Validate category as single field (always required for part categories)
	category, err := ValidateRequired(c, "category", "Category")
	if err != nil {
		return ad.AdDetail{}, nil, nil, err
	}

	// Validate subcategory as single field (always required for part categories)
	subcategoryName, err := ValidateRequired(c, "subcategory", "Subcategory")
	if err != nil {
		return ad.AdDetail{}, nil, nil, err
	}

	// Extract image files
	imageFiles := form.File["images"]

	// Validate image files before processing
	if len(imageFiles) > 0 {
		if err := ValidateImageFiles(imageFiles); err != nil {
			return ad.AdDetail{}, nil, nil, err
		}
	}

	description, err := ValidateCleanText(c, "description", "Description", 500)
	if err != nil {
		return ad.AdDetail{}, nil, nil, err
	}
	price, err := ValidateAndParsePrice(c)
	if err != nil {
		return ad.AdDetail{}, nil, nil, err
	}

	id := 0
	if len(adID) > 0 {
		id = adID[0]
	}

	// Parse deleted images
	deletedImagesStr := c.FormValue("deleted_images")
	deletedImages := []int{}
	if deletedImagesStr != "" {
		for _, s := range strings.Split(deletedImagesStr, ",") {
			if n, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
				deletedImages = append(deletedImages, n)
			}
		}
	}

	// Calculate final image count: new images - deleted images
	imageCount := len(imageFiles) - len(deletedImages)
	if imageCount < 0 {
		imageCount = 0
	}

	// Create a simple AdDetail struct with the specified category
	// This is a temporary compatibility layer - AdDetail has hard-coded fields
	newAd := ad.AdDetail{
		Ad: ad.Ad{
			ID:           id,
			AdCategoryID: adCat,
			Title:        title,
			Price:        price,
			UserID:       userID,
			ImageCount:   imageCount,
		},
		Description:     description,
		LocationID:      locationID,
		HasVector:       false, // New ads start without vectors
		PartCategory:    category,
		PartSubcategory: subcategoryName,
		Make:            make,
		Years:           years,
		Models:          models,
		Engines:         engines,
	}

	return newAd, imageFiles, deletedImages, nil
}
