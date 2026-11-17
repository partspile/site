package currency

import "testing"

func TestDefaultFromPhone(t *testing.T) {
	tests := []struct {
		phone string
		want  string
	}{
		{"", Default},
		{"+14155552671", "USD"},
		{"+442079460123", "GBP"},
		{"+33123456789", "EUR"},
		{"+819012345678", "JPY"},
	}
	for _, tt := range tests {
		if got := DefaultFromPhone(tt.phone); got != tt.want {
			t.Errorf("DefaultFromPhone(%q) = %q, want %q", tt.phone, got, tt.want)
		}
	}
}

func TestFormat(t *testing.T) {
	tests := []struct {
		amount int
		code   string
		want   string
	}{
		{0, "USD", "FREE"},
		{0, "EUR", "FREE"},
		{WantedAmount, "USD", WantedLabel},
		{100, "USD", "$ 100"},
		{250, "GBP", "£ 250"},
		{1245, "USD", "$ 1,245"},
	}
	for _, tt := range tests {
		if got := Format(tt.amount, tt.code); got != tt.want {
			t.Errorf("Format(%d, %q) = %q, want %q", tt.amount, tt.code, got, tt.want)
		}
	}
}

func TestIsSupported(t *testing.T) {
	if !IsSupported("usd") {
		t.Error("expected USD to be supported")
	}
	if IsSupported("XYZ") {
		t.Error("expected XYZ to be unsupported")
	}
}
