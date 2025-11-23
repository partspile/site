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
	loc := cookie.GetLocation(c)
	category := cookie.GetAdCategory(c)
	path := c.Params("*")

	name, level, parts := parsePath(path)

	// Get all category fields - these are already flattened in order
	// Tree structure: first field of first chain -> last field of last chain
	fields := ad.GetCategoryFields(category)

	// Get children for the current level using dynamic field configuration
	var children []string
	var ads []ad.Ad
	var err error

	// Map level to field index
	// Level 0 = first field (e.g., make)
	// Level 1 = second field (e.g., year)
	// ...
	// Level len(fields)-1 = last field (e.g., part_subcategory)
	// Level len(fields) = ads

	if level < len(fields) {
		// This is a field level
		field := fields[level]
		parentValues := buildParentValuesFromPath(parts, fields, level)

		// Get field values dynamically using Field interface
		children, err = field.GetValues(parentValues)
		if err != nil {
			return fmt.Errorf("failed to get field values for %s: %w", field.Name(), err)
		}
	} else if level == len(fields) {
		// Ads level - show ads filtered by all field values in path
		fieldValues := buildParentValuesFromPath(parts, fields, level)
		ads, err = ad.GetAdsForAll(fieldValues, category, fields)
		if err != nil {
			return fmt.Errorf("failed to get ads: %w", err)
		}
		logger.Debug("Found ads",
			"component", "tree-view", "count", len(ads))
	} else {
		return fmt.Errorf("invalid tree level: %d (max: %d)", level, len(fields))
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
func buildParentValuesFromPath(parts []string, fields []ad.Field, currentLevel int) map[string]string {
	parentValues := make(map[string]string)
	for i := 0; i < currentLevel && i < len(parts) && i < len(fields); i++ {
		field := fields[i]
		parentValues[field.Name()] = parts[i]
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
	loc := cookie.GetLocation(c)
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

	// Get all category fields - these are already flattened in order
	fields := ad.GetCategoryFields(category)

	// Get children for the current level using dynamic field configuration
	var children []string
	var ads []ad.Ad

	// Map level to field index (same as browse mode)
	if level < len(fields) {
		// This is a field level
		field := fields[level]
		parentValues := buildParentValuesFromPath(parts, fields, level)

		// TODO: Need to filter field values by ad IDs for search mode
		// For now, get all values (will need new method or approach)
		children, err = field.GetValues(parentValues)
		if err != nil {
			logger.Warn("Failed to get field values for ad IDs",
				"error", err, "component", "tree-search", "field", field.Name(), "adCategoryID", category)
			children = []string{}
		}
		// TODO: Filter children to only include values that exist in the ad IDs
	} else if level == len(fields) {
		// Ads level
		ads, err = ad.GetAdsForAdIDs(adIDs)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("invalid tree level: %d (max: %d)", level, len(fields))
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
