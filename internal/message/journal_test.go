package message

import (
	"testing"
	"time"

	"github.com/partspile/site/internal/journal"
)

func TestMessagesAndRockEventsFromJournal(t *testing.T) {
	loc := time.UTC
	t1 := time.Date(2026, 7, 10, 22, 33, 0, 0, loc)
	t2 := time.Date(2026, 7, 10, 22, 35, 0, 0, loc)
	t3 := time.Date(2026, 7, 10, 22, 40, 0, 0, loc)

	j := journal.AppendMessage("", 12, "hello", t1, loc)
	j = journal.AppendRock(j, journal.RockThrown, 12, "conduct", t2, loc)
	j = journal.AppendMessage(j, 34, "reply", t3, loc)

	msgs := MessagesFromJournal(7, j, loc)
	if len(msgs) != 2 {
		t.Fatalf("messages = %d, want 2", len(msgs))
	}
	if msgs[0].ConversationID != 7 || msgs[0].SenderID != 12 ||
		msgs[0].Content != "hello" {
		t.Errorf("first message: %+v", msgs[0])
	}
	if msgs[1].SenderID != 34 || msgs[1].Content != "reply" {
		t.Errorf("second message: %+v", msgs[1])
	}

	events := RockEventsFromJournal(j, loc)
	if len(events) != 1 {
		t.Fatalf("events = %d, want 1", len(events))
	}
	if events[0].Kind != "thrown" || events[0].UserID != 12 {
		t.Errorf("event: %+v", events[0])
	}
	if events[0].Reason != "conduct" {
		t.Errorf("reason = %q", events[0].Reason)
	}
	if LatestThrownReason(j) != "conduct" {
		t.Errorf("LatestThrownReason = %q", LatestThrownReason(j))
	}
}

func TestCloseEventsFromJournal(t *testing.T) {
	loc := time.UTC
	t1 := time.Date(2026, 7, 10, 22, 33, 0, 0, loc)
	t2 := time.Date(2026, 7, 10, 22, 40, 0, 0, loc)

	j := journal.AppendMessage("", 12, "hello", t1, loc)
	j = journal.AppendClose(j, journal.AccountDeleted, 34, t2, loc)

	closes := CloseEventsFromJournal(j, loc)
	if len(closes) != 1 {
		t.Fatalf("closes = %d, want 1", len(closes))
	}
	if closes[0].Kind != "account" || closes[0].UserID != 34 {
		t.Errorf("close: %+v", closes[0])
	}
	if CloseEventText("account", "Bob") !=
		"User Bob deleted their account and all their ads, so this conversation is done" {
		t.Errorf("unexpected account text")
	}
	if CloseEventText("ad", "Bob") !=
		"Ad owner Bob deleted this ad, so this conversation is done" {
		t.Errorf("unexpected ad text")
	}
	if CloseEventText("paused", "Bob") !=
		"Ad owner Bob paused this ad, so this conversation is suspended" {
		t.Errorf("unexpected paused text")
	}
	if CloseEventText("unpaused", "Bob") !=
		"Ad owner Bob unpaused this ad, so this conversation can continue" {
		t.Errorf("unexpected unpaused text")
	}
}
