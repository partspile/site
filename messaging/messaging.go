package messaging

import (
	"database/sql"
	"fmt"
	"time"

	"sync"

	"github.com/parts-pile/site/db"
)

// Table name constants
const (
	TableConversation = "Conversation"
	TableMessage      = "Message"
)

// Conversation represents a conversation between two users about an ad
type Conversation struct {
	ID        int       `json:"id"`
	User1ID   int       `json:"user1_id"`
	User2ID   int       `json:"user2_id"`
	AdID      int       `json:"ad_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	User1Read bool      `json:"user1_read"`
	User2Read bool      `json:"user2_read"`
	// Runtime fields
	User1Name     string    `json:"user1_name,omitempty"`
	User2Name     string    `json:"user2_name,omitempty"`
	AdTitle       string    `json:"ad_title,omitempty"`
	LastMessage   string    `json:"last_message,omitempty"`
	LastMessageAt time.Time `json:"last_message_at,omitempty"`
	UnreadCount   int       `json:"unread_count,omitempty"`
	IsUnread      bool      `json:"is_unread,omitempty"` // Whether current user has unread messages
}

// Message represents a single message in a conversation
type Message struct {
	ID             int        `json:"id"`
	ConversationID int        `json:"conversation_id"`
	SenderID       int        `json:"sender_id"`
	Content        string     `json:"content"`
	CreatedAt      time.Time  `json:"created_at"`
	ReadAt         *time.Time `json:"read_at,omitempty"`
	// Runtime fields
	SenderName string `json:"sender_name,omitempty"`
}

// CreateConversation creates a new conversation between two users about an ad
func CreateConversation(user1ID, user2ID, adID int) (int, error) {
	// Ensure user1ID is always the smaller ID for consistent ordering
	if user1ID > user2ID {
		user1ID, user2ID = user2ID, user1ID
	}

	res, err := db.Exec(`INSERT INTO Conversation (user1_id, user2_id, ad_id) VALUES (?, ?, ?)`, user1ID, user2ID, adID)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	return int(id), nil
}

// GetOrCreateConversation gets an existing conversation or creates a new one
func GetOrCreateConversation(user1ID, user2ID, adID int) (int, error) {
	// Ensure user1ID is always the smaller ID for consistent ordering
	if user1ID > user2ID {
		user1ID, user2ID = user2ID, user1ID
	}

	// Try to find existing conversation
	var id int
	err := db.QueryRow(`SELECT id FROM Conversation WHERE user1_id = ? AND user2_id = ? AND ad_id = ?`, user1ID, user2ID, adID).Scan(&id)
	if err == nil {
		return id, nil
	}

	if err != sql.ErrNoRows {
		return 0, err
	}

	// Create new conversation
	return CreateConversation(user1ID, user2ID, adID)
}

// GetConversationByID retrieves a conversation by ID
func GetConversationByID(id int) (Conversation, error) {
	row := db.QueryRow(`SELECT id, user1_id, user2_id, ad_id, created_at, updated_at, user1_read, user2_read FROM Conversation WHERE id = ?`, id)
	var conv Conversation
	var createdAt, updatedAt string
	err := row.Scan(&conv.ID, &conv.User1ID, &conv.User2ID, &conv.AdID, &createdAt, &updatedAt, &conv.User1Read, &conv.User2Read)
	if err != nil {
		return Conversation{}, err
	}

	conv.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	conv.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)

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
	row := db.QueryRow(`SELECT title FROM Ad WHERE id = ?`, conv.AdID)
	var adTitle string
	err = row.Scan(&adTitle)
	if err != nil {
		return Conversation{}, err
	}
	conv.AdTitle = adTitle

	// Get last message info
	row = db.QueryRow(`
		SELECT content, created_at 
		FROM Message 
		WHERE conversation_id = ? 
		ORDER BY created_at DESC 
		LIMIT 1
	`, id)
	var lastMessage string
	var lastMessageAt string
	err = row.Scan(&lastMessage, &lastMessageAt)
	if err == nil {
		conv.LastMessage = lastMessage
		conv.LastMessageAt, _ = time.Parse(time.RFC3339Nano, lastMessageAt)
	}

	return conv, nil
}

// GetConversationsForUser retrieves all conversations for a user
func GetConversationsForUser(userID int) ([]Conversation, error) {
	rows, err := db.Query(`
		SELECT c.id, c.user1_id, c.user2_id, c.ad_id, c.created_at, c.updated_at,
		       c.user1_read, c.user2_read,
		       u1.name as user1_name, u2.name as user2_name,
		       a.title as ad_title,
		       m.content as last_message, m.created_at as last_message_at,
		       COUNT(CASE WHEN m2.read_at IS NULL AND m2.sender_id != ? THEN 1 END) as unread_count
		FROM Conversation c
		JOIN User u1 ON c.user1_id = u1.id
		JOIN User u2 ON c.user2_id = u2.id
		JOIN Ad a ON c.ad_id = a.id
		LEFT JOIN Message m ON m.id = (
			SELECT id FROM Message 
			WHERE conversation_id = c.id 
			ORDER BY created_at DESC 
			LIMIT 1
		)
		LEFT JOIN Message m2 ON m2.conversation_id = c.id
		WHERE c.user1_id = ? OR c.user2_id = ?
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
		var lastMessage sql.NullString
		var lastMessageAtStr sql.NullString
		err := rows.Scan(&conv.ID, &conv.User1ID, &conv.User2ID, &conv.AdID, &createdAt, &updatedAt,
			&conv.User1Read, &conv.User2Read,
			&conv.User1Name, &conv.User2Name, &conv.AdTitle,
			&lastMessage, &lastMessageAtStr, &conv.UnreadCount)
		if err != nil {
			return nil, err
		}

		conv.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
		conv.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)

		if lastMessage.Valid {
			conv.LastMessage = lastMessage.String
		}
		if lastMessageAtStr.Valid {
			conv.LastMessageAt, _ = time.Parse(time.RFC3339Nano, lastMessageAtStr.String)
		}

		// Calculate if conversation is unread for current user
		conv.IsUnread = conv.UnreadCount > 0

		conversations = append(conversations, conv)
	}

	return conversations, nil
}

