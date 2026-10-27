package cookie

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/partspile/site/internal/config"
)

const recoverSessionCookie = "recover_session"

func SetRecoverSession(c *fiber.Ctx, token, code string) {
	c.Cookie(&fiber.Cookie{
		Name:     recoverSessionCookie,
		Value:    token + ":" + code,
		HTTPOnly: true,
		Secure:   config.CookieSecure,
		Path:     "/",
		SameSite: "Strict",
		MaxAge:   int(config.RecoverySessionTTL.Seconds()),
	})
}

func ClearRecoverSession(c *fiber.Ctx) {
	c.Cookie(&fiber.Cookie{
		Name:     recoverSessionCookie,
		Value:    "",
		HTTPOnly: true,
		Secure:   config.CookieSecure,
		Path:     "/",
		SameSite: "Strict",
		MaxAge:   -1,
	})
}

func GetRecoverSession(c *fiber.Ctx) string {
	token, _, _ := strings.Cut(c.Cookies(recoverSessionCookie), ":")
	return token
}

func GetRecoverCode(c *fiber.Ctx) string {
	_, code, ok := strings.Cut(c.Cookies(recoverSessionCookie), ":")
	if !ok {
		return ""
	}
	return code
}
