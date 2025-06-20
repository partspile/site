package components

import (
	"fmt"
	"sort"
	"time"

	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"

	"github.com/parts-pile/site/ad"
)

// ---- Ad Components ----

func AdDetails(ad ad.Ad) g.Node {
	sortedYears := append([]string{}, ad.Years...)
	sortedModels := append([]string{}, ad.Models...)
	sortedEngines := append([]string{}, ad.Engines...)
	sort.Strings(sortedYears)
	sort.Strings(sortedModels)
	sort.Strings(sortedEngines)
	return GridContainer(1,
		P(Class("text-gray-600"), g.Text(fmt.Sprintf("Years: %v", sortedYears))),
		P(Class("text-gray-600"), g.Text(fmt.Sprintf("Models: %v", sortedModels))),
		P(Class("text-gray-600"), g.Text(fmt.Sprintf("Engines: %v", sortedEngines))),
		P(Class("mt-4"), g.Text(ad.Description)),
		P(Class("text-2xl font-bold mt-4"), g.Text(fmt.Sprintf("$%.2f", ad.Price))),
	)
}

func AdCard(ad ad.Ad, loc *time.Location) g.Node {
	sortedYears := append([]string{}, ad.Years...)
	sort.Strings(sortedYears)
	sortedModels := append([]string{}, ad.Models...)
	sort.Strings(sortedModels)
	posted := ad.CreatedAt.In(loc).Format("Jan 2, 2006 3:04:05 PM MST")
	return A(
		Href(fmt.Sprintf("/ad/%d", ad.ID)),
		Class("block border p-4 mb-4 rounded hover:bg-gray-50"),
		Div(
			H3(Class("text-xl font-bold"), g.Text(ad.Make)),
			P(Class("text-gray-600"), g.Text(fmt.Sprintf("Years: %v", sortedYears))),
			P(Class("text-gray-600"), g.Text(fmt.Sprintf("Models: %v", sortedModels))),
			P(Class("mt-2"), g.Text(ad.Description)),
			P(Class("text-xl font-bold mt-2"), g.Text(fmt.Sprintf("$%.2f", ad.Price))),
			P(
				Class("text-xs text-gray-400 mt-4"),
				g.Text(fmt.Sprintf("ID: %d • Posted: %s", ad.ID, posted)),
			),
		),
	)
}

func AdListContainer(children ...g.Node) g.Node {
	return Div(
		ID("adsList"),
		Class("space-y-4"),
		g.Group(children),
	)
}

func BuildAdListNodes(ads map[int]ad.Ad, loc *time.Location) []g.Node {
	// Convert map to slice
	adSlice := make([]ad.Ad, 0, len(ads))
	for _, ad := range ads {
		adSlice = append(adSlice, ad)
	}
	// Sort by CreatedAt DESC, ID DESC
	sort.Slice(adSlice, func(i, j int) bool {
		if adSlice[i].CreatedAt.Equal(adSlice[j].CreatedAt) {
			return adSlice[i].ID > adSlice[j].ID
		}
		return adSlice[i].CreatedAt.After(adSlice[j].CreatedAt)
	})
	// Build nodes
	adsList := []g.Node{}
	for _, ad := range adSlice {
		adsList = append(adsList, AdCard(ad, loc))
	}
	return adsList
}
