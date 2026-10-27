package handler

import (
	"slices"

	"github.com/gofiber/fiber/v2"
	"github.com/partspile/site/internal/ui"
)

var passwordFieldNames = []string{
	"password",
	"password2",
	"current_password",
	"new_password",
	"confirm_password",
}

// passwordFieldIDs are DOM id stems; may differ from Name when the same
// form field name appears more than once on a page.
var passwordFieldIDs = []string{
	"password",
	"password2",
	"current_password",
	"phone_current_password",
	"settings_current_password",
	"new_password",
	"confirm_password",
}

var passwordFieldAutocompletes = []string{
	"current-password",
	"new-password",
	"off",
}

func PasswordFieldHandler(c *fiber.Ctx) error {
	name := c.Query("name")
	if !slices.Contains(passwordFieldNames, name) {
		return fiber.ErrBadRequest
	}
	id := c.Query("id")
	if id == "" {
		id = name
	}
	if !slices.Contains(passwordFieldIDs, id) {
		return fiber.ErrBadRequest
	}
	autocomplete := c.Query("autocomplete")
	if !slices.Contains(passwordFieldAutocompletes, autocomplete) {
		return fiber.ErrBadRequest
	}
	visible := c.Query("visible") == "true"
	return render(c, ui.PasswordField(ui.PasswordFieldView{
		Name:            name,
		ID:              id,
		Autocomplete:    autocomplete,
		Value:           c.FormValue(name),
		Visible:         visible,
		PreventAutofill: c.Query("prevent_autofill") == "true",
	}))
}
