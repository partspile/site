package search

import (
	"strings"
	"testing"
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

func TestBuildVectorMetadataWhereNoGeoFilter(t *testing.T) {
	var pa pgArgs
	p := Params{
		CategoryID: 6,
		Expanded:   true,
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
