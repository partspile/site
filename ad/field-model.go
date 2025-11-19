package ad

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/qdrant/go-client/qdrant"
	g "maragu.dev/gomponents"
)

// FieldModel handles the model field
type FieldModel struct {
	adField
}

func (f *FieldModel) BuildFilterCondition(c *fiber.Ctx) *qdrant.Condition {
	fieldValue := c.FormValue("model")
	if fieldValue == "" {
		return nil
	}
	return qdrant.NewMatchKeywords("model", fieldValue)
}

func (f *FieldModel) RenderNewAdField(selectedValues map[string]bool, fieldValues []string, defaultValue string) g.Node {
	return fieldCheckboxGrid(
		f.name+"Div",
		f.name,
		f.DisplayName,
		fieldValues,
		selectedValues,
		f.NextFieldName,
	)
}

func (f *FieldModel) Validate(value string) error {
	if f.IsRequired && value == "" {
		return fmt.Errorf("%s is required", f.DisplayName)
	}
	return nil
}
