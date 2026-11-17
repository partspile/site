package phoneformat

import "github.com/nyaruka/phonenumbers"

// Display formats an E.164 phone number for display (e.g. +12025550123 → (202) 555-0123).
// Falls back to the original string if parsing fails.
func Display(phoneE64 string) string {
	if phoneE64 == "" {
		return ""
	}
	num, err := phonenumbers.Parse(phoneE64, "")
	if err != nil {
		return phoneE64
	}
	return phonenumbers.Format(num, phonenumbers.NATIONAL)
}
