package handler

import (
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/cookie"
	"github.com/partspile/site/internal/local"
	"github.com/partspile/site/internal/user"
)

// JWTMiddleware is a middleware that validates a JWT token and sets the user in the context.
func JWTMiddleware(c *fiber.Ctx) error {
	// Get JWT token from cookie
	tokenString := cookie.GetJWT(c)
	if tokenString == "" {
		// No token present - don't clear anything, just continue
		// (clearing would set a cookie in response even when none exists)
		return c.Next()
	}

	// Validate JWT token
	claims, err := validateJWTToken(tokenString)
	if err != nil {
		// Invalid token, clear cookie
		clearAuth(c)
		return c.Next()
	}

	userID := getUserID(claims)
	salt, isAdmin, ok := user.SessionAuth(userID)
	if !ok || salt != claims.PasswordSalt {
		// User gone or password changed — revoke session
		clearAuth(c)
		return c.Next()
	}

	local.SetUserID(c, userID)
	local.SetUserName(c, getUserName(claims))
	local.SetUserIsAdmin(c, isAdmin)

	return c.Next()
}

type claims struct {
	UserID       int    `json:"user_id"`
	UserName     string `json:"user_name"`
	PasswordSalt string `json:"pwd_salt"`
	jwt.RegisteredClaims
}

// generateJWTToken creates a JWT token for a user
func generateJWTToken(u *user.User) (string, error) {
	if u.PasswordSalt == "" {
		return "", errors.New("user password salt required for token")
	}
	claims := claims{
		UserID:       u.ID,
		UserName:     u.Name,
		PasswordSalt: u.PasswordSalt,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Subject:   strconv.Itoa(u.ID),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(config.JWTSecret)
}

// validateJWTToken validates a JWT token and returns the claims
func validateJWTToken(tokenString string) (*claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &claims{}, func(token *jwt.Token) (any, error) {
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

// getUserID extracts the user ID from validated claims
func getUserID(claims *claims) int {
	return claims.UserID
}

// getUserName extracts the username from validated claims
func getUserName(claims *claims) string {
	return claims.UserName
}
