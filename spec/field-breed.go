package spec

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/qdrant/go-client/qdrant"
	g "maragu.dev/gomponents"
)

// FieldBreed handles the breed field
type FieldBreed struct {
	BaseField
}

func (f *FieldBreed) BuildFilterCondition(c *fiber.Ctx, _ LocationResolver) *qdrant.Condition {
	fieldValue := c.FormValue("breed")
	if fieldValue == "" {
		return nil
	}
	return qdrant.NewMatch("breed", fieldValue)
}

func (f *FieldBreed) RenderNewAdField(selectedValues map[string]bool, fieldValues []string, defaultValue string) g.Node {
	if f.FieldMultiValue {
		return fieldCheckboxGrid(
			f.Name+"Div",
			f.Name,
			f.DisplayName,
			fieldValues,
			selectedValues,
			f.NextFieldName,
		)
	}
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

func (f *FieldBreed) Validate(value string) error {
	if f.IsRequired && value == "" {
		return fmt.Errorf("%s is required", f.DisplayName)
	}
	return nil
}
