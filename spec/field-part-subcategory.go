package spec

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/qdrant/go-client/qdrant"
	g "maragu.dev/gomponents"
)

// FieldPartSubcategory handles the part_subcategory field
type FieldPartSubcategory struct {
	BaseField
}

func (f *FieldPartSubcategory) BuildFilterCondition(c *fiber.Ctx, _ LocationResolver) *qdrant.Condition {
	fieldValue := c.FormValue("part_subcategory")
	if fieldValue == "" {
		return nil
	}
	return qdrant.NewMatch("part_subcategory", fieldValue)
}

func (f *FieldPartSubcategory) RenderNewAdField(selectedValues map[string]bool, fieldValues []string, defaultValue string) g.Node {
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

func (f *FieldPartSubcategory) Validate(value string) error {
	if f.config.IsRequired && value == "" {
		return fmt.Errorf("%s is required", f.config.DisplayName)
	}
	return nil
}
