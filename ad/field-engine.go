package ad

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/qdrant/go-client/qdrant"
	g "maragu.dev/gomponents"
)

// FieldEngine handles the engine field
type FieldEngine struct {
	adField
}

func (f *FieldEngine) BuildFilterCondition(c *fiber.Ctx) *qdrant.Condition {
	fieldValue := c.FormValue("engine")
	if fieldValue == "" {
		return nil
	}
	return qdrant.NewMatchKeywords("engine", fieldValue)
}

func (f *FieldEngine) RenderNewAdField(selectedValues map[string]bool, fieldValues []string, defaultValue string) g.Node {
	return fieldCheckboxGrid(
		f.FieldName+"Div",
		f.FieldName,
		f.DisplayName,
		fieldValues,
		selectedValues,
		f.NextFieldName,
	)
}

func (f *FieldEngine) Validate(value string) error {
	if f.IsRequired && value == "" {
		return fmt.Errorf("%s is required", f.DisplayName)
	}
	return nil
}
