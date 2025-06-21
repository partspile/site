package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/components"
	"github.com/parts-pile/site/grok"
	"github.com/parts-pile/site/part"
	"github.com/parts-pile/site/user"
	"github.com/parts-pile/site/vehicle"
	"golang.org/x/crypto/bcrypt"
	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"
)

func HandleLoginSubmission(c *fiber.Ctx) error {
	name := c.FormValue("name")
	password := c.FormValue("password")

	u, err := user.GetUserByName(name)
	if err != nil {
		c.Response().SetStatusCode(fiber.StatusUnauthorized)
		return render(c, components.ValidationError("Invalid username or password"))
	}

	err = bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	if err != nil {
		c.Response().SetStatusCode(fiber.StatusUnauthorized)
		return render(c, components.ValidationError("Invalid username or password"))
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

	c.Response().Header.Set("HX-Redirect", "/")
	return c.SendStatus(fiber.StatusOK)
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
	return c.Redirect("/", fiber.StatusSeeOther)
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

// render sets the content type to HTML and renders the component.
func render(c *fiber.Ctx, component g.Node) error {
	c.Set(fiber.HeaderContentType, fiber.MIMETextHTML)
	return component.Render(c.Response().BodyWriter())
}

func HandleHome(c *fiber.Ctx) error {
	currentUser, _ := GetCurrentUser(c) // this might return an error, but we ignore it, same as original.

	var newAdButton g.Node
	if currentUser != nil {
		newAdButton = components.StyledLink("New Ad", "/new-ad", components.ButtonPrimary)
	} else {
		newAdButton = components.StyledLinkDisabled("New Ad", components.ButtonPrimary)
	}

	return render(c, components.Page(
		"Parts Pile - Auto Parts and Sales",
		currentUser,
		c.Path(),
		[]g.Node{
			components.SearchWidget(newAdButton),
			components.InitialSearchResults(),
		},
	))
}

func HandleNewAd(c *fiber.Ctx) error {
	currentUser := c.Locals("user").(*user.User)

	makes := vehicle.GetMakes()
	makeOptions := []g.Node{}
	for _, makeName := range makes {
		makeOptions = append(makeOptions,
			Option(Value(makeName), g.Text(makeName)),
		)
	}

	return render(c, components.Page(
		"New Ad - Parts Pile",
		currentUser,
		c.Path(),
		[]g.Node{
			components.PageHeader("Create New Ad"),
			Form(
				ID("newAdForm"),
				Class("space-y-6"),
				components.ValidationErrorContainer(),
				components.FormGroup("Make", "make",
					Select(
						ID("make"),
						Name("make"),
						Class("w-full p-2 border rounded"),
						hx.Trigger("change"),
						hx.Get("/api/years"),
						hx.Target("#yearsDiv"),
						hx.Include("this"),
						g.Attr("onchange", "document.getElementById('modelsDiv').innerHTML = ''; document.getElementById('enginesDiv').innerHTML = '';"),
						Option(Value(""), g.Text("Select a make")),
						g.Group(makeOptions),
					),
				),
				Div(
					ID("yearsDiv"),
					Class("space-y-2"),
				),
				Div(
					ID("modelsDiv"),
					Class("space-y-2"),
				),
				Div(
					ID("enginesDiv"),
					Class("space-y-2"),
				),
				components.FormGroup("Description", "description",
					Textarea(
						ID("description"),
						Name("description"),
						Class("w-full p-2 border rounded"),
						Rows("4"),
					),
				),
				components.FormGroup("Price", "price",
					Input(
						Type("number"),
						ID("price"),
						Name("price"),
						Class("w-full p-2 border rounded"),
						Step("0.01"),
					),
				),
				components.StyledButton("Submit", components.ButtonPrimary,
					Type("submit"),
					hx.Post("/api/new-ad"),
					hx.Target("#result"),
				),
				components.ResultContainer(),
			),
		},
	))
}

func HandleNewAdSubmission(c *fiber.Ctx) error {
	currentUser := c.Locals("user").(*user.User)

	// Validate make selection first
	if c.FormValue("make") == "" {
		return render(c, components.ValidationError("Please select a make first"))
	}

	form, err := c.MultipartForm()
	if err != nil {
		return err
	}

	// Validate required selections
	if len(form.Value["years"]) == 0 {
		return render(c, components.ValidationError("Please select at least one year"))
	}

	if len(form.Value["models"]) == 0 {
		return render(c, components.ValidationError("Please select at least one model"))
	}

	if len(form.Value["engines"]) == 0 {
		return render(c, components.ValidationError("Please select at least one engine size"))
	}

	price := 0.0
	fmt.Sscanf(c.FormValue("price"), "%f", &price)

	make := c.FormValue("make")
	years := form.Value["years"]
	models := form.Value["models"]
	engines := form.Value["engines"]
	description := c.FormValue("description")

	newAd := ad.Ad{
		ID:          ad.GetNextAdID(),
		Make:        make,
		Years:       years,
		Models:      models,
		Engines:     engines,
		Description: description,
		Price:       price,
		UserID:      currentUser.ID,
	}

	ad.AddAd(newAd)

	c.Response().Header.Set("HX-Redirect", "/")
	return c.SendStatus(fiber.StatusOK)
}

func HandleViewAd(c *fiber.Ctx) error {
	adID, err := c.ParamsInt("id")
	if err != nil {
		return fiber.ErrBadRequest
	}

	ad, ok := ad.GetAd(adID)
	if !ok || ad.ID == 0 {
		return fiber.ErrNotFound
	}

	currentUser, _ := GetCurrentUser(c)
	var editButton, deleteButton g.Node
	if currentUser != nil && currentUser.ID == ad.UserID {
		editButton = components.StyledLink("Edit Ad", fmt.Sprintf("/edit-ad/%d", ad.ID), components.ButtonPrimary)
		deleteButton = components.DeleteButton(ad.ID)
	} else {
		editButton = components.StyledLinkDisabled("Edit Ad", components.ButtonPrimary)
		deleteButton = components.StyledLinkDisabled("Delete Ad", components.ButtonDanger)
	}

	return render(c, components.Page(
		fmt.Sprintf("Ad %d - Parts Pile", ad.ID),
		currentUser,
		c.Path(),
		[]g.Node{
			Div(
				Class("max-w-2xl mx-auto"),
				components.PageHeader(ad.Make),
				components.AdDetails(ad),
				components.ActionButtons(
					components.BackToListingsButton(),
					editButton,
					deleteButton,
				),
				Div(
					ID("result"),
					Class("mt-4"),
				),
			),
		},
	))
}

func HandleEditAd(c *fiber.Ctx) error {
	currentUser := c.Locals("user").(*user.User)

	adID, err := c.ParamsInt("id")
	if err != nil {
		return fiber.ErrBadRequest
	}

	ad, ok := ad.GetAd(adID)
	if !ok || ad.ID == 0 {
		return fiber.ErrNotFound
	}

	if ad.UserID != currentUser.ID {
		return fiber.ErrForbidden
	}

	// Prepare make options
	makes := vehicle.GetMakes()
	makeOptions := []g.Node{}
	for _, makeName := range makes {
		attrs := []g.Node{Value(makeName)}
		if makeName == ad.Make {
			attrs = append(attrs, Selected())
		}
		attrs = append(attrs, g.Text(makeName))
		makeOptions = append(makeOptions, Option(attrs...))
	}

	// Prepare year checkboxes
	years := vehicle.GetYears(ad.Make)
	yearCheckboxes := []g.Node{}
	for _, year := range years {
		isChecked := false
		for _, adYear := range ad.Years {
			if year == adYear {
				isChecked = true
				break
			}
		}
		yearCheckboxes = append(yearCheckboxes,
			components.Checkbox("years", year, year, isChecked, false,
				hx.Trigger("change"),
				hx.Get("/api/models"),
				hx.Target("#modelsDiv"),
				hx.Include("[name='make'],[name='years']:checked"),
			),
		)
	}

	// Prepare model checkboxes
	modelAvailability := vehicle.GetModelsWithAvailability(ad.Make, ad.Years)
	modelCheckboxes := []g.Node{}
	models := make([]string, 0, len(modelAvailability))
	for m := range modelAvailability {
		models = append(models, m)
	}
	sort.Strings(models)

	for _, modelName := range models {
		isAvailable := modelAvailability[modelName]
		isChecked := false
		for _, adModel := range ad.Models {
			if modelName == adModel {
				isChecked = true
				break
			}
		}
		modelCheckboxes = append(modelCheckboxes,
			components.Checkbox("models", modelName, modelName, isChecked, !isAvailable,
				hx.Trigger("change"),
				hx.Get("/api/engines"),
				hx.Target("#enginesDiv"),
				hx.Include("[name='make'],[name='years']:checked,[name='models']:checked"),
			),
		)
	}

	// Prepare engine checkboxes
	engineAvailability := vehicle.GetEnginesWithAvailability(ad.Make, ad.Years, ad.Models)
	engineCheckboxes := []g.Node{}
	engines := make([]string, 0, len(engineAvailability))
	for e := range engineAvailability {
		engines = append(engines, e)
	}
	sort.Strings(engines)

	for _, engineName := range engines {
		isAvailable := engineAvailability[engineName]
		isChecked := false
		for _, adEngine := range ad.Engines {
			if engineName == adEngine {
				isChecked = true
				break
			}
		}
		engineCheckboxes = append(engineCheckboxes,
			components.Checkbox("engines", engineName, engineName, isChecked, !isAvailable),
		)
	}

	return render(c, components.Page(
		"Edit Ad - Parts Pile",
		currentUser,
		c.Path(),
		[]g.Node{
			components.PageHeader("Edit Ad"),
			Form(
				ID("editAdForm"),
				Class("space-y-6"),
				components.ValidationErrorContainer(),
				components.FormGroup("Make", "make",
					Select(
						ID("make"),
						Name("make"),
						Class("w-full p-2 border rounded"),
						hx.Trigger("change"),
						hx.Get("/api/years"),
						hx.Target("#yearsDiv"),
						hx.Include("this"),
						g.Attr("onchange", "document.getElementById('modelsDiv').innerHTML = ''; document.getElementById('enginesDiv').innerHTML = '';"),
						Option(Value(""), g.Text("Select a make")),
						g.Group(makeOptions),
					),
				),
				components.FormGroup("Years", "years", Div(ID("yearsDiv"), Class("space-y-2"), g.Group(yearCheckboxes))),
				components.FormGroup("Models", "models", Div(ID("modelsDiv"), Class("space-y-2"), g.Group(modelCheckboxes))),
				components.FormGroup("Engines", "engines", Div(ID("enginesDiv"), Class("space-y-2"), g.Group(engineCheckboxes))),
				components.FormGroup("Description", "description",
					Textarea(
						ID("description"),
						Name("description"),
						Class("w-full p-2 border rounded"),
						Rows("4"),
						g.Text(ad.Description),
					),
				),
				components.FormGroup("Price", "price",
					Input(
						Type("number"),
						ID("price"),
						Name("price"),
						Class("w-full p-2 border rounded"),
						Step("0.01"),
						Value(fmt.Sprintf("%.2f", ad.Price)),
					),
				),
				components.StyledButton("Submit", components.ButtonPrimary,
					Type("submit"),
					hx.Post(fmt.Sprintf("/api/update-ad/%d", ad.ID)),
					hx.Target("#result"),
				),
				components.ResultContainer(),
			),
		},
	))
}

func HandleUpdateAdSubmission(c *fiber.Ctx) error {
	currentUser := c.Locals("user").(*user.User)

	adID, err := strconv.Atoi(c.Query("id"))
	if err != nil {
		return fiber.ErrBadRequest
	}

	existingAd, ok := ad.GetAd(adID)
	if !ok || existingAd.ID == 0 {
		return fiber.ErrNotFound
	}
	if existingAd.UserID != currentUser.ID {
		return fiber.ErrForbidden
	}

	form, err := c.MultipartForm()
	if err != nil {
		return err
	}

	if len(form.Value["years"]) == 0 || len(form.Value["models"]) == 0 || len(form.Value["engines"]) == 0 {
		return render(c, components.ValidationError("Please make sure you have selected a year, model, and engine"))
	}

	price := 0.0
	fmt.Sscanf(c.FormValue("price"), "%f", &price)

	updatedAd := ad.Ad{
		ID:          adID,
		Make:        c.FormValue("make"),
		Years:       form.Value["years"],
		Models:      form.Value["models"],
		Engines:     form.Value["engines"],
		Description: c.FormValue("description"),
		Price:       price,
		UserID:      currentUser.ID,
	}

	ad.UpdateAd(updatedAd)

	c.Response().Header.Set("HX-Redirect", fmt.Sprintf("/ad/%d", adID))
	return c.SendStatus(fiber.StatusOK)
}

func HandleDeleteAd(c *fiber.Ctx) error {
	currentUser := c.Locals("user").(*user.User)

	adID, err := c.ParamsInt("id")
	if err != nil {
		return fiber.ErrBadRequest
	}

	existingAd, ok := ad.GetAd(adID)
	if !ok {
		return fiber.ErrNotFound
	}
	if existingAd.UserID != currentUser.ID {
		return fiber.ErrForbidden
	}

	ad.DeleteAd(adID)

	c.Response().Header.Set("HX-Redirect", "/")
	return c.SendStatus(fiber.StatusOK)
}

func HandleAdminDashboard(c *fiber.Ctx) error {
	currentUser := c.Locals("user").(*user.User)
	return render(c, components.AdminDashboard(currentUser, c.Path()))
}

func HandleAdminUsers(c *fiber.Ctx) error {
	currentUser := c.Locals("user").(*user.User)
	users, err := user.GetAllUsers()
	if err != nil {
		return fiber.ErrInternalServerError
	}
	return render(c, components.AdminUsers(currentUser, c.Path(), users))
}

func HandleSetAdmin(c *fiber.Ctx) error {
	userID, err := strconv.Atoi(c.FormValue("user_id"))
	if err != nil {
		return fiber.ErrBadRequest
	}
	isAdmin := c.FormValue("is_admin") == "true"

	if err := user.SetAdmin(userID, isAdmin); err != nil {
		return fiber.ErrInternalServerError
	}

	users, err := user.GetAllUsers()
	if err != nil {
		return fiber.ErrInternalServerError
	}

	return render(c, components.AdminUserTable(users))
}

func HandleAdminAds(c *fiber.Ctx) error {
	return c.SendString("Admin Ads")
}

func HandleAdminTransactions(c *fiber.Ctx) error {
	return c.SendString("Admin Transactions")
}

func HandleAdminExport(c *fiber.Ctx) error {
	return c.SendString("Admin Export")
}

func HandleMakes(c *fiber.Ctx) error {
	makes := vehicle.GetMakes()
	return c.JSON(makes)
}

func HandleYears(c *fiber.Ctx) error {
	makeName := c.Query("make")
	if makeName == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Make is required")
	}

	years := vehicle.GetYears(makeName)
	checkboxes := []g.Node{}

	for _, year := range years {
		checkboxes = append(checkboxes,
			components.Checkbox("years", year, year, false, false,
				hx.Trigger("change"),
				hx.Get("/api/models"),
				hx.Target("#modelsDiv"),
				hx.Include("[name='make'],[name='years']:checked"),
				hx.Swap("innerHTML"),
				g.Attr("onclick", "document.getElementById('enginesDiv').innerHTML = ''"),
			),
		)
	}

	return render(c, components.FormGroup("Years", "years", components.GridContainer(5, checkboxes...)))
}

func HandleModels(c *fiber.Ctx) error {
	makeName := c.Query("make")
	if makeName == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Make is required")
	}

	q, err := url.ParseQuery(string(c.Request().URI().QueryString()))
	if err != nil {
		return err
	}
	years := q["years"]
	if len(years) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "At least one year is required")
	}

	modelAvailability := vehicle.GetModelsWithAvailability(makeName, years)
	checkboxes := []g.Node{}
	for model, isAvailable := range modelAvailability {
		checkboxes = append(checkboxes,
			components.Checkbox("models", model, model, false, !isAvailable,
				hx.Trigger("change"),
				hx.Get("/api/engines"),
				hx.Target("#enginesDiv"),
				hx.Include("[name='make'],[name='years']:checked,[name='models']:checked"),
				hx.Swap("innerHTML"),
			),
		)
	}

	return render(c, components.FormGroup("Models", "models", components.GridContainer(5, checkboxes...)))
}

