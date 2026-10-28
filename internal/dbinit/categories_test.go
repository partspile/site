package dbinit

import (
	"encoding/json"
	"testing"
)

func TestPartsCategoriesIncludeLocalPickup(t *testing.T) {
	var cats []categoryJSON
	if err := json.Unmarshal(categoriesJSON, &cats); err != nil {
		t.Fatal(err)
	}
	wantPickup := map[string]bool{
		"Car & Truck Parts":            true,
		"Motorcycle Parts":             true,
		"Bicycle Parts":                true,
		"Agricultural Equipment Parts": true,
	}
	found := make(map[string]bool)
	for _, cat := range cats {
		hasPickup := false
		for _, key := range cat.Facets {
			if key == "local_pickup" {
				hasPickup = true
				break
			}
		}
		if want, ok := wantPickup[cat.Name]; ok {
			found[cat.Name] = true
			if hasPickup != want {
				t.Errorf("%s local_pickup=%v, want %v",
					cat.Name, hasPickup, want)
			}
			continue
		}
		if hasPickup {
			t.Errorf("%s should not have local_pickup facet", cat.Name)
		}
	}
	for name := range wantPickup {
		if !found[name] {
			t.Errorf("missing category %s", name)
		}
	}
}
