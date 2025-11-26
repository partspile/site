package ad

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// FieldMake handles the make field
type FieldMake struct {
	adField
}

func (f *FieldMake) BuildFilterCondition(c *fiber.Ctx) (string, []interface{}) {
	fieldValue := c.FormValue("make")
	if fieldValue == "" {
		return "", nil
	}
	return "vector_metadata->>'make' = $1::text", []interface{}{fieldValue}
}

func (f *FieldMake) RenderFilterField(params map[string]string) g.Node {
	value := params[f.FieldName]

	values, err := f.GetValues(params)
	if err != nil {
		values = []string{}
	}

	var options []g.Node
	options = append(options, Option(Value(""), g.Text("All"), g.If(value == "", Selected())))
	for _, val := range values {
		optionAttrs := []g.Node{Value(val), g.Text(val)}
		if val == value {
			optionAttrs = append(optionAttrs, Selected())
		}
		options = append(options, Option(optionAttrs...))
	}

	return Div(
		Label(Class("block text-sm font-medium mb-1"), g.Text(f.DisplayName)),
		Select(
			Name(f.FieldName),
			ID(f.FieldName+"Filter"),
			Class("w-full p-2 border rounded-md"),
			g.Group(options),
		),
	)
}

func (f *FieldMake) RenderNewAdField(parentValues map[string]string) g.Node {
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

func (f *FieldMake) Validate(value string) error {
	if f.IsRequired && value == "" {
		return fmt.Errorf("%s is required", f.DisplayName)
	}
	return nil
}
