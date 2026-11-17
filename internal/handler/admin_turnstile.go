package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/partspile/site/internal/logger"
	"github.com/partspile/site/internal/service/turnstile"
	"github.com/partspile/site/internal/ui"
)

func AdminTurnstileSubmitHandler(c *fiber.Ctx) error {
	token := c.FormValue("cf-turnstile-response")
	err := turnstile.VerifyAlways(token, VisitorIP(c))
	if err != nil {
		logger.Warn("Turnstile probe failed", "error", err)
		return render(c, ui.AdminTurnstileResult(false,
			"Cloudflare did not accept the token. Complete the check and try again."))
	}
	return render(c, ui.AdminTurnstileResult(true,
		"Cloudflare accepted the token."))
}
