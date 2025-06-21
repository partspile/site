package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/user"
	"golang.org/x/crypto/bcrypt"
	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
)

func HandleLoginSubmission(c *fiber.Ctx) error {
	name := c.FormValue("name")
	password := c.FormValue("password")

	u, err := user.GetUserByName(name)
	if err != nil {
		c.Response().SetStatusCode(fiber.StatusUnauthorized)
		return render(c, ui.ValidationError("Invalid username or password"))
	}

	err = bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	if err != nil {
		c.Response().SetStatusCode(fiber.StatusUnauthorized)
		return render(c, ui.ValidationError("Invalid username or password"))
	}

	store := c.Locals("session_store").(*session.Store)
	sess, err := store.Get(c)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Server error, unable to log you in.")
	}

	sess.Set("userID", u.ID)

	if err := sess.Save(); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Server error, unable to save session.")
	}

	return render(c, ui.SuccessMessageWithRedirect("Login successful", "/"))
}

func HandleLogout(c *fiber.Ctx) error {
	store := c.Locals("session_store").(*session.Store)
	sess, err := store.Get(c)
	if err != nil {
		// can't get session, maybe it's already gone. redirect anyway.
		return c.Redirect("/", fiber.StatusSeeOther)
	}

	if err := sess.Destroy(); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Server error, unable to log you out.")
	}
	return render(c, ui.SuccessMessageWithRedirect("You have been logged out", "/"))
}

func GetCurrentUser(c *fiber.Ctx) (*user.User, error) {
	store := c.Locals("session_store").(*session.Store)
	sess, err := store.Get(c)
	if err != nil {
		return nil, err // No session
	}

	userID, ok := sess.Get("userID").(int)
	if !ok || userID == 0 {
		return nil, nil // No user ID in session
	}

	u, err := user.GetUserByID(userID)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// AuthRequired is a middleware that requires a user to be logged in.
func AuthRequired(c *fiber.Ctx) error {
	user, err := GetCurrentUser(c)
	if err != nil || user == nil {
		// You might want to redirect to login page
		return c.Redirect("/login", fiber.StatusSeeOther)
	}

	// Store user in context for downstream handlers
	c.Locals("user", user)

	return c.Next()
}

// OptionalAuth is a middleware that checks for a user but does not require one.
func OptionalAuth(c *fiber.Ctx) error {
	user, err := GetCurrentUser(c)
	if err == nil && user != nil {
		c.Locals("user", user)
	}
	return c.Next()
}

// AdminRequired is a middleware that requires a user to be an admin.
func AdminRequired(c *fiber.Ctx) error {
	user, err := GetCurrentUser(c)
	if err != nil || user == nil {
		return c.Redirect("/login", fiber.StatusSeeOther)
	}

	if !user.IsAdmin {
		return c.Status(fiber.StatusForbidden).SendString("Forbidden")
	}

	c.Locals("user", user)

	return c.Next()
}

func HandleRegister(c *fiber.Ctx) error {
	currentUser, _ := GetCurrentUser(c)
	return render(c, ui.Page(
		"Register",
		currentUser,
		c.Path(),
		[]g.Node{
			ui.PageHeader("Register"),
			ui.ContentContainer(
				ui.FormContainer("registerForm",
					ui.FormGroup("Username", "name",
						ui.TextInput("name", "name", ""),
					),
					ui.FormGroup("Phone Number", "phone",
						ui.TextInput("phone", "phone", ""),
					),
					ui.FormGroup("Password", "password",
						ui.PasswordInput("password", "password"),
					),
					ui.FormGroup("Confirm Password", "password2",
						ui.PasswordInput("password2", "password2"),
					),
					ui.ActionButtons(
						ui.StyledButton("Register", ui.ButtonPrimary,
							hx.Post("/api/register"),
							hx.Target("#result"),
							hx.Indicator("#registerForm"),
						),
					),
					ui.ResultContainer(),
				),
			),
		},
	))
}

func HandleRegisterSubmission(c *fiber.Ctx) error {
	name := c.FormValue("name")
	phone := c.FormValue("phone")
	password := c.FormValue("password")
	password2 := c.FormValue("password2")

	if password != password2 {
		return render(c, ui.ValidationError("Passwords do not match"))
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Server error, unable to create your account.")
	}

	if _, err := user.CreateUser(name, phone, string(hashedPassword)); err != nil {
		return render(c, ui.ValidationError("User already exists or another error occurred."))
	}

	return render(c, ui.SuccessMessageWithRedirect("Registration successful", "/login"))
}

func HandleLogin(c *fiber.Ctx) error {
	currentUser, _ := GetCurrentUser(c)
	return render(c, ui.Page(
		"Login",
		currentUser,
		c.Path(),
		[]g.Node{
			ui.PageHeader("Login"),
			ui.ContentContainer(
				ui.FormContainer("loginForm",
					ui.FormGroup("Username", "name",
						ui.TextInput("name", "name", ""),
					),
					ui.FormGroup("Password", "password",
						ui.PasswordInput("password", "password"),
					),
					ui.ActionButtons(
						ui.StyledButton("Login", ui.ButtonPrimary,
							hx.Post("/api/login"),
							hx.Target("#result"),
							hx.Indicator("#loginForm"),
						),
					),
					ui.ResultContainer(),
				),
			),
		},
	))
}

func HandleChangePassword(c *fiber.Ctx) error {
	currentPassword := c.FormValue("currentPassword")
	newPassword := c.FormValue("newPassword")
	confirmNewPassword := c.FormValue("confirmNewPassword")

	if newPassword != confirmNewPassword {
		return render(c, ui.ValidationError("New passwords do not match"))
	}

	currentUser := c.Locals("user").(*user.User)

	err := bcrypt.CompareHashAndPassword([]byte(currentUser.PasswordHash), []byte(currentPassword))
	if err != nil {
		return render(c, ui.ValidationError("Invalid current password"))
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Server error, unable to update password.")
	}

	if _, err := user.UpdateUserPassword(currentUser.ID, string(newHash)); err != nil {
		return render(c, ui.ValidationError("Failed to update password"))
	}
	return render(c, ui.SuccessMessage("Password changed successfully", ""))
}

func HandleDeleteAccount(c *fiber.Ctx) error {
	password := c.FormValue("password")

	currentUser := c.Locals("user").(*user.User)
	if currentUser == nil {
		c.Response().SetStatusCode(fiber.StatusUnauthorized)
		return render(c, ui.ValidationError("You must be logged in to delete your account"))
	}

	err := bcrypt.CompareHashAndPassword([]byte(currentUser.PasswordHash), []byte(password))
	if err != nil {
		c.Response().SetStatusCode(fiber.StatusUnauthorized)
		return render(c, ui.ValidationError("Invalid password"))
	}

	err = ad.DeleteAdsByUserID(currentUser.ID)
	if err != nil {
		c.Response().SetStatusCode(fiber.StatusInternalServerError)
		return render(c, ui.ValidationError("Could not delete ads for user"))
	}

	err = user.DeleteUser(currentUser.ID)
	if err != nil {
		c.Response().SetStatusCode(fiber.StatusInternalServerError)
		return render(c, ui.ValidationError("Could not delete user"))
	}

	return render(c, ui.SuccessMessageWithRedirect("Account deleted successfully", "/"))
}