func HandleEngines(c *fiber.Ctx) error {
	makeName := c.Query("make")
	if makeName == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Make is required")
	}

	q, err := url.ParseQuery(string(c.Request().URI().QueryString()))
	if err != nil {
		return err
	}
	years := q["years"]
	if len(years) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "At least one year is required")
	}

	models := q["models"]
	if len(models) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "At least one model is required")
	}

	engineAvailability := vehicle.GetEnginesWithAvailability(makeName, years, models)
	checkboxes := []g.Node{}
	for engine, isAvailable := range engineAvailability {
		checkboxes = append(checkboxes,
			components.Checkbox("engines", engine, engine, false, !isAvailable),
		)
	}
	return render(c, components.FormGroup("Engines", "engines", components.GridContainer(5, checkboxes...)))
}

func HandleRegister(c *fiber.Ctx) error {
	currentUser, _ := GetCurrentUser(c)
	return render(c, components.Page(
		"Register",
		currentUser,
		c.Path(),
		[]g.Node{
			components.PageHeader("Register"),
			components.ContentContainer(
				components.FormContainer("registerForm",
					components.FormGroup("Username", "name",
						components.TextInput("name", "name", ""),
					),
					components.FormGroup("Phone Number", "phone",
						components.TextInput("phone", "phone", ""),
					),
					components.FormGroup("Password", "password",
						components.PasswordInput("password", "password"),
					),
					components.FormGroup("Confirm Password", "password2",
						components.PasswordInput("password2", "password2"),
					),
					components.ActionButtons(
						components.StyledButton("Register", components.ButtonPrimary,
							hx.Post("/api/register"),
							hx.Target("#result"),
							hx.Indicator("#registerForm"),
						),
					),
					components.ResultContainer(),
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
		return render(c, components.ValidationError("Passwords do not match"))
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Server error, unable to create your account.")
	}

	if _, err := user.CreateUser(name, phone, string(hashedPassword)); err != nil {
		return render(c, components.ValidationError("User already exists or another error occurred."))
	}

	c.Response().Header.Set("HX-Redirect", "/login")
	return c.SendStatus(fiber.StatusOK)
}

func HandleLogin(c *fiber.Ctx) error {
	currentUser, _ := GetCurrentUser(c)
	return render(c, components.Page(
		"Login",
		currentUser,
		c.Path(),
		[]g.Node{
			components.PageHeader("Login"),
			components.ContentContainer(
				components.FormContainer("loginForm",
					components.FormGroup("Username", "name",
						components.TextInput("name", "name", ""),
					),
					components.FormGroup("Password", "password",
						components.PasswordInput("password", "password"),
					),
					components.ActionButtons(
						components.StyledButton("Login", components.ButtonPrimary,
							hx.Post("/api/login"),
							hx.Target("#result"),
							hx.Indicator("#loginForm"),
						),
					),
					components.ResultContainer(),
				),
			),
		},
	))
}

func HandleSearch(c *fiber.Ctx) error {
	userPrompt := c.Query("q")
	query, err := ParseSearchQuery(userPrompt)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Could not parse query")
	}

	ads, nextCursor, err := GetNextPage(query, nil, 10)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Could not get ads")
	}

	loc, _ := time.LoadLocation(c.Get("X-Timezone"))

	adsMap := make(map[int]ad.Ad)
	for _, ad := range ads {
		adsMap[ad.ID] = ad
	}

	// For the initial search, we render the whole container.
	render(c, components.SearchResultsContainer(components.SearchSchema(query), adsMap, loc))

	// Add the loader if there are more results
	if nextCursor != nil {
		nextCursorStr := EncodeCursor(*nextCursor)
		loaderURL := fmt.Sprintf("/search-page?q=%s&cursor=%s",
			htmlEscape(userPrompt),
			htmlEscape(nextCursorStr))
		loaderHTML := fmt.Sprintf(`<div id="loader" hx-get="%s" hx-trigger="revealed" hx-swap="outerHTML">Loading more...</div>`, loaderURL)
		fmt.Fprint(c.Response().BodyWriter(), loaderHTML)
	}
	return nil
}

func HandleSearchPage(c *fiber.Ctx) error {
	c.Set("Content-Type", "text/html")
	userPrompt := c.Query("q")
	cursorStr := c.Query("cursor")

	if cursorStr == "" {
		// This page should not be called without a cursor.
		return nil
	}

	cursor, err := DecodeCursor(cursorStr)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid cursor")
	}

	ads, nextCursor, err := GetNextPage(cursor.Query, &cursor, 10)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Could not get ads")
	}

	loc, _ := time.LoadLocation(c.Get("X-Timezone"))

	// For subsequent loads, we just render the new ad cards, and the next loader
	for _, ad := range ads {
		render(c, components.AdCard(ad, loc))
	}

	if nextCursor != nil {
		nextCursorStr := EncodeCursor(*nextCursor)
		loaderURL := fmt.Sprintf("/search-page?q=%s&cursor=%s",
			htmlEscape(userPrompt),
			htmlEscape(nextCursorStr))
		loaderHTML := fmt.Sprintf(`<div id="loader" hx-get="%s" hx-trigger="revealed" hx-swap="outerHTML">Loading more...</div>`, loaderURL)
		fmt.Fprint(c.Response().BodyWriter(), loaderHTML)
	}
	return nil
}

