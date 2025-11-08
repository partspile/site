package handlers

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/cookie"
	"github.com/parts-pile/site/local"
	"github.com/parts-pile/site/logger"
	"github.com/parts-pile/site/part"
	"github.com/parts-pile/site/spec"
	"github.com/parts-pile/site/ui"
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
	userID := local.GetUserID(c)
	userName := local.GetUserName(c)
	loc := getLocation(c)
	path := c.Params("*")
	name, level, parts := parsePath(path)

	category := cookie.GetAdCategory(c)

	// Get category fields to determine tree structure
	fieldConfigs, err := spec.GetCategoryFields(category)
	if err != nil {
		return fmt.Errorf("failed to get category fields: %w", err)
	}

	// Browse mode: No ad IDs filtering needed
	logger.Debug("Browse mode: no ad ID filtering",
		"component", "tree-view")

	// Get children for the current level using dynamic field configuration
	var children []string
	var ads []ad.Ad

	// Map level to field index
	// Level 0 = first field (usually make)
	// Level 1 = second field (usually year)
	// etc.
	// After all spec fields, we have part category and subcategory
	// Then finally ads

	if level < len(fieldConfigs) {
		// This is a spec field level
		fieldConfig := fieldConfigs[level]
		parentValues := buildParentValuesFromPath(parts, fieldConfigs, level)

		// Get field values dynamically
		values, err := spec.GetFieldValues(category, fieldConfig.Name, parentValues)
		if err != nil {
			return fmt.Errorf("failed to get field values: %w", err)
		}
		children = values
	} else if level == len(fieldConfigs) {
		// Part category level
		parentValues := buildParentValuesFromPath(parts, fieldConfigs, level)
		children, err = part.GetAdCategories(category, parentValues)
	} else if level == len(fieldConfigs)+1 {
		// Part subcategory level
		parentValues := buildParentValuesFromPath(parts[:len(parts)-1], fieldConfigs, level-1)
		partCategory := parts[len(parts)-1]
		children, err = part.GetAdSubCategories(category, parentValues, partCategory)
	} else if level == len(fieldConfigs)+2 {
		// Ads level
		ads, err = ad.GetAdsForAll()
		if err != nil {
			return err
		}
		logger.Debug("Found ads",
			"component", "tree-view", "count", len(ads))
	} else {
		return fmt.Errorf("invalid tree level: %d", level)
	}

	if err != nil {
		return err
	}

	// At root level, show empty response if no children available
	if level == 0 && len(children) == 0 {
		return render(c, ui.EmptyResponse())
	}

	csrfToken := local.GetCSRFToken(c)
	return render(c, ui.ExpandedTreeNodeBrowse(name, path, level, loc, userID, userName, children, ads, csrfToken))
}

// buildParentValuesFromPath builds parent field values from path parts
func buildParentValuesFromPath(parts []string, fieldConfigs []spec.FieldConfig, currentLevel int) map[string]string {
	parentValues := make(map[string]string)
	for i := 0; i < currentLevel && i < len(parts) && i < len(fieldConfigs); i++ {
		fieldConfig := fieldConfigs[i]
		parentValues[fieldConfig.Name] = parts[i]
	}
	return parentValues
}

func HandleTreeCollapseSearch(c *fiber.Ctx) error {
	path := c.Params("*")
	name, _, _ := parsePath(path)
	return render(c, ui.CollapsedTreeNodeSearch(name, path))
}

func HandleTreeExpandSearch(c *fiber.Ctx) error {
	userID := local.GetUserID(c)
	userName := local.GetUserName(c)
	loc := getLocation(c)
	path := c.Params("*")
	name, level, parts := parsePath(path)

	category := cookie.GetAdCategory(c)

	// Search mode: Get ad IDs from DOM storage (passed via HTMX)
	adIDsStr := c.Query("adIDs")
	if adIDsStr == "" {
		return fmt.Errorf("no adIDs provided for search mode")
	}

	adIDs, err := decodeAdIDs(adIDsStr)
	if err != nil {
		return fmt.Errorf("failed to parse adIDs: %w", err)
	}
	logger.Debug("Using ad IDs from DOM storage",
		"component", "tree-search", "count", len(adIDs))

	// Get category fields to determine tree structure
	fieldConfigs, err := spec.GetCategoryFields(category)
	if err != nil {
		return fmt.Errorf("failed to get category fields: %w", err)
	}

	// Get children for the current level using dynamic field configuration
	var children []string
	var ads []ad.Ad

	// Map level to field index (same as browse mode)
	if level < len(fieldConfigs) {
		// This is a spec field level - use generic search mode function
		fieldConfig := fieldConfigs[level]
		parentValues := buildParentValuesFromPath(parts, fieldConfigs, level)

		// Use generic function that works with any field
		children, err = spec.GetAdFieldValuesForAdIDs(category, fieldConfig.Name, adIDs, parentValues)
		if err != nil {
			logger.Warn("Failed to get field values for ad IDs",
				"error", err, "component", "tree-search", "field", fieldConfig.Name, "adCategoryID", category)
			children = []string{}
		}
	} else if level == len(fieldConfigs) {
		// Part category level
		parentValues := buildParentValuesFromPath(parts, fieldConfigs, level)
		children, err = part.GetCategoriesForAds(adIDs, parentValues)
	} else if level == len(fieldConfigs)+1 {
		// Part subcategory level
		parentValues := buildParentValuesFromPath(parts[:len(parts)-1], fieldConfigs, level-1)
		category := parts[len(parts)-1]
		children, err = part.GetSubCategoriesForAds(adIDs, parentValues, category)
	} else if level == len(fieldConfigs)+2 {
		// Ads level
		ads, err = ad.GetAdsForAdIDs(adIDs)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("invalid tree level: %d", level)
	}

	if err != nil {
		return err
	}

	// At root level, show empty response if no makes available
	if level == 0 && len(children) == 0 {
		return render(c, ui.EmptyResponse())
	}

	csrfToken := local.GetCSRFToken(c)
	return render(c, ui.ExpandedTreeNodeSearch(name, path, level, loc, userID, userName, children, ads, csrfToken))
}
