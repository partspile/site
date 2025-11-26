package ad

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// FieldYear handles the year field
type FieldYear struct {
	adField
}

func (f *FieldYear) BuildFilterCondition(c *fiber.Ctx) (string, []interface{}) {
	minKey := "min_year"
	maxKey := "max_year"
	minValue := c.FormValue(minKey)
	maxValue := c.FormValue(maxKey)

	if minValue == "" && maxValue == "" {
		return "", nil
	}

	// Preset defaults
	currentYear := time.Now().Year()
	minVal := 1900
	maxVal := currentYear + 2

	// Override with provided values if valid
	if minValue != "" {
		if val, err := strconv.Atoi(minValue); err == nil {
			minVal = val
		}
	}
	if maxValue != "" {
		if val, err := strconv.Atoi(maxValue); err == nil {
			maxVal = val
		}
	}

	var conditions []string
	var args []interface{}

	// Check if any year in the JSONB array falls within the range
	// Extract each year from the array, convert to int, and check if it's in range
	condition := `EXISTS (
		SELECT 1 
		FROM jsonb_array_elements_text(vector_metadata->'year') AS year_val
		WHERE year_val::int BETWEEN $1::int AND $2::int
	)`
	conditions = append(conditions, condition)
	args = append(args, minVal, maxVal)

	return strings.Join(conditions, " AND "), args
}

func (f *FieldYear) RenderNewAdField(parentValues map[string]string) g.Node {
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

func (f *FieldYear) RenderFilterField(params map[string]string) g.Node {
	minYear := params["min_year"]
	maxYear := params["max_year"]

	// Calculate max year as current year + 2
	maxYearValue := time.Now().Year() + 2
	maxYearStr := fmt.Sprintf("%d", maxYearValue)

	return Div(
		Label(Class("block text-sm font-medium mb-1"), g.Text(f.DisplayName+" Range")),
		Div(
			Class("flex gap-2 flex-nowrap"),
			Input(
				Type("number"),
				Name("min_year"),
				ID("minYearFilter"),
				Class("w-20 flex-shrink-0 p-2 border rounded-md"),
				Placeholder("Min"),
				Min("1900"),
				Max(maxYearStr),
				Value(minYear),
			),
			Input(
				Type("number"),
				Name("max_year"),
				ID("maxYearFilter"),
				Class("w-20 flex-shrink-0 p-2 border rounded-md"),
				Placeholder("Max"),
				Min("1900"),
				Max(maxYearStr),
				Value(maxYear),
			),
		),
	)
}

func (f *FieldYear) Validate(value string) error {
	if f.IsRequired && value == "" {
		return fmt.Errorf("%s is required", f.DisplayName)
	}
	return nil
}
