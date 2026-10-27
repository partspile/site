package user

import (
	"database/sql"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/partspile/site/internal/config"
)

func setShareSecret(t *testing.T) {
	t.Helper()
	prev := config.ShareSecret
	key, err := base64.StdEncoding.DecodeString(
		"AgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgI=")
	if err != nil {
		t.Fatal(err)
	}
	config.ShareSecret = key
	t.Cleanup(func() { config.ShareSecret = prev })
}

func TestShareTokenRoundTrip(t *testing.T) {
	setShareSecret(t)
	resetSchema(t)

	u, err := CreateUser("shareme", "+15559874001", "password1")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if _, err := GetIDByShareToken("not-a-token"); err != sql.ErrNoRows {
		t.Fatalf("invalid token: %v", err)
	}

	tok, err := ShareToken(u.ID)
	if err != nil {
		t.Fatalf("share: %v", err)
	}
	if !ValidShareToken(tok) {
		t.Fatalf("token %q not valid", tok)
	}
	again, err := ShareToken(u.ID)
	if err != nil || again != tok {
		t.Fatalf("ShareToken not stable: %q %v", again, err)
	}
	if strings.Contains(tok, ".") {
		t.Fatalf("token should be opaque, got %q", tok)
	}

	id, err := GetIDByShareToken(tok)
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if id != u.ID {
		t.Fatalf("id %d, want %d", id, u.ID)
	}

	tampered := tok[:len(tok)-1] + "x"
	if _, err := GetIDByShareToken(tampered); err != sql.ErrNoRows {
		t.Fatalf("tampered token: %v", err)
	}
	other, err := ShareToken(u.ID + 1)
	if err != nil {
		t.Fatalf("other token: %v", err)
	}
	if _, err := GetIDByShareToken(other); err != sql.ErrNoRows {
		t.Fatalf("other user token: %v", err)
	}
}

func TestShareTokenDeletedUser(t *testing.T) {
	setShareSecret(t)
	resetSchema(t)

	u, err := CreateUser("sharegone", "+15559874002", "password1")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	tok, err := ShareToken(u.ID)
	if err != nil {
		t.Fatalf("share: %v", err)
	}
	if err := DeleteUser(u.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := GetIDByShareToken(tok); err != sql.ErrNoRows {
		t.Fatalf("deleted user token: %v", err)
	}
}

func TestValidShareToken(t *testing.T) {
	setShareSecret(t)
	tok, err := ShareToken(1)
	if err != nil {
		t.Fatal(err)
	}
	if !ValidShareToken(tok) {
		t.Fatalf("generated token invalid: %q", tok)
	}
	for _, s := range []string{"", "abc", "1", "1.mac"} {
		if ValidShareToken(s) {
			t.Fatalf("accepted %q", s)
		}
	}
}
