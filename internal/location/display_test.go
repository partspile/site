package location

import "testing"

func TestDisplayText(t *testing.T) {
	got := DisplayText("Klamath Falls", "OR", "US")
	if got != "Klamath Falls, OR" {
		t.Fatalf("unexpected display: %q", got)
	}
}
