package spec

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/qdrant/go-client/qdrant"
	g "maragu.dev/gomponents"
)

// FieldPartCategory handles the part_category field
type FieldPartCategory struct {
	BaseField
}

func (f *FieldPartCategory) BuildFilterCondition(c *fiber.Ctx, _ LocationResolver) *qdrant.Condition {
	fieldValue := c.FormValue("part_category")
	if fieldValue == "" {
		return nil
	}
	return qdrant.NewMatch("part_category", fieldValue)
}

func (f *FieldPartCategory) RenderFilterField(params map[string]string) g.Node {
	// Handled by existing UI code
	return nil
}

func (f *FieldPartCategory) RenderNewAdField(selectedValues map[string]bool, fieldValues []string, defaultValue string) g.Node {
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

func (f *FieldPartCategory) Validate(value string) error {
	if f.config.IsRequired && value == "" {
		return fmt.Errorf("%s is required", f.config.DisplayName)
	}
	return nil
}

