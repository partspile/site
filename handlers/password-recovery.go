package handlers

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/local"
	"github.com/parts-pile/site/logger"
	"github.com/parts-pile/site/password"
	"github.com/parts-pile/site/sms"
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/user"
	"github.com/parts-pile/site/verification"
)

// HandlePasswordRecovery renders the password recovery step 1 page
func HandlePasswordRecovery(c *fiber.Ctx) error {
	userID := local.GetUserID(c)
	userName := local.GetUserName(c)
	csrfToken := local.GetCSRFToken(c)
	return render(c, ui.PasswordRecoveryPage(userID, userName, c.Path(), csrfToken))
}

// sendPasswordRecoveryCode attempts to send a verification code via SMS for password recovery.
// Always ensures minimum processing time to prevent timing attacks.
// Returns smsOptedOut status and an error if SMS sending fails (non-blocked errors).
func sendPasswordRecoveryCode(username string) (smsOptedOut bool, err error) {
	startTime := time.Now()

	// Try to get user
	u, err := user.GetUserByName(username)
	userExists := err == nil
	isAdmin := userExists && u.IsAdmin

	if userExists && !isAdmin {
		phone := u.Phone
		smsOptedOut = u.SMSOptedOut

		// Only send SMS if not opted out
		if !smsOptedOut {
			// Generate verification code
			code, err := verification.GenerateCode()
			if err != nil {
				logger.Error("Failed to generate verification code",
					"error", err)
				return smsOptedOut, fmt.Errorf("unable to generate verification code")
			}

			// Store verification code
			err = verification.CreateVerificationCode(phone, code)
			if err != nil {
				logger.Error("Failed to store verification code",
					"error", err, "phone", phone)
				return smsOptedOut, fmt.Errorf("unable to create verification code")
			}

			// Send SMS
			message := fmt.Sprintf("Your Parts Pile password recovery code is: %s. "+
				"This code expires in 10 minutes. Reply STOP to unsubscribe.", code)
			err = sms.SendMessage(phone, message)
			if err != nil {
				logger.Error("Failed to send SMS",
					"error", err, "phone", phone)
				// Check if this is a blocked number error
				if errors.Is(err, sms.ErrBlockedNumber) {
					smsOptedOut = true
				} else {
					return smsOptedOut, fmt.Errorf("unable to send verification code")
				}
			}
		}
	}

	// Prevent timing attacks: always ensure minimum processing time
	// SMS sending typically takes 200-800ms, add delay to match when SMS wasn't sent
	// This prevents attackers from determining if SMS was sent based on response time
	elapsed := time.Since(startTime)
	// Use 300-700ms range to simulate SMS API call time
	targetDelay := time.Duration(300+rand.Intn(400)) * time.Millisecond
	if elapsed < targetDelay {
		time.Sleep(targetDelay - elapsed)
	}

	return smsOptedOut, nil
}

// HandlePasswordRecoveryStep1 handles username submission and sends verification code
// Always proceeds to step 2, even if username doesn't exist or is admin (prevents user enumeration)
func HandlePasswordRecoveryStep1(c *fiber.Ctx) error {
	username := c.FormValue("username")
	username = strings.TrimSpace(username)

	// Validate username format
	if err := ValidateUsername(username); err != nil {
		return ValidationErrorResponse(c, err.Error())
	}

	// Send verification code (with timing attack protection)
	smsOptedOut, err := sendPasswordRecoveryCode(username)
	if err != nil {
		return ValidationErrorResponse(c, err.Error()+". Please try again.")
	}

	// Always proceed to step 2 (never reveal if user exists or is admin)
	// Return verification form with out-of-band swap to replace password-recovery-content
	return render(c, ui.PasswordRecoveryVerifyPageWithSwapOob(username, smsOptedOut))
}

// HandlePasswordRecoveryResend handles resending the verification code
func HandlePasswordRecoveryResend(c *fiber.Ctx) error {
	username := c.FormValue("username")
	username = strings.TrimSpace(username)

	if username == "" {
		return ValidationErrorResponse(c, "Username is required.")
	}

	// Send verification code (with timing attack protection)
	smsOptedOut, err := sendPasswordRecoveryCode(username)
	if err != nil {
		return ValidationErrorResponse(c, err.Error()+". Please try again.")
	}

	// Check if user exists to determine response message
	u, err := user.GetUserByName(username)
	userExists := err == nil
	isAdmin := userExists && u.IsAdmin

	// If opted out or user doesn't exist/admin, show generic message (don't reveal status)
	if smsOptedOut || !userExists || isAdmin {
		return render(c, ui.SuccessMessage("If your phone number is registered and SMS is enabled, a verification code has been sent.", "", 0))
	}

	return render(c, ui.SuccessMessage("Verification code sent successfully!", "", 0))
}

// HandlePasswordRecoveryStep2 handles verification code and password reset
// Shows generic error if user doesn't exist, is admin, or code is invalid
func HandlePasswordRecoveryStep2(c *fiber.Ctx) error {
	username := c.FormValue("username")
	code := c.FormValue("verification_code")
	userPassword := c.FormValue("password")
	password2 := c.FormValue("password2")

	if username == "" {
		return c.Redirect("/password-recovery")
	}

	if code == "" {
		return ValidationErrorResponse(c, "Please enter the verification code.")
	}

	// Try to get user
	u, err := user.GetUserByName(username)
	userExists := err == nil
	isAdmin := userExists && u.IsAdmin

	// If user doesn't exist or is admin, show generic error
	if !userExists || isAdmin {
		return ValidationErrorResponse(c, "Invalid or expired verification code. Please check your code and try again.")
	}

	// Validate the code
	valid, err := verification.ValidateCode(u.Phone, code)
	if err != nil {
		logger.Error("Code validation error",
			"error", err, "phone", u.Phone)
		return ValidationErrorResponse(c, "Invalid or expired verification code. Please check your code and try again.")
	}

	if !valid {
		return ValidationErrorResponse(c, "Invalid or expired verification code. Please check your code and try again.")
	}

	// Validate password confirmation
	if err := password.ValidatePasswordConfirmation(userPassword, password2); err != nil {
		return ValidationErrorResponse(c, err.Error())
	}

	// Validate password strength
	if err := password.ValidatePasswordStrength(userPassword); err != nil {
		return ValidationErrorResponse(c, err.Error())
	}

	// Hash new password
	hash, salt, err := password.HashPassword(userPassword)
	if err != nil {
		logger.Error("Failed to hash password",
			"error", err)
		return ValidationErrorResponse(c, "Server error, unable to reset your password.")
	}

	// Update password
	if _, err := user.UpdateUserPassword(u.ID, hash, salt, "argon2id"); err != nil {
		logger.Error("Failed to update password",
			"error", err, "userID", u.ID)
		return ValidationErrorResponse(c, "Failed to reset password. Please try again.")
	}

	// Invalidate verification codes
	err = verification.InvalidateVerificationCodes(u.Phone)
	if err != nil {
		logger.Error("Failed to cleanup verification codes",
			"error", err, "phone", u.Phone)
		// Don't fail password reset for cleanup errors
	}

	logger.Info("Password recovery successful",
		"userID", u.ID, "name", username)

	// Return success response with delay before redirecting to login
	redirectDelayMs := config.ServerRedirectDelay.Milliseconds()
	return render(c, ui.SuccessMessage("Password reset successful! Please log in with your new password.", "/login", redirectDelayMs))
}