func HandleSettings(c *fiber.Ctx) error {
	currentUser := c.Locals("user").(*user.User)
	return render(c, components.SettingsPage(currentUser, c.Path()))
}

func HandleChangePassword(c *fiber.Ctx) error {
	currentPassword := c.FormValue("currentPassword")
	newPassword := c.FormValue("newPassword")
	confirmNewPassword := c.FormValue("confirmNewPassword")

	if newPassword != confirmNewPassword {
		return render(c, components.ValidationError("New passwords do not match"))
	}

	currentUser := c.Locals("user").(*user.User)

	err := bcrypt.CompareHashAndPassword([]byte(currentUser.PasswordHash), []byte(currentPassword))
	if err != nil {
		return render(c, components.ValidationError("Invalid current password"))
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Server error, unable to update password.")
	}

	if _, err := user.UpdateUserPassword(currentUser.ID, string(newHash)); err != nil {
		return render(c, components.ValidationError("Failed to update password"))
	}
	return render(c, components.SuccessMessage("Password changed successfully", ""))
}

func HandleDeleteAccount(c *fiber.Ctx) error {
	password := c.FormValue("password")

	currentUser := c.Locals("user").(*user.User)
	if currentUser == nil {
		c.Response().SetStatusCode(fiber.StatusUnauthorized)
		return render(c, components.ValidationError("You must be logged in to delete your account"))
	}

	err := bcrypt.CompareHashAndPassword([]byte(currentUser.PasswordHash), []byte(password))
	if err != nil {
		c.Response().SetStatusCode(fiber.StatusUnauthorized)
		return render(c, components.ValidationError("Invalid password"))
	}

	err = ad.DeleteAdsByUserID(currentUser.ID)
	if err != nil {
		c.Response().SetStatusCode(fiber.StatusInternalServerError)
		return render(c, components.ValidationError("Could not delete ads for user"))
	}

	err = user.DeleteUser(currentUser.ID)
	if err != nil {
		c.Response().SetStatusCode(fiber.StatusInternalServerError)
		return render(c, components.ValidationError("Could not delete user"))
	}

	c.Response().Header.Set("HX-Refresh", "true")
	return c.SendStatus(fiber.StatusOK)
}

