package ad

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	g "maragu.dev/gomponents"
)

// FieldEngine handles the engine field
type FieldEngine struct {
	adField
}

func (f *FieldEngine) BuildFilterCondition(c *fiber.Ctx) (string, []interface{}) {
	fieldValue := c.FormValue("engine")
	if fieldValue == "" {
		return "", nil
	}
	return "vector_metadata->>'engine' = $1::text", []interface{}{fieldValue}
}

func (f *FieldEngine) RenderNewAdField(parentValues map[string]string) g.Node {
	values, err := f.GetValues(parentValues)
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

func (f *FieldEngine) Validate(value string) error {
	if f.IsRequired && value == "" {
		return fmt.Errorf("%s is required", f.DisplayName)
	}
	return nil
}
