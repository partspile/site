package main

import (
	"testing"

	"github.com/partspile/site/internal/db"
	"github.com/partspile/site/internal/location"
)

func TestIntegrationResolveLocationCacheHit(t *testing.T) {
	var rawText string
	var wantLat, wantLon float64
	err := db.QueryRow(`
		SELECT raw_text, latitude, longitude FROM locations
		WHERE city = 'Portland' LIMIT 1`,
	).Scan(&rawText, &wantLat, &wantLon)
	if err != nil {
		t.Fatalf("seed Portland location: %v", err)
	}

	lat, lon, ok, err := location.ResolveLocation(rawText)
	if err != nil || !ok {
		t.Fatalf("ResolveLocation: ok=%v err=%v", ok, err)
	}
	if lat != wantLat || lon != wantLon {
		t.Fatalf("unexpected coords: %v, %v want %v, %v", lat, lon, wantLat, wantLon)
	}
}
