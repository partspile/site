package ad

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/qdrant/go-client/qdrant"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// FieldPartSubcategory handles the part_subcategory field
type FieldPartSubcategory struct {
	adField
}

func (f *FieldPartSubcategory) BuildFilterCondition(c *fiber.Ctx) *qdrant.Condition {
	fieldValue := c.FormValue("part_subcategory")
	if fieldValue == "" {
		return nil
	}
	return qdrant.NewMatch("part_subcategory", fieldValue)
}

func (f *FieldPartSubcategory) RenderNewAdField(selectedValues map[string]bool, defaultValue string, parentValues map[string]string) g.Node {
	// If field has a parent and no default value, render empty div for HTMX
	if f.ParentField != nil && defaultValue == "" {
		fieldID := f.FieldName + "Div"
		return Div(ID(fieldID))
	}

	values, err := f.GetValues(parentValues)
	if err != nil {
		values = []string{}
	}

	return fieldSelect(
		f.FieldName+"Div",
		f.FieldName,
		f.DisplayName,
		values,
		defaultValue,
		f.IsRequired,
		f.NextFieldName,
	)
}

func (f *FieldPartSubcategory) Validate(value string) error {
	if f.IsRequired && value == "" {
		return fmt.Errorf("%s is required", f.DisplayName)
	}
	return nil
}
