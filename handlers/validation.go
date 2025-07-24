package handlers

import (
	"fmt"
	"mime/multipart"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ad"
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

// ValidateRequiredMultipart validates that a required multipart form field has at least one value
func ValidateRequiredMultipart(form *multipart.Form, fieldName, displayName string) ([]string, error) {
	values := form.Value[fieldName]
	if len(values) == 0 {
		return nil, fmt.Errorf("Please select at least one %s", displayName)
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

// ValidateAdFormAndReturn validates ad form and returns the values
func ValidateAdFormAndReturn(form *multipart.Form) (years, models, engines []string, err error) {
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
		return 0, fmt.Errorf("Price is required")
	}

	price, err := strconv.ParseFloat(priceStr, 64)
	if err != nil {
		return 0, fmt.Errorf("Invalid price format")
	}

	if price < 0 {
		return 0, fmt.Errorf("Price cannot be negative")
	}

	return price, nil
}

// BuildAdFromForm builds an Ad struct from form data
func BuildAdFromForm(c *fiber.Ctx, userID int, locationID int, adID ...int) (ad.Ad, []*multipart.FileHeader, []int, error) {
	title, err := ValidateRequired(c, "title", "Title")
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
	years, models, engines, err := ValidateAdFormAndReturn(form)
	if err != nil {
		return ad.Ad{}, nil, nil, err
	}
	// Extract image files
	imageFiles := form.File["images"]
	// Don't require at least one image for edit
	description, err := ValidateRequired(c, "description", "Description")
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
	} else {
		id = ad.GetNextAdID()
	}
	imageOrderStr := c.FormValue("image_order")
	imageOrder := []int{}
	if imageOrderStr != "" {
		for _, s := range strings.Split(imageOrderStr, ",") {
			if n, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
				imageOrder = append(imageOrder, n)
			}
		}
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
	return ad.Ad{
		ID:          id,
		Title:       title,
		Make:        make,
		Years:       years,
		Models:      models,
		Engines:     engines,
		Description: description,
		Price:       price,
		UserID:      userID,
		LocationID:  locationID,
		ImageOrder:  imageOrder,
	}, imageFiles, deletedImages, nil
}
