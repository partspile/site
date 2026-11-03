package flagopinion

import (
	"regexp"
	"strings"
)

var (
	phonePattern = regexp.MustCompile(
		`(?:\+?1[-.\s]?)?(?:\(?\d{3}\)?[-.\s]?)\d{3}[-.\s]?\d{4}`,
	)
	emailPattern = regexp.MustCompile(
		`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`,
	)
	streetPattern = regexp.MustCompile(
		`(?i)\b\d{1,5}\s+[\w.'\-]+\s+` +
			`(?:St|Street|Ave|Avenue|Rd|Road|Dr|Drive|Ln|Lane|` +
			`Blvd|Boulevard|Way|Ct|Court|Pl|Place|Ter|Terrace)\b` +
			`(?:\s*(?:,?\s*(?:Apt|Unit|Suite|Ste|#)\s*[\w\d\-]+)?)?`,
	)
)

const redactedPlaceholder = "[redacted]"

// RedactText replaces likely PII with a placeholder.
func RedactText(s string) string {
	s = phonePattern.ReplaceAllString(s, redactedPlaceholder)
	s = emailPattern.ReplaceAllString(s, redactedPlaceholder)
	s = streetPattern.ReplaceAllString(s, redactedPlaceholder)
	return s
}

// ContainsPII reports whether s still has likely PII patterns.
func ContainsPII(s string) bool {
	if phonePattern.MatchString(s) {
		return true
	}
	if emailPattern.MatchString(s) {
		return true
	}
	if streetPattern.MatchString(s) {
		return true
	}
	return false
}

// SanitizeOpinionFields returns false if any field still contains PII.
func SanitizeOpinionFields(fields ...string) bool {
	for _, f := range fields {
		if ContainsPII(f) {
			return false
		}
	}
	return true
}

// RedactNames replaces known participant names in text.
func RedactNames(s string, names ...string) string {
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		s = strings.ReplaceAll(s, name, redactedPlaceholder)
	}
	return s
}
