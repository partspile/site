package search

import (
	"testing"
)

func TestSearchGeoBandsUseColumns(t *testing.T) {
	var pa pgArgs
	p := Params{
		CategoryID: 1,
		HasGeo:     true,
		CenterLat:  45.5,
		CenterLon:  -122.6,
		WithinKm:   25,
		Limit:      20,
		Offset:     0,
	}
	where := buildVectorMetadataWhere(p, &pa)
	bbox := geoInAreaExpr(p, &pa)
	inWhere := where + " AND (" + bbox + ")"
	outWhere := where + " AND (" + geoOutOfAreaExpr(bbox) + ")"
	if outWhere == where || inWhere == where {
		t.Fatal("geo bands should extend base where")
	}
	if got := len(pa.args); got != 6 {
		t.Fatalf("expected category+flag+bbox args (6), got %d", got)
	}
}
