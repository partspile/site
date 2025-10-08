package handlers

import (
	"database/sql"
	"fmt"
	"mime/multipart"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/part"
	"github.com/parts-pile/site/ui"
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

// ParseFormInt parses a form value as an integer with consistent error handling
func ParseFormInt(c *fiber.Ctx, fieldName string) (int, error) {
	value, err := strconv.Atoi(c.FormValue(fieldName))
	if err != nil {
		return 0, fiber.NewError(fiber.StatusBadRequest, "Invalid integer value for field: "+fieldName)
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

// ValidateAdForm validates the common ad form fields (years, models, engines)
func ValidateAdForm(form *multipart.Form) error {
	if _, err := ValidateRequiredMultipart(form, "years", "year"); err != nil {
		return err
	}
	if _, err := ValidateRequiredMultipart(form, "models", "model"); err != nil {
		return err
	}
	if _, err := ValidateRequiredMultipart(form, "engines", "engine size"); err != nil {
		return err
	}
	return nil
}

// validateAdFormMultipart validates ad form and returns the values
func validateAdFormMultipart(form *multipart.Form) (years, models, engines []string, err error) {
	years, err = ValidateRequiredMultipart(form, "years", "year")
	if err != nil {
		return nil, nil, nil, err
	}

	models, err = ValidateRequiredMultipart(form, "models", "model")
	if err != nil {
		return nil, nil, nil, err
	}

	engines, err = ValidateRequiredMultipart(form, "engines", "engine size")
	if err != nil {
		return nil, nil, nil, err
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

// BuildAdFromForm builds an Ad struct from form data
func BuildAdFromForm(c *fiber.Ctx, userID int, locationID int, adID ...int) (ad.Ad, []*multipart.FileHeader, []int, error) {
	title, err := ValidateCleanText(c, "title", "Title", 35)
	if err != nil {
		return ad.Ad{}, nil, nil, err
	}
	make, err := ValidateRequired(c, "make", "Make")
	if err != nil {
		return ad.Ad{}, nil, nil, err
	}
	form, err := c.MultipartForm()
	if err != nil {
		return ad.Ad{}, nil, nil, err
	}
	years, models, engines, err := validateAdFormMultipart(form)
	if err != nil {
		return ad.Ad{}, nil, nil, err
	}

	// Validate category as single field
	category, err := ValidateRequired(c, "category", "Category")
	if err != nil {
		return ad.Ad{}, nil, nil, err
	}

	// Validate subcategory as single field
	subcategoryName, err := ValidateRequired(c, "subcategory", "Subcategory")
	if err != nil {
		return ad.Ad{}, nil, nil, err
	}

	// Look up subcategory ID by name
	subcategoryID, err := part.GetSubCategoryIDByName(subcategoryName)
	if err != nil {
		return ad.Ad{}, nil, nil, fmt.Errorf("invalid subcategory: %s", subcategoryName)
	}

	// Extract image files
	imageFiles := form.File["images"]

	description, err := ValidateCleanText(c, "description", "Description", 500)
	if err != nil {
		return ad.Ad{}, nil, nil, err
	}
	price, err := ValidateAndParsePrice(c)
	if err != nil {
		return ad.Ad{}, nil, nil, err
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

	return ad.Ad{
		ID:            id,
		Title:         title,
		Make:          make,
		Years:         years,
		Models:        models,
		Engines:       engines,
		Category:      sql.NullString{String: category, Valid: category != ""},
		SubCategoryID: subcategoryID,
		Description:   description,
		Price:         price,
		UserID:        userID,
		LocationID:    locationID,
		ImageCount:    imageCount,
	}, imageFiles, deletedImages, nil
}
