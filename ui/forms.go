package ui

import (
	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"

	"github.com/parts-pile/site/part"
)

// ---- Form Components ----

func formContainer(formID string, content ...g.Node) g.Node {
	return Form(
		ID(formID),
		Class("space-y-6"),
		g.Group(content),
	)
}

func formGroup(labelText string, fieldID string, input g.Node) g.Node {
	return Div(
		Class("space-y-2"),
		Label(For(fieldID), Class("block"), g.Text(labelText)),
		input,
	)
}

func Checkbox(id string, value string, label string, checked bool, disabled bool, attrs ...g.Node) g.Node {
	inputAttrs := []g.Node{
		Type("checkbox"),
		Name(id),
		Value(value),
		ID(id + "-" + value),
	}

	if checked {
		inputAttrs = append(inputAttrs, Checked())
	}
	if disabled {
		inputAttrs = append(inputAttrs, Disabled())
		inputAttrs = append(inputAttrs, g.Attr("class", "opacity-50 cursor-not-allowed"))
	}

	inputAttrs = append(inputAttrs, attrs...)

	labelNode := Label(
		For(id+"-"+value),
		func() g.Node {
			if disabled {
				return Class("text-gray-400")
			}
			return g.Text("")
		}(),
		g.Text(label),
	)

	return Div(
		Class("flex items-center space-x-2"),
		Input(inputAttrs...),
		labelNode,
	)
}

func TextInput(id, name, value string) g.Node {
	return Input(
		Type("text"),
		ID(id),
		Name(name),
		Value(value),
		Class("w-full p-2 border rounded"),
	)
}

func passwordInput(id, name string) g.Node {
	return Input(
		Type("password"),
		ID(id),
		Name(name),
		Class("w-full p-2 border rounded"),
	)
}

func YearsFormGroup(years []string) g.Node {
	checkboxes := []g.Node{}
	for _, year := range years {
		checkboxes = append(checkboxes,
			Checkbox("years", year, year, false, false,
				hx.Trigger("change"),
				hx.Get("/api/models"),
				hx.Target("#modelsDiv"),
				hx.Include("[name='make'],[name='years']:checked"),
				hx.Swap("innerHTML"),
				g.Attr("onclick", "document.getElementById('enginesDiv').innerHTML = ''"),
			),
		)
	}
	return formGroup("Years", "years", GridContainer(5, checkboxes...))
}

func ModelsFormGroup(models []string) g.Node {
	checkboxes := []g.Node{}
	for _, model := range models {
		checkboxes = append(checkboxes,
			Checkbox("models", model, model, false, false,
				hx.Trigger("change"),
				hx.Get("/api/engines"),
				hx.Target("#enginesDiv"),
				hx.Include("[name='make'],[name='years']:checked,[name='models']:checked"),
				hx.Swap("innerHTML"),
			),
		)
	}
	return formGroup("Models", "models", GridContainer(5, checkboxes...))
}

func EnginesFormGroup(engines []string) g.Node {
	checkboxes := []g.Node{}
	for _, engine := range engines {
		checkboxes = append(checkboxes,
			Checkbox("engines", engine, engine, false, false),
		)
	}
	return formGroup("Engines", "engines", GridContainer(5, checkboxes...))
}

func catogorySelector(categories []string, selectedCategory string) g.Node {
	options := []g.Node{
		Option(Value(""), g.Text("Select a category")),
	}

	for _, category := range categories {
		attrs := []g.Node{Value(category), g.Text(category)}
		if category == selectedCategory {
			attrs = append(attrs, Selected())
		}
		options = append(options, Option(attrs...))
	}

	return formGroup("Category", "category",
		Select(
			ID("category"),
			Name("category"),
			Class("w-full p-2 border rounded"),
			Required(),
			hx.Trigger("change"),
			hx.Get("/api/subcategories"),
			hx.Target("#subcategoriesDiv"),
			hx.Include("this"),
			g.Group(options),
		),
	)
}

