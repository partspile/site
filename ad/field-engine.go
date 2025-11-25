package ad

import (
	"encoding/json"
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
	// For array fields, check if the value is contained in the JSONB array
	jsonValue, _ := json.Marshal([]string{fieldValue})
	return "vector_metadata->$1 @> $2::jsonb", []interface{}{"engine", string(jsonValue)}
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
