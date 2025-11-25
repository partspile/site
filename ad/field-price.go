package ad

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// FieldPrice handles the price field
type FieldPrice struct {
	adField
}

func (f *FieldPrice) BuildFilterCondition(c *fiber.Ctx) (string, []interface{}) {
	minPriceStr := c.FormValue("min_price")
	maxPriceStr := c.FormValue("max_price")
	if minPriceStr == "" && maxPriceStr == "" {
		return "", nil
	}

	var minPrice, maxPrice *float64

	if minPriceStr != "" {
		if price, err := strconv.ParseFloat(minPriceStr, 64); err == nil {
			minPrice = &price
		}
	}
	if maxPriceStr != "" {
		if price, err := strconv.ParseFloat(maxPriceStr, 64); err == nil {
			maxPrice = &price
		}
	}

	if minPrice == nil && maxPrice == nil {
		return "", nil
	}

	var conditions []string
	var args []interface{}
	argIndex := 1

	if minPrice != nil && maxPrice != nil {
		// Both min and max - use BETWEEN
		conditions = append(conditions, fmt.Sprintf("(vector_metadata->>$%d)::float BETWEEN $%d AND $%d", argIndex, argIndex+1, argIndex+2))
		args = append(args, "price", *minPrice, *maxPrice)
	} else if minPrice != nil {
		// Only min - use >=
		conditions = append(conditions, fmt.Sprintf("(vector_metadata->>$%d)::float >= $%d", argIndex, argIndex+1))
		args = append(args, "price", *minPrice)
	} else if maxPrice != nil {
		// Only max - use <=
		conditions = append(conditions, fmt.Sprintf("(vector_metadata->>$%d)::float <= $%d", argIndex, argIndex+1))
		args = append(args, "price", *maxPrice)
	}

	return strings.Join(conditions, " AND "), args
}

func (f *FieldPrice) RenderNewAdField(parentValues map[string]string) g.Node {
	return fieldInput(
		f.FieldName+"Div",
		f.FieldName,
		f.DisplayName,
		"number",
		f.IsRequired,
		"",
		Step("1"),
		Min("0"),
		g.Attr("title", "Price must be >= 0"),
		g.Attr("oninput", "this.checkValidity()"),
	)
}

func (f *FieldPrice) RenderFilterField(params map[string]string) g.Node {
	minPrice := params["min_price"]
	maxPrice := params["max_price"]

	return Div(
		Label(Class("block text-sm font-medium mb-1"), g.Text(f.DisplayName+" Range")),
		Div(
			Class("flex gap-2 flex-nowrap"),
			Input(
				Type("number"),
				Name("min_price"),
				ID("minPriceFilter"),
				Class("w-24 flex-shrink-0 p-2 border rounded-md"),
				Placeholder("Min $"),
				Min("0"),
				Step("1"),
				Value(minPrice),
			),
			Input(
				Type("number"),
				Name("max_price"),
				ID("maxPriceFilter"),
				Class("w-24 flex-shrink-0 p-2 border rounded-md"),
				Placeholder("Max $"),
				Min("0"),
				Step("1"),
				Value(maxPrice),
			),
		),
	)
}

func (f *FieldPrice) Validate(value string) error {
	if f.IsRequired && value == "" {
		return fmt.Errorf("%s is required", f.DisplayName)
	}
	if value != "" {
		price, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("%s must be a valid number", f.DisplayName)
		}
		if price < 0 {
			return fmt.Errorf("%s cannot be negative", f.DisplayName)
		}
	}
	return nil
}
