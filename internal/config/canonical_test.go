package config

import "testing"

func TestCanonicalURL(t *testing.T) {
	prev := PublicSiteURL
	PublicSiteURL = "https://parts-pile.com"
	t.Cleanup(func() { PublicSiteURL = prev })

	tests := []struct {
		path, want string
	}{
		{"", "https://parts-pile.com/"},
		{"/", "https://parts-pile.com/"},
		{"/login", "https://parts-pile.com/login"},
		{"/login?return=/", "https://parts-pile.com/login"},
		{"about", "https://parts-pile.com/about"},
	}
	for _, tt := range tests {
		if got := CanonicalURL(tt.path); got != tt.want {
			t.Errorf("CanonicalURL(%q) = %q, want %q",
				tt.path, got, tt.want)
		}
	}
}

func TestPublicHost(t *testing.T) {
	prev := PublicSiteURL
	t.Cleanup(func() { PublicSiteURL = prev })

	PublicSiteURL = "https://parts-pile.com"
	if got := PublicHost(); got != "parts-pile.com" {
		t.Errorf("PublicHost() = %q, want parts-pile.com", got)
	}

	PublicSiteURL = "https://www.example.com:8443/path"
	if got := PublicHost(); got != "www.example.com" {
		t.Errorf("PublicHost() = %q, want www.example.com", got)
	}

	PublicSiteURL = ""
	if got := PublicHost(); got != "" {
		t.Errorf("PublicHost() = %q, want empty", got)
	}
}
