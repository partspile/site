package spec

import (
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/qdrant/go-client/qdrant"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// FieldPrice handles the price field
type FieldPrice struct {
	BaseField
}

func (f *FieldPrice) BuildFilterCondition(c *fiber.Ctx, _ LocationResolver) *qdrant.Condition {
	minPriceStr := c.FormValue("min_price")
	maxPriceStr := c.FormValue("max_price")
	if minPriceStr == "" && maxPriceStr == "" {
		return nil
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

	if minPrice != nil || maxPrice != nil {
		return qdrant.NewRange("price", &qdrant.Range{
			Gte: minPrice,
			Lte: maxPrice,
		})
	}

	return nil
}

func (f *FieldPrice) RenderNewAdField(selectedValues map[string]bool, fieldValues []string, defaultValue string) g.Node {
	config := f.config
	return fieldInput(
		config.Name+"Div",
		config.Name,
		config.DisplayName,
		"number",
		defaultValue,
		config.IsRequired,
		"",
		Step("1"),
		Min("0"),
		g.Attr("title", "Price must be >= 0"),
		g.Attr("oninput", "this.checkValidity()"),
	)
}

func (f *FieldPrice) RenderFilterField(params map[string]string) g.Node {
	// Handled by existing UI code (priceFilter)
	return nil
}

func (f *FieldPrice) Validate(value string) error {
	if f.config.IsRequired && value == "" {
		return fmt.Errorf("%s is required", f.config.DisplayName)
	}
	if value != "" {
		if _, err := strconv.ParseFloat(value, 64); err != nil {
			return fmt.Errorf("%s must be a valid number", f.config.DisplayName)
		}
	}
	return nil
}
