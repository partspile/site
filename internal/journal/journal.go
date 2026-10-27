package journal

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/partspile/site/internal/entrylog"
)

const (
	Message        = "message"
	RockThrown     = "rock thrown"
	RockUnthrown   = "rock unthrown"
	AccountDeleted = "account deleted"
	AdDeleted      = "ad deleted"
	AdPaused       = "ad paused"
	AdUnpaused     = "ad unpaused"
)

type Entry struct {
	Kind   string
	At     time.Time
	UserID int
	Body   string
}

func AppendEntry(j, label, meta, body string, at time.Time,
	tz *time.Location) string {
	return entrylog.Append(j, label, meta, body, at, tz)
}

func AppendMessage(j string, senderID int, body string, at time.Time,
	tz *time.Location) string {
	meta := fmt.Sprintf("sender:%d", senderID)
	return entrylog.Append(j, Message, meta, body, at, tz)
}

func AppendRock(j, kind string, userID int, reason string, at time.Time,
	tz *time.Location) string {
	meta := fmt.Sprintf("user:%d", userID)
	return entrylog.Append(j, kind, meta, reason, at, tz)
}

func AppendClose(j, kind string, userID int, at time.Time,
	tz *time.Location) string {
	meta := fmt.Sprintf("user:%d", userID)
	return entrylog.Append(j, kind, meta, "", at, tz)
}

func Parse(j string) []Entry {
	blocks := entrylog.Parse(j)
	var entries []Entry
	for _, b := range blocks {
		switch b.Label {
		case Message, RockThrown, RockUnthrown, AccountDeleted, AdDeleted,
			AdPaused, AdUnpaused:
			entries = append(entries, Entry{
				Kind:   b.Label,
				At:     b.At,
				UserID: parseUserMeta(b.Meta),
				Body:   b.Body,
			})
		}
	}
	return entries
}

func parseUserMeta(meta string) int {
	for _, field := range strings.Fields(meta) {
		key, val, ok := strings.Cut(field, ":")
		if !ok {
			continue
		}
		if key == "sender" || key == "user" {
			id, err := strconv.Atoi(val)
			if err == nil {
				return id
			}
		}
	}
	return 0
}

func LastMessagePreview(j string) (content string, at time.Time, ok bool) {
	entries := Parse(j)
	for i := len(entries) - 1; i >= 0; i-- {
		if entries[i].Kind == Message {
			return entries[i].Body, entries[i].At, true
		}
	}
	return "", time.Time{}, false
}

func FirstEntryAt(j string) (time.Time, bool) {
	entries := Parse(j)
	if len(entries) == 0 {
		return time.Time{}, false
	}
	return entries[0].At, true
}
