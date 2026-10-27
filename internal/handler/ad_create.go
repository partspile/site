package handler

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/partspile/site/internal/ad"
	"github.com/partspile/site/internal/cookie"
	"github.com/partspile/site/internal/currency"
	"github.com/partspile/site/internal/facet"
	"github.com/partspile/site/internal/local"
	"github.com/partspile/site/internal/vector"
)

func CreateAdHandler(c *fiber.Ctx) error {
	userID := local.GetUserID(c)

	category := ad.GetCategory(cookie.GetCategoryID(c))

	facets, err := parseAdFacets(c, category)
	if err != nil {
		return showCreateAdError(c, err.Error())
	}

	adID, err := ad.CreateAd(ad.CreateInput{
		CategoryID:   category.ID,
		UserID:       userID,
		Title:        c.FormValue("title"),
		Description:  c.FormValue("description"),
		LocationText: c.FormValue("location"),
		Facets:       facets,
		Suggestions:  parseAdSuggestions(c),
		ImageCount:   0,
	})
	if err != nil {
		return showCreateAdError(c, err.Error())
	}

	if in, err := ad.GetForEmbedding(adID); err != nil {
		vector.QueueAd(adID)
	} else if err := vector.BuildAdEmbedding(in); err != nil {
		vector.QueueAd(adID)
	}

	return respondAdSaved(c, adID, 0)
}

func parseAdFacets(c *fiber.Ctx,
	category ad.Category) (map[string]facet.Value, error) {
	values := make(map[string]facet.Value)
	for _, d := range category.Facets() {
		switch d.Kind {
		case facet.Money:
			free := c.FormValue("price_free") == "1"
			raw := strings.TrimSpace(c.FormValue(d.Key))
			code := currency.Normalize(c.FormValue("price_currency"))
			if !currency.IsSupported(code) {
				code = defaultCurrencyForUser(c)
			}
			if free {
				amount := 0
				values[d.Key] = facet.Value{Num: &amount, Text: &code}
				continue
			}
			if raw == "" {
				if d.Required {
					return nil, fmt.Errorf("%s is required", d.Label)
				}
				continue
			}
			n, err := strconv.Atoi(raw)
			if err != nil || n < 0 {
				return nil, fmt.Errorf("%s must be a non-negative whole number", d.Label)
			}
			values[d.Key] = facet.Value{Num: &n, Text: &code}
		case facet.Enum:
			val := strings.TrimSpace(c.FormValue(d.Key))
			if val != "" {
				values[d.Key] = facet.Value{Text: &val}
			} else if d.Required {
				return nil, fmt.Errorf("%s is required", d.Label)
			}
		case facet.Date:
			raw := strings.TrimSpace(c.FormValue(d.Key))
			if raw == "" {
				if d.Required {
					return nil, fmt.Errorf("%s is required", d.Label)
				}
				continue
			}
			v, err := facet.ParseDateValue(raw)
			if err != nil {
				return nil, fmt.Errorf("%s must be a valid date", d.Label)
			}
			values[d.Key] = v
		case facet.MultiEnum:
			vals := parseFormEnumCheckboxes(c, d.Key, d.Enum)
			if len(vals) == 0 {
				if d.Required {
					return nil, fmt.Errorf("%s is required", d.Label)
				}
				continue
			}
			values[d.Key] = facet.EncodeMultiEnum(vals)
		case facet.Flag:
			if c.FormValue(d.Key) == "1" {
				values[d.Key] = facet.EncodeFlag(true)
			}
		case facet.Location:
			raw := strings.TrimSpace(c.FormValue(d.Key))
			if raw == "" {
				if d.Required {
					return nil, fmt.Errorf("%s is required", d.Label)
				}
				continue
			}
			values[d.Key] = facet.Value{Text: &raw}
		default:
			num, err := parseOptionalFacet(c.FormValue(d.Key))
			if err != nil {
				return nil, fmt.Errorf("%s must be a non-negative whole number", d.Label)
			}
			if num == nil {
				if d.Required {
					return nil, fmt.Errorf("%s is required", d.Label)
				}
				continue
			}
			v := facet.Value{Num: num}
			if len(d.Units) > 0 {
				unit := strings.TrimSpace(c.FormValue(d.Key + "_unit"))
				if unit == "" {
					unit = cookie.GetDistanceUnit(c)
				}
				if !d.ValidUnit(unit) {
					return nil, fmt.Errorf("%s requires a valid unit", d.Label)
				}
				u := d.NormalizeUnit(unit)
				v.Text = &u
			}
			values[d.Key] = v
		}
	}
	if err := validateSaleDates(values); err != nil {
		return nil, err
	}
	return values, nil
}

func validateSaleDates(values map[string]facet.Value) error {
	start, startOK := values["sale_start_date"]
	end, endOK := values["sale_end_date"]
	if !startOK || !endOK {
		return nil
	}
	if start.DateString() > end.DateString() {
		return fmt.Errorf("Sale End Date must be on or after Sale Start Date")
	}
	return nil
}

func parseFormEnumCheckboxes(c *fiber.Ctx, key string,
	allowed []string) []string {
	allowedSet := make(map[string]bool, len(allowed))
	for _, a := range allowed {
		allowedSet[a] = true
	}
	var vals []string
	add := func(raw string) {
		s := strings.TrimSpace(raw)
		if s == "" || !allowedSet[s] {
			return
		}
		for _, existing := range vals {
			if existing == s {
				return
			}
		}
		vals = append(vals, s)
	}
	c.Context().PostArgs().VisitAll(func(k, v []byte) {
		if string(k) == key {
			add(string(v))
		}
	})
	if form, err := c.MultipartForm(); err == nil && form != nil {
		for _, v := range form.Value[key] {
			add(v)
		}
	}
	return vals
}

func parseOptionalFacet(raw string) (*int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return nil, err
	}
	return &value, nil
}

func showCreateAdError(c *fiber.Ctx, errMsg string) error {
	if wantsJSON(c) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": errMsg,
		})
	}
	return showError(c, errMsg)
}
