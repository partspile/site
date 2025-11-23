package ad

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/qdrant/go-client/qdrant"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// FieldEngine handles the engine field
type FieldEngine struct {
	adField
}

func (f *FieldEngine) BuildFilterCondition(c *fiber.Ctx) *qdrant.Condition {
	fieldValue := c.FormValue("engine")
	if fieldValue == "" {
		return nil
	}
	return qdrant.NewMatchKeywords("engine", fieldValue)
}

func (f *FieldEngine) RenderNewAdField(selectedValues map[string]bool, defaultValue string) g.Node {
	// If field has a parent and no selected values, render empty div for HTMX
	if f.ParentField != nil && len(selectedValues) == 0 {
		fieldID := f.FieldName + "Div"
		return Div(ID(fieldID))
	}

	values, err := f.GetValues(map[string]string{})
	if err != nil {
		values = []string{}
	}

	return fieldCheckboxGrid(
		f.FieldName+"Div",
		f.FieldName,
		f.DisplayName,
		values,
		selectedValues,
		f.NextFieldName,
	)
}

func (f *FieldEngine) Validate(value string) error {
	if f.IsRequired && value == "" {
		return fmt.Errorf("%s is required", f.DisplayName)
	}
	return nil
}
