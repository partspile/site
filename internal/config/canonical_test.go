package config

import "testing"

func TestCanonicalURL(t *testing.T) {
	prev := PublicSiteURL
	PublicSiteURL = "https://rockyads.com"
	t.Cleanup(func() { PublicSiteURL = prev })

	tests := []struct {
		path, want string
	}{
		{"", "https://rockyads.com/"},
		{"/", "https://rockyads.com/"},
		{"/login", "https://rockyads.com/login"},
		{"/login?return=/", "https://rockyads.com/login"},
		{"about", "https://rockyads.com/about"},
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

	PublicSiteURL = "https://rockyads.com"
	if got := PublicHost(); got != "rockyads.com" {
		t.Errorf("PublicHost() = %q, want rockyads.com", got)
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
