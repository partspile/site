package ad

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/qdrant/go-client/qdrant"
	g "maragu.dev/gomponents"
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

func (f *FieldBreed) RenderNewAdField(selectedValues map[string]bool, fieldValues []string, defaultValue string) g.Node {
	nextFieldName := ""
	if f.NextInChain != 0 {
		nextFieldName = fmt.Sprintf("%d", f.NextInChain)
	}
	if f.QdrantType == "keyword_array" {
		return fieldCheckboxGrid(
			f.FieldName+"Div",
			f.FieldName,
			f.DisplayName,
			fieldValues,
			selectedValues,
			nextFieldName,
		)
	}
	return fieldSelect(
		f.FieldName+"Div",
		f.FieldName,
		f.DisplayName,
		fieldValues,
		defaultValue,
		f.IsRequired,
		nextFieldName,
	)
}

func (f *FieldBreed) Validate(value string) error {
	if f.IsRequired && value == "" {
		return fmt.Errorf("%s is required", f.DisplayName)
	}
	return nil
}
