package cookie

import (
	"github.com/gofiber/fiber/v2"
	"github.com/partspile/site/internal/config"
)

const registerTicketCookie = "reg_ticket"

func SetRegisterTicket(c *fiber.Ctx, token string) {
	c.Cookie(&fiber.Cookie{
		Name:     registerTicketCookie,
		Value:    token,
		HTTPOnly: true,
		Secure:   config.CookieSecure,
		Path:     "/",
		SameSite: "Strict",
		MaxAge:   int(config.RegistrationTicketTTL.Seconds()),
	})
}

func ClearRegisterTicket(c *fiber.Ctx) {
	c.Cookie(&fiber.Cookie{
		Name:     registerTicketCookie,
		Value:    "",
		HTTPOnly: true,
		Secure:   config.CookieSecure,
		Path:     "/",
		SameSite: "Strict",
		MaxAge:   -1,
	})
}

func GetRegisterTicket(c *fiber.Ctx) string {
	return c.Cookies(registerTicketCookie)
}
