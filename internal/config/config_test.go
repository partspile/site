package config

import "testing"

func TestTrimURLBase(t *testing.T) {
	if got := trimURLBase(" https://parts-pile.com/ "); got != "https://parts-pile.com" {
		t.Fatalf("got %q", got)
	}
}
