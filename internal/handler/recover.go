package handler

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/partspile/site/internal/accountrecovery"
	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/cookie"
	"github.com/partspile/site/internal/logger"
	"github.com/partspile/site/internal/ui"
)

func setNoStore(c *fiber.Ctx) {
	c.Set("Cache-Control", "no-store")
}

func reusableRecoverSession(c *fiber.Ctx) (code string, expiresAt time.Time,
	ok bool) {
	token := cookie.GetRecoverSession(c)
	code = cookie.GetRecoverCode(c)
	if token == "" || len(code) != accountrecovery.CodeLength {
		return "", time.Time{}, false
	}
	st, err := accountrecovery.GetStatus(token)
	if err != nil {
		return "", time.Time{}, false
	}
	switch st.Kind {
	case accountrecovery.StatusPending,
		accountrecovery.StatusVerified,
		accountrecovery.StatusFailed:
		if st.ExpiresAt.IsZero() {
			return "", time.Time{}, false
		}
		return code, st.ExpiresAt, true
	default:
		return "", time.Time{}, false
	}
}

func RecoverHandler(c *fiber.Ctx) error {
	setNoStore(c)

	if config.TwilioFromNumber == "" && !config.AllowTestRegistration {
		return renderPage(c, "Recover account", ui.RecoverUnavailablePage())
	}

	if code, exp, ok := reusableRecoverSession(c); ok {
		return renderPage(c, "Recover account",
			ui.RecoverPage(code, config.TwilioFromNumber, exp))
	}

	if prev := cookie.GetRecoverSession(c); prev != "" {
		_ = accountrecovery.Cancel(prev)
		cookie.ClearRecoverSession(c)
	}

	session, err := accountrecovery.Start()
	if err != nil {
		logger.Error("Failed to start recovery session", "error", err)
		return fiber.NewError(fiber.StatusInternalServerError,
			"Unable to start account recovery. Please try again.")
	}

	cookie.SetRecoverSession(c, session.Token, session.Code)
	return renderPage(c, "Recover account",
		ui.RecoverPage(session.Code, config.TwilioFromNumber, session.ExpiresAt))
}

func RecoverStatusHandler(c *fiber.Ctx) error {
	setNoStore(c)

	token := cookie.GetRecoverSession(c)
	status, err := accountrecovery.GetStatus(token)
	if err != nil {
		logger.Error("Failed to get recovery status", "error", err)
		return fiber.NewError(fiber.StatusInternalServerError,
			"Unable to check recovery status")
	}

	switch status.Kind {
	case accountrecovery.StatusVerified:
		return render(c, ui.RecoverResetForm(status.Username))
	case accountrecovery.StatusFailed:
		return render(c, ui.RecoverFailedPanel(status.Message))
	case accountrecovery.StatusExpired:
		cookie.ClearRecoverSession(c)
		return render(c, ui.RecoverExpiredPanel())
	default:
		// Leave waiting panel (code + countdown) in place.
		return c.SendStatus(fiber.StatusNoContent)
	}
}

func RecoverPasswordHandler(c *fiber.Ctx) error {
	setNoStore(c)

	token := cookie.GetRecoverSession(c)
	newPassword := c.FormValue("password")
	confirmPassword := c.FormValue("password2")

	err := accountrecovery.ResetPassword(token, newPassword, confirmPassword)
	if err != nil {
		switch {
		case errors.Is(err, accountrecovery.ErrNotFound),
			errors.Is(err, accountrecovery.ErrExpired),
			errors.Is(err, accountrecovery.ErrNotVerified):
			cookie.ClearRecoverSession(c)
			return showErrorTo(c, ui.RecoverPasswordErrorID,
				"Recovery session expired. Please start again.")
		default:
			return showErrorTo(c, ui.RecoverPasswordErrorID, err.Error())
		}
	}

	cookie.ClearRecoverSession(c)
	c.Set("HX-Redirect", "/login")
	return c.SendStatus(fiber.StatusOK)
}
