package jwt

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/user"
)

type claims struct {
	UserID   int    `json:"user_id"`
	UserName string `json:"user_name"`
	jwt.RegisteredClaims
}

// GenerateToken creates a JWT token for a user
func GenerateToken(u *user.User) (string, error) {
	claims := claims{
		UserID:   u.ID,
		UserName: u.Name,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Subject:   string(rune(u.ID)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(config.JWTSecret)
}

// ValidateToken validates a JWT token and returns the claims
func ValidateToken(tokenString string) (*claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &claims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected signing method")
		}
		return config.JWTSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// GetUserID extracts the user ID from validated claims
func GetUserID(claims *claims) int {
	return claims.UserID
}

// GetUserName extracts the username from validated claims
func GetUserName(claims *claims) string {
	return claims.UserName
}

// ValidateSecret validates that JWT secret is set and sufficiently strong
// Minimum requirements: at least 32 bytes (256 bits) of entropy
func ValidateSecret(secret []byte) error {
	if len(secret) == 0 {
		return fmt.Errorf("JWT_SECRET environment variable is required but not set")
	}

	// Require at least 32 bytes (256 bits) for HS256
	minLength := 32
	if len(secret) < minLength {
		return fmt.Errorf("JWT_SECRET must be at least %d characters long for security", minLength)
	}

	return nil
}
