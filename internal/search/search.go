package search

import (
	"fmt"

	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/logger"
	"github.com/partspile/site/internal/vector"
)

func Search(p Params) (Results, error) {
	logger.Debug("search request",
		"categoryID", p.CategoryID,
		"userID", p.UserID,
		"q", p.Q,
		"expanded", p.Expanded,
		"limit", p.Limit,
		"offset", p.Offset,
		"hasGeo", p.HasGeo,
		"facetFilters", len(p.FacetFilters),
		"threshold", config.SearchThreshold,
	)

	embedding, err := vector.ResolveSearchEmbedding(p.UserID, p.CategoryID, p.Q)
	if err != nil {
		if vector.IsEmbeddingUnavailable(err) {
			logger.Info("search embedding unavailable",
				"error", err, "q", p.Q)
			return Results{}, nil
		}
		logger.Debug("search embedding error", "error", err)
		return Results{}, fmt.Errorf("search embedding: %w", err)
	}

	var pa pgArgs
	where := buildVectorMetadataWhere(p, &pa)

	if !p.HasGeo {
		ids, err := vector.QuerySimilarAdIDs(
			embedding, where, pa.args,
			p.Limit, p.Offset, config.SearchThreshold,
		)
		if err != nil {
			logger.Debug("search vector query error", "error", err)
			return Results{}, err
		}
		logger.Debug("search complete",
			"categoryID", p.CategoryID,
			"q", p.Q,
			"resultCount", len(ids),
		)
		return Results{IDs: ids}, nil
	}

	results, err := searchGeo(embedding, where, &pa, p)
	if err != nil {
		logger.Debug("search vector query error", "error", err)
		return Results{}, err
	}
	logger.Debug("search complete",
		"categoryID", p.CategoryID,
		"q", p.Q,
		"resultCount", len(results.IDs),
		"inAreaOnPage", results.InAreaOnPage,
		"hasAnyInArea", results.HasAnyInArea,
	)
	return results, nil
}

func searchGeo(embedding []float32, where string, pa *pgArgs,
	p Params) (Results, error) {
	bbox := geoInAreaExpr(p, pa)
	inWhere := where + " AND (" + bbox + ")"
	outWhere := where + " AND (" + geoOutOfAreaExpr(bbox) + ")"
	args := pa.args
	thresh := config.SearchThreshold

	inIDs, err := vector.QuerySimilarAdIDs(
		embedding, inWhere, args, p.Limit, p.Offset, thresh,
	)
	if err != nil {
		return Results{}, err
	}
	if len(inIDs) == p.Limit {
		return Results{
			IDs:          inIDs,
			InAreaOnPage: len(inIDs),
			HasAnyInArea: true,
		}, nil
	}

	need := p.Limit - len(inIDs)
	outOffset := 0
	hasAnyInArea := len(inIDs) > 0
	if len(inIDs) == 0 && p.Offset > 0 {
		earlier, err := vector.QuerySimilarAdIDs(
			embedding, inWhere, args, p.Offset, 0, thresh,
		)
		if err != nil {
			return Results{}, err
		}
		hasAnyInArea = len(earlier) > 0
		outOffset = p.Offset - len(earlier)
	}

	outIDs, err := vector.QuerySimilarAdIDs(
		embedding, outWhere, args, need, outOffset, thresh,
	)
	if err != nil {
		return Results{}, err
	}

	ids := make([]int, 0, len(inIDs)+len(outIDs))
	ids = append(ids, inIDs...)
	ids = append(ids, outIDs...)
	return Results{
		IDs:          ids,
		InAreaOnPage: len(inIDs),
		HasAnyInArea: hasAnyInArea,
	}, nil
}
