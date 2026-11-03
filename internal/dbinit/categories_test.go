package dbinit

import (
	"encoding/json"
	"testing"
)

func TestAllCategoriesIncludeLocalPickup(t *testing.T) {
	var cats []categoryJSON
	if err := json.Unmarshal(categoriesJSON, &cats); err != nil {
		t.Fatal(err)
	}
	if len(cats) == 0 {
		t.Fatal("expected categories")
	}
	for _, cat := range cats {
		hasPickup := false
		for _, key := range cat.Facets {
			if key == "local_pickup" {
				hasPickup = true
				break
			}
		}
		if !hasPickup {
			t.Errorf("%s missing local_pickup facet", cat.Name)
		}
	}
}
