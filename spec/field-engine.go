package spec

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/qdrant/go-client/qdrant"
	g "maragu.dev/gomponents"
)

// FieldEngine handles the engine field
type FieldEngine struct {
	BaseField
}

func (f *FieldEngine) BuildFilterCondition(c *fiber.Ctx, _ LocationResolver) *qdrant.Condition {
	fieldValue := c.FormValue("engine")
	if fieldValue == "" {
		return nil
	}
	return qdrant.NewMatchKeywords("engine", fieldValue)
}

func (f *FieldEngine) RenderFilterField(params map[string]string) g.Node {
	// Handled by existing UI code
	return nil
}

func (f *FieldEngine) RenderNewAdField(selectedValues map[string]bool, fieldValues []string, defaultValue string) g.Node {
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

func (f *FieldEngine) Validate(value string) error {
	if f.config.IsRequired && value == "" {
		return fmt.Errorf("%s is required", f.config.DisplayName)
	}
	return nil
}

