package ad

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/qdrant/go-client/qdrant"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// FieldLocation handles the location field
type FieldLocation struct {
	adField
}

func (f *FieldLocation) BuildFilterCondition(c *fiber.Ctx) *qdrant.Condition {
	l := c.FormValue("location")
	if l == "" {
		return nil
	}

	rStr := c.FormValue("radius")
	var r float64 = 25 // default to 25 miles
	if rStr != "" {
		if _, err := fmt.Sscanf(rStr, "%f", &r); err != nil {
			r = 25 // default to 25 miles if invalid
		}
	}

	/*
		if locationResolver == nil {
			return nil
		}
		lat, lon, err := locationResolver(l)
		if err != nil {
			return nil
		}

	*/

	var lat, lon float64

	// Convert miles to meters and create geo radius condition
	rm := r * 1609.34
	return qdrant.NewGeoRadius("location", lat, lon, float32(rm))
}

func (f *FieldLocation) RenderNewAdField(selectedValues map[string]bool, defaultValue string) g.Node {
	// If field has a parent and no default value, render empty div for HTMX
	if f.ParentField != nil && defaultValue == "" {
		fieldID := f.FieldName + "Div"
		return Div(ID(fieldID))
	}

	return fieldInput(
		f.FieldName+"Div",
		f.FieldName,
		f.DisplayName,
		"text",
		defaultValue,
		f.IsRequired,
		"(Optional)",
	)
}

func (f *FieldLocation) RenderFilterField(params map[string]string) g.Node {
	locationValue := params["location"]
	radiusValue := params["radius"]
	if radiusValue == "" {
		radiusValue = "25"
	}

	return g.Group([]g.Node{
		// Location input
		Div(
			Label(Class("block text-sm font-medium mb-1"), g.Text("Location")),
			Input(
				Type("text"),
				Name("location"),
				ID("locationFilter"),
				Class("w-full p-2 border rounded-md"),
				Placeholder("City, State or ZIP"),
				Value(locationValue),
			),
		),
		// Radius dropdown
		Div(
			Label(Class("block text-sm font-medium mb-1"), g.Text("Radius")),
			Select(
				Name("radius"),
				ID("radiusFilter"),
				Class("w-full p-2 border rounded-md"),
				Option(Value("25"), g.Text("25 miles"), g.If(radiusValue == "25", Selected())),
				Option(Value("50"), g.Text("50 miles"), g.If(radiusValue == "50", Selected())),
				Option(Value("100"), g.Text("100 miles"), g.If(radiusValue == "100", Selected())),
				Option(Value("250"), g.Text("250 miles"), g.If(radiusValue == "250", Selected())),
				Option(Value("500"), g.Text("500 miles"), g.If(radiusValue == "500", Selected())),
			),
		),
	})
}

func (f *FieldLocation) Validate(value string) error {
	if f.IsRequired && value == "" {
		return fmt.Errorf("%s is required", f.DisplayName)
	}
	return nil
}
