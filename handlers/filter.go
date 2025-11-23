package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/cookie"
	"github.com/parts-pile/site/logger"
	"github.com/qdrant/go-client/qdrant"
)

func buildSearchFilter(c *fiber.Ctx) *qdrant.Filter {
	var conditions []*qdrant.Condition

	ac := cookie.GetAdCategory(c)
	conditions = append(conditions, qdrant.NewMatchInt("ad_category_id", int64(ac)))
	logger.Debug("Added ad_category_id filter",
		"component", "buildSearchFilters", "adCategoryID", ac)

	fields := ad.GetCategoryFields(ac)

	// Build filter conditions using Field interface
	for _, field := range fields {
		condition := field.BuildFilterCondition(c)
		if condition != nil {
			conditions = append(conditions, condition)
			logger.Debug("Added filter condition",
				"component", "buildSearchFilters", "field", field.Name(),
				"qdrantType", field.GetQdrantType())
		}
	}

	// Rock count filter - exclude ads with too many rocks
	rockCondition := qdrant.NewRange("rock_count", &qdrant.Range{
		Gte: qdrant.PtrOf(0.0),
		Lte: qdrant.PtrOf(float64(config.MaxRockCount)),
	})
	conditions = append(conditions, rockCondition)
	logger.Debug("Added rock_count filter",
		"component", "buildSearchFilters", "maxRockCount",
		config.MaxRockCount)

	// Create filter, all conditions MUST match
	filter := &qdrant.Filter{
		Must: conditions,
	}

	logger.Debug("Built filter",
		"component", "buildSearchFilters", "conditionCount",
		len(conditions))
	return filter
}
