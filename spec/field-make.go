package spec

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/qdrant/go-client/qdrant"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// FieldMake handles the make field
type FieldMake struct {
	BaseField
}

func (f *FieldMake) BuildFilterCondition(c *fiber.Ctx, _ LocationResolver) *qdrant.Condition {
	fieldValue := c.FormValue("make")
	if fieldValue == "" {
		return nil
	}
	return qdrant.NewMatch("make", fieldValue)
}

func (f *FieldMake) RenderFilterField(params map[string]string) g.Node {
	value := params[f.Name]

	// Get all available values for this field (filter fields are independent)
	values, err := GetFieldValues(f.AdCategoryID, f.Name, map[string]string{})
	if err != nil {
		// If error, return empty dropdown
		values = []string{}
	}

	// Build options list
	var options []g.Node
	options = append(options, Option(Value(""), g.Text("All"), g.If(value == "", Selected())))
	for _, val := range values {
		optionAttrs := []g.Node{Value(val), g.Text(val)}
		if val == value {
			optionAttrs = append(optionAttrs, Selected())
		}
		options = append(options, Option(optionAttrs...))
	}

	return Div(
		Label(Class("block text-sm font-medium mb-1"), g.Text(f.DisplayName)),
		Select(
			Name(f.Name),
			ID(f.Name+"Filter"),
			Class("w-full p-2 border rounded-md"),
			g.Group(options),
		),
	)
}

func (f *FieldMake) RenderNewAdField(selectedValues map[string]bool, fieldValues []string, defaultValue string) g.Node {
	return fieldSelect(
		f.Name+"Div",
		f.Name,
		f.DisplayName,
		fieldValues,
		defaultValue,
		f.IsRequired,
		f.NextFieldName,
	)
}

func (f *FieldMake) Validate(value string) error {
	if f.IsRequired && value == "" {
		return fmt.Errorf("%s is required", f.DisplayName)
	}
	return nil
}
