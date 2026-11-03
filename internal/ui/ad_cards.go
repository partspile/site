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

// AdNodes renders search or list ad cards into gomponents nodes.
func AdNodes(cards []AdCard, userID, view, page int, csrfToken string,
	pagination bool) []g.Node {
	results := make([]g.Node, len(cards))
	for i, card := range cards {
		isLast := i == len(cards)-1 && pagination
		results[i] = adCardNode(card, userID, view, page+1, csrfToken, isLast)
	}
	return results
}

func adCardNode(card AdCard, userID, view, nextPage int, csrfToken string,
	isLast bool) g.Node {
	switch view {
	case ViewGrid:
		return AdGridNode(
			userID, card.ID, card.ImageCount, nextPage,
			card.PriceDisplay, card.Title, card.Location, card.FacetLabel, csrfToken,
			card.HasPrice, card.CreatedAt, card.Active, card.Bookmarked, isLast, card.FlagCount,
		)
	case ViewList:
		return AdListNode(
			userID, card.ID, card.PriceDisplay, card.Title, card.Location, card.FacetLabel,
			card.HasPrice, card.CreatedAt, card.Active, card.Bookmarked, csrfToken, isLast, nextPage,
			card.FlagCount,
		)
	default:
		return g.Text("bad view")
	}
}
