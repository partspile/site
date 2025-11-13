package messaging

import (
	"database/sql"
	"fmt"
	"time"

	"sync"

	"github.com/parts-pile/site/db"
	"github.com/parts-pile/site/logger"
	"github.com/parts-pile/site/user"
)

// Table name constants
const (
	TableConversation = "Conversation"
	TableMessage      = "Message"
)

// Conversation represents a conversation between two users about an ad
type Conversation struct {
	ID                    int        `json:"id" db:"id"`
	User1ID               int        `json:"user1_id" db:"user1_id"`
	User2ID               int        `json:"user2_id" db:"user2_id"`
	AdID                  int        `json:"ad_id" db:"ad_id"`
	CreatedAt             time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at" db:"updated_at"`
	User1Read             bool       `json:"user1_read" db:"user1_read"`
	User2Read             bool       `json:"user2_read" db:"user2_read"`
	User1FirstMsgNotified bool       `json:"user1_first_msg_notified" db:"user1_first_msg_notified"`
	User2FirstMsgNotified bool       `json:"user2_first_msg_notified" db:"user2_first_msg_notified"`
	PingedByUserID        *int       `json:"pinged_by_user_id,omitempty" db:"pinged_by_user_id"`
	PingedAt              *time.Time `json:"pinged_at,omitempty" db:"pinged_at"`
	// Runtime fields
	User1Name     string     `json:"user1_name,omitempty" db:"user1_name"`
	User2Name     string     `json:"user2_name,omitempty" db:"user2_name"`
	AdTitle       string     `json:"ad_title,omitempty" db:"ad_title"`
	LastMessage   *string    `json:"last_message,omitempty" db:"last_message"`
	LastMessageAt *time.Time `json:"last_message_at,omitempty" db:"last_message_at"`
	UnreadCount   int        `json:"unread_count,omitempty" db:"unread_count"`
	IsUnread      bool       `json:"is_unread,omitempty"` // Whether current user has unread messages
}

