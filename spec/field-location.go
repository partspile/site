package spec

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/qdrant/go-client/qdrant"
	g "maragu.dev/gomponents"
)

// FieldLocation handles the location field
type FieldLocation struct {
	BaseField
}

func (f *FieldLocation) BuildFilterCondition(c *fiber.Ctx, locationResolver LocationResolver) *qdrant.Condition {
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

	if locationResolver == nil {
		return nil
	}
	lat, lon, err := locationResolver(l)
	if err != nil {
		return nil
	}

	// Convert miles to meters and create geo radius condition
	rm := r * 1609.34
	return qdrant.NewGeoRadius("location", lat, lon, float32(rm))
}

func (f *FieldLocation) RenderNewAdField(selectedValues map[string]bool, fieldValues []string, defaultValue string) g.Node {
	config := f.config
	return fieldInput(
		config.Name+"Div",
		config.Name,
		config.DisplayName,
		"text",
		defaultValue,
		config.IsRequired,
		"(Optional)",
	)
}

func (f *FieldLocation) RenderFilterField(params map[string]string) g.Node {
	// Handled by existing UI code (locationFilter, radiusFilter)
	return nil
}

func (f *FieldLocation) Validate(value string) error {
	if f.config.IsRequired && value == "" {
		return fmt.Errorf("%s is required", f.config.DisplayName)
	}
	return nil
}

