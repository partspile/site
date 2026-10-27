package ad_test

import (
	"testing"
	"time"

	"github.com/partspile/site/internal/ad"
	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/facet"
)

func TestExpiresAtFromSaleEnd(t *testing.T) {
	got, err := ad.ExpiresAtFromSaleEnd("2026-07-04")
	if err != nil {
		t.Fatal(err)
	}
	// End of 2026-07-04 UTC is 2026-07-05 00:00; plus 1 week => 2026-07-12.
	want := time.Date(2026, 7, 12, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("ExpiresAtFromSaleEnd = %v, want %v", got, want)
	}
}

func TestComputeExpiresAtPrefersSaleEnd(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := "2026-01-10"
	facets := map[string]facet.Value{
		"sale_end_date": {Text: &end},
	}
	got := ad.ComputeExpiresAt(facets, now)
	want, err := ad.ExpiresAtFromSaleEnd(end)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Equal(want) {
		t.Fatalf("ComputeExpiresAt = %v, want %v", got, want)
	}
}

func TestComputeExpiresAtDefaultMonths(t *testing.T) {
	now := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	got := ad.ComputeExpiresAt(nil, now)
	want := now.AddDate(0, config.AdExpireMonths, 0)
	if !got.Equal(want) {
		t.Fatalf("ComputeExpiresAt = %v, want %v", got, want)
	}
}

func TestRenewEligible(t *testing.T) {
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		at   time.Time
		want bool
	}{
		{"exact_1mo", now.AddDate(0, 1, 0), true},
		{"under_1mo", now.AddDate(0, 0, 20), true},
		{"soon", now.Add(2 * time.Hour), true},
		{"past", now.Add(-time.Hour), true},
		{"over_1mo", now.AddDate(0, 1, 1), false},
		{"fresh_3mo", now.AddDate(0, 3, 0), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ad.RenewEligible(tt.at, now)
			if got != tt.want {
				t.Fatalf("RenewEligible(%v) = %v, want %v", tt.at, got, tt.want)
			}
		})
	}
}
