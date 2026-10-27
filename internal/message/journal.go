package message

import (
	"time"

	"github.com/partspile/site/internal/journal"
)

const (
	JournalMessage        = journal.Message
	JournalRockThrown     = journal.RockThrown
	JournalRockUnthrown   = journal.RockUnthrown
	JournalAccountDeleted = journal.AccountDeleted
	JournalAdDeleted      = journal.AdDeleted
	JournalAdPaused       = journal.AdPaused
	JournalAdUnpaused     = journal.AdUnpaused
)

type JournalEntry = journal.Entry

func AppendJournalEntry(j, label, meta, body string, at time.Time,
	tz *time.Location) string {
	return journal.AppendEntry(j, label, meta, body, at, tz)
}

func AppendMessageEntry(j string, senderID int, body string, at time.Time,
	tz *time.Location) string {
	return journal.AppendMessage(j, senderID, body, at, tz)
}

func AppendRockEntry(j, kind string, userID int, reason string, at time.Time,
	tz *time.Location) string {
	return journal.AppendRock(j, kind, userID, reason, at, tz)
}

func AppendCloseEntry(j, kind string, userID int, at time.Time,
	tz *time.Location) string {
	return journal.AppendClose(j, kind, userID, at, tz)
}

func ParseJournal(j string) []JournalEntry {
	return journal.Parse(j)
}

func LastMessagePreview(j string) (content string, at time.Time, ok bool) {
	return journal.LastMessagePreview(j)
}

func FirstEntryAt(j string) (time.Time, bool) {
	return journal.FirstEntryAt(j)
}

func MessagesFromJournal(conversationID int, j string, tz *time.Location) []Message {
	entries := journal.Parse(j)
	var messages []Message
	id := 1
	for _, e := range entries {
		if e.Kind != journal.Message {
			continue
		}
		at := e.At
		if tz != nil {
			at = at.In(tz)
		}
		messages = append(messages, Message{
			ID:             id,
			ConversationID: conversationID,
			SenderID:       e.UserID,
			Content:        e.Body,
			CreatedAt:      at,
		})
		id++
	}
	return messages
}

// RockJournalEvent is a rock throw/unthrow timeline entry from the journal.
type RockJournalEvent struct {
	UserID    int
	Kind      string
	Reason    string
	CreatedAt time.Time
}

func RockEventsFromJournal(j string, tz *time.Location) []RockJournalEvent {
	entries := journal.Parse(j)
	var events []RockJournalEvent
	for _, e := range entries {
		var kind string
		switch e.Kind {
		case journal.RockThrown:
			kind = "thrown"
		case journal.RockUnthrown:
			kind = "unthrown"
		default:
			continue
		}
		at := e.At
		if tz != nil {
			at = at.In(tz)
		}
		events = append(events, RockJournalEvent{
			UserID:    e.UserID,
			Kind:      kind,
			Reason:    e.Body,
			CreatedAt: at,
		})
	}
	return events
}

// LatestThrownReason returns the reason on the most recent rock-thrown entry.
func LatestThrownReason(j string) string {
	entries := journal.Parse(j)
	for i := len(entries) - 1; i >= 0; i-- {
		if entries[i].Kind == journal.RockThrown {
			return entries[i].Body
		}
	}
	return ""
}

// CloseJournalEvent is an ad/account deletion timeline entry from the journal.
type CloseJournalEvent struct {
	UserID    int
	Kind      string
	CreatedAt time.Time
}

func CloseEventsFromJournal(j string, tz *time.Location) []CloseJournalEvent {
	entries := journal.Parse(j)
	var events []CloseJournalEvent
	for _, e := range entries {
		var kind string
		switch e.Kind {
		case journal.AccountDeleted:
			kind = "account"
		case journal.AdDeleted:
			kind = "ad"
		case journal.AdPaused:
			kind = "paused"
		case journal.AdUnpaused:
			kind = "unpaused"
		default:
			continue
		}
		at := e.At
		if tz != nil {
			at = at.In(tz)
		}
		events = append(events, CloseJournalEvent{
			UserID:    e.UserID,
			Kind:      kind,
			CreatedAt: at,
		})
	}
	return events
}
