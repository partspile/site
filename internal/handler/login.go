package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/partspile/site/internal/cookie"
	"github.com/partspile/site/internal/local"
	"github.com/partspile/site/internal/logger"
	"github.com/partspile/site/internal/loginlimit"
	"github.com/partspile/site/internal/password"
	"github.com/partspile/site/internal/ui"
	"github.com/partspile/site/internal/user"
)

func LoginHandler(c *fiber.Ctx) error {
	returnPath := safeReturnPath(c.Query("return"))
	return renderPage(c, "Login", ui.LoginPage(returnPath))
}

func clearAuth(c *fiber.Ctx) {
	cookie.ClearJWT(c)
	cookie.ResetDistanceUnit(c)
	local.SetUserID(c, 0)
	local.SetUserName(c, "")
	local.SetUserIsAdmin(c, false)
}

func logout(c *fiber.Ctx) {
	clearAuth(c)
	cookie.ClearSearchState(c)
}

func LogoutHandler(c *fiber.Ctx) error {
	logout(c)
	return c.Redirect("/")
}

func LoginSubmitHandler(c *fiber.Ctx) error {
	userName := c.FormValue("username")
	passwd := c.FormValue("password")

	logger.Info("Login attempt", "userName", userName)

	if userName == "" {
		return showError(c, "Username is required")
	}
	if passwd == "" {
		return showError(c, "Password is required")
	}

	if err := loginlimit.Allow(userName); err != nil {
		return showError(c, err.Error())
	}

	u, err := user.GetByName(userName)
	if err != nil {
		loginlimit.RecordFailure(userName)
		return showError(c, "Invalid username or password")
	}

	if !password.VerifyPassword(passwd, u.PasswordHash, u.PasswordSalt) {
		loginlimit.RecordFailure(userName)
		return showError(c, "Invalid username or password")
	}

	loginlimit.Clear(userName)

	token, err := generateJWTToken(&u)
	if err != nil {
		logger.Error("Failed to generate token", "error", err)
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	cookie.SetJWT(c, token)
	cookie.SetDistanceUnitForUser(c, u.PhoneE64)

	c.Set("HX-Redirect", loginRedirectPath(c.FormValue("return")))
	return c.SendStatus(fiber.StatusOK)
}
