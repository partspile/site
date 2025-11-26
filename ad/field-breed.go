package ad

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	g "maragu.dev/gomponents"
)

// FieldBreed handles the breed field
type FieldBreed struct {
	adField
}

func (f *FieldBreed) BuildFilterCondition(c *fiber.Ctx) (string, []interface{}) {
	fieldValue := c.FormValue("breed")
	if fieldValue == "" {
		return "", nil
	}
	return "vector_metadata->>'breed' = $1::text", []interface{}{fieldValue}
}

func (f *FieldBreed) RenderNewAdField(parentValues map[string]string) g.Node {
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

func (f *FieldBreed) Validate(value string) error {
	if f.IsRequired && value == "" {
		return fmt.Errorf("%s is required", f.DisplayName)
	}
	return nil
}
