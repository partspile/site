package turnstile

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/logger"
)

func TestMain(m *testing.M) {
	if err := logger.Init("error", "text", ""); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

func TestVerifySkippedWhenNotRequired(t *testing.T) {
	orig := config.AllowTestRegistration
	reflect.ValueOf(&config.AllowTestRegistration).Elem().SetBool(true)
	t.Cleanup(func() {
		reflect.ValueOf(&config.AllowTestRegistration).Elem().SetBool(orig)
	})
	if err := Verify("", ""); err != nil {
		t.Fatalf("expected skip: %v", err)
	}
}

func TestVerifySuccessAndFailure(t *testing.T) {
	origAllow := config.AllowTestRegistration
	origSecret := config.TurnstileSecretKey
	origURL := siteverifyURL
	reflect.ValueOf(&config.AllowTestRegistration).Elem().SetBool(false)
	reflect.ValueOf(&config.TurnstileSecretKey).Elem().SetString("test-secret")
	t.Cleanup(func() {
		reflect.ValueOf(&config.AllowTestRegistration).Elem().SetBool(origAllow)
		reflect.ValueOf(&config.TurnstileSecretKey).Elem().SetString(origSecret)
		siteverifyURL = origURL
	})

	success := true
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse form: %v", err)
		}
		if r.FormValue("secret") != "test-secret" {
			t.Errorf("secret = %q", r.FormValue("secret"))
		}
		_ = json.NewEncoder(w).Encode(siteverifyResponse{Success: success})
	}))
	t.Cleanup(srv.Close)
	siteverifyURL = srv.URL

	if err := Verify("tok", "1.2.3.4"); err != nil {
		t.Fatalf("success path: %v", err)
	}

	success = false
	if err := Verify("tok", ""); err == nil {
		t.Fatal("expected failure when success=false")
	}

	if err := Verify("", ""); err == nil {
		t.Fatal("expected failure for empty token")
	}
}

func TestVerifyAlwaysWhenTestRegistration(t *testing.T) {
	origAllow := config.AllowTestRegistration
	origSecret := config.TurnstileSecretKey
	origURL := siteverifyURL
	config.AllowTestRegistration = true
	config.TurnstileSecretKey = "test-secret"
	t.Cleanup(func() {
		config.AllowTestRegistration = origAllow
		config.TurnstileSecretKey = origSecret
		siteverifyURL = origURL
	})

	called := false
	srv := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			called = true
			if err := r.ParseForm(); err != nil {
				t.Errorf("parse form: %v", err)
			}
			if r.FormValue("response") != "probe-token" {
				t.Errorf("response = %q", r.FormValue("response"))
			}
			_ = json.NewEncoder(w).Encode(siteverifyResponse{Success: true})
		},
	))
	t.Cleanup(srv.Close)
	siteverifyURL = srv.URL

	if err := Verify("probe-token", ""); err != nil {
		t.Fatalf("Verify should skip: %v", err)
	}
	if called {
		t.Fatal("Verify called siteverify while test registration is on")
	}

	if err := VerifyAlways("probe-token", "127.0.0.1"); err != nil {
		t.Fatalf("VerifyAlways: %v", err)
	}
	if !called {
		t.Fatal("VerifyAlways did not call siteverify")
	}
}
