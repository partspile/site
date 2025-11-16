package spec

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/qdrant/go-client/qdrant"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// FieldYear handles the year field
type FieldYear struct {
	BaseField
}

func (f *FieldYear) BuildFilterCondition(c *fiber.Ctx, _ LocationResolver) *qdrant.Condition {
	minKey := "min_year"
	maxKey := "max_year"
	minValue := c.FormValue(minKey)
	maxValue := c.FormValue(maxKey)
	fieldValue := c.FormValue("year")

	if minValue != "" || maxValue != "" {
		// Range filter
		var minVal, maxVal int
		var hasMin, hasMax bool

		if minValue != "" {
			if val, err := strconv.Atoi(minValue); err == nil {
				minVal = val
				hasMin = true
			}
		}
		if maxValue != "" {
			if val, err := strconv.Atoi(maxValue); err == nil {
				maxVal = val
				hasMax = true
			}
		}

		// Convert range to list of values
		var valueList []string
		if hasMin && hasMax {
			for val := minVal; val <= maxVal; val++ {
				valueList = append(valueList, strconv.Itoa(val))
			}
		} else if hasMin {
			// Default max to 2029 for year fields
			maxVal = 2029
			for val := minVal; val <= maxVal; val++ {
				valueList = append(valueList, strconv.Itoa(val))
			}
		} else if hasMax {
			// Default min to 1900 for year fields
			minVal = 1900
			for val := minVal; val <= maxVal; val++ {
				valueList = append(valueList, strconv.Itoa(val))
			}
		}

		if len(valueList) > 0 {
			return qdrant.NewMatchKeywords("year", valueList...)
		}
	} else if fieldValue != "" {
		// Single value
		return qdrant.NewMatchKeywords("year", fieldValue)
	}

	return nil
}

func (f *FieldYear) RenderNewAdField(selectedValues map[string]bool, fieldValues []string, defaultValue string) g.Node {
	config := f.config
	return fieldCheckboxGrid(
		config.Name+"Div",
		config.Name,
		config.DisplayName,
		fieldValues,
		selectedValues,
		config.NextFieldName,
	)
}

func (f *FieldYear) RenderFilterField(params map[string]string) g.Node {
	minYear := params["min_year"]
	maxYear := params["max_year"]
	config := f.config

	// Calculate max year as current year + 2
	maxYearValue := time.Now().Year() + 2
	maxYearStr := fmt.Sprintf("%d", maxYearValue)

	return Div(
		Label(Class("block text-sm font-medium mb-1"), g.Text(config.DisplayName+" Range")),
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
	if f.config.IsRequired && value == "" {
		return fmt.Errorf("%s is required", f.config.DisplayName)
	}
	return nil
}