// Message represents a single message in a conversation
type Message struct {
	ID             int        `json:"id" db:"id"`
	ConversationID int        `json:"conversation_id" db:"conversation_id"`
	SenderID       int        `json:"sender_id" db:"sender_id"`
	Content        string     `json:"content" db:"content"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	ReadAt         *time.Time `json:"read_at,omitempty" db:"read_at"`
	// Runtime fields
	SenderName string `json:"sender_name,omitempty" db:"sender_name"`
}

// CreateConversation creates a new conversation between two users about an ad
func CreateConversation(user1ID, user2ID, adID int) (int, error) {
	// Ensure user1ID is always the smaller ID for consistent ordering
	if user1ID > user2ID {
		user1ID, user2ID = user2ID, user1ID
	}

	var id int
	err := db.QueryRow(`INSERT INTO Conversation (user1_id, user2_id, ad_id) VALUES ($1, $2, $3) RETURNING id`, user1ID, user2ID, adID).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

// GetOrCreateConversation gets an existing conversation or creates a new one
func GetOrCreateConversation(user1ID, user2ID, adID int) (int, error) {
	// Ensure user1ID is always the smaller ID for consistent ordering
	if user1ID > user2ID {
		user1ID, user2ID = user2ID, user1ID
	}

	// Try to find existing conversation
	var id int
	err := db.QueryRow(`SELECT id FROM Conversation WHERE user1_id = $1 AND user2_id = $2 AND ad_id = $3`, user1ID, user2ID, adID).Scan(&id)
	if err == nil {
		return id, nil
	}

	if err != sql.ErrNoRows {
		return 0, err
	}

	// Create new conversation
	return CreateConversation(user1ID, user2ID, adID)
}

// CreateConversationWithTx creates a new conversation between two users about an ad within a transaction
func CreateConversationWithTx(tx *sql.Tx, user1ID, user2ID, adID int) (int, error) {
	// Ensure user1ID is always the smaller ID for consistent ordering
	if user1ID > user2ID {
		user1ID, user2ID = user2ID, user1ID
	}

	var id int
	err := tx.QueryRow(`INSERT INTO Conversation (user1_id, user2_id, ad_id) VALUES ($1, $2, $3) RETURNING id`, user1ID, user2ID, adID).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

// GetConversationByID retrieves a conversation by ID
func GetConversationByID(id int) (Conversation, error) {
	row := db.QueryRow(`SELECT id, user1_id, user2_id, ad_id, created_at, updated_at, user1_read, user2_read, 
		user1_first_msg_notified, user2_first_msg_notified, pinged_by_user_id, pinged_at 
		FROM Conversation WHERE id = $1`, id)
	var conv Conversation
	var createdAt, updatedAt string
	var user1FirstMsgNotified, user2FirstMsgNotified int
	var pingedByUserID sql.NullInt64
	var pingedAt sql.NullString
	err := row.Scan(&conv.ID, &conv.User1ID, &conv.User2ID, &conv.AdID, &createdAt, &updatedAt,
		&conv.User1Read, &conv.User2Read, &user1FirstMsgNotified, &user2FirstMsgNotified,
		&pingedByUserID, &pingedAt)
	if err != nil {
		return Conversation{}, err
	}

	conv.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	conv.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)
	conv.User1FirstMsgNotified = user1FirstMsgNotified == 1
	conv.User2FirstMsgNotified = user2FirstMsgNotified == 1
	if pingedByUserID.Valid {
		userID := int(pingedByUserID.Int64)
		conv.PingedByUserID = &userID
	}
	if pingedAt.Valid && pingedAt.String != "" {
		if t, err := time.Parse(time.RFC3339Nano, pingedAt.String); err == nil {
			conv.PingedAt = &t
		}
	}

	return conv, nil
}

// GetConversationWithDetails retrieves a conversation by ID with runtime data populated
func GetConversationWithDetails(id int) (Conversation, error) {
	conv, err := GetConversationByID(id)
	if err != nil {
		return Conversation{}, err
	}

	// Get user names
	user1Name, user2Name, err := GetConversationParticipantNames(id)
	if err != nil {
		return Conversation{}, err
	}
	conv.User1Name = user1Name
	conv.User2Name = user2Name

	// Get ad title
	row := db.QueryRow(`SELECT title FROM Ad WHERE id = $1`, conv.AdID)
	var adTitle string
	err = row.Scan(&adTitle)
	if err != nil {
		return Conversation{}, err
	}
	conv.AdTitle = adTitle

	// Get last message info (need to decrypt)
	row = db.QueryRow(`
		SELECT encrypted_content, nonce, created_at 
		FROM Message 
		WHERE conversation_id = $1 
		ORDER BY created_at DESC 
		LIMIT 1
	`, id)
	var encryptedContent, nonce string
	var lastMessageAt time.Time
	err = row.Scan(&encryptedContent, &nonce, &lastMessageAt)
	if err == nil {
		// Decrypt the last message
		plaintext, err := DecryptMessage(id, encryptedContent, nonce)
		if err == nil {
			conv.LastMessage = &plaintext
			conv.LastMessageAt = &lastMessageAt
		}
	}

	return conv, nil
}

// GetConversationsForUser retrieves all conversations for a user
func GetConversationsForUser(userID int) ([]Conversation, error) {
	// First get basic conversation data
	rows, err := db.Query(`
		SELECT c.id, c.user1_id, c.user2_id, c.ad_id, c.created_at, c.updated_at,
		       c.user1_read, c.user2_read,
		       c.user1_first_msg_notified, c.user2_first_msg_notified,
		       c.pinged_by_user_id, c.pinged_at,
		       u1.id, u1.encrypted_name, u1.name_nonce,
		       u2.id, u2.encrypted_name, u2.name_nonce,
		       a.title as ad_title,
		       COUNT(CASE WHEN m2.read_at IS NULL AND m2.sender_id != $1 THEN 1 END) as unread_count
		FROM Conversation c
		JOIN Users u1 ON c.user1_id = u1.id
		JOIN Users u2 ON c.user2_id = u2.id
		JOIN Ad a ON c.ad_id = a.id
		LEFT JOIN Message m2 ON m2.conversation_id = c.id
		WHERE c.user1_id = $2 OR c.user2_id = $3
		GROUP BY c.id
		ORDER BY c.updated_at DESC
	`, userID, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conversations []Conversation
	for rows.Next() {
		var conv Conversation
		var createdAt, updatedAt string
		var user1FirstMsgNotified, user2FirstMsgNotified int
		var pingedByUserID sql.NullInt64
		var pingedAt sql.NullString
		var user1ID, user2ID int
		var user1EncryptedName, user1NameNonce, user2EncryptedName, user2NameNonce string

		err := rows.Scan(&conv.ID, &conv.User1ID, &conv.User2ID, &conv.AdID, &createdAt, &updatedAt,
			&conv.User1Read, &conv.User2Read, &user1FirstMsgNotified, &user2FirstMsgNotified,
			&pingedByUserID, &pingedAt,
			&user1ID, &user1EncryptedName, &user1NameNonce,
			&user2ID, &user2EncryptedName, &user2NameNonce,
			&conv.AdTitle, &conv.UnreadCount)
		if err != nil {
			continue
		}

		// Decrypt user names
		conv.User1Name, err = user.DecryptName(user1ID, user1EncryptedName, user1NameNonce)
		if err != nil {
			logger.Error("Failed to decrypt user1 name", "error", err, "conversationID", conv.ID)
			continue
		}
		conv.User2Name, err = user.DecryptName(user2ID, user2EncryptedName, user2NameNonce)
		if err != nil {
			logger.Error("Failed to decrypt user2 name", "error", err, "conversationID", conv.ID)
			continue
		}

		conv.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
		conv.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)
		conv.User1FirstMsgNotified = user1FirstMsgNotified == 1
		conv.User2FirstMsgNotified = user2FirstMsgNotified == 1
		if pingedByUserID.Valid {
			userIDVal := int(pingedByUserID.Int64)
			conv.PingedByUserID = &userIDVal
		}
		if pingedAt.Valid && pingedAt.String != "" {
			if t, err := time.Parse(time.RFC3339Nano, pingedAt.String); err == nil {
				conv.PingedAt = &t
			}
		}

		// Get last message (need to decrypt)
		row := db.QueryRow(`
			SELECT encrypted_content, nonce, created_at 
			FROM Message 
			WHERE conversation_id = $1 
			ORDER BY created_at DESC 
			LIMIT 1
		`, conv.ID)
		var encryptedContent, nonce string
		var lastMessageAt time.Time
		err = row.Scan(&encryptedContent, &nonce, &lastMessageAt)
		if err == nil {
			plaintext, err := DecryptMessage(conv.ID, encryptedContent, nonce)
			if err == nil {
				conv.LastMessage = &plaintext
				conv.LastMessageAt = &lastMessageAt
			}
		}

		conversations = append(conversations, conv)
	}

	// Calculate if conversation is unread for current user
	for i := range conversations {
		conversations[i].IsUnread = conversations[i].UnreadCount > 0
	}

	return conversations, nil
}

// MarkConversationAsRead marks a conversation as read for a specific user and resets ping state
func MarkConversationAsRead(conversationID, userID int) error {
	// Check if this user was pinged (get current state)
	var pingedByUserID sql.NullInt64
	err := db.QueryRow(`
		SELECT pinged_by_user_id
		FROM Conversation
		WHERE id = $1
	`, conversationID).Scan(&pingedByUserID)
	if err != nil {
		return err
	}

	// If other user pinged this user, reset ping state
	if pingedByUserID.Valid && int(pingedByUserID.Int64) != userID {
		_, err = db.Exec(`
			UPDATE Conversation
			SET pinged_by_user_id = NULL, pinged_at = NULL
			WHERE id = $1
		`, conversationID)
		if err != nil {
			return err
		}
	}

	// Get conversation to determine which user field to update
	conv, err := GetConversationByID(conversationID)
	if err != nil {
		return err
	}

	// Mark as read
	if conv.User1ID == userID {
		_, err = db.Exec(`UPDATE Conversation SET user1_read = TRUE WHERE id = $1`, conversationID)
	} else if conv.User2ID == userID {
		_, err = db.Exec(`UPDATE Conversation SET user2_read = TRUE WHERE id = $1`, conversationID)
	} else {
		return fmt.Errorf("user %d is not part of conversation %d", userID, conversationID)
	}
	return err
}

// GetConversationWithUser retrieves a conversation between two specific users about an ad
func GetConversationWithUser(user1ID, user2ID, adID int) (Conversation, error) {
	// Ensure user1ID is always the smaller ID for consistent ordering
	if user1ID > user2ID {
		user1ID, user2ID = user2ID, user1ID
	}

	row := db.QueryRow(`
		SELECT c.id, c.user1_id, c.user2_id, c.ad_id, c.created_at, c.updated_at,
		       u1.id, u1.encrypted_name, u1.name_nonce,
		       u2.id, u2.encrypted_name, u2.name_nonce,
		       a.title as ad_title
		FROM Conversation c
		JOIN Users u1 ON c.user1_id = u1.id
		JOIN Users u2 ON c.user2_id = u2.id
		JOIN Ad a ON c.ad_id = a.id
		WHERE c.user1_id = $1 AND c.user2_id = $2 AND c.ad_id = $3
	`, user1ID, user2ID, adID)

	var conv Conversation
	var createdAt, updatedAt string
	var u1ID, u2ID int
	var u1EncryptedName, u1NameNonce, u2EncryptedName, u2NameNonce string
	err := row.Scan(&conv.ID, &conv.User1ID, &conv.User2ID, &conv.AdID, &createdAt, &updatedAt,
		&u1ID, &u1EncryptedName, &u1NameNonce,
		&u2ID, &u2EncryptedName, &u2NameNonce,
		&conv.AdTitle)
	if err != nil {
		return Conversation{}, err
	}

	// Decrypt user names
	conv.User1Name, err = user.DecryptName(u1ID, u1EncryptedName, u1NameNonce)
	if err != nil {
		return Conversation{}, fmt.Errorf("failed to decrypt user1 name: %w", err)
	}
	conv.User2Name, err = user.DecryptName(u2ID, u2EncryptedName, u2NameNonce)
	if err != nil {
		return Conversation{}, fmt.Errorf("failed to decrypt user2 name: %w", err)
	}

	conv.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	conv.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)
	return conv, nil
}

// AddMessage adds a new message to a conversation (with encryption)
// Returns messageID and whether a notification should be sent
func AddMessage(conversationID, senderID int, content string) (messageID int, shouldNotify bool, err error) {
	// Encrypt message before storing
	encryptedContent, nonce, err := EncryptMessage(conversationID, content)
	if err != nil {
		return 0, false, fmt.Errorf("failed to encrypt message: %w", err)
	}

	// Store encrypted message
	tx, err := db.Begin()
	if err != nil {
		return 0, false, err
	}
	defer tx.Rollback()

	var msgID int64
	err = tx.QueryRow(`
		INSERT INTO Message (conversation_id, sender_id, encrypted_content, nonce, created_at)
		VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP)
		RETURNING id
	`, conversationID, senderID, encryptedContent, nonce).Scan(&msgID)
	if err != nil {
		return 0, false, err
	}
	messageID = int(msgID)

	// Get conversation to check notification state
	// Note: We need to query within transaction to get current state
	var user1ID, user2ID int
	var user1FirstMsgNotified, user2FirstMsgNotified int
	err = tx.QueryRow(`
		SELECT user1_id, user2_id, user1_first_msg_notified, user2_first_msg_notified
		FROM Conversation
		WHERE id = $1
	`, conversationID).Scan(&user1ID, &user2ID, &user1FirstMsgNotified, &user2FirstMsgNotified)
	if err != nil {
		return 0, false, err
	}

	// Determine if this is sender's first message
	var isFirstMessage bool
	var notifyField string
	if user1ID == senderID {
		isFirstMessage = user1FirstMsgNotified == 0
		notifyField = "user1_first_msg_notified"
	} else {
		isFirstMessage = user2FirstMsgNotified == 0
		notifyField = "user2_first_msg_notified"
	}

	// Mark first message as notified if this is the first
	if isFirstMessage {
		_, err = tx.Exec(fmt.Sprintf(`UPDATE Conversation SET %s = 1 WHERE id = $1`, notifyField), conversationID)
		if err != nil {
			return 0, false, err
		}
	}

	// Mark conversation as unread for recipient
	if user1ID == senderID {
		_, err = tx.Exec(`UPDATE Conversation SET user2_read = FALSE WHERE id = $1`, conversationID)
	} else {
		_, err = tx.Exec(`UPDATE Conversation SET user1_read = FALSE WHERE id = $1`, conversationID)
	}
	if err != nil {
		return 0, false, err
	}

	// Update conversation timestamp
	_, err = tx.Exec(`UPDATE Conversation SET updated_at = CURRENT_TIMESTAMP WHERE id = $1`, conversationID)
	if err != nil {
		return 0, false, err
	}

	if err := tx.Commit(); err != nil {
		return 0, false, err
	}

	// Create a message struct for notification
	msg := Message{
		ID:             messageID,
		ConversationID: conversationID,
		SenderID:       senderID,
		Content:        content,
		CreatedAt:      time.Now(),
	}

	// Notify all participants about the new message
	NotifyConversationUpdate(conversationID, "new_message", &msg)

	// Return whether notification should be sent (only for first message)
	return messageID, isFirstMessage, nil
}

// AddMessageWithTx adds a new message to a conversation within a transaction
func AddMessageWithTx(tx *sql.Tx, conversationID, senderID int, content string) (int, error) {
	// Encrypt message before storing
	encryptedContent, nonce, err := EncryptMessage(conversationID, content)
	if err != nil {
		return 0, fmt.Errorf("failed to encrypt message: %w", err)
	}

	var id int
	err = tx.QueryRow(`INSERT INTO Message (conversation_id, sender_id, encrypted_content, nonce) VALUES ($1, $2, $3, $4) RETURNING id`,
		conversationID, senderID, encryptedContent, nonce).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

// GetMessages retrieves all messages for a conversation (with decryption)
func GetMessages(conversationID int) ([]Message, error) {
	rows, err := db.Query(`
		SELECT m.id, m.conversation_id, m.sender_id, m.encrypted_content, m.nonce, 
		       m.created_at, m.read_at, u.encrypted_name, u.name_nonce
		FROM Message m
		JOIN Users u ON m.sender_id = u.id
		WHERE m.conversation_id = $1
		ORDER BY m.created_at ASC
	`, conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		var encryptedContent, nonce string
		var createdAt, readAt sql.NullString
		var senderEncryptedName, senderNameNonce string

		err := rows.Scan(&msg.ID, &msg.ConversationID, &msg.SenderID,
			&encryptedContent, &nonce, &createdAt, &readAt, &senderEncryptedName, &senderNameNonce)
		if err != nil {
			logger.Error("Failed to scan message row", "error", err, "conversationID", conversationID)
			continue // Skip this message but continue with others
		}

		// Parse timestamps
		if createdAt.Valid {
			msg.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt.String)
		}
		if readAt.Valid && readAt.String != "" {
			readAtTime, _ := time.Parse(time.RFC3339Nano, readAt.String)
			msg.ReadAt = &readAtTime
		}

		// Validate encrypted fields are not empty
		if encryptedContent == "" || nonce == "" {
			logger.Error("Message has empty encrypted fields", "messageID", msg.ID, "conversationID", conversationID)
			msg.Content = "[Unable to decrypt message - missing encryption data]"
			msg.SenderName = "[Unknown]"
			messages = append(messages, msg)
			continue
		}

		// Decrypt sender name (use msg.SenderID which was already scanned)
		msg.SenderName, err = user.DecryptName(msg.SenderID, senderEncryptedName, senderNameNonce)
		if err != nil {
			logger.Error("Failed to decrypt sender name", "error", err, "senderID", msg.SenderID, "messageID", msg.ID)
			msg.SenderName = "[Unknown]" // Fallback
		}

		// Decrypt message
		plaintext, err := DecryptMessage(conversationID, encryptedContent, nonce)
		if err != nil {
			// Log error but continue (might be corrupted data)
			logger.Error("Failed to decrypt message", "error", err, "message_id", msg.ID, "conversationID", conversationID)
			msg.Content = "[Unable to decrypt message]"
		} else {
			msg.Content = plaintext
		}

		messages = append(messages, msg)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating message rows: %w", err)
	}

	return messages, nil
}

// MarkMessagesAsRead marks all messages in a conversation as read by a specific user
func MarkMessagesAsRead(conversationID, userID int) error {
	_, err := db.Exec(`
		UPDATE Message 
		SET read_at = CURRENT_TIMESTAMP 
		WHERE conversation_id = $1 AND sender_id != $2 AND read_at IS NULL
	`, conversationID, userID)
	return err
}

// GetUnreadCount returns the number of unread messages for a user
func GetUnreadCount(userID int) (int, error) {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM Message m
		JOIN Conversation c ON m.conversation_id = c.id
		WHERE (c.user1_id = $1 OR c.user2_id = $2) 
		AND m.sender_id != $3 
		AND m.read_at IS NULL
	`, userID, userID, userID).Scan(&count)
	return count, err
}

