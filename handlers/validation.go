package handlers

import (
	"fmt"
	"mime/multipart"
	"strconv"

	"github.com/gofiber/fiber/v2"
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
		return 0, fiber.ErrBadRequest
	}
	return value, nil
}

// ParseFormInt parses a form value as an integer with consistent error handling
func ParseFormInt(c *fiber.Ctx, fieldName string) (int, error) {
	value, err := strconv.Atoi(c.FormValue(fieldName))
	if err != nil {
		return 0, fiber.ErrBadRequest
	}
	return value, nil
}

// ParseFormFloat parses a form value as a float64 with consistent error handling
func ParseFormFloat(c *fiber.Ctx, fieldName string) (float64, error) {
	value, err := strconv.ParseFloat(c.FormValue(fieldName), 64)
	if err != nil {
		return 0, fiber.ErrBadRequest
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

// ValidateOwnership checks if the current user owns the resource
func ValidateOwnership(resourceUserID, currentUserID int) error {
	if resourceUserID != currentUserID {
		return fiber.ErrForbidden
	}
	return nil
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
