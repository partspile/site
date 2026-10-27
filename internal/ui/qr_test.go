package ui

import (
	"strings"
	"testing"
)

func TestQRPNGDataURI(t *testing.T) {
	if got := qrPNGDataURI("", 192); got != "" {
		t.Fatal("empty content should yield empty src")
	}
	got := qrPNGDataURI("https://parts-pile.com/ad/1", 192)
	if !strings.HasPrefix(got, "data:image/png;base64,") {
		t.Fatalf("unexpected src prefix: %q", got)
	}
	if len(got) < 100 {
		t.Fatalf("QR data URI too short: %d", len(got))
	}
}
