package message

import (
	"fmt"
	"time"

	"github.com/partspile/site/internal/db"
	"github.com/partspile/site/internal/journal"
)

func CloseConversationsForDeletedAd(adID, actorID int) ([]Conversation, error) {
	return appendCloseToConversations(
		`SELECT `+conversationColumns+` FROM conversations c WHERE c.ad_id = $1`,
		[]any{adID}, journal.AdDeleted, actorID,
	)
}

func CloseConversationsForDeletedAccount(userID int) ([]Conversation, error) {
	return appendCloseToConversations(
		`SELECT `+conversationColumns+`
		 FROM conversations c
		 WHERE c.owner_id = $1 OR c.inquirer_id = $1`,
		[]any{userID}, journal.AccountDeleted, userID,
	)
}

func SuspendConversationsForPausedAd(adID, actorID int) ([]Conversation, error) {
	return appendCloseToConversations(
		`SELECT `+conversationColumns+` FROM conversations c WHERE c.ad_id = $1`,
		[]any{adID}, journal.AdPaused, actorID,
	)
}

func ResumeConversationsForUnpausedAd(adID, actorID int) ([]Conversation, error) {
	return appendCloseToConversations(
		`SELECT `+conversationColumns+` FROM conversations c WHERE c.ad_id = $1`,
		[]any{adID}, journal.AdUnpaused, actorID,
	)
}

func appendCloseToConversations(query string, args []any, kind string,
	actorID int) ([]Conversation, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list conversations for close: %w", err)
	}
	defer rows.Close()

	var convs []Conversation
	for rows.Next() {
		conv, err := scanConversationRow(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan conversation for close: %w", err)
		}
		convs = append(convs, conv)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	updated := make([]Conversation, 0, len(convs))
	for _, conv := range convs {
		newJournal := AppendCloseEntry(conv.Journal, kind, actorID, now, time.UTC)
		sealed, err := sealJournal(conv.ID, newJournal)
		if err != nil {
			return nil, fmt.Errorf("seal journal: %w", err)
		}
		_, err = db.Exec(`
			UPDATE conversations
			SET journal = $1, updated_at = $2
			WHERE id = $3
		`, sealed, now, conv.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to append close entry: %w", err)
		}
		conv.Journal = newJournal
		conv.UpdatedAt = now
		updated = append(updated, conv)
	}
	return updated, nil
}

func CloseEventText(kind, userName string) string {
	switch kind {
	case "account":
		return fmt.Sprintf(
			"User %s deleted their account and all their ads, so this conversation is done",
			userName,
		)
	case "ad":
		return fmt.Sprintf(
			"Ad owner %s deleted this ad, so this conversation is done",
			userName,
		)
	case "paused":
		return fmt.Sprintf(
			"Ad owner %s paused this ad, so this conversation is suspended",
			userName,
		)
	case "unpaused":
		return fmt.Sprintf(
			"Ad owner %s unpaused this ad, so this conversation can continue",
			userName,
		)
	default:
		return fmt.Sprintf("User %s closed this conversation", userName)
	}
}

func OtherParticipantID(conv Conversation, userID int) int {
	if conv.OwnerID == userID {
		return conv.InquirerID
	}
	return conv.OwnerID
}
