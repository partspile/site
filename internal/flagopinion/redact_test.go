package flagopinion

import "testing"

func TestRedactTextPhone(t *testing.T) {
	in := "Call me at 310-555-0142 tomorrow"
	got := RedactText(in)
	if ContainsPII(got) {
		t.Fatalf("phone not redacted: %q", got)
	}
	if got == in {
		t.Fatal("expected redaction")
	}
}

func TestRedactTextEmail(t *testing.T) {
	in := "Reach me at seller@example.com please"
	got := RedactText(in)
	if ContainsPII(got) {
		t.Fatalf("email not redacted: %q", got)
	}
}

func TestRedactTextStreet(t *testing.T) {
	in := "Meet at 4821 Melrose Ave, Apt 3B"
	got := RedactText(in)
	if ContainsPII(got) {
		t.Fatalf("street not redacted: %q", got)
	}
}

func TestRedactNames(t *testing.T) {
	in := "john_doe said the price is firm"
	got := RedactNames(in, "john_doe")
	if got == in {
		t.Fatal("expected name redaction")
	}
}

func TestSanitizeOpinionFieldsRejectsPII(t *testing.T) {
	if SanitizeOpinionFields("Clean summary", "Call 555-123-4567") {
		t.Fatal("expected PII rejection")
	}
	if !SanitizeOpinionFields("Clean summary", "No contact info here") {
		t.Fatal("expected clean fields to pass")
	}
}
