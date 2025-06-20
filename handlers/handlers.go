package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/grok"
	"github.com/parts-pile/site/part"
	"github.com/parts-pile/site/vehicle"
)

const sysPrompt = `You are an expert vehicle parts assistant.

Your job is to extract a structured query from a user's search request.

Extract the make, years, models, engine sizes, category, and subcategory from
the user's search request.  Use your best judgement as a vehicle parts export
to fill out the structured query as much as possible.  When filling out the
structured query, only use values from the lists below, and not the user's values.
For example, if user entered "Ford", the structure query would use "FORD".

<Makes>
%s
</Makes>

<Years>
%s
</Years>

<Models>
%s
</Models>

<EngineSizes>
%s
</EngineSizes>

<Categories>
%s
</Categories>

<SubCategories>
%s
</SubCategories>

Return JSON encoding this Go structure with the vehicle parts data:

struct {
	Make        string
	Years       []string
	Models      []string
	EngineSizes []string
	Category    string
	SubCategory string
}

Only return the JSON.  Nothing else.
`

type SearchQuery = ad.SearchQuery

type SearchCursor = ad.SearchCursor

func EncodeCursor(c SearchCursor) string {
	jsonCursor, _ := json.Marshal(c)
	return base64.StdEncoding.EncodeToString(jsonCursor)
}

func DecodeCursor(s string) (SearchCursor, error) {
	var c SearchCursor
	if s == "" {
		return c, nil
	}
	jsonCursor, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return c, err
	}
	err = json.Unmarshal(jsonCursor, &c)
	return c, err
}

func ParseSearchQuery(q string) (SearchQuery, error) {
	if q == "" {
		return SearchQuery{}, nil
	}

	allMakes := vehicle.GetMakes()
	allYears := vehicle.GetYearRange()
	allModels := vehicle.GetAllModels()
	allEngineSizes := vehicle.GetAllEngineSizes()
	allCategories := part.GetAllCategories()
	allSubCategories := part.GetAllSubCategories()

	prompt := fmt.Sprintf(sysPrompt,
		strings.Join(allMakes, "\n"),
		strings.Join(allYears, "\n"),
		strings.Join(allModels, "\n"),
		strings.Join(allEngineSizes, "\n"),
		strings.Join(allCategories, "\n"),
		strings.Join(allSubCategories, "\n"),
	)

	var query SearchQuery
	resp, err := grok.CallGrok(prompt, q)
	if err != nil {
		return SearchQuery{}, fmt.Errorf("error grokking query: %w", err)
	}

	err = json.Unmarshal([]byte(resp), &query)
	if err != nil {
		return SearchQuery{}, fmt.Errorf("error unmarshalling grok response: %w", err)
	}

	return query, nil
}

func FilterAds(query SearchQuery, ads []ad.Ad) []ad.Ad {
	if query.Make == "" && len(query.Years) == 0 && len(query.Models) == 0 &&
		len(query.EngineSizes) == 0 && query.Category == "" && query.SubCategory == "" {
		return ads
	}
	var filteredAds []ad.Ad
	for _, ad := range ads {
		var makeMatch, yearMatch, modelMatch, engineMatch bool

		if query.Make == "" || ad.Make == query.Make {
			makeMatch = true
		}

		if len(query.Years) == 0 || anyStringInSlice(ad.Years, query.Years) {
			yearMatch = true
		}

		if len(query.Models) == 0 || anyStringInSlice(ad.Models, query.Models) {
			modelMatch = true
		}

		if len(query.EngineSizes) == 0 || anyStringInSlice(ad.Engines, query.EngineSizes) {
			engineMatch = true
		}

		if makeMatch && yearMatch && modelMatch && engineMatch {
			filteredAds = append(filteredAds, ad)
		}
	}
	return filteredAds
}

func GetNextPage(query SearchQuery, cursor *SearchCursor, limit int) ([]ad.Ad, *SearchCursor, error) {
	// Get filtered page from database
	ads, hasMore, err := ad.GetFilteredAdsPageDB(query, cursor, limit)
	if err != nil {
		return nil, nil, err
	}

	// Create next cursor if there are more results
	var nextCursor *SearchCursor
	if hasMore && len(ads) > 0 {
		last := ads[len(ads)-1]
		nextCursor = &SearchCursor{
			Query:      query,
			LastID:     last.ID,
			LastPosted: last.CreatedAt,
		}
	}

	return ads, nextCursor, nil
}

func anyStringInSlice(a, b []string) bool {
	for _, aVal := range a {
		for _, bVal := range b {
			if aVal == bVal {
				return true
			}
		}
	}
	return false
}

func htmlEscape(s string) string {
	return strings.ReplaceAll(s, `"`, `\"`)
}
