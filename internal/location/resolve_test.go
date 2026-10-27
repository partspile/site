package location

import (
	"testing"

	"github.com/partspile/site/internal/service/geoapify"
)

func TestLocationFromGeoapifyUS(t *testing.T) {
	got := locationFromGeoapify(&geoapify.Result{
		City:        "Corvallis",
		State:       "Oregon",
		StateCode:   "OR",
		Country:     "United States",
		CountryCode: "us",
		Lat:         44.5646,
		Lon:         -123.2620,
	})
	if got.City != "Corvallis" || got.AdminArea != "OR" ||
		got.Country != "US" {
		t.Fatalf("unexpected location: %+v", got)
	}
	if got.Latitude != 44.5646 || got.Longitude != -123.2620 {
		t.Fatalf("coords = %v,%v", got.Latitude, got.Longitude)
	}
}

func TestLocationFromGeoapifyCanada(t *testing.T) {
	got := locationFromGeoapify(&geoapify.Result{
		City:        "Vancouver",
		State:       "British Columbia",
		StateCode:   "BC",
		CountryCode: "ca",
		Lat:         49.2827,
		Lon:         -123.1207,
	})
	if got.AdminArea != "BC" || got.Country != "CA" {
		t.Fatalf("unexpected location: %+v", got)
	}
}

func TestLocationFromGeoapifyOtherCountry(t *testing.T) {
	got := locationFromGeoapify(&geoapify.Result{
		City:        "Berlin",
		State:       "Berlin",
		StateCode:   "BE",
		CountryCode: "de",
		Lat:         52.52,
		Lon:         13.405,
	})
	if got.AdminArea != "Berlin" || got.Country != "DE" {
		t.Fatalf("unexpected location: %+v", got)
	}
}

func TestLocationFromGeoapifyUSMissingStateCode(t *testing.T) {
	got := locationFromGeoapify(&geoapify.Result{
		City:        "Corvallis",
		State:       "Oregon",
		CountryCode: "us",
		Lat:         44.5646,
		Lon:         -123.2620,
	})
	if got.AdminArea != "Oregon" {
		t.Fatalf("admin_area = %q", got.AdminArea)
	}
}

func TestLocationFromGeoapifyCountyFallback(t *testing.T) {
	got := locationFromGeoapify(&geoapify.Result{
		County:      "Lincoln County",
		State:       "Oregon",
		StateCode:   "OR",
		CountryCode: "us",
		Lat:         44.6215,
		Lon:         -123.7579,
	})
	if got.City != "Lincoln County" || got.AdminArea != "OR" {
		t.Fatalf("unexpected location: %+v", got)
	}
}
