package handlers

import (
	"fmt"
	"mime/multipart"
	"regexp"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/ui"
	"golang.org/x/crypto/bcrypt"
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

// ValidatePasswordConfirmation validates that password and confirmation match
func ValidatePasswordConfirmation(password, confirmation string) error {
	if password != confirmation {
		return fmt.Errorf("Passwords do not match")
	}
	return nil
}

// VerifyPassword verifies a password against a bcrypt hash
func VerifyPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
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

// ValidateAndParsePrice validates the price field and returns the parsed float64 value or an error
func ValidateAndParsePrice(c *fiber.Ctx) (float64, error) {
	priceStr, err := ValidateRequired(c, "price", "Price")
	if err != nil {
		return 0, err
	}
	price, err := strconv.ParseFloat(priceStr, 64)
	if err != nil {
		return 0, fmt.Errorf("Price must be a valid number")
	}
	if price < 0 {
		return 0, fmt.Errorf("Price cannot be negative")
	}
	if !regexp.MustCompile(`^\d+(\.\d{1,2})?$`).MatchString(priceStr) {
		return 0, fmt.Errorf("Price must have at most two decimal places")
	}
	return price, nil
}

// BuildAdFromForm validates and constructs an ad.Ad from the form data
func BuildAdFromForm(c *fiber.Ctx, userID int, adID ...int) (ad.Ad, []*multipart.FileHeader, error) {
	title, err := ValidateRequired(c, "title", "Title")
	if err != nil {
		return ad.Ad{}, nil, err
	}
	make, err := ValidateRequired(c, "make", "Make")
	if err != nil {
		return ad.Ad{}, nil, err
	}
	form, err := c.MultipartForm()
	if err != nil {
		return ad.Ad{}, nil, err
	}
	years, models, engines, err := ValidateAdFormAndReturn(form)
	if err != nil {
		return ad.Ad{}, nil, err
	}
	description, err := ValidateRequired(c, "description", "Description")
	if err != nil {
		return ad.Ad{}, nil, err
	}
	price, err := ValidateAndParsePrice(c)
	if err != nil {
		return ad.Ad{}, nil, err
	}
	id := 0
	if len(adID) > 0 {
		id = adID[0]
	} else {
		id = ad.GetNextAdID()
	}
	// Extract image files
	imageFiles := form.File["images"]
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
	}, imageFiles, nil
}
