package handler

import (
	"reflect"
	"testing"

	"github.com/partspile/site/internal/config"
)

func TestValidatePhoneTestRegistration(t *testing.T) {
	orig := config.AllowTestRegistration
	t.Cleanup(func() {
		reflect.ValueOf(&config.AllowTestRegistration).Elem().SetBool(orig)
	})

	tests := []struct {
		name    string
		phone   string
		allow   bool
		wantErr bool
	}{
		{"agent phone allowed", "+15550101001", true, false},
		{"agent phone blocked", "+15550101001", false, true},
		{"seed phone allowed", "+15550100123", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reflect.ValueOf(&config.AllowTestRegistration).Elem().SetBool(tt.allow)
			got, err := validatePhone(tt.phone)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validatePhone(%q) err=%v wantErr=%v", tt.phone, err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.phone {
				t.Fatalf("validatePhone(%q) = %q", tt.phone, got)
			}
		})
	}
}
