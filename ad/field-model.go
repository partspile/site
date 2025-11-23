package ad

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/qdrant/go-client/qdrant"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// FieldModel handles the model field
type FieldModel struct {
	adField
}

func (f *FieldModel) BuildFilterCondition(c *fiber.Ctx) *qdrant.Condition {
	fieldValue := c.FormValue("model")
	if fieldValue == "" {
		return nil
	}
	return qdrant.NewMatchKeywords("model", fieldValue)
}

func (f *FieldModel) RenderNewAdField(selectedValues map[string]bool, defaultValue string, parentValues map[string]string) g.Node {
	// If field has a parent and no selected values, render empty div for HTMX
	if f.ParentField != nil && len(selectedValues) == 0 {
		fieldID := f.FieldName + "Div"
		return Div(ID(fieldID))
	}

	values, err := f.GetValues(parentValues)
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

func (f *FieldModel) Validate(value string) error {
	if f.IsRequired && value == "" {
		return fmt.Errorf("%s is required", f.DisplayName)
	}
	return nil
}
