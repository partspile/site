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
	config := f.config
	if config.FieldMultiValue {
		return fieldCheckboxGrid(
			config.Name+"Div",
			config.Name,
			config.DisplayName,
			fieldValues,
			selectedValues,
			config.NextFieldName,
		)
	}
	return fieldSelect(
		config.Name+"Div",
		config.Name,
		config.DisplayName,
		fieldValues,
		defaultValue,
		config.IsRequired,
		config.NextFieldName,
	)
}

func (f *FieldBreed) Validate(value string) error {
	if f.config.IsRequired && value == "" {
		return fmt.Errorf("%s is required", f.config.DisplayName)
	}
	return nil
}
