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
	return fieldSelect(
		f.Name+"Div",
		f.Name,
		f.DisplayName,
		fieldValues,
		defaultValue,
		f.IsRequired,
		config.NextFieldName,
	)
}

func (f *FieldPartSubcategory) Validate(value string) error {
	if f.IsRequired && value == "" {
		return fmt.Errorf("%s is required", f.DisplayName)
	}
	return nil
}
