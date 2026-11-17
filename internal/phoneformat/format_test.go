package phoneformat

import "testing"

func TestDisplay(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"+12025550123", "(202) 555-0123"},
		{"", ""},
		{"not-a-phone", "not-a-phone"},
	}
	for _, tc := range tests {
		got := Display(tc.in)
		if got != tc.want {
			t.Errorf("Display(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
