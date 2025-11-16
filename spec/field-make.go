package spec

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/qdrant/go-client/qdrant"
	g "maragu.dev/gomponents"
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
	// Handled by existing UI code
	return nil
}

func (f *FieldMake) RenderNewAdField(selectedValues map[string]bool, fieldValues []string, defaultValue string) g.Node {
	config := f.config
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

func (f *FieldMake) Validate(value string) error {
	if f.config.IsRequired && value == "" {
		return fmt.Errorf("%s is required", f.config.DisplayName)
	}
	return nil
}

