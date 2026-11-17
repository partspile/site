package search

import (
	"strings"
	"testing"

	"github.com/partspile/site/internal/currency"
	"github.com/partspile/site/internal/facet"
)

func TestBuildVectorMetadataWhereUsesFlagColumn(t *testing.T) {
	var pa pgArgs
	p := Params{CategoryID: 6}
	clause := buildVectorMetadataWhere(p, &pa)
	if strings.Contains(clause, "flag_count')") {
		t.Fatalf("should not use metadata flag_count: %s", clause)
	}
	if !strings.Contains(clause, "flag_count <=") {
		t.Fatalf("expected flag_count column filter: %s", clause)
	}
	if !strings.Contains(clause, "category_id") {
		t.Fatalf("expected category filter when CategoryID set: %s", clause)
	}
}

func TestBuildVectorMetadataWhereOptionalCategory(t *testing.T) {
	var pa pgArgs
	clause := buildVectorMetadataWhere(Params{}, &pa)
	if strings.Contains(clause, "category_id") {
		t.Fatalf("expected no category filter when CategoryID is 0: %s", clause)
	}
	if !strings.Contains(clause, "flag_count <=") {
		t.Fatalf("expected flag_count column filter: %s", clause)
	}
}

func TestMoneyMaxFilterExcludesWanted(t *testing.T) {
	var pa pgArgs
	max := 50
	p := Params{FacetFilters: map[string]facet.Filter{
		"price": {Max: &max},
	}}
	clause := buildVectorMetadataWhere(p, &pa)
	if !strings.Contains(clause, ">= $2") || !strings.Contains(clause, "<= $3") {
		t.Fatalf("max price filter should bound both sides: %s", clause)
	}
	low, lowOK := pa.args[1].(int)
	high, highOK := pa.args[2].(int)
	if !lowOK || !highOK || low != 0 || high != max {
		t.Fatalf("price args = %v", pa.args[1:])
	}
}

func TestMoneyWantedFilter(t *testing.T) {
	var pa pgArgs
	wanted := currency.WantedFilter
	p := Params{FacetFilters: map[string]facet.Filter{
		"price": {Value: &wanted},
	}}
	clause := buildVectorMetadataWhere(p, &pa)
	if !strings.Contains(clause, "= $2") {
		t.Fatalf("wanted filter = %s", clause)
	}
	got, ok := pa.args[1].(int)
	if !ok || got != currency.WantedAmount {
		t.Fatalf("wanted arg = %v", pa.args[1])
	}
}

func TestBuildVectorMetadataWhereNoGeoFilter(t *testing.T) {
	var pa pgArgs
	p := Params{
		CategoryID: 6,
		HasGeo:     true,
		CenterLat:  34.0,
		CenterLon:  -118.0,
		WithinKm:   50,
	}
	clause := buildVectorMetadataWhere(p, &pa)
	if strings.Contains(clause, "BETWEEN") {
		t.Fatalf("geo should not hard-filter results: %s", clause)
	}
}
