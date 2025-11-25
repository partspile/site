package ad

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// FieldLocation handles the location field
type FieldLocation struct {
	adField
}

func (f *FieldLocation) BuildFilterCondition(c *fiber.Ctx) (string, []interface{}) {
	locationRaw := c.FormValue("location")
	if locationRaw == "" {
		return "", nil
	}

	lat, lon, err := ResolveLocationToCoords(locationRaw)
	if err != nil {
		return "", nil
	}

	rStr := c.FormValue("radius")
	var r float64 = 25 // default to 25 miles
	if rStr != "" {
		if _, err := fmt.Sscanf(rStr, "%f", &r); err != nil {
			r = 25 // default to 25 miles if invalid
		}
	}

	// Convert miles to meters for ST_Distance calculation
	radiusMeters := r * 1609.34

	// Use ST_Distance with PostGIS for accurate geographic distance calculation
	// ST_MakePoint(longitude, latitude) creates a point
	// Cast to geography for accurate distance calculations in meters
	// ST_Distance returns distance in meters for geography types
	return `ST_Distance(
		ST_MakePoint((vector_metadata->'location'->>'lon')::float,
		(vector_metadata->'location'->>'lat')::float)::geography,
		ST_MakePoint($1::float, $2::float)::geography
	) <= $3`,
		[]interface{}{lon, lat, radiusMeters}
}

func (f *FieldLocation) RenderNewAdField(parentValues map[string]string) g.Node {
	return fieldInput(
		f.FieldName+"Div",
		f.FieldName,
		f.DisplayName+" (Optional)",
		"text",
		f.IsRequired,
		"City, State or ZIP",
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
