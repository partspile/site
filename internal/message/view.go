package message

import (
	"errors"
	"fmt"
	"time"

	"github.com/partspile/site/internal/ad"
	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/rock"
	"github.com/partspile/site/internal/user"
)

var (
	ErrModalAdNotFound = errors.New("modal ad not found")
	ErrModalAccess     = errors.New("modal conversation access denied")
)

// ConversationModalView holds data needed to render a conversation modal.
type ConversationModalView struct {
	Conversation      Conversation
	AdTitle           string
	OwnerName         string
	InquirerName      string
	OwnerDeleted      bool
	InquirerDeleted   bool
	Messages          []Message
	RockEvents        []RockJournalEvent
	CloseEvents       []CloseJournalEvent
	InquirerRockCount int
	OwnerRockCount    int
	CanPost           bool
	HasThrownRock     bool
	CanThrowRock      bool
}

func CanViewConversation(conv Conversation, userID int) bool {
	return userID > 0 && IsParticipant(conv, userID)
}

func IsParticipant(conv Conversation, userID int) bool {
	return conv.OwnerID == userID || conv.InquirerID == userID
}

// OpenConversation loads a conversation the user may view (participants only)
// and marks it read. The second return value is true when marked.
func OpenConversation(conversationID, userID int) (Conversation, bool, error) {
	conv, err := GetConversationByID(conversationID)
	if err != nil {
		return Conversation{}, false, err
	}
	if !CanViewConversation(conv, userID) {
		return Conversation{}, false, ErrModalAccess
	}
	markedRead := false
	if err := MarkConversationAsRead(conversationID, userID); err == nil {
		markedRead = true
	}
	return conv, markedRead, nil
}

func OwnerAndInquirerNames(conv Conversation) (ownerName, inquirerName string,
	err error) {
	ownerName, _ = displayName(conv.OwnerID)
	inquirerName, _ = displayName(conv.InquirerID)
	return ownerName, inquirerName, nil
}

func displayName(userID int) (name string, deleted bool) {
	u, err := user.GetByIDIncludingDeleted(userID)
	if err != nil {
		return "Unknown", true
	}
	return u.Name, u.DeletedAt != nil
}

func DisplayName(userID int) (name string, deleted bool) {
	return displayName(userID)
}

func OtherUserName(conv Conversation, currentUserID int) (string, error) {
	var otherUserID int
	if conv.OwnerID == currentUserID {
		otherUserID = conv.InquirerID
	} else {
		otherUserID = conv.OwnerID
	}
	name, _ := displayName(otherUserID)
	return name, nil
}

func BuildConversationModal(conv Conversation, currentUserID int,
	tz *time.Location) (ConversationModalView, error) {
	a, err := ad.GetAd(currentUserID, conv.AdID, tz)
	if err != nil {
		return ConversationModalView{}, fmt.Errorf("%w: %v", ErrModalAdNotFound, err)
	}

	ownerName, ownerDeleted := displayName(conv.OwnerID)
	inquirerName, inquirerDeleted := displayName(conv.InquirerID)

	view := ConversationModalView{
		Conversation:      conv,
		AdTitle:           a.Title,
		OwnerName:         ownerName,
		InquirerName:      inquirerName,
		OwnerDeleted:      ownerDeleted,
		InquirerDeleted:   inquirerDeleted,
		InquirerRockCount: rockCountForUser(conv.InquirerID),
		OwnerRockCount:    rockCountForUser(conv.OwnerID),
		Messages:          []Message{},
	}

	messagingOK := MessagingAllowed(conv)
	if conv.ID == 0 {
		view.CanPost = messagingOK && a.IsActive()
		view.CanThrowRock = view.CanPost && userUnderRockLimit(currentUserID)
		return view, nil
	}

	view.Messages = MessagesFromJournal(conv.ID, conv.Journal, tz)
	view.RockEvents = RockEventsFromJournal(conv.Journal, tz)
	view.CloseEvents = CloseEventsFromJournal(conv.Journal, tz)

	canPost, err := CanUserPost(conv.ID, currentUserID)
	if err != nil {
		return ConversationModalView{}, fmt.Errorf("failed to check permissions: %w", err)
	}
	view.CanPost = canPost && messagingOK

	hasThrown, canThrow := rockThrowPermissions(conv.ID, currentUserID, view.CanPost)
	view.HasThrownRock = hasThrown
	view.CanThrowRock = canThrow

	return view, nil
}

func rockCountForUser(userID int) int {
	count, err := rock.GetRockCountForUser(userID)
	if err != nil {
		return 0
	}
	return count
}

func userUnderRockLimit(userID int) bool {
	userRockCount, err := rock.GetUserRockCount(userID)
	if err != nil || userRockCount >= config.MaxOutstandingRocks {
		return false
	}
	return true
}

func rockThrowPermissions(conversationID, userID int,
	canPost bool) (hasThrown, canThrow bool) {
	hasThrown, err := rock.HasUserThrownRock(userID, conversationID)
	if err != nil {
		hasThrown = false
	}
	if !canPost {
		return hasThrown, false
	}
	rockCount, err := rock.GetRockCountForConversation(conversationID)
	if err != nil || rockCount > 0 {
		return hasThrown, false
	}
	if !userUnderRockLimit(userID) {
		return hasThrown, false
	}
	return hasThrown, true
}