// MarkConversationAsRead marks a conversation as read for a specific user
func MarkConversationAsRead(conversationID, userID int) error {
	// Determine which read field to update based on user position in conversation
	conv, err := GetConversationByID(conversationID)
	if err != nil {
		return err
	}

	if conv.User1ID == userID {
		_, err = db.Exec(`UPDATE Conversation SET user1_read = TRUE WHERE id = ?`, conversationID)
	} else if conv.User2ID == userID {
		_, err = db.Exec(`UPDATE Conversation SET user2_read = TRUE WHERE id = ?`, conversationID)
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
		       u1.name as user1_name, u2.name as user2_name,
		       a.title as ad_title
		FROM Conversation c
		JOIN User u1 ON c.user1_id = u1.id
		JOIN User u2 ON c.user2_id = u2.id
		JOIN Ad a ON c.ad_id = a.id
		WHERE c.user1_id = ? AND c.user2_id = ? AND c.ad_id = ?
	`, user1ID, user2ID, adID)

	var conv Conversation
	var createdAt, updatedAt string
	err := row.Scan(&conv.ID, &conv.User1ID, &conv.User2ID, &conv.AdID, &createdAt, &updatedAt,
		&conv.User1Name, &conv.User2Name, &conv.AdTitle)
	if err != nil {
		return Conversation{}, err
	}

	conv.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	conv.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)
	return conv, nil
}

// AddMessage adds a new message to a conversation
func AddMessage(conversationID, senderID int, content string) (int, error) {
	// Start a transaction to update both the message and conversation
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	// Insert the message
	res, err := tx.Exec(`INSERT INTO Message (conversation_id, sender_id, content) VALUES (?, ?, ?)`, conversationID, senderID, content)
	if err != nil {
		return 0, err
	}
	messageID, _ := res.LastInsertId()

	// Update the conversation's updated_at timestamp
	_, err = tx.Exec(`UPDATE Conversation SET updated_at = CURRENT_TIMESTAMP WHERE id = ?`, conversationID)
	if err != nil {
		return 0, err
	}

	// Mark conversation as unread for the recipient
	conv, err := GetConversationByID(conversationID)
	if err != nil {
		return 0, err
	}

	if conv.User1ID == senderID {
		_, err = tx.Exec(`UPDATE Conversation SET user2_read = FALSE WHERE id = ?`, conversationID)
	} else {
		_, err = tx.Exec(`UPDATE Conversation SET user1_read = FALSE WHERE id = ?`, conversationID)
	}
	if err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	// Create a message struct for notification
	msg := Message{
		ID:             int(messageID),
		ConversationID: conversationID,
		SenderID:       senderID,
		Content:        content,
		CreatedAt:      time.Now(),
	}

	// Notify all participants about the new message
	NotifyConversationUpdate(conversationID, "new_message", &msg)

	return int(messageID), nil
}

// GetMessages retrieves all messages for a conversation
func GetMessages(conversationID int) ([]Message, error) {
	rows, err := db.Query(`
		SELECT m.id, m.conversation_id, m.sender_id, m.content, m.created_at, m.read_at,
		       u.name as sender_name
		FROM Message m
		JOIN User u ON m.sender_id = u.id
		WHERE m.conversation_id = ?
		ORDER BY m.created_at ASC
	`, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		var createdAt string
		var readAt sql.NullString
		err := rows.Scan(&msg.ID, &msg.ConversationID, &msg.SenderID, &msg.Content, &createdAt, &readAt, &msg.SenderName)
		if err != nil {
			return nil, err
		}

		msg.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
		if readAt.Valid {
			readTime, _ := time.Parse(time.RFC3339Nano, readAt.String)
			msg.ReadAt = &readTime
		}

		messages = append(messages, msg)
	}

	return messages, nil
}

// MarkMessagesAsRead marks all messages in a conversation as read by a specific user
func MarkMessagesAsRead(conversationID, userID int) error {
	_, err := db.Exec(`
		UPDATE Message 
		SET read_at = CURRENT_TIMESTAMP 
		WHERE conversation_id = ? AND sender_id != ? AND read_at IS NULL
	`, conversationID, userID)
	return err
}

// GetUnreadCount returns the number of unread messages for a user
func GetUnreadCount(userID int) (int, error) {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM Message m
		JOIN Conversation c ON m.conversation_id = c.id
		WHERE (c.user1_id = ? OR c.user2_id = ?) 
		AND m.sender_id != ? 
		AND m.read_at IS NULL
	`, userID, userID, userID).Scan(&count)
	return count, err
}

// GetConversationParticipantNames returns the names of both participants in a conversation
func GetConversationParticipantNames(conversationID int) (string, string, error) {
	row := db.QueryRow(`
		SELECT u1.name, u2.name
		FROM Conversation c
		JOIN User u1 ON c.user1_id = u1.id
		JOIN User u2 ON c.user2_id = u2.id
		WHERE c.id = ?
	`, conversationID)

	var user1Name, user2Name string
	err := row.Scan(&user1Name, &user2Name)
	return user1Name, user2Name, err
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
	conv, err := GetConversationByID(conversationID)
	if err != nil {
		return
	}

	// Create update for both users
	update := ConversationUpdate{
		Type:           updateType,
		ConversationID: conversationID,
		Message:        message,
	}

	// Notify both users
	NotifyUserUpdate(conv.User1ID, update)
	NotifyUserUpdate(conv.User2ID, update)
}
