package ad

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/qdrant/go-client/qdrant"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// FieldPartCategory handles the part_category field
type FieldPartCategory struct {
	adField
}

func (f *FieldPartCategory) BuildFilterCondition(c *fiber.Ctx) *qdrant.Condition {
	fieldValue := c.FormValue("part_category")
	if fieldValue == "" {
		return nil
	}
	return qdrant.NewMatch("part_category", fieldValue)
}

func (f *FieldPartCategory) RenderFilterField(params map[string]string) g.Node {
	value := params[f.name]

	// Get all available values for this field (filter fields are independent)
	values, err := GetFieldValues(f.adCategoryID, f.name, map[string]string{})
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
			Name(f.name),
			ID(f.name+"Filter"),
			Class("w-full p-2 border rounded-md"),
			g.Group(options),
		),
	)
}

func (f *FieldPartCategory) RenderNewAdField(selectedValues map[string]bool, fieldValues []string, defaultValue string) g.Node {
	return fieldSelect(
		f.name+"Div",
		f.name,
		f.DisplayName,
		fieldValues,
		defaultValue,
		f.IsRequired,
		f.NextFieldName,
	)
}

func (f *FieldPartCategory) Validate(value string) error {
	if f.IsRequired && value == "" {
		return fmt.Errorf("%s is required", f.DisplayName)
	}
	return nil
}
