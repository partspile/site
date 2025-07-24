package password

import (
	"unicode"
)

// ValidationError represents a password validation error
type ValidationError struct {
	Message string
}

func (e ValidationError) Error() string {
	return e.Message
}

// ValidatePasswordConfirmation checks if password and confirmation match
func ValidatePasswordConfirmation(password, confirmation string) error {
	if password != confirmation {
		return ValidationError{Message: "Passwords do not match"}
	}
	return nil
}

// ValidatePasswordStrength checks if a password meets minimum requirements
func ValidatePasswordStrength(password string) error {
	if len(password) < 8 {
		return ValidationError{Message: "Password must be at least 8 characters long"}
	}

	// Check for at least one letter and one number
	hasLetter := false
	hasNumber := false

	for _, char := range password {
		if unicode.IsLetter(char) {
			hasLetter = true
		}
		if unicode.IsNumber(char) {
			hasNumber = true
		}
	}

	if !hasLetter {
		return ValidationError{Message: "Password must contain at least one letter"}
	}

	if !hasNumber {
		return ValidationError{Message: "Password must contain at least one number"}
	}

	return nil
}

// ValidatePasswordChange validates a password change operation
func ValidatePasswordChange(currentPassword, newPassword, confirmPassword string) error {
	// Check if current password is provided
	if currentPassword == "" {
		return ValidationError{Message: "Current password is required"}
	}

	// Check if new password is provided
	if newPassword == "" {
		return ValidationError{Message: "New password is required"}
	}

	// Check if new password is different from current
	if currentPassword == newPassword {
		return ValidationError{Message: "New password must be different from current password"}
	}

	// Validate password confirmation
	if err := ValidatePasswordConfirmation(newPassword, confirmPassword); err != nil {
		return err
	}

	// Validate new password strength
	if err := ValidatePasswordStrength(newPassword); err != nil {
		return err
	}

	return nil
}
