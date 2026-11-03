package flagopinion

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	MinAssessment = 1
	MaxAssessment = 10
)

type rawOpinion struct {
	Summary          string          `json:"summary"`
	Assessment       json.RawMessage `json:"assessment"`
	AssessmentDetail string          `json:"assessment_detail"`
	Resolution       string          `json:"resolution"`
	Reasoning        string          `json:"reasoning"`
}

func parseAssessmentScore(raw json.RawMessage) (int, error) {
	if len(raw) == 0 {
		return 0, fmt.Errorf("missing assessment")
	}

	var n int
	if err := json.Unmarshal(raw, &n); err == nil {
		if n < MinAssessment || n > MaxAssessment {
			return 0, fmt.Errorf("assessment out of range: %d", n)
		}
		return n, nil
	}

	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return 0, fmt.Errorf("invalid assessment")
	}
	s = strings.TrimSpace(s)
	switch strings.ToLower(s) {
	case "inquirer":
		return 2, nil
	case "owner":
		return 8, nil
	case "neither":
		return 5, nil
	case "insufficient":
		return 5, nil
	}
	return 0, fmt.Errorf("invalid assessment: %q", s)
}

func parseOpinionResponse(resp string) (Opinion, error) {
	resp = strings.TrimSpace(resp)
	resp = trimCodeFence(resp)

	var raw rawOpinion
	if err := json.Unmarshal([]byte(resp), &raw); err != nil {
		return Opinion{}, fmt.Errorf("parse opinion json: %w", err)
	}

	score, err := parseAssessmentScore(raw.Assessment)
	if err != nil {
		return Opinion{}, err
	}

	fields := []string{
		raw.Summary, raw.AssessmentDetail, raw.Resolution, raw.Reasoning,
	}
	for i, f := range fields {
		fields[i] = strings.TrimSpace(f)
		if fields[i] == "" {
			return Opinion{}, fmt.Errorf("empty opinion field")
		}
	}

	if !SanitizeOpinionFields(fields...) {
		return Opinion{}, fmt.Errorf("opinion contains PII")
	}

	return Opinion{
		Summary:          fields[0],
		Assessment:       score,
		AssessmentDetail: fields[1],
		Resolution:       fields[2],
		Reasoning:        fields[3],
	}, nil
}

func trimCodeFence(s string) string {
	if !strings.HasPrefix(s, "```") {
		return s
	}
	lines := strings.Split(s, "\n")
	if len(lines) < 2 {
		return s
	}
	end := len(lines) - 1
	for end > 0 && strings.TrimSpace(lines[end]) == "" {
		end--
	}
	if strings.HasPrefix(strings.TrimSpace(lines[end]), "```") {
		return strings.Join(lines[1:end], "\n")
	}
	return s
}
