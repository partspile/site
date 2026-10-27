package message

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/partspile/site/internal/db"
	"github.com/partspile/site/internal/entrylog"
	"github.com/partspile/site/internal/rock"
	"github.com/partspile/site/internal/user"
)

type Conversation struct {
	ID                int        `db:"id"`
	AdID              int        `db:"ad_id"`
	OwnerID           int        `db:"owner_id"`
	InquirerID        int        `db:"inquirer_id"`
	CreatedAt         time.Time  `db:"created_at"`
	UpdatedAt         time.Time  `db:"updated_at"`
	OwnerHasUnread    bool       `db:"owner_has_unread"`
	InquirerHasUnread bool       `db:"inquirer_has_unread"`
	RockThrowerID     *int       `db:"rock_thrower_id"`
	RockThrownAt      *time.Time `db:"rock_thrown_at"`
	Journal           string     `db:"journal"`
}

type Message struct {
	ID             int       `db:"id"`
	ConversationID int       `db:"conversation_id"`
	SenderID       int       `db:"sender_id"`
	Content        string    `db:"content"`
	CreatedAt      time.Time `db:"created_at"`
}

type ConversationWithAd struct {
	Conversation
	AdTitle string `db:"ad_title"`
}

type ConversationWithLastMessage struct {
	ConversationWithAd
	LastMessageContent string     `db:"last_message_content"`
	LastMessageAt      *time.Time `db:"last_message_at"`
	OtherUserID        int        `db:"other_user_id"`
	HasUnread          bool       `db:"has_unread"`
	RockCount          int        `db:"rock_count"`
	OtherUserName      string
	OtherUserRockCount int
	OtherUserDeleted   bool
}

var ErrConversationNotFound = errors.New("conversation not found")
var ErrNotParticipant = errors.New("user is not a participant in this conversation")
var ErrMessagingClosed = errors.New("messaging is closed for this conversation")

func applyJournalDerivedTimes(conv *Conversation) {
	if at, ok := FirstEntryAt(conv.Journal); ok {
		conv.CreatedAt = at
		return
	}
	if conv.RockThrownAt != nil {
		conv.CreatedAt = *conv.RockThrownAt
		return
	}
	conv.CreatedAt = conv.UpdatedAt
}

func scanConversationRow(scanner interface {
	Scan(dest ...any) error
}) (Conversation, error) {
	var conv Conversation
	var rockThrowerID sql.NullInt64
	var rockThrownAt sql.NullTime
	err := scanner.Scan(
		&conv.ID,
		&conv.AdID,
		&conv.OwnerID,
		&conv.InquirerID,
		&conv.OwnerHasUnread,
		&conv.InquirerHasUnread,
		&rockThrowerID,
		&rockThrownAt,
		&conv.Journal,
		&conv.UpdatedAt,
	)
	if err != nil {
		return Conversation{}, err
	}
	if rockThrowerID.Valid {
		id := int(rockThrowerID.Int64)
		conv.RockThrowerID = &id
	}
	if rockThrownAt.Valid {
		conv.RockThrownAt = &rockThrownAt.Time
	}
	plain, err := openJournal(conv.ID, conv.Journal)
	if err != nil {
		return Conversation{}, fmt.Errorf("open journal: %w", err)
	}
	conv.Journal = plain
	applyJournalDerivedTimes(&conv)
	return conv, nil
}

const conversationColumns = `
	c.id, c.ad_id, c.owner_id, c.inquirer_id,
	c.owner_has_unread, c.inquirer_has_unread, c.rock_thrower_id, c.rock_thrown_at,
	c.journal, c.updated_at`