const sysPrompt = `You are an expert vehicle parts assistant.

Your job is to extract a structured query from a user's search request.

Extract the make, years, models, engine sizes, category, and subcategory from
the user's search request.  Use your best judgement as a vehicle parts export
to fill out the structured query as much as possible.  When filling out the
structured query, only use values from the lists below, and not the user's values.
For example, if user entered "Ford", the structure query would use "FORD".

<Makes>
%s
</Makes>

<Years>
%s
</Years>

<Models>
%s
</Models>

<EngineSizes>
%s
</EngineSizes>

<Categories>
%s
</Categories>

<SubCategories>
%s
</SubCategories>

Return JSON encoding this Go structure with the vehicle parts data:

struct {
	Make        string
	Years       []string
	Models      []string
	EngineSizes []string
	Category    string
	SubCategory string
}

Only return the JSON.  Nothing else.
`

type SearchQuery = ad.SearchQuery

type SearchCursor = ad.SearchCursor

func EncodeCursor(c SearchCursor) string {
	jsonCursor, _ := json.Marshal(c)
	return base64.StdEncoding.EncodeToString(jsonCursor)
}

func DecodeCursor(s string) (SearchCursor, error) {
	var c SearchCursor
	if s == "" {
		return c, nil
	}
	jsonCursor, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return c, err
	}
	err = json.Unmarshal(jsonCursor, &c)
	return c, err
}

