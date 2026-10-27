package search

import (
	"fmt"
	"sort"
	"strings"

	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/facet"
)

func buildVectorMetadataWhere(p Params, pa *pgArgs) string {
	clause := `(vector_metadata->>'category_id')::int = ` +
		pa.add(p.CategoryID)
	clause += ` AND rock_count <= ` + pa.add(config.MaxRockCount)

	if !p.Expanded {
		return clause
	}

	for _, key := range sortedFacetKeys(p.FacetFilters) {
		f := p.FacetFilters[key]
		if !f.Active() {
			continue
		}
		d, ok := facet.Get(key)
		if !ok {
			continue
		}
		part := vectorFilterClause(d, f, pa)
		if part != "" {
			clause += " AND " + part
		}
	}

	return clause
}

func vectorFilterClause(d facet.Def, f facet.Filter, pa *pgArgs) string {
	path := `vector_metadata->>'` + d.Key + `'`
	switch d.Kind {
	case facet.MultiEnum:
		if len(f.Values) == 0 {
			return ""
		}
		valuePH := make([]string, len(f.Values))
		for i, v := range f.Values {
			valuePH[i] = pa.add(v)
		}
		return fmt.Sprintf(
			`EXISTS (SELECT 1 FROM jsonb_array_elements_text(`+
				`vector_metadata->'%s') AS je(value) `+
				`WHERE je.value IN (%s))`,
			d.Key, strings.Join(valuePH, ","),
		)
	case facet.Date:
		clause := ""
		if f.TextMin != nil {
			clause += path + ` >= ` + pa.add(*f.TextMin)
		}
		if f.TextMax != nil {
			if clause != "" {
				clause += " AND "
			}
			clause += path + ` <= ` + pa.add(*f.TextMax)
		}
		return clause
	case facet.Flag:
		if f.Value != nil && *f.Value == "1" {
			return `(` + path + `)::int = ` + pa.add(1)
		}
		return ""
	case facet.Money, facet.Int:
		numPath := `(` + path + `)::int`
		clause := ""
		if f.Min != nil {
			clause += numPath + ` >= ` + pa.add(*f.Min)
		}
		if f.Max != nil {
			if clause != "" {
				clause += " AND "
			}
			clause += numPath + ` <= ` + pa.add(*f.Max)
		}
		return clause
	default:
		clause := ""
		if len(f.Values) > 0 {
			valuePH := make([]string, len(f.Values))
			for i, v := range f.Values {
				valuePH[i] = pa.add(v)
			}
			clause = path + ` IN (` + strings.Join(valuePH, ",") + `)`
		} else if f.Value != nil {
			clause = path + ` = ` + pa.add(*f.Value)
		}
		if f.Min != nil {
			numPath := `(` + path + `)::int`
			if clause != "" {
				clause += " AND "
			}
			clause += numPath + ` >= ` + pa.add(*f.Min)
		}
		if f.Max != nil {
			numPath := `(` + path + `)::int`
			if clause != "" {
				clause += " AND "
			}
			clause += numPath + ` <= ` + pa.add(*f.Max)
		}
		return clause
	}
}

func sortedFacetKeys(filters map[string]facet.Filter) []string {
	keys := make([]string, 0, len(filters))
	for k := range filters {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
