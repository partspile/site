package ad

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/qdrant/go-client/qdrant"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// FieldBreed handles the breed field
type FieldBreed struct {
	adField
}

func (f *FieldBreed) BuildFilterCondition(c *fiber.Ctx) *qdrant.Condition {
	fieldValue := c.FormValue("breed")
	if fieldValue == "" {
		return nil
	}
	return qdrant.NewMatch("breed", fieldValue)
}

func (f *FieldBreed) RenderNewAdField(selectedValues map[string]bool, defaultValue string) g.Node {
	// If field has a parent and no values to pre-populate, render empty div for HTMX
	if f.ParentField != nil && len(selectedValues) == 0 && defaultValue == "" {
		fieldID := f.FieldName + "Div"
		return Div(ID(fieldID))
	}

	values, err := f.GetValues(map[string]string{})
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

func (f *FieldBreed) Validate(value string) error {
	if f.IsRequired && value == "" {
		return fmt.Errorf("%s is required", f.DisplayName)
	}
	return nil
}
