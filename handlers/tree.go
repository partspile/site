package handlers

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"

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

func HandleTreeCollapseBrowse(c *fiber.Ctx) error {
	fieldName := c.Query("field")
	fieldValue := c.Query(fieldName)
	category := cookie.GetAdCategory(c)
	parentValues := ad.BuildParentValues(c, category, []string{"field", "adIDs"})
	return render(c, ui.CollapsedTreeNodeBrowse(fieldValue, fieldName, parentValues))
}

func HandleTreeExpandBrowse(c *fiber.Ctx) error {
	userID := local.GetUserID(c)
	userName := local.GetUserName(c)
	loc := cookie.GetLocation(c)
	category := cookie.GetAdCategory(c)

	// Get field name from query parameter
	fieldName := c.Query("field")
	if fieldName == "" {
		return fmt.Errorf("field parameter is required")
	}

	// Get parent values from query parameters
	parentValues := ad.BuildParentValues(c, category, []string{"field", "adIDs"})

	// Get the current field value (the one being expanded)
	fieldValue := c.Query(fieldName)

	// Get field chains to determine next field and level
	chains := ad.GetFieldChains(category)
	if len(chains) == 0 {
		return fmt.Errorf("no field chains found for category %d", category)
	}

	// Flatten chains into a single list
	var fields []ad.Field
	for _, chain := range chains {
		fields = append(fields, chain...)
	}

	// Find current field index
	var currentFieldIndex int = -1
	for i, field := range fields {
		if field.Name() == fieldName {
			currentFieldIndex = i
			break
		}
	}
	if currentFieldIndex == -1 {
		return fmt.Errorf("field %s not found in chains", fieldName)
	}

	// Determine next field name (for building child URLs)
	var nextFieldName string
	if currentFieldIndex+1 < len(fields) {
		nextFieldName = fields[currentFieldIndex+1].Name()
	}

	// Get children for the current field
	var children []string
	var ads []ad.Ad
	var err error

	if currentFieldIndex+1 < len(fields) {
		// This is a field level - get children for the NEXT field
		nextField := fields[currentFieldIndex+1]

		// Add current field value to parentValues so next field can filter by it
		if fieldValue != "" {
			parentValues[fieldName] = fieldValue
		}

		children, err = ad.GetValuesForActiveAds(nextField, parentValues)
		if err != nil {
			return fmt.Errorf("failed to get field values for %s: %w", nextField.Name(), err)
		}
	} else {
		// This is the last field - show ads
		// Include current field value in filter
		fieldValues := make(map[string]string)
		for k, v := range parentValues {
			fieldValues[k] = v
		}
		fieldValues[fieldName] = fieldValue

		ads, err = ad.GetAdsForAll(fieldValues, category)
		if err != nil {
			return fmt.Errorf("failed to get ads: %w", err)
		}
		logger.Debug("Found ads",
			"component", "tree-view", "count", len(ads))
	}

	if err != nil {
		return err
	}

	// At root level, show empty response if no children available
	if currentFieldIndex == 0 && len(children) == 0 {
		return render(c, ui.EmptyResponse())
	}

	// Build up auto-expanded path: track nodes that were auto-expanded
	var expandedPath []ui.ExpandedPathNode

	// Initial node
	initialParentValues := make(map[string]string)
	for k, v := range parentValues {
		initialParentValues[k] = v
	}
	expandedPath = append(expandedPath, ui.ExpandedPathNode{
		FieldValue:   fieldValue,
		FieldName:    fieldName,
		FieldIndex:   currentFieldIndex,
		ParentValues: initialParentValues,
	})

	// Auto-expand single-child nodes recursively
	// Continue expanding until we hit multiple children or ads
	currentFieldIdx := currentFieldIndex
	currentFieldName := fieldName
	currentFieldVal := fieldValue
	currentParentVals := make(map[string]string)
	for k, v := range parentValues {
		currentParentVals[k] = v
	}
	currentParentVals[fieldName] = fieldValue

	for len(children) == 1 && currentFieldIdx+1 < len(fields) {
		// Auto-expand the single child
		singleChildValue := children[0]
		currentFieldIdx++
		nextField := fields[currentFieldIdx]
		nextFieldName := nextField.Name()

		// Add to expanded path
		pathParentVals := make(map[string]string)
		for k, v := range currentParentVals {
			pathParentVals[k] = v
		}
		expandedPath = append(expandedPath, ui.ExpandedPathNode{
			FieldValue:   singleChildValue,
			FieldName:    nextFieldName,
			FieldIndex:   currentFieldIdx,
			ParentValues: pathParentVals,
		})

		// Get children for the next field
		children, err = ad.GetValuesForActiveAds(nextField, currentParentVals)
		if err != nil {
			return fmt.Errorf("failed to get field values for %s: %w", nextField.Name(), err)
		}

		// Update for next iteration
		currentFieldName = nextFieldName
		currentFieldVal = singleChildValue
		currentParentVals[nextFieldName] = singleChildValue
	}

	// If we've reached the last field and have a single child, check for ads
	if len(children) == 1 && currentFieldIdx+1 >= len(fields) {
		// This is the last field - check for ads
		fieldValues := make(map[string]string)
		for k, v := range currentParentVals {
			fieldValues[k] = v
		}
		fieldValues[currentFieldName] = currentFieldVal

		ads, err = ad.GetAdsForAll(fieldValues, category)
		if err != nil {
			return fmt.Errorf("failed to get ads: %w", err)
		}
		logger.Debug("Found ads",
			"component", "tree-view", "count", len(ads))
		// Clear children since we're showing ads
		children = nil
		nextFieldName = ""
	}

	csrfToken := local.GetCSRFToken(c)

	// If we have an auto-expanded path, render nested expanded nodes
	if len(expandedPath) > 1 {
		return render(c, ui.ExpandedTreeNodeBrowseAutoExpanded(expandedPath, fields, nextFieldName, children, ads, loc, userID, userName, csrfToken))
	}

	// Otherwise, render normally
	return render(c, ui.ExpandedTreeNodeBrowse(fieldValue, fieldName, nextFieldName, parentValues, currentFieldIndex, loc, userID, userName, children, ads, csrfToken))
}

