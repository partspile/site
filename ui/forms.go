package ui

import (
	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"

	"github.com/parts-pile/site/part"
)

// ---- Form Components ----

func FormContainer(formID string, content ...g.Node) g.Node {
	return Form(
		ID(formID),
		Class("space-y-6"),
		ValidationErrorContainer(),
		g.Group(content),
	)
}

func FormGroup(labelText string, fieldID string, input g.Node) g.Node {
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

func PasswordInput(id, name string) g.Node {
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
	return FormGroup("Years", "years", GridContainer(5, checkboxes...))
}

func ModelsFormGroup(modelAvailability map[string]bool) g.Node {
	checkboxes := []g.Node{}
	for model, isAvailable := range modelAvailability {
		checkboxes = append(checkboxes,
			Checkbox("models", model, model, false, !isAvailable,
				hx.Trigger("change"),
				hx.Get("/api/engines"),
				hx.Target("#enginesDiv"),
				hx.Include("[name='make'],[name='years']:checked,[name='models']:checked"),
				hx.Swap("innerHTML"),
			),
		)
	}
	return FormGroup("Models", "models", GridContainer(5, checkboxes...))
}

func EnginesFormGroup(engineAvailability map[string]bool) g.Node {
	checkboxes := []g.Node{}
	for engine, isAvailable := range engineAvailability {
		checkboxes = append(checkboxes,
			Checkbox("engines", engine, engine, false, !isAvailable),
		)
	}
	return FormGroup("Engines", "engines", GridContainer(5, checkboxes...))
}

func CategoriesFormGroup(categories []string, selectedCategory string) g.Node {
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

	return FormGroup("Category", "category",
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

	return FormGroup("Subcategory", "subcategory",
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

	return FormGroup("Subcategory", "subcategory",
		Select(
			ID("subcategory"),
			Name("subcategory"),
			Class("w-full p-2 border rounded"),
			Required(),
			g.Group(options),
		),
	)
}
