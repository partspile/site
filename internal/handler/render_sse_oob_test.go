package handler_test

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/partspile/site/internal/ui"
)

func TestConversationMessagesAppendSwap(t *testing.T) {
	got := ui.ConversationMessagesAppendSwapForTest()
	want := "beforeend"
	if got != want {
		t.Fatalf("swap = %q, want %q", got, want)
	}
}

func TestRenderMessageAppendOOB(t *testing.T) {
	msgData := ui.MessageItemData{
		SenderID: 2, CurrentUserID: 4,
		Content:   "Hi this is Bob",
		CreatedAt: time.Now().UTC(),
	}
	var buf bytes.Buffer
	node := ui.MessageItem(msgData, ui.ConversationMessageAppendOOB(5))
	if err := node.Render(&buf); err != nil {
		t.Fatal(err)
	}
	html := buf.String()
	t.Logf("HTML:\n%s", html)
	if !strings.Contains(html, `hx-swap-oob="beforeend:#conversation-5-messages-list"`) {
		t.Fatalf("missing oob attr: %s", html)
	}
	if strings.Contains(html, "scroll:bottom") {
		t.Fatalf("scroll:bottom must not appear in oob attr: %s", html)
	}
	if strings.Contains(html, "\n") {
		t.Log("WARNING: HTML contains newlines - may break SSE")
	}
}

func TestRenderRockEventAppendOOB(t *testing.T) {
	event := ui.RockEventData{
		ConversationID: 5,
		ThrowerID:      4, CurrentUserID: 3,
		Kind:    ui.RockEventThrown,
		EventAt: time.Now().UTC(),
		OwnerID: 3, InquirerID: 4,
		OwnerName:          "sfeldma",
		InquirerName:       "bob",
		ShowAssessmentHint: true,
	}
	var buf bytes.Buffer
	node := ui.RockEventMessage(event, ui.ConversationMessageAppendOOB(5))
	if err := node.Render(&buf); err != nil {
		t.Fatal(err)
	}
	html := buf.String()
	if !strings.Contains(html, `hx-swap-oob="beforeend:#conversation-5-messages-list"`) {
		t.Fatalf("missing oob attr: %s", html)
	}
	if !strings.Contains(html, "Rock thrown at ad by bob") {
		t.Fatalf("missing rock event label: %s", html)
	}
	if !strings.Contains(html, "for dispute assessment") {
		t.Fatalf("missing assessment hint: %s", html)
	}
	if !strings.Contains(html, "rock-assessment-hint") {
		t.Fatalf("missing assessment hint class: %s", html)
	}

	event.ShowAssessmentHint = false
	buf.Reset()
	if err := ui.RockEventMessage(event).Render(&buf); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "for dispute assessment") {
		t.Fatalf("hint should be absent without ShowAssessmentHint: %s",
			buf.String())
	}
}

func TestConversationMessagesAreaStructure(t *testing.T) {
	var buf bytes.Buffer
	node := ui.ConversationMessagesArea(5, true, nil)
	if err := node.Render(&buf); err != nil {
		t.Fatal(err)
	}
	html := buf.String()
	t.Logf("HTML:\n%s", html)
	for _, want := range []string{
		`id="conversation-5-messages"`,
		`flex-col-reverse`,
		`id="conversation-5-messages-list"`,
		`conversation-5-empty-message`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q in:\n%s", want, html)
		}
	}
	if strings.Contains(html, "sentinel") {
		t.Fatalf("sentinel should be removed: %s", html)
	}
}

func TestSSEEventNames(t *testing.T) {
	if ui.SSEEventUnread != "unread" {
		t.Fatalf("unread = %q", ui.SSEEventUnread)
	}
	if ui.SSEEventConversationList != "conversation-list" {
		t.Fatalf("list = %q", ui.SSEEventConversationList)
	}
	if ui.SSEEventNotifications != "notifications" {
		t.Fatalf("notifications = %q", ui.SSEEventNotifications)
	}
	if got := ui.SSEEventConversation(5); got != "conversation-5" {
		t.Fatalf("conversation = %q", got)
	}
}

func TestConversationSSESink(t *testing.T) {
	var buf bytes.Buffer
	if err := ui.ConversationSSESink(5).Render(&buf); err != nil {
		t.Fatal(err)
	}
	html := buf.String()
	for _, want := range []string{
		`id="conversation-5-sse"`,
		`sse-swap="conversation-5"`,
		`hx-swap="none"`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q in:\n%s", want, html)
		}
	}
}

func TestUnreadAndListSSESinks(t *testing.T) {
	var buf bytes.Buffer
	if err := ui.UnreadSSESink().Render(&buf); err != nil {
		t.Fatal(err)
	}
	unread := buf.String()
	if !strings.Contains(unread, `sse-swap="unread"`) {
		t.Fatalf("unread sink: %s", unread)
	}

	buf.Reset()
	if err := ui.ConversationListSSESink().Render(&buf); err != nil {
		t.Fatal(err)
	}
	list := buf.String()
	if !strings.Contains(list, `sse-swap="conversation-list"`) {
		t.Fatalf("list sink: %s", list)
	}

	buf.Reset()
	if err := ui.NotificationsSSESink().Render(&buf); err != nil {
		t.Fatal(err)
	}
	notes := buf.String()
	if !strings.Contains(notes, `sse-swap="notifications"`) {
		t.Fatalf("notifications sink: %s", notes)
	}
}