// GetConversationByAdAndInquirer gets an existing conversation by ad ID and inquirer ID
// Returns ErrConversationNotFound if the conversation does not exist
func GetConversationByAdAndInquirer(adID, ownerID,
	inquirerID int) (Conversation, error) {
	if ownerID == inquirerID {
		return Conversation{}, fmt.Errorf("owner and inquirer cannot be the same")
	}

	query := `
		SELECT ` + conversationColumns + `
		FROM conversations c
		WHERE c.ad_id = $1 AND c.inquirer_id = $2
	`
	conv, err := scanConversationRow(db.QueryRow(query, adID, inquirerID))
	if err != nil {
		if err == sql.ErrNoRows {
			return Conversation{}, ErrConversationNotFound
		}
		return Conversation{}, fmt.Errorf("failed to query conversation: %w", err)
	}
	// Verify owner_id matches (in case ad ownership changed)
	if conv.OwnerID != ownerID {
		_, updateErr := db.Exec(`
			UPDATE conversations
			SET owner_id = $1
			WHERE id = $2
		`, ownerID, conv.ID)
		if updateErr != nil {
			return Conversation{}, fmt.Errorf("failed to update conversation owner: %w", updateErr)
		}
		conv.OwnerID = ownerID
	}
	return conv, nil
}

// CreateConversation creates a new conversation
func CreateConversation(adID, ownerID, inquirerID int) (Conversation, error) {
	if ownerID == inquirerID {
		return Conversation{}, fmt.Errorf("owner and inquirer cannot be the same")
	}

	var id int
	var updatedAt time.Time
	err := db.QueryRow(`
		INSERT INTO conversations (ad_id, owner_id, inquirer_id, owner_has_unread, inquirer_has_unread, rock_thrower_id, journal, updated_at)
		VALUES ($1, $2, $3, 0, 0, NULL, '', CURRENT_TIMESTAMP)
		RETURNING id, updated_at
	`, adID, ownerID, inquirerID).Scan(&id, &updatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return GetConversationByAdAndInquirer(adID, ownerID, inquirerID)
		}
		return Conversation{}, fmt.Errorf("failed to create conversation: %w", err)
	}

	return Conversation{
		ID:         id,
		AdID:       adID,
		OwnerID:    ownerID,
		InquirerID: inquirerID,
		UpdatedAt:  updatedAt,
		CreatedAt:  updatedAt,
		Journal:    "",
	}, nil
}

func GetConversation(conversationID, userID int) (Conversation, error) {
	query := `
		SELECT ` + conversationColumns + `
		FROM conversations c
		WHERE c.id = $1 AND (c.owner_id = $2 OR c.inquirer_id = $3)
	`
	conv, err := scanConversationRow(db.QueryRow(query, conversationID, userID, userID))
	if err != nil {
		if err == sql.ErrNoRows {
			return Conversation{}, ErrConversationNotFound
		}
		return Conversation{}, fmt.Errorf("failed to get conversation: %w", err)
	}
	return conv, nil
}

// GetConversationByID gets a conversation by ID without user check (for public access)
func GetConversationByID(conversationID int) (Conversation, error) {
	query := `
		SELECT ` + conversationColumns + `
		FROM conversations c
		WHERE c.id = $1
	`
	conv, err := scanConversationRow(db.QueryRow(query, conversationID))
	if err != nil {
		if err == sql.ErrNoRows {
			return Conversation{}, ErrConversationNotFound
		}
		return Conversation{}, fmt.Errorf("failed to get conversation: %w", err)
	}
	return conv, nil
}

// HasRock reports whether a conversation currently has a rock thrown
// (assessment may be public; the message thread is not).
func IsPublic(conversationID int) (bool, error) {
	var rockThrowerID sql.NullInt64
	err := db.QueryRow(`
		SELECT rock_thrower_id
		FROM conversations
		WHERE id = $1
	`, conversationID).Scan(&rockThrowerID)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, ErrConversationNotFound
		}
		return false, fmt.Errorf("failed to check if conversation is public: %w", err)
	}
	return rockThrowerID.Valid, nil
}

// CanUserPost checks if a user can post to a conversation (must be participant)
func CanUserPost(conversationID, userID int) (bool, error) {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*)
		FROM conversations
		WHERE id = $1 AND (owner_id = $2 OR inquirer_id = $3)
	`, conversationID, userID, userID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check if user can post: %w", err)
	}
	return count > 0, nil
}

// MessagingAllowed reports whether new messages/rocks are allowed on this thread.
func MessagingAllowed(conv Conversation) bool {
	if !user.Exists(conv.OwnerID) || !user.Exists(conv.InquirerID) {
		return false
	}
	var live bool
	err := db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM ads
			WHERE id = $1 AND inactive_at IS NULL AND deleted_at IS NULL
		)
	`, conv.AdID).Scan(&live)
	return err == nil && live
}