func SubCategoriesFormGroup(subCategories []string, selectedSubCategory string) g.Node {
	options := []g.Node{
		Option(Value(""), g.Text("Select a subcategory")),
	}

	for _, subCategory := range subCategories {
		attrs := []g.Node{Value(subCategory), g.Text(subCategory)}
		if subCategory == selectedSubCategory {
			attrs = append(attrs, Selected())
		}
		options = append(options, Option(attrs...))
	}

	return formGroup("Subcategory", "subcategory",
		Select(
			ID("subcategory"),
			Name("subcategory"),
			Class("w-full p-2 border rounded"),
			Required(),
			g.Group(options),
		),
	)
}

func SubCategoriesFormGroupFromStruct(subCategories []part.SubCategory, selectedSubCategory string) g.Node {
	options := []g.Node{
		Option(Value(""), g.Text("Select a subcategory")),
	}

	for _, subCategory := range subCategories {
		attrs := []g.Node{Value(subCategory.Name), g.Text(subCategory.Name)}
		if subCategory.Name == selectedSubCategory {
			attrs = append(attrs, Selected())
		}
		options = append(options, Option(attrs...))
	}

	return formGroup("Subcategory", "subcategory",
		Select(
			ID("subcategory"),
			Name("subcategory"),
			Class("w-full p-2 border rounded"),
			Required(),
			g.Group(options),
		),
	)
}

// notificationMethodRadioGroup creates radio buttons for selecting notification method
func notificationMethodRadioGroup(selectedMethod string, emailAddress *string, phoneNumber string) g.Node {
	radioButtons := []g.Node{
		Div(Class("flex items-center"),
			Input(Type("radio"), Name("notificationMethod"), Value("sms"), ID("notificationMethod-sms"), g.If(selectedMethod == "sms", Checked()), Class("mr-2"),
				hx.Post("/api/notification-method-changed"),
				hx.Target("#emailField"),
				hx.Swap("innerHTML"),
				hx.Include("this"),
			),
			Label(For("notificationMethod-sms"), g.Text("Text to "+phoneNumber)),
		),
		Div(Class("flex items-center"),
			Input(Type("radio"), Name("notificationMethod"), Value("email"), ID("notificationMethod-email"), g.If(selectedMethod == "email", Checked()), Class("mr-2"),
				hx.Post("/api/notification-method-changed"),
				hx.Target("#emailField"),
				hx.Swap("innerHTML"),
				hx.Include("this"),
			),
			Label(For("notificationMethod-email"), g.Text("Email to:")),
		),
		Div(ID("emailField"), Class("ml-6 mt-2"),
			g.If(selectedMethod == "email",
				Input(
					Type("text"),
					ID("emailAddress"),
					Name("emailAddress"),
					Placeholder("Enter email address"),
					Value(func() string {
						if emailAddress != nil {
							return *emailAddress
						}
						return ""
					}()),
					Class("w-full p-2 border rounded"),
					Required(),
				),
			),
			g.If(selectedMethod != "email",
				Input(
					Type("text"),
					ID("emailAddress"),
					Name("emailAddress"),
					Placeholder("Enter email address"),
					Value(func() string {
						if emailAddress != nil {
							return *emailAddress
						}
						return ""
					}()),
					Class("w-full p-2 border rounded opacity-50 cursor-not-allowed"),
					Disabled(),
				),
			),
		),
		Div(Class("flex items-center"),
			Input(Type("radio"), Name("notificationMethod"), Value("signal"), ID("notificationMethod-signal"), g.If(selectedMethod == "signal", Checked()), Class("mr-2"),
				hx.Post("/api/notification-method-changed"),
				hx.Target("#emailField"),
				hx.Swap("innerHTML"),
				hx.Include("this"),
			),
			Label(For("notificationMethod-signal"), g.Text("Signal")),
		),
	}

	return formGroup("Notification Method", "notificationMethod",
		Div(Class("space-y-3"),
			g.Group(radioButtons),
		),
	)
}
