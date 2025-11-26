package handlers

import (
	"fmt"
	"regexp"
	"strconv"
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
	acClause := "(vector_metadata->>'ad_category_id')::int = $1::int"
	conditions = append(conditions, acClause)
	args = append(args, ac)
	argIndex += 1
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
	rockClause := "(vector_metadata->>'rock_count')::int >= $1::int AND (vector_metadata->>'rock_count')::int <= $2::int"
	rockClause = replacePlaceholders(rockClause, argIndex)
	conditions = append(conditions, rockClause)
	args = append(args, 0, config.MaxRockCount)
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
	// Use regex to match $N patterns and replace them correctly
	// This avoids replacing $1 inside $10, $11, etc.
	re := regexp.MustCompile(`\$(\d+)`)
	return re.ReplaceAllStringFunc(query, func(match string) string {
		numStr := match[1:]
		num, _ := strconv.Atoi(numStr)
		return fmt.Sprintf("$%d", num+startIndex-1)
	})
}
