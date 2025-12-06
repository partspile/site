package ad

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	g "maragu.dev/gomponents"
)

// FieldModel handles the model field
type FieldModel struct {
	adField
}

func (f *FieldModel) BuildFilterCondition(c *fiber.Ctx) (string, []interface{}) {
	fieldValue := c.FormValue("model")
	if fieldValue == "" {
		return "", nil
	}
	return "vector_metadata->>'model' = $1::text", []interface{}{fieldValue}
}

func (f *FieldModel) RenderNewAdField(parentValues map[string]string) g.Node {
	values, err := GetValues(f, parentValues)
	if err != nil {
		values = []string{}
	}

	return fieldCheckboxGrid(
		f.FieldName+"Div",
		f.FieldName,
		f.DisplayName,
		values,
		f.NextFieldName,
	)
}

func (f *FieldModel) Validate(value string) error {
	if f.IsRequired && value == "" {
		return fmt.Errorf("%s is required", f.DisplayName)
	}
	return nil
}
