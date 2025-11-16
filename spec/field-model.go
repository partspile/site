package spec

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/qdrant/go-client/qdrant"
	g "maragu.dev/gomponents"
)

// FieldModel handles the model field
type FieldModel struct {
	BaseField
}

func (f *FieldModel) BuildFilterCondition(c *fiber.Ctx, _ LocationResolver) *qdrant.Condition {
	fieldValue := c.FormValue("model")
	if fieldValue == "" {
		return nil
	}
	return qdrant.NewMatchKeywords("model", fieldValue)
}

func (f *FieldModel) RenderNewAdField(selectedValues map[string]bool, fieldValues []string, defaultValue string) g.Node {
	config := f.config
	return fieldCheckboxGrid(
		config.Name+"Div",
		config.Name,
		config.DisplayName,
		fieldValues,
		selectedValues,
		config.NextFieldName,
	)
}

func (f *FieldModel) Validate(value string) error {
	if f.config.IsRequired && value == "" {
		return fmt.Errorf("%s is required", f.config.DisplayName)
	}
	return nil
}
