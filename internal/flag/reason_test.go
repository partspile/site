package flag

import "testing"

func TestValidReason(t *testing.T) {
	if !ValidReason(ReasonPolicy) || !ValidReason(ReasonConduct) ||
		!ValidReason(ReasonDeal) {
		t.Fatal("expected canned reasons valid")
	}
	if ValidReason("") || ValidReason("other") {
		t.Fatal("expected invalid reasons rejected")
	}
	if ReasonLabel(ReasonPolicy) == "" {
		t.Fatal("expected policy label")
	}
}

func TestReasonsForTarget(t *testing.T) {
	ad := ReasonsForTarget(true)
	user := ReasonsForTarget(false)
	if len(ad) != 3 || len(user) != 3 {
		t.Fatalf("want 3 reasons each, got ad=%d user=%d", len(ad), len(user))
	}
	if ad[0].Label == user[0].Label {
		t.Fatal("expected different policy labels for ad vs user")
	}
	if ReasonLabelForTarget(ReasonPolicy, false) == "" {
		t.Fatal("expected user policy label")
	}
}
