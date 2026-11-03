package flagopinion

import "testing"

func TestParseOpinionResponse(t *testing.T) {
	resp := `{"summary":"Both parties disagree on price.",
"assessment":4,
"assessment_detail":"Negotiation is within normal bounds.",
"resolution":"Owner should state a firm floor price.",
"reasoning":"No harassment occurred."}`
	op, err := parseOpinionResponse(resp)
	if err != nil {
		t.Fatal(err)
	}
	if op.Assessment != 4 {
		t.Fatalf("assessment = %d", op.Assessment)
	}
	if op.Summary == "" || op.Resolution == "" {
		t.Fatal("expected populated fields")
	}
}

func TestParseOpinionResponseStripsFence(t *testing.T) {
	resp := "```json\n" +
		`{"summary":"Summary text here for the dispute.",
"assessment":8,
"assessment_detail":"Owner disclosed terms clearly.",
"resolution":"Inquirer should confirm before traveling.",
"reasoning":"Ad history supports the owner."}` +
		"\n```"
	op, err := parseOpinionResponse(resp)
	if err != nil {
		t.Fatal(err)
	}
	if op.Assessment != 8 {
		t.Fatalf("assessment = %d", op.Assessment)
	}
}

func TestParseOpinionResponseRejectsPII(t *testing.T) {
	resp := `{"summary":"Meet at 123 Main Street",
"assessment":3,
"assessment_detail":"Detail without PII here for test.",
"resolution":"Parties should message again.",
"reasoning":"Reasoning without PII here for test."}`
	_, err := parseOpinionResponse(resp)
	if err == nil {
		t.Fatal("expected PII rejection")
	}
}

func TestParseOpinionResponseInvalidAssessment(t *testing.T) {
	resp := `{"summary":"Summary text here for the dispute.",
"assessment":11,
"assessment_detail":"Some detail text here.",
"resolution":"Do something useful.",
"reasoning":"Because reasons."}`
	_, err := parseOpinionResponse(resp)
	if err == nil {
		t.Fatal("expected invalid assessment error")
	}
}

func TestParseAssessmentScoreLegacyStrings(t *testing.T) {
	tests := []struct {
		raw  string
		want int
	}{
		{`"inquirer"`, 2},
		{`"owner"`, 8},
		{`"neither"`, 5},
		{`5`, 5},
	}
	for _, tt := range tests {
		got, err := parseAssessmentScore([]byte(tt.raw))
		if err != nil {
			t.Fatalf("parseAssessmentScore(%s): %v", tt.raw, err)
		}
		if got != tt.want {
			t.Fatalf("parseAssessmentScore(%s) = %d, want %d",
				tt.raw, got, tt.want)
		}
	}
}
