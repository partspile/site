package ad

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	g "maragu.dev/gomponents"
)

// FieldPartSubcategory handles the part_subcategory field
type FieldPartSubcategory struct {
	adField
}

func (f *FieldPartSubcategory) BuildFilterCondition(c *fiber.Ctx) (string, []interface{}) {
	fieldValue := c.FormValue("part_subcategory")
	if fieldValue == "" {
		return "", nil
	}
	return "vector_metadata->>$1 = $2", []interface{}{"part_subcategory", fieldValue}
}

func (f *FieldPartSubcategory) RenderNewAdField(parentValues map[string]string) g.Node {
	values, err := f.GetValues(parentValues)
	if err != nil {
		values = []string{}
	}

	return fieldSelect(
		f.FieldName+"Div",
		f.FieldName,
		f.DisplayName,
		values,
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
