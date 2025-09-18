package handlers

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/part"
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/vector"
	"github.com/parts-pile/site/vehicle"
	g "maragu.dev/gomponents"
)

func HandleTreeCollapse(c *fiber.Ctx) error {
	q := c.Query("q")
	path := c.Params("*")
	parts := strings.Split(strings.Trim(path, "/"), "/")

	threshold := getThreshold(c)

	name := parts[len(parts)-1]
	level := len(parts) - 1

	return render(c, ui.CollapsedTreeNodeWithThreshold(name, "/"+path, q, fmt.Sprintf("%.1f", threshold), level))
}

func HandleTreeViewNavigation(c *fiber.Ctx) error {
	q := c.Query("q")
	threshold := getThreshold(c)
	currentUser, _ := getUser(c)

	path := c.Params("*")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 1 && parts[0] == "" {
		parts = []string{}
	}
	level := len(parts)

	var childNodes []g.Node

	// Determine if we're in browse mode (q=="") or search mode (q!="")
	var adIDs []int
	var err error

	if q != "" {
		// Search mode: Get ad IDs from vector search
		log.Printf("[tree-view] Search mode: getting ad IDs for query: %s, threshold: %.2f", q, threshold)

		// Generate embedding for search query
		embedding, err := vector.GetQueryEmbedding(q)
		if err != nil {
			return fmt.Errorf("failed to generate embedding: %w", err)
		}

		// Get results with threshold filtering
		adIDs, _, err = vector.QuerySimilarAdIDs(embedding, nil, config.QdrantSearchInitialK, "", threshold)
		if err != nil {
			return fmt.Errorf("failed to query Qdrant: %w", err)
		}
		log.Printf("[tree-view] Vector search returned %d ad IDs", len(adIDs))
	} else {
		// Browse mode: No ad IDs filtering needed
		log.Printf("[tree-view] Browse mode: no ad ID filtering")
	}

	// Get children for the current level using appropriate SQL function
	var children []string
	switch level {
	case 0: // Root level - get makes
		if q != "" {
			// Search mode: get makes filtered by ad IDs
			children, err = part.GetMakesForAdIDs(adIDs)
		} else {
			// Browse mode: get makes with existing ads (cached)
			children, err = vehicle.GetAdMakes()
		}
	case 1: // Make level - get years
		makeName := parts[0]
		if q != "" {
			children, err = part.GetYearsForAdIDs(adIDs, makeName)
		} else {
			// Browse mode: get years with existing ads (cached)
			children, err = vehicle.GetAdYears(makeName)
		}
	case 2: // Year level - get models
		makeName, year := parts[0], parts[1]
		if q != "" {
			children, err = part.GetModelsForAdIDs(adIDs, makeName, year)
		} else {
			// Browse mode: get models with existing ads (cached)
			children, err = vehicle.GetAdModels(makeName, year)
		}
	case 3: // Model level - get engines
		makeName, year, model := parts[0], parts[1], parts[2]
		if q != "" {
			children, err = part.GetEnginesForAdIDs(adIDs, makeName, year, model)
		} else {
			// Browse mode: get engines with existing ads (cached)
			children, err = vehicle.GetAdEngines(makeName, year, model)
		}
	case 4: // Engine level - get categories
		makeName, year, model, engine := parts[0], parts[1], parts[2], parts[3]
		if q != "" {
			children, err = part.GetCategoriesForAdIDs(adIDs, makeName, year, model, engine)
		} else {
			// Browse mode: get categories with existing ads
			children, err = part.GetAdCategories(makeName, year, model, engine)
		}
	case 5: // Category level - get subcategories
		makeName, year, model, engine, category := parts[0], parts[1], parts[2], parts[3], parts[4]
		if q != "" {
			children, err = part.GetSubCategoriesForAdIDs(adIDs, makeName, year, model, engine, category)
		} else {
			// Browse mode: get subcategories with existing ads
			children, err = part.GetAdSubCategories(makeName, year, model, engine, category)
		}
	case 6: // Subcategory level - get ads
		makeName, year, model, engine, category, subcategory := parts[0], parts[1], parts[2], parts[3], parts[4], parts[5]
		var ads []ad.Ad
		if q != "" {
			ads, err = part.GetAdsForAdIDs(adIDs, makeName, year, model, engine, category, subcategory)
		} else {
			ads, err = part.GetAdsForAll(makeName, year, model, engine, category, subcategory)
		}

		if err != nil {
			return err
		}

		// Sort ads by CreatedAt DESC, ID DESC
		sort.Slice(ads, func(i, j int) bool {
			if ads[i].CreatedAt.Equal(ads[j].CreatedAt) {
				return ads[i].ID > ads[j].ID
			}
			return ads[i].CreatedAt.After(ads[j].CreatedAt)
		})

		// Show ads at leaf level
		if len(ads) > 0 {
			loc, _ := time.LoadLocation(c.Get("X-Timezone"))
			for _, ad := range ads {
				childNodes = append(childNodes, ui.AdCardCompactTree(ad, loc, currentUser))
			}
		} else {
			childNodes = append(childNodes, ui.NoSearchResultsMessage())
		}

		// Return expanded node with ads
		name := parts[len(parts)-1]
		return render(c, ui.ExpandedTreeNodeWithThreshold(name, "/"+path, q, fmt.Sprintf("%.1f", threshold), level, g.Group(childNodes)))
	}

	if err != nil {
		return err
	}

	// At root level, show empty response if no makes available
	if level == 0 && len(children) == 0 {
		return render(c, ui.EmptyResponse())
	}

	// Show children as collapsed tree nodes
	for _, child := range children {
		childNodes = append(childNodes, ui.CollapsedTreeNodeWithThreshold(child, "/"+path+"/"+child, q, fmt.Sprintf("%.1f", threshold), level+1))
	}

	// Return appropriate response based on level
	if level == 0 {
		return render(c, g.Group(childNodes))
	}

	name := parts[len(parts)-1]
	return render(c, ui.ExpandedTreeNodeWithThreshold(name, "/"+path, q, fmt.Sprintf("%.1f", threshold), level, g.Group(childNodes)))
}
