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

	// Check for at least one letter, one number, one uppercase, one lowercase, and one special character
	hasLetter := false
	hasNumber := false
	hasUpper := false
	hasLower := false
	hasSpecial := false

	for _, char := range password {
		if unicode.IsLetter(char) {
			hasLetter = true
			if unicode.IsUpper(char) {
				hasUpper = true
			}
			if unicode.IsLower(char) {
				hasLower = true
			}
		}
		if unicode.IsNumber(char) {
			hasNumber = true
		}
		// Check for special characters (printable ASCII that's not letter, number, or space)
		if !unicode.IsLetter(char) && !unicode.IsNumber(char) && !unicode.IsSpace(char) && char >= 33 && char <= 126 {
			hasSpecial = true
		}
	}

	if !hasLetter {
		return ValidationError{Message: "Password must contain at least one letter"}
	}

	if !hasNumber {
		return ValidationError{Message: "Password must contain at least one number"}
	}

	if !hasUpper {
		return ValidationError{Message: "Password must contain at least one uppercase letter"}
	}

	if !hasLower {
		return ValidationError{Message: "Password must contain at least one lowercase letter"}
	}

	if !hasSpecial {
		return ValidationError{Message: "Password must contain at least one special character (e.g., !@#$%^&*)"}
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
