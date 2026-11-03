package message

import "testing"

func TestCanViewConversationParticipantsOnly(t *testing.T) {
	thrower := 2
	conv := Conversation{
		OwnerID:      1,
		InquirerID:   2,
		FlagRaiserID: &thrower,
	}

	if !CanViewConversation(conv, 1) {
		t.Fatal("owner should view thread")
	}
	if !CanViewConversation(conv, 2) {
		t.Fatal("inquirer should view thread")
	}
	if CanViewConversation(conv, 99) {
		t.Fatal("bystander must not view thread even when flaged")
	}
	if CanViewConversation(conv, 0) {
		t.Fatal("anonymous must not view thread")
	}

	conv.FlagRaiserID = nil
	if CanViewConversation(conv, 99) {
		t.Fatal("bystander must not view private thread")
	}
}