// GetPublicConversations returns public conversations for an ad
func GetPublicConversations(adID int) ([]Conversation, error) {
	query := `
		SELECT ` + conversationColumns + `
		FROM conversations c
		WHERE c.ad_id = $1 AND c.rock_thrower_id IS NOT NULL
		ORDER BY c.updated_at DESC
	`
	rows, err := db.Query(query, adID)
	if err != nil {
		return nil, fmt.Errorf("failed to get public conversations: %w", err)
	}
	defer rows.Close()

	var conversations []Conversation
	for rows.Next() {
		conv, err := scanConversationRow(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan conversation: %w", err)
		}
		conversations = append(conversations, conv)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate conversations: %w", err)
	}

	return conversations, nil
}

func GetUserConversations(userID int,
	tz *time.Location) ([]ConversationWithLastMessage, error) {
	query := `
		SELECT
			c.id,
			c.ad_id,
			c.owner_id,
			c.inquirer_id,
			c.owner_has_unread,
			c.inquirer_has_unread,
			c.rock_thrower_id,
			c.rock_thrown_at,
			c.journal,
			c.updated_at,
			a.title AS ad_title,
			COALESCE((
				SELECT COUNT(*)
				FROM conversations c2
				WHERE c2.ad_id = c.ad_id
					AND c2.rock_thrower_id IS NOT NULL
					AND c2.rock_thrower_id = c2.inquirer_id
			), 0) AS rock_count,
			CASE
				WHEN c.owner_id = $1 THEN c.inquirer_id
				ELSE c.owner_id
			END AS other_user_id,
			CASE
				WHEN c.owner_id = $2 THEN c.owner_has_unread
				ELSE c.inquirer_has_unread
			END AS has_unread
		FROM conversations c
		JOIN ads a ON c.ad_id = a.id
		WHERE (c.owner_id = $3 OR c.inquirer_id = $4)
		ORDER BY c.updated_at DESC
	`
	rows, err := db.Query(query, userID, userID, userID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user conversations: %w", err)
	}
	defer rows.Close()

	var conversations []ConversationWithLastMessage
	for rows.Next() {
		var conv ConversationWithLastMessage
		var rockThrowerID sql.NullInt64
		var rockThrownAt sql.NullTime

		err := rows.Scan(
			&conv.ID,
			&conv.AdID,
			&conv.OwnerID,
			&conv.InquirerID,
			&conv.OwnerHasUnread,
			&conv.InquirerHasUnread,
			&rockThrowerID,
			&rockThrownAt,
			&conv.Journal,
			&conv.UpdatedAt,
			&conv.AdTitle,
			&conv.RockCount,
			&conv.OtherUserID,
			&conv.HasUnread,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan conversation: %w", err)
		}

		if rockThrowerID.Valid {
			id := int(rockThrowerID.Int64)
			conv.RockThrowerID = &id
		}
		if rockThrownAt.Valid {
			conv.RockThrownAt = &rockThrownAt.Time
		}
		plain, err := openJournal(conv.ID, conv.Journal)
		if err != nil {
			return nil, fmt.Errorf("open journal: %w", err)
		}
		conv.Journal = plain
		applyJournalDerivedTimes(&conv.Conversation)

		if content, at, ok := LastMessagePreview(conv.Journal); ok {
			conv.LastMessageContent = content
			lastAt := at
			conv.LastMessageAt = &lastAt
		}

		conversations = append(conversations, conv)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate conversations: %w", err)
	}

	for i := range conversations {
		conversations[i].CreatedAt = conversations[i].CreatedAt.In(tz)
		conversations[i].UpdatedAt = conversations[i].UpdatedAt.In(tz)
		if conversations[i].LastMessageAt != nil {
			converted := (*conversations[i].LastMessageAt).In(tz)
			conversations[i].LastMessageAt = &converted
		}
	}

	enrichConversationListItems(conversations)

	return conversations, nil
}

