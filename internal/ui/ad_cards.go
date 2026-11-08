package ui

import (
	"time"

	g "maragu.dev/gomponents"
)

// AdCardFromFields builds an AdCard from presentation field values.
func AdCardFromFields(id, imageCount, flagCount int, priceDisplay, title,
	location, facetLabel string, hasPrice bool, createdAt time.Time, active,
	bookmarked bool) AdCard {
	return AdCard{
		ID:           id,
		PriceDisplay: priceDisplay,
		HasPrice:     hasPrice,
		Title:        title,
		Location:     location,
		FacetLabel:   facetLabel,
		CreatedAt:    createdAt,
		ImageCount:   imageCount,
		Active:       active,
		Bookmarked:   bookmarked,
		FlagCount:    flagCount,
	}
}

const adsGridClass = "grid grid-cols-2 md:grid-cols-3 gap-2 -mx-5 md:mx-0"

// AdNodes renders listing cards into gomponents nodes.
func AdNodes(cards []AdCard, userID, page int, csrfToken string,
	pagination bool) []g.Node {
	results := make([]g.Node, len(cards))
	for i, card := range cards {
		isLast := i == len(cards)-1 && pagination
		results[i] = AdGridNode(
			userID, card.ID, card.ImageCount, page+1,
			card.PriceDisplay, card.Title, card.Location, card.FacetLabel,
			csrfToken, card.HasPrice, card.CreatedAt, card.Active,
			card.Bookmarked, isLast, card.FlagCount,
		)
	}
	return results
}