// GetConversationParticipantNames returns the names of both participants in a conversation
func GetConversationParticipantNames(conversationID int) (string, string, error) {
	row := db.QueryRow(`
		SELECT u1.id, u1.encrypted_name, u1.name_nonce, u2.id, u2.encrypted_name, u2.name_nonce
		FROM Conversation c
		JOIN Users u1 ON c.user1_id = u1.id
		JOIN Users u2 ON c.user2_id = u2.id
		WHERE c.id = $1
	`, conversationID)

	var user1ID, user2ID int
	var user1EncryptedName, user1NameNonce, user2EncryptedName, user2NameNonce string
	err := row.Scan(&user1ID, &user1EncryptedName, &user1NameNonce, &user2ID, &user2EncryptedName, &user2NameNonce)
	if err != nil {
		return "", "", err
	}

	// Decrypt user names
	user1Name, err := user.DecryptName(user1ID, user1EncryptedName, user1NameNonce)
	if err != nil {
		return "", "", fmt.Errorf("failed to decrypt user1 name: %w", err)
	}

	user2Name, err := user.DecryptName(user2ID, user2EncryptedName, user2NameNonce)
	if err != nil {
		return "", "", fmt.Errorf("failed to decrypt user2 name: %w", err)
	}

	return user1Name, user2Name, nil
}

