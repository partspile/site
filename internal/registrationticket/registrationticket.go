package registrationticket

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/cookie"
	"github.com/partspile/site/internal/logger"
)

var ErrInvalid = errors.New("invalid or expired registration ticket")

type claims struct {
	Username string `json:"username"`
	PhoneE64 string `json:"phone_e64"`
	jwt.RegisteredClaims
}

// Issue sets an HttpOnly cookie proving OTP succeeded for username+phone.
func Issue(c *fiber.Ctx, username, phoneE64 string) error {
	token, err := sign(username, phoneE64)
	if err != nil {
		return err
	}
	cookie.SetRegisterTicket(c, token)
	logger.Info("Issued registration ticket",
		"component", "registrationticket", "phoneE64", phoneE64)
	return nil
}

// Consume verifies the cookie matches username+phone and clears it.
func Consume(c *fiber.Ctx, username, phoneE64 string) error {
	token := cookie.GetRegisterTicket(c)
	if token == "" {
		return ErrInvalid
	}

	cl, err := parse(token)
	cookie.ClearRegisterTicket(c)
	if err != nil {
		return ErrInvalid
	}

	if subtle.ConstantTimeCompare([]byte(cl.Username), []byte(username)) != 1 ||
		subtle.ConstantTimeCompare([]byte(cl.PhoneE64), []byte(phoneE64)) != 1 {
		return ErrInvalid
	}

	logger.Info("Consumed registration ticket",
		"component", "registrationticket", "phoneE64", phoneE64)
	return nil
}

func sign(username, phoneE64 string) (string, error) {
	now := time.Now().UTC()
	cl := claims{
		Username: username,
		PhoneE64: phoneE64,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(config.RegistrationTicketTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, cl)
	signed, err := tok.SignedString(config.JWTSecret)
	if err != nil {
		return "", fmt.Errorf("sign registration ticket: %w", err)
	}
	return signed, nil
}

func parse(tokenString string) (*claims, error) {
	tok, err := jwt.ParseWithClaims(tokenString, &claims{},
		func(t *jwt.Token) (any, error) {
			if t.Method != jwt.SigningMethodHS256 {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return config.JWTSecret, nil
		})
	if err != nil {
		return nil, err
	}
	cl, ok := tok.Claims.(*claims)
	if !ok || !tok.Valid {
		return nil, ErrInvalid
	}
	if cl.Username == "" || cl.PhoneE64 == "" {
		return nil, ErrInvalid
	}
	return cl, nil
}
