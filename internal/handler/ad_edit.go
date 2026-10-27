package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/partspile/site/internal/ad"
	"github.com/partspile/site/internal/cookie"
	"github.com/partspile/site/internal/facet"
	"github.com/partspile/site/internal/local"
	"github.com/partspile/site/internal/logger"
	"github.com/partspile/site/internal/param"
	"github.com/partspile/site/internal/rockopinion"
	"github.com/partspile/site/internal/ui"
	uiads "github.com/partspile/site/internal/ui/ads"
	"github.com/partspile/site/internal/vector"
)

func EditAdHandler(c *fiber.Ctx) error {
	userID := local.GetUserID(c)

	adID, err := param.GetAdID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}

	tz := cookie.GetTimezone(c)
	a, err := ad.GetAd(userID, adID, tz)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}
	if a.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "You are not the owner of this ad")
	}
	if !a.IsActive() {
		return fiber.NewError(fiber.StatusBadRequest, "Cannot edit a deleted or inactive ad")
	}

	category := ad.GetCategory(a.CategoryID)

	if err := ad.LoadTags(&a); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	values := adFormValuesFrom(a)
	cfg := uiads.EditFormConfig(adID, values, newAdFormDefaults(c))
	fieldsNode := uiads.AdFieldsPartial(cfg, category.Facets())

	return renderPage(c, "Edit Ad", ui.EditAd(categoryOption(category), cfg, fieldsNode))
}

func UpdateAdHandler(c *fiber.Ctx) error {
	userID := local.GetUserID(c)

	adID, err := param.GetAdID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}

	tz := cookie.GetTimezone(c)
	a, err := ad.GetAd(userID, adID, tz)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}
	if a.UserID != userID {
		return fiber.NewError(fiber.StatusForbidden, "You are not the owner of this ad")
	}
	if !a.IsActive() {
		return fiber.NewError(fiber.StatusBadRequest, "Cannot edit a deleted or inactive ad")
	}

	category := ad.GetCategory(a.CategoryID)

	facets, err := parseAdFacets(c, category)
	if err != nil {
		return showError(c, err.Error())
	}

	err = ad.UpdateAd(ad.UpdateInput{
		AdID:         adID,
		UserID:       userID,
		Title:        c.FormValue("title"),
		Description:  c.FormValue("description"),
		LocationText: c.FormValue("location"),
		Facets:       facets,
		Suggestions:  parseAdSuggestions(c),
		ImagesAdded:  0,
		Tz:           tz,
	})
	if err != nil {
		return showError(c, err.Error())
	}
	if err := rockopinion.InvalidateForAd(adID); err != nil {
		logger.Error("Failed to invalidate rock opinions",
			"error", err, "adID", adID)
	}
	vector.QueueAd(adID)

	return respondAdSaved(c, adID, a.ImageCount)
}

func adFormValuesFrom(a ad.Ad) uiads.AdFormValues {
	desc := ad.ParseDescriptionForDisplay(a.Description)
	values := uiads.AdFormValues{
		Title:               a.Title,
		OriginalDescription: desc.Body,
		Location:            a.RawLocation,
		ImageCount:          a.ImageCount,
		PriceRow:            priceRowFromAd(a),
		Facets:              make(map[string]string),
		FacetUnits:          make(map[string]string),
		FacetMulti:          make(map[string][]string),
		FacetFlags:          make(map[string]bool),
	}
	for key, v := range a.Facets {
		if key == "price" {
			continue
		}
		d, ok := facet.Get(key)
		if ok && d.Kind == facet.MultiEnum {
			values.FacetMulti[key] = v.MultiEnumValues()
			continue
		}
		if ok && d.Kind == facet.Flag {
			values.FacetFlags[key] = v.FlagSet()
			continue
		}
		if v.Num != nil {
			values.Facets[key] = strconv.Itoa(*v.Num)
		}
		if v.Text != nil {
			if v.Num != nil {
				values.FacetUnits[key] = *v.Text
			} else {
				values.Facets[key] = *v.Text
			}
		}
	}
	for _, s := range a.Tags {
		values.Suggestions = append(values.Suggestions, uiads.SuggestionOption{
			Label:    s.Label,
			Value:    s.Value,
			Selected: true,
		})
	}
	return values
}

func priceRowFromAd(a ad.Ad) uiads.PriceRowView {
	v, ok := a.Facets["price"]
	if !ok || v.Num == nil {
		return uiads.PriceRowView{}
	}
	currency := ""
	if v.Text != nil {
		currency = *v.Text
	}
	if *v.Num == 0 {
		return uiads.PriceRowView{IsFree: true, Currency: currency}
	}
	return uiads.PriceRowView{
		Amount:   strconv.Itoa(*v.Num),
		Currency: currency,
	}
}
