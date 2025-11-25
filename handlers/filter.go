package handlers

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/cookie"
	"github.com/parts-pile/site/logger"
)

// FilterClause represents a SQL WHERE clause and its arguments
type FilterClause struct {
	WhereClause string
	Args        []interface{}
}

func buildSearchFilter(c *fiber.Ctx) FilterClause {
	var conditions []string
	var args []interface{}
	argIndex := 1

	ac := cookie.GetAdCategory(c)
	conditions = append(conditions, fmt.Sprintf("(vector_metadata->>$%d)::int = $%d", argIndex, argIndex+1))
	args = append(args, "ad_category_id", ac)
	argIndex += 2
	logger.Debug("Added ad_category_id filter",
		"component", "buildSearchFilters", "adCategoryID", ac)

	fields := ad.GetCategoryFields(ac)

	// Build filter conditions using Field interface
	for _, field := range fields {
		whereClause, fieldArgs := field.BuildFilterCondition(c)
		if whereClause != "" && len(fieldArgs) > 0 {
			// Replace placeholders in whereClause to account for current argIndex
			adjustedClause := replacePlaceholders(whereClause, argIndex)
			conditions = append(conditions, "("+adjustedClause+")")
			args = append(args, fieldArgs...)
			argIndex += len(fieldArgs)
			logger.Debug("Added filter condition",
				"component", "buildSearchFilters", "field", field.Name(),
				"vectorType", field.GetVectorType())
		}
	}

	// Rock count filter - exclude ads with too many rocks
	conditions = append(conditions, fmt.Sprintf("(vector_metadata->>$%d)::int >= $%d AND (vector_metadata->>$%d)::int <= $%d", argIndex, argIndex+1, argIndex, argIndex+2))
	args = append(args, "rock_count", 0, config.MaxRockCount)
	logger.Debug("Added rock_count filter",
		"component", "buildSearchFilters", "maxRockCount",
		config.MaxRockCount)

	// Combine all conditions with AND
	whereClause := strings.Join(conditions, " AND ")

	logger.Debug("Built filter",
		"component", "buildSearchFilters", "conditionCount",
		len(conditions))
	return FilterClause{WhereClause: whereClause, Args: args}
}

// replacePlaceholders replaces $1, $2, etc. in a SQL query with $startIndex, $startIndex+1, etc.
func replacePlaceholders(query string, startIndex int) string {
	// Simple regex replacement for $N patterns
	result := query
	for i := 1; i <= 100; i++ { // Support up to $100
		oldPlaceholder := fmt.Sprintf("$%d", i)
		newPlaceholder := fmt.Sprintf("$%d", i+startIndex-1)
		result = strings.ReplaceAll(result, oldPlaceholder, newPlaceholder)
	}
	return result
}
