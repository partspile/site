package password

import (
	"testing"
)

func TestHashPassword(t *testing.T) {
	password := "testpassword123"

	hash, salt, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	if hash == "" {
		t.Error("Hash should not be empty")
	}

	if salt == "" {
		t.Error("Salt should not be empty")
	}

	// Verify the password works
	if !VerifyPassword(password, hash, salt) {
		t.Error("Password verification failed")
	}

	// Verify wrong password fails
	if VerifyPassword("wrongpassword", hash, salt) {
		t.Error("Wrong password should not verify")
	}
}

func TestVerifyPassword(t *testing.T) {
	password := "testpassword123"
	hash, salt, _ := HashPassword(password)

	// Test correct password
	if !VerifyPassword(password, hash, salt) {
		t.Error("Correct password should verify")
	}

	// Test wrong password
	if VerifyPassword("wrongpassword", hash, salt) {
		t.Error("Wrong password should not verify")
	}

	// Test wrong salt
	wrongSalt, _ := GenerateSalt()
	if VerifyPassword(password, hash, wrongSalt) {
		t.Error("Wrong salt should not verify")
	}

	// Test invalid salt
	if VerifyPassword(password, hash, "invalid-salt") {
		t.Error("Invalid salt should not verify")
	}
}

func TestGenerateSalt(t *testing.T) {
	salt1, err := GenerateSalt()
	if err != nil {
		t.Fatalf("GenerateSalt failed: %v", err)
	}

	if salt1 == "" {
		t.Error("Generated salt should not be empty")
	}

	// Generate another salt to ensure they're different
	salt2, err := GenerateSalt()
	if err != nil {
		t.Fatalf("GenerateSalt failed: %v", err)
	}

	if salt1 == salt2 {
		t.Error("Generated salts should be different")
	}
}

func TestHashPasswordWithSalt(t *testing.T) {
	password := "testpassword123"
	salt, _ := GenerateSalt()

	hash, err := HashPasswordWithSalt(password, salt)
	if err != nil {
		t.Fatalf("HashPasswordWithSalt failed: %v", err)
	}

	if hash == "" {
		t.Error("Hash should not be empty")
	}

	// Verify the password works
	if !VerifyPassword(password, hash, salt) {
		t.Error("Password verification failed")
	}
}

func TestValidatePasswordConfirmation(t *testing.T) {
	password := "testpassword123"
	confirm := "testpassword123"

	// Test matching passwords
	if err := ValidatePasswordConfirmation(password, confirm); err != nil {
		t.Errorf("Matching passwords should not error: %v", err)
	}

	// Test non-matching passwords
	if err := ValidatePasswordConfirmation(password, "different"); err == nil {
		t.Error("Non-matching passwords should error")
	}
}

func TestValidatePasswordStrength(t *testing.T) {
	// Test valid password
	if err := ValidatePasswordStrength("password123"); err != nil {
		t.Errorf("Valid password should not error: %v", err)
	}

	// Test too short password
	if err := ValidatePasswordStrength("pass1"); err == nil {
		t.Error("Short password should error")
	}

	// Test password without letters
	if err := ValidatePasswordStrength("12345678"); err == nil {
		t.Error("Password without letters should error")
	}

	// Test password without numbers
	if err := ValidatePasswordStrength("password"); err == nil {
		t.Error("Password without numbers should error")
	}
}

func TestValidatePasswordChange(t *testing.T) {
	current := "oldpassword123"
	newPass := "newpassword456"
	confirm := "newpassword456"

	// Test valid password change
	if err := ValidatePasswordChange(current, newPass, confirm); err != nil {
		t.Errorf("Valid password change should not error: %v", err)
	}

	// Test same password
	if err := ValidatePasswordChange(current, current, current); err == nil {
		t.Error("Same password should error")
	}

	// Test non-matching confirmation
	if err := ValidatePasswordChange(current, newPass, "different"); err == nil {
		t.Error("Non-matching confirmation should error")
	}

	// Test weak new password
	if err := ValidatePasswordChange(current, "weak", "weak"); err == nil {
		t.Error("Weak password should error")
	}
}