// CanUserMessageAd checks if a user can send a message about an ad
func CanUserMessageAd(userID, adUserID int) error {
	if userID == adUserID {
		return fmt.Errorf("users cannot message themselves")
	}
	return nil
}

// ConversationUpdate represents a real-time update for a conversation
type ConversationUpdate struct {
	Type           string                 `json:"type"`
	ConversationID int                    `json:"conversation_id"`
	Message        *Message               `json:"message,omitempty"`
	UnreadCount    int                    `json:"unread_count,omitempty"`
	Data           map[string]interface{} `json:"data,omitempty"`
}

// Global map to store user update channels
var userUpdateChannels = make(map[int]chan ConversationUpdate)
var userUpdateMutex sync.RWMutex

// RegisterUserUpdates registers a channel for a user to receive real-time updates
func RegisterUserUpdates(userID int, ch chan ConversationUpdate) {
	userUpdateMutex.Lock()
	defer userUpdateMutex.Unlock()
	userUpdateChannels[userID] = ch
}

// UnregisterUserUpdates removes a user's update channel
func UnregisterUserUpdates(userID int) {
	userUpdateMutex.Lock()
	defer userUpdateMutex.Unlock()
	delete(userUpdateChannels, userID)
}

// NotifyUserUpdate sends an update to a specific user
func NotifyUserUpdate(userID int, update ConversationUpdate) {
	userUpdateMutex.RLock()
	defer userUpdateMutex.RUnlock()

	if ch, exists := userUpdateChannels[userID]; exists {
		select {
		case ch <- update:
			// Update sent successfully
		default:
			// Channel is full, skip this update
		}
	}
}

// NotifyConversationUpdate notifies all participants in a conversation about an update
func NotifyConversationUpdate(conversationID int, updateType string, message *Message) {
	conv, err := GetConversationWithDetails(conversationID)
	if err != nil {
		return
	}

	// Create update for both users
	update := ConversationUpdate{
		Type:           updateType,
		ConversationID: conversationID,
		Message:        message,
		UnreadCount:    1, // Increment unread count
	}

	// Notify both users
	NotifyUserUpdate(conv.User1ID, update)
	NotifyUserUpdate(conv.User2ID, update)
}

// HandlePing sends a ping notification to the other participant
func HandlePing(conversationID, userID int) error {
	// Update ping state
	_, err := db.Exec(`
		UPDATE Conversation
		SET pinged_by_user_id = $1, pinged_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`, userID, conversationID)
	if err != nil {
		return err
	}

	return nil
}
