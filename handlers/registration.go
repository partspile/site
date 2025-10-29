package handlers

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/grok"
	"github.com/parts-pile/site/password"
	"github.com/parts-pile/site/sms"
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/user"
	"github.com/parts-pile/site/verification"
)

// HandleRegistration handles the first step of registration (collecting user info)
func HandleRegistration(c *fiber.Ctx) error {
	// Log out any currently logged-in user since they're starting a new registration
	logoutUser(c)
	return render(c, ui.RegisterPage(0, "", c.Path()))
}

// HandleRegistrationStep1 handles the first step submission and sends SMS
func HandleRegistrationStep1(c *fiber.Ctx) error {
	name := c.FormValue("name")
	phone := c.FormValue("phone")
	phone = strings.TrimSpace(phone)

	// Validate username format
	if err := ValidateUsername(name); err != nil {
		return ValidationErrorResponse(c, err.Error())
	}

	if phone == "" {
		return ValidationErrorResponse(c, "Phone number is required.")
	}

	// Validate required checkbox
	offers := c.FormValue("offers")

	if offers != "true" {
		return ValidationErrorResponse(c, "You must agree to receive informational text messages to continue.")
	}

	// Validate phone format
	if strings.HasPrefix(phone, "+") {
		matched, _ := regexp.MatchString(`^\+[1-9][0-9]{7,14}$`, phone)
		if !matched {
			return ValidationErrorResponse(c, "Phone must be in international format, e.g. +12025550123")
		}
	} else {
		digits := regexp.MustCompile(`\D`).ReplaceAllString(phone, "")
		if len(digits) == 10 {
			phone = "+1" + digits
		} else {
			return ValidationErrorResponse(c, "US/Canada numbers must have 10 digits")
		}
	}

	// Check for existing username
	if _, err := user.GetUserByName(name); err == nil {
		return ValidationErrorResponse(c,
			"Unable to complete registration with these credentials. Please try different information.")
	}

	// GROK username screening
	systemPrompt := `Your job is to screen potential user names for a web site.
Reject user names that the general public would find offensive or inappropriate.
The user name is displayed on the site for other to see and interact with, so we
want polite names.

Unacceptable usernames:
- racial slurs
- hate speech
- explicit sexual content

If the user name is acceptable, return only: OK

If the user name is unacceptable, return a short, direct error message (1-2
sentences), and do not mention yourself, AI, or Grok in the response.

Only reject names that are truly offensive to a general audience.`

	userPrompt := `Screen the following user name for the web site: ` + name
	resp, err := grok.CallGrok(systemPrompt, userPrompt)
	if err != nil {
		return ValidationErrorResponse(c,
			"Unable to complete registration with these credentials. Please try different information.")
	}
	if resp != "OK" {
		return ValidationErrorResponse(c, resp)
	}

	// Check for existing phone (do this after username checks to avoid revealing phone existence prematurely)
	if _, err := user.GetUserByPhone(phone); err == nil {
		return ValidationErrorResponse(c,
			"Unable to complete registration with these credentials. Please try different information.")
	}

	// Generate verification code
	code, err := verification.GenerateCode()
	if err != nil {
		log.Printf("[REGISTRATION] Failed to generate verification code: %v", err)
		return ValidationErrorResponse(c, "Unable to generate verification code. Please try again.")
	}

	// Store verification code with registration name
	err = verification.CreateRegistrationVerificationCode(phone, code, name)
	if err != nil {
		log.Printf("[REGISTRATION] Failed to store verification code: %v", err)
		return ValidationErrorResponse(c, "Unable to create verification code. Please try again.")
	}

	// Send SMS
	message := fmt.Sprintf("Your Parts Pile verification code is: %s. "+
		"This code expires in 10 minutes. Reply STOP to unsubscribe.", code)
	err = sms.SendMessage(phone, message)
	if err != nil {
		log.Printf("[REGISTRATION] Failed to send SMS: %v", err)
		return ValidationErrorResponse(c, "Unable to send verification code. Please try again.")
	}

	// Render verification form content (content only, no Page wrapper)
	return render(c, ui.VerificationPageContent(name, phone))
}

// HandleRegistrationStep2 handles verification code submission and completes registration
func HandleRegistrationStep2(c *fiber.Ctx) error {
	// Get phone from form (stored in hidden field)
	phone := c.FormValue("reg_phone")
	if phone == "" {
		return c.Redirect("/register")
	}

	code := c.FormValue("verification_code")
	if code == "" {
		return ValidationErrorResponse(c, "Please enter the verification code.")
	}

	// Get the registration name from the verification record
	vc, err := verification.GetVerificationCode(phone)
	if err != nil {
		log.Printf("[REGISTRATION] Failed to get verification code: %v", err)
		return c.Redirect("/register")
	}

	name := vc.RegistrationName

	// Validate the code
	valid, err := verification.ValidateCode(phone, code)
	if err != nil {
		log.Printf("[REGISTRATION] Code validation error: %v", err)
		return ValidationErrorResponse(c, "Verification code validation failed. Please try again.")
	}

	if !valid {
		return ValidationErrorResponse(c, "Invalid or expired verification code. Please check your code and try again.")
	}

	// Get password from form
	userPassword := c.FormValue("password")
	password2 := c.FormValue("password2")

	if err := password.ValidatePasswordConfirmation(userPassword, password2); err != nil {
		return ValidationErrorResponse(c, err.Error())
	}

	// Validate password strength
	if err := password.ValidatePasswordStrength(userPassword); err != nil {
		return ValidationErrorResponse(c, err.Error())
	}

	// Validate terms acceptance
	terms := c.FormValue("terms")
	if terms != "accepted" {
		return ValidationErrorResponse(c, "You must accept the Terms of Service and Privacy Policy to continue.")
	}

	// Create the user
	hash, salt, err := password.HashPassword(userPassword)
	if err != nil {
		log.Printf("[REGISTRATION] Failed to hash password: %v", err)
		return ValidationErrorResponse(c, "Server error, unable to create your account.")
	}

	userID, err := user.CreateUser(name, phone, hash, salt, "argon2id")
	if err != nil {
		log.Printf("[REGISTRATION] Failed to create user: %v", err)
		return ValidationErrorResponse(c, "Unable to create account. Please try again.")
	}

	// Mark phone as verified
	err = user.MarkPhoneVerified(userID)
	if err != nil {
		log.Printf("[REGISTRATION] Failed to mark phone verified: %v", err)
		// Don't fail registration for this, but log it
	}

	// Clean up registration verification record
	err = verification.InvalidateVerificationCodes(phone)
	if err != nil {
		log.Printf("[REGISTRATION] Failed to cleanup verification codes: %v", err)
		// Don't fail registration for cleanup errors
	}

	// Get the newly created user to generate JWT
	u, err := user.GetUser(userID)
	if err != nil {
		log.Printf("[REGISTRATION] Failed to get newly created user: %v", err)
		return ValidationErrorResponse(c, "Registration completed but unable to log you in. Please log in manually.")
	}

	// Generate JWT token and log the user in
	if err := loginUser(c, &u); err != nil {
		log.Printf("[REGISTRATION] Failed to log in: %v", err)
		return ValidationErrorResponse(c, "Registration completed but unable to log you in. Please log in manually.")
	}

	log.Printf("[REGISTRATION] Registration successful: userID=%d, name=%s", userID, name)

	// Return success response with delay before redirecting to rocks page
	return render(c, ui.SuccessMessage("Registration successful! Redirecting to rocks page...", "/rocks"))
}
