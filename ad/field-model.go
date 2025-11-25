package ad

import (
	"encoding/json"
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
	// For array fields, check if the value is contained in the JSONB array
	jsonValue, _ := json.Marshal([]string{fieldValue})
	return "vector_metadata->$1 @> $2::jsonb", []interface{}{"model", string(jsonValue)}
}

func (f *FieldModel) RenderNewAdField(parentValues map[string]string) g.Node {
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

func (f *FieldModel) Validate(value string) error {
	if f.IsRequired && value == "" {
		return fmt.Errorf("%s is required", f.DisplayName)
	}
	return nil
}
