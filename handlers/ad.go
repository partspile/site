package handlers

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/user"
	"github.com/parts-pile/site/vehicle"
	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"
)

func HandleNewAd(c *fiber.Ctx) error {
	currentUser := c.Locals("user").(*user.User)

	makes := vehicle.GetMakes()
	makeOptions := []g.Node{}
	for _, makeName := range makes {
		makeOptions = append(makeOptions,
			Option(Value(makeName), g.Text(makeName)),
		)
	}

	return render(c, ui.Page(
		"New Ad - Parts Pile",
		currentUser,
		c.Path(),
		[]g.Node{
			ui.PageHeader("Create New Ad"),
			Form(
				ID("newAdForm"),
				Class("space-y-6"),
				EncType("multipart/form-data"),
				ui.ValidationErrorContainer(),
				ui.FormGroup("Make", "make",
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
				ui.FormGroup("Description", "description",
					Textarea(
						ID("description"),
						Name("description"),
						Class("w-full p-2 border rounded"),
						Rows("4"),
					),
				),
				ui.FormGroup("Price", "price",
					Input(
						Type("number"),
						ID("price"),
						Name("price"),
						Class("w-full p-2 border rounded"),
						Step("0.01"),
					),
				),
				ui.StyledButton("Submit", ui.ButtonPrimary,
					Type("submit"),
					hx.Post("/api/new-ad"),
					hx.Encoding("multipart/form-data"),
					hx.Target("#result"),
				),
				ui.ResultContainer(),
			),
		},
	))
}

func HandleNewAdSubmission(c *fiber.Ctx) error {
	currentUser := c.Locals("user").(*user.User)

	// Validate make selection first
	if c.FormValue("make") == "" {
		return render(c, ui.ValidationError("Please select a make first"))
	}

	form, err := c.MultipartForm()
	if err != nil {
		return render(c, ui.ValidationError(err.Error()))
	}

	// Validate required selections
	if len(form.Value["years"]) == 0 {
		return render(c, ui.ValidationError("Please select at least one year"))
	}

	if len(form.Value["models"]) == 0 {
		return render(c, ui.ValidationError("Please select at least one model"))
	}

	if len(form.Value["engines"]) == 0 {
		return render(c, ui.ValidationError("Please select at least one engine size"))
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

	return render(c, ui.SuccessMessageWithRedirect("Ad created successfully", "/"))
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

	currentUser, _ := c.Locals("user").(*user.User)

	actionButtons := []g.Node{
		ui.BackToListingsButton(),
	}

	if currentUser != nil && currentUser.ID == ad.UserID {
		editButton := ui.StyledLink("Edit Ad", fmt.Sprintf("/edit-ad/%d", ad.ID), ui.ButtonPrimary)
		deleteButton := ui.DeleteButton(ad.ID)
		actionButtons = append(actionButtons, editButton, deleteButton)
	}

	return render(c, ui.Page(
		fmt.Sprintf("Ad %d - Parts Pile", ad.ID),
		currentUser,
		c.Path(),
		[]g.Node{
			Div(
				Class("max-w-2xl mx-auto"),
				ui.PageHeader(ad.Make),
				ui.AdDetails(ad),
				ui.ActionButtons(actionButtons...),
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
			ui.Checkbox("years", year, year, isChecked, false,
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
			ui.Checkbox("models", modelName, modelName, isChecked, !isAvailable,
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
			ui.Checkbox("engines", engineName, engineName, isChecked, !isAvailable),
		)
	}

	return render(c, ui.Page(
		"Edit Ad - Parts Pile",
		currentUser,
		c.Path(),
		[]g.Node{
			ui.PageHeader("Edit Ad"),
			Form(
				ID("editAdForm"),
				Class("space-y-6"),
				EncType("multipart/form-data"),
				ui.ValidationErrorContainer(),
				ui.FormGroup("Make", "make",
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
				ui.FormGroup("Years", "years", Div(ID("yearsDiv"), ui.GridContainer(5, yearCheckboxes...))),
				ui.FormGroup("Models", "models", Div(ID("modelsDiv"), ui.GridContainer(5, modelCheckboxes...))),
				ui.FormGroup("Engines", "engines", Div(ID("enginesDiv"), ui.GridContainer(5, engineCheckboxes...))),
				ui.FormGroup("Description", "description",
					Textarea(
						ID("description"),
						Name("description"),
						Class("w-full p-2 border rounded"),
						Rows("4"),
						g.Text(ad.Description),
					),
				),
				ui.FormGroup("Price", "price",
					Input(
						Type("number"),
						ID("price"),
						Name("price"),
						Class("w-full p-2 border rounded"),
						Step("0.01"),
						Value(fmt.Sprintf("%.2f", ad.Price)),
					),
				),
				ui.StyledButton("Submit", ui.ButtonPrimary,
					Type("submit"),
					hx.Post(fmt.Sprintf("/api/update-ad/%d", ad.ID)),
					hx.Encoding("multipart/form-data"),
					hx.Target("#result"),
				),
				ui.ResultContainer(),
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
		return render(c, ui.ValidationError(err.Error()))
	}

	if len(form.Value["years"]) == 0 || len(form.Value["models"]) == 0 || len(form.Value["engines"]) == 0 {
		return render(c, ui.ValidationError("Please make sure you have selected a year, model, and engine"))
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

	return render(c, ui.SuccessMessageWithRedirect("Ad updated successfully", fmt.Sprintf("/ad/%d", adID)))
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

	return render(c, ui.SuccessMessageWithRedirect("Ad deleted successfully", "/"))
}
