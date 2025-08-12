package handlers

import (
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/parts-pile/site/grok"
	"github.com/parts-pile/site/password"
	"github.com/parts-pile/site/sms"
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/user"
	"github.com/parts-pile/site/verification"
)

// HandleRegistrationStep1 handles the first step of registration (collecting user info)
func HandleRegistrationStep1(c *fiber.Ctx) error {
	currentUser, _ := GetCurrentUser(c)
	return render(c, ui.RegisterPage(currentUser, c.Path()))
}

// HandleRegistrationStep1Submission handles the first step submission and sends SMS
func HandleRegistrationStep1Submission(c *fiber.Ctx) error {
	name := c.FormValue("name")
	phone := c.FormValue("phone")
	phone = strings.TrimSpace(phone)

	// Validate required fields
	if strings.TrimSpace(name) == "" {
		return ValidationErrorResponse(c, "Username is required.")
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

	// Check for existing username and phone
	if _, err := user.GetUserByName(name); err == nil {
		return ValidationErrorResponse(c, "Username already exists. Please choose a different username.")
	}

	if _, err := user.GetUserByPhone(phone); err == nil {
		return ValidationErrorResponse(c, "Phone number is already registered. Please use a different phone number.")
	}

	// GROK username screening
	systemPrompt := `You are an expert parts technician. Your job is to screen potential user names for the parts-pile web site.
Reject user names that the general public would find offensive.
Car-guy humor, double entendres, and puns are allowed unless they are widely considered offensive or hateful.
Examples of acceptable usernames:
- rusty nuts
- lugnut
- fast wrench
- shift happens

Examples of unacceptable usernames:
- racial slurs
- hate speech
- explicit sexual content

If the user name is acceptable, return only: OK
If the user name is unacceptable, return a short, direct error message (1-2 sentences), and do not mention yourself, AI, or Grok in the response.
Only reject names that are truly offensive to a general audience.`

	resp, err := grok.CallGrok(systemPrompt, name)
	if err != nil {
		return ValidationErrorResponse(c, "Could not validate username. Please try again later.")
	}
	if resp != "OK" {
		return ValidationErrorResponse(c, resp)
	}

	// Generate verification code
	code, err := verification.GenerateCode()
	if err != nil {
		log.Printf("[REGISTRATION] Failed to generate verification code: %v", err)
		return ValidationErrorResponse(c, "Unable to generate verification code. Please try again.")
	}

	// Store verification code
	err = verification.CreateVerificationCode(phone, code)
	if err != nil {
		log.Printf("[REGISTRATION] Failed to store verification code: %v", err)
		return ValidationErrorResponse(c, "Unable to create verification code. Please try again.")
	}

	// Send SMS
	smsService, err := sms.NewSMSService()
	if err != nil {
		log.Printf("[REGISTRATION] Failed to create SMS service: %v", err)
		// For development, you might want to use mock service here
		return ValidationErrorResponse(c, "Unable to send verification code. Please try again.")
	}

	messageSid, err := smsService.SendVerificationCode(phone, code)
	if err != nil {
		log.Printf("[REGISTRATION] Failed to send SMS: %v", err)
		return ValidationErrorResponse(c, "Unable to send verification code. Please try again.")
	}

	// Store registration data in session for step 2
	store := c.Locals("session_store").(*session.Store)
	sess, err := store.Get(c)
	if err != nil {
		log.Printf("[REGISTRATION] Failed to get session: %v", err)
		return ValidationErrorResponse(c, "Session error. Please try again.")
	}

	sess.Set("reg_name", name)
	sess.Set("reg_phone", phone)
	sess.Set("reg_step", "waiting")
	sess.Set("reg_message_sid", messageSid)

	if err := sess.Save(); err != nil {
		log.Printf("[REGISTRATION] Failed to save session: %v", err)
		return ValidationErrorResponse(c, "Session error. Please try again.")
	}

	// Wait for SMS delivery confirmation
	delivered, err := waitForSMSDelivery(messageSid, 30*time.Second)
	if err != nil {
		log.Printf("[REGISTRATION] SMS delivery wait failed: %v", err)
		return ValidationErrorResponse(c, "SMS delivery confirmation failed. Please try again.")
	}

	if !delivered {
		return ValidationErrorResponse(c, "SMS delivery failed. Please check your phone number and try again.")
	}

	// SMS delivered successfully, update session
	sess.Set("reg_step", "verification")
	if err := sess.Save(); err != nil {
		log.Printf("[REGISTRATION] Failed to save session: %v", err)
		return ValidationErrorResponse(c, "Session error. Please try again.")
	}

	// Return success response with redirect
	return render(c, ui.SuccessMessage("SMS delivered successfully! Redirecting to verification...", "/register/verify"))
}

// waitForSMSDelivery waits for SMS delivery confirmation with a timeout
func waitForSMSDelivery(messageSid string, timeout time.Duration) (bool, error) {
	start := time.Now()
	ticker := time.NewTicker(500 * time.Millisecond) // Check every 500ms
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			status := sms.GetGlobalStatusTracker().GetMessageStatus(messageSid)

			switch status {
			case sms.SMSStatusDelivered:
				return true, nil
			case sms.SMSStatusFailed, sms.SMSStatusUndelivered:
				return false, nil
			}

			// Check if we've exceeded the timeout
			if time.Since(start) > timeout {
				return false, nil
			}
		}
	}
}

// HandleRegistrationVerification shows the verification code input page
func HandleRegistrationVerification(c *fiber.Ctx) error {
	store := c.Locals("session_store").(*session.Store)
	sess, err := store.Get(c)
	if err != nil {
		return c.Redirect("/register")
	}

	regStep := sess.Get("reg_step")
	if regStep != "verification" {
		return c.Redirect("/register")
	}

	currentUser, _ := GetCurrentUser(c)
	return render(c, ui.VerificationPage(currentUser, c.Path()))
}

// HandleRegistrationStep2Submission handles verification code submission and completes registration
func HandleRegistrationStep2Submission(c *fiber.Ctx) error {
	store := c.Locals("session_store").(*session.Store)
	sess, err := store.Get(c)
	if err != nil {
		return c.Redirect("/register")
	}

	regStep := sess.Get("reg_step")
	if regStep != "verification" {
		return c.Redirect("/register")
	}

	name := sess.Get("reg_name").(string)
	phone := sess.Get("reg_phone").(string)
	code := c.FormValue("verification_code")

	if code == "" {
		return ValidationErrorResponse(c, "Please enter the verification code.")
	}

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

	// Clear registration session data
	sess.Delete("reg_name")
	sess.Delete("reg_phone")
	sess.Delete("reg_step")
	sess.Save()

	return render(c, ui.SuccessMessage("Registration successful! Your phone number has been verified.", "/login"))
}