func HandleTreeCollapseSearch(c *fiber.Ctx) error {
	fieldName := c.Query("field")
	fieldValue := c.Query(fieldName)
	category := cookie.GetAdCategory(c)
	parentValues := ad.BuildParentValues(c, category, []string{"field", "adIDs"})
	return render(c, ui.CollapsedTreeNodeSearch(fieldValue, fieldName, parentValues))
}

func HandleTreeExpandSearch(c *fiber.Ctx) error {
	userID := local.GetUserID(c)
	userName := local.GetUserName(c)
	loc := cookie.GetLocation(c)
	category := cookie.GetAdCategory(c)

	// Get field name from query parameter
	fieldName := c.Query("field")
	if fieldName == "" {
		return fmt.Errorf("field parameter is required")
	}

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

	// Get parent values from query parameters
	parentValues := ad.BuildParentValues(c, category, []string{"field", "adIDs"})

	// Get the current field value (the one being expanded)
	fieldValue := c.Query(fieldName)

	// Get field chains to determine next field and level
	chains := ad.GetFieldChains(category)
	if len(chains) == 0 {
		return fmt.Errorf("no field chains found for category %d", category)
	}

	// Flatten chains into a single list
	var fields []ad.Field
	for _, chain := range chains {
		fields = append(fields, chain...)
	}

	// Find current field index
	var currentFieldIndex int = -1
	var currentField ad.Field
	for i, field := range fields {
		if field.Name() == fieldName {
			currentFieldIndex = i
			currentField = field
			break
		}
	}
	if currentFieldIndex == -1 {
		return fmt.Errorf("field %s not found in chains", fieldName)
	}

	// Determine next field name (for building child URLs)
	var nextFieldName string
	if currentFieldIndex+1 < len(fields) {
		nextFieldName = fields[currentFieldIndex+1].Name()
	}

	// Get children for the current field
	var children []string
	var ads []ad.Ad

	if currentFieldIndex+1 < len(fields) {
		// This is a field level - filter by ad IDs
		children, err = ad.GetValuesForAdIDs(currentField, adIDs)
		if err != nil {
			logger.Warn("Failed to get field values for ad IDs",
				"error", err, "component", "tree-search", "field", fieldName, "adCategoryID", category)
			children = []string{}
		}
	} else {
		// Ads level
		ads, err = ad.GetAdsForAdIDs(adIDs)
		if err != nil {
			return err
		}
	}

	if err != nil {
		return err
	}

	// At root level, show empty response if no children available
	if currentFieldIndex == 0 && len(children) == 0 {
		return render(c, ui.EmptyResponse())
	}

	csrfToken := local.GetCSRFToken(c)
	return render(c, ui.ExpandedTreeNodeSearch(fieldValue, fieldName, nextFieldName, parentValues, currentFieldIndex, loc, userID, userName, children, ads, csrfToken))
}

// IsSearching checks if any filters are applied (search query or field filters)
// Used to differentiate between browse mode (no filters) and search mode (has filters)
func IsSearching(c *fiber.Ctx, adCat int) bool {
	// Check if search query is present
	if c.FormValue("q") != "" {
		return true
	}

	// Check if any field filters are applied
	fields := ad.GetCategoryFields(adCat)
	for _, field := range fields {
		whereClause, args := field.BuildFilterCondition(c)
		if whereClause != "" && len(args) > 0 {
			return true
		}
	}

	return false
}