func enrichConversationListItems(conversations []ConversationWithLastMessage) {
	if len(conversations) == 0 {
		return
	}

	otherIDs := make(map[int]struct{})
	for _, conv := range conversations {
		otherIDs[conv.OtherUserID] = struct{}{}
	}

	names := make(map[int]string, len(otherIDs))
	deleted := make(map[int]bool, len(otherIDs))
	rockCounts := make(map[int]int, len(otherIDs))
	for id := range otherIDs {
		if u, err := user.GetByIDIncludingDeleted(id); err == nil {
			names[id] = u.Name
			deleted[id] = u.DeletedAt != nil
		} else {
			names[id] = "Unknown"
			deleted[id] = true
		}
		count, err := rock.GetRockCountForUser(id)
		if err == nil {
			rockCounts[id] = count
		}
	}

	for i := range conversations {
		id := conversations[i].OtherUserID
		conversations[i].OtherUserName = names[id]
		conversations[i].OtherUserDeleted = deleted[id]
		conversations[i].OtherUserRockCount = rockCounts[id]
	}
}

func GetConversationMessages(conversationID, userID int,
	tz *time.Location) ([]Message, error) {
	conv, err := GetConversation(conversationID, userID)
	if err != nil {
		return nil, err
	}
	return MessagesFromJournal(conv.ID, conv.Journal, tz), nil
}

func CreateMessage(conversationID, senderID int,
	content string) (Message, error) {
	content = strings.TrimSpace(entrylog.SanitizeText(content))
	if content == "" {
		return Message{}, fmt.Errorf("message content cannot be empty")
	}

	if conversationID == 0 {
		return Message{}, fmt.Errorf("conversation ID is required")
	}

	conv, err := GetConversationByID(conversationID)
	if err != nil {
		return Message{}, err
	}

	canPost, err := CanUserPost(conversationID, senderID)
	if err != nil {
		return Message{}, err
	}
	if !canPost {
		return Message{}, ErrNotParticipant
	}
	if !MessagingAllowed(conv) {
		return Message{}, ErrMessagingClosed
	}

	now := time.Now().UTC()
	newJournal := AppendMessageEntry(conv.Journal, senderID, content, now, time.UTC)
	sealed, err := sealJournal(conversationID, newJournal)
	if err != nil {
		return Message{}, fmt.Errorf("seal journal: %w", err)
	}

	var recipientField string
	if conv.OwnerID == senderID {
		recipientField = "inquirer_has_unread"
	} else {
		recipientField = "owner_has_unread"
	}

	_, err = db.Exec(fmt.Sprintf(`
		UPDATE conversations
		SET journal = $1, updated_at = $2, %s = 1
		WHERE id = $3
	`, recipientField), sealed, now, conversationID)
	if err != nil {
		return Message{}, fmt.Errorf("failed to create message: %w", err)
	}

	msgs := MessagesFromJournal(conversationID, newJournal, time.UTC)
	if len(msgs) == 0 {
		return Message{}, fmt.Errorf("failed to read created message")
	}
	return msgs[len(msgs)-1], nil
}

// GetHasUnread checks if the user has any unread conversations
func GetHasUnread(userID int) (bool, error) {
	query := `
		SELECT COUNT(*) > 0
		FROM conversations
		WHERE (owner_id = $1 AND owner_has_unread = 1)
		   OR (inquirer_id = $2 AND inquirer_has_unread = 1)
	`
	var hasUnread bool
	err := db.QueryRow(query, userID, userID).Scan(&hasUnread)
	if err != nil {
		return false, fmt.Errorf("failed to get unread status: %w", err)
	}
	return hasUnread, nil
}

// MarkConversationAsRead marks a conversation as read for the given user
func MarkConversationAsRead(conversationID, userID int) error {
	conv, err := GetConversation(conversationID, userID)
	if err != nil {
		return err
	}

	var field string
	if conv.OwnerID == userID {
		field = "owner_has_unread"
	} else {
		field = "inquirer_has_unread"
	}

	_, err = db.Exec(fmt.Sprintf(`
		UPDATE conversations
		SET %s = 0
		WHERE id = $1
	`, field), conversationID)
	if err != nil {
		return fmt.Errorf("failed to mark conversation as read: %w", err)
	}

	return nil
}
