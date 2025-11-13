package spec

import (
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/qdrant/go-client/qdrant"
	g "maragu.dev/gomponents"
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
	// Handled by existing UI code
	return nil
}

func (f *FieldYear) Validate(value string) error {
	if f.config.IsRequired && value == "" {
		return fmt.Errorf("%s is required", f.config.DisplayName)
	}
	return nil
}
