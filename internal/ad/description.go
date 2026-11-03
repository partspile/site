package ad

import (
	"strings"
	"unicode"
)

// SanitizeAdText normalizes common paste artifacts and removes
// invisible or control characters unsuitable for stored ad title/description.
func SanitizeAdText(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case '\n', '\r':
			b.WriteRune(r)
		case '\t':
			b.WriteByte(' ')
		default:
			if r == '\uFFFC' || r == '\uFFFD' {
				continue
			}
			if unicode.IsControl(r) || unicode.In(r, unicode.Cf) {
				continue
			}
			b.WriteRune(normalizeDescriptionRune(r))
		}
	}
	return b.String()
}

// TitleContainsEmoji reports whether s includes emoji or other pictographs
// unsuitable for ad titles (e.g. flag or warning icons).
func TitleContainsEmoji(s string) bool {
	for _, r := range s {
		if isEmojiRune(r) {
			return true
		}
	}
	return false
}

func isEmojiRune(r rune) bool {
	switch {
	case r == 0x200D, r == 0xFE0F:
		return true
	case r >= 0x2300 && r <= 0x23FF:
		return true
	case r >= 0x2600 && r <= 0x27BF:
		return true
	case r >= 0x2B05 && r <= 0x2B07:
		return true
	case r == 0x2B1B, r == 0x2B1C, r == 0x2B50, r == 0x2B55:
		return true
	case r >= 0x1F000 && r <= 0x1FAFF:
		return true
	case r >= 0x1F1E6 && r <= 0x1F1FF:
		return true
	default:
		return false
	}
}

func normalizeDescriptionRune(r rune) rune {
	switch r {
	case '\u2018', '\u2019', '\u201A', '\u201B':
		return '\''
	case '\u201C', '\u201D', '\u201E', '\u201F':
		return '"'
	case '\u2013', '\u2014':
		return '-'
	case '\u00A0', '\u202F', '\u2007', '\u2008', '\u2009', '\u200A':
		return ' '
	default:
		return r
	}
}