func ParseSearchQuery(q string) (SearchQuery, error) {
	if q == "" {
		return SearchQuery{}, nil
	}

	allMakes := vehicle.GetMakes()
	allYears := vehicle.GetYearRange()
	allModels := vehicle.GetAllModels()
	allEngineSizes := vehicle.GetAllEngineSizes()
	allCategories := part.GetAllCategories()
	allSubCategories := part.GetAllSubCategories()

	prompt := fmt.Sprintf(sysPrompt,
		strings.Join(allMakes, "\n"),
		strings.Join(allYears, "\n"),
		strings.Join(allModels, "\n"),
		strings.Join(allEngineSizes, "\n"),
		strings.Join(allCategories, "\n"),
		strings.Join(allSubCategories, "\n"),
	)

	var query SearchQuery
	resp, err := grok.CallGrok(prompt, q)
	if err != nil {
		return SearchQuery{}, fmt.Errorf("error grokking query: %w", err)
	}

	err = json.Unmarshal([]byte(resp), &query)
	if err != nil {
		return SearchQuery{}, fmt.Errorf("error unmarshalling grok response: %w", err)
	}

	return query, nil
}

func FilterAds(query SearchQuery, ads []ad.Ad) []ad.Ad {
	if query.Make == "" && len(query.Years) == 0 && len(query.Models) == 0 &&
		len(query.EngineSizes) == 0 && query.Category == "" && query.SubCategory == "" {
		return ads
	}
	var filteredAds []ad.Ad
	for _, ad := range ads {
		var makeMatch, yearMatch, modelMatch, engineMatch bool

		if query.Make == "" || ad.Make == query.Make {
			makeMatch = true
		}

		if len(query.Years) == 0 || anyStringInSlice(ad.Years, query.Years) {
			yearMatch = true
		}

		if len(query.Models) == 0 || anyStringInSlice(ad.Models, query.Models) {
			modelMatch = true
		}

		if len(query.EngineSizes) == 0 || anyStringInSlice(ad.Engines, query.EngineSizes) {
			engineMatch = true
		}

		if makeMatch && yearMatch && modelMatch && engineMatch {
			filteredAds = append(filteredAds, ad)
		}
	}
	return filteredAds
}

func GetNextPage(query SearchQuery, cursor *SearchCursor, limit int) ([]ad.Ad, *SearchCursor, error) {
	// Get filtered page from database
	ads, hasMore, err := ad.GetFilteredAdsPageDB(query, cursor, limit)
	if err != nil {
		return nil, nil, err
	}

	// Create next cursor if there are more results
	var nextCursor *SearchCursor
	if hasMore && len(ads) > 0 {
		last := ads[len(ads)-1]
		nextCursor = &SearchCursor{
			Query:      query,
			LastID:     last.ID,
			LastPosted: last.CreatedAt,
		}
	}

	return ads, nextCursor, nil
}

func anyStringInSlice(a, b []string) bool {
	for _, aVal := range a {
		for _, bVal := range b {
			if aVal == bVal {
				return true
			}
		}
	}
	return false
}

func htmlEscape(s string) string {
	return strings.ReplaceAll(s, `"`, `\"`)
}
