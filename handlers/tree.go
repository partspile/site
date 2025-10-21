package handlers

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/part"
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/vehicle"
)

// decodeAdIDs decodes a base64 string back to a slice of integers
func decodeAdIDs(adIDsStr string) ([]int, error) {
	if adIDsStr == "" {
		return []int{}, nil
	}

	// Decode from base64
	buf, err := base64.URLEncoding.DecodeString(adIDsStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}

	// Convert bytes back to integers
	if len(buf)%4 != 0 {
		return nil, fmt.Errorf("invalid buffer length: %d", len(buf))
	}

	adIDs := make([]int, len(buf)/4)
	for i := 0; i < len(adIDs); i++ {
		adIDs[i] = int(binary.LittleEndian.Uint32(buf[i*4:]))
	}

	return adIDs, nil
}

// parsePath extracts name, level, and parts from the path parameter
func parsePath(path string) (name string, level int, parts []string) {
	parts = strings.Split(strings.Trim(path, "/"), "/")

	if len(parts) == 1 && parts[0] == "" {
		parts = []string{}
	}

	level = len(parts)
	if level > 0 {
		name = parts[len(parts)-1]
	}

	return name, level, parts
}

func HandleTreeCollapseBrowse(c *fiber.Ctx) error {
	path := c.Params("*")
	name, _, _ := parsePath(path)
	return render(c, ui.CollapsedTreeNodeBrowse(name, path))
}

func HandleTreeExpandBrowse(c *fiber.Ctx) error {
	_, userID := CurrentUser(c)
	loc := getLocation(c)
	path := c.Params("*")
	name, level, parts := parsePath(path)

	// Get category from query parameter
	category := AdCategory(c)

	// Browse mode: No ad IDs filtering needed
	log.Printf("[tree-view] Browse mode: no ad ID filtering")

	// Get children for the current level using browse mode SQL functions
	var children []string
	var ads []ad.Ad
	var err error
	switch level {
	case 0: // Root level - get makes
		children, err = vehicle.GetAdMakes(category)
	case 1: // Make level - get years
		makeName := parts[0]
		children, err = vehicle.GetAdYears(category, makeName)
	case 2: // Year level - get models
		makeName, year := parts[0], parts[1]
		children, err = vehicle.GetAdModels(category, makeName, year)
	case 3: // Model level - get engines
		makeName, year, model := parts[0], parts[1], parts[2]
		children, err = vehicle.GetAdEngines(category, makeName, year, model)
	case 4: // Engine level - get categories
		makeName, year, model, engine := parts[0], parts[1], parts[2], parts[3]
		children, err = part.GetAdCategories(category, makeName, year, model, engine)
	case 5: // Category level - get subcategories
		makeName, year, model, engine, partCategory := parts[0], parts[1], parts[2], parts[3], parts[4]
		children, err = part.GetAdSubCategories(category, makeName, year, model, engine, partCategory)
	case 6: // Subcategory level - get ads
		makeName, year, model, engine, partCategory, subcategory := parts[0], parts[1], parts[2], parts[3], parts[4], parts[5]
		log.Printf("[tree-view] Getting ads for make=%s, year=%s, model=%s, engine=%s, category=%s, subcategory=%s", makeName, year, model, engine, partCategory, subcategory)
		ads, err = ad.GetAdsForAll()
		if err != nil {
			return err
		}
		log.Printf("[tree-view] Found %d ads", len(ads))
	}

	if err != nil {
		return err
	}

	// At root level, show empty response if no makes available
	if level == 0 && len(children) == 0 {
		return render(c, ui.EmptyResponse())
	}

	return render(c, ui.ExpandedTreeNodeBrowse(name, path, level, loc, userID, children, ads))
}

func HandleTreeCollapseSearch(c *fiber.Ctx) error {
	path := c.Params("*")
	name, _, _ := parsePath(path)
	return render(c, ui.CollapsedTreeNodeSearch(name, path))
}

func HandleTreeExpandSearch(c *fiber.Ctx) error {
	_, userID := CurrentUser(c)
	loc := getLocation(c)
	path := c.Params("*")
	name, level, parts := parsePath(path)

	// Search mode: Get ad IDs from DOM storage (passed via HTMX)
	adIDsStr := c.Query("adIDs")
	if adIDsStr == "" {
		return fmt.Errorf("no adIDs provided for search mode")
	}

	adIDs, err := decodeAdIDs(adIDsStr)
	if err != nil {
		return fmt.Errorf("failed to parse adIDs: %w", err)
	}
	log.Printf("[tree-search] Using %d ad IDs from DOM storage", len(adIDs))

	// Get children for the current level using search mode SQL functions
	var children []string
	var ads []ad.Ad
	switch level {
	case 0: // Root level - get makes
		children, err = vehicle.GetAdMakesForAdIDs(adIDs)
	case 1: // Make level - get years
		makeName := parts[0]
		children, err = vehicle.GetAdYearsForAdIDs(adIDs, makeName)
	case 2: // Year level - get models
		makeName, year := parts[0], parts[1]
		children, err = vehicle.GetAdModelsForAdIDs(adIDs, makeName, year)
	case 3: // Model level - get engines
		makeName, year, model := parts[0], parts[1], parts[2]
		children, err = vehicle.GetAdEnginesForAdIDs(adIDs, makeName, year, model)
	case 4: // Engine level - get categories
		makeName, year, model, engine := parts[0], parts[1], parts[2], parts[3]
		children, err = part.GetCategoriesForAds(adIDs, makeName, year, model, engine)
	case 5: // Category level - get subcategories
		makeName, year, model, engine, category := parts[0], parts[1], parts[2], parts[3], parts[4]
		children, err = part.GetSubCategoriesForAds(adIDs, makeName, year, model, engine, category)
	case 6: // Subcategory level - get ads
		ads, err = ad.GetAdsForAdIDs(adIDs)
		if err != nil {
			return err
		}
	}

	if err != nil {
		return err
	}

	// At root level, show empty response if no makes available
	if level == 0 && len(children) == 0 {
		return render(c, ui.EmptyResponse())
	}

	return render(c, ui.ExpandedTreeNodeSearch(name, path, level, loc, userID, children, ads))
}
