# Option 1: Server-Side Encryption Implementation

This document details the implementation of server-side encryption for messages and other sensitive data. This approach encrypts data at rest in the database, protecting against database breaches while maintaining compatibility with your async messaging model.

**Package Structure:**
- `encryption/encryption.go` - Generic encryption package (reusable for any data type)
- `messaging/encryption.go` - Message-specific convenience wrappers

## Architecture Overview

**Key Principles:**
- Messages encrypted on server before storing in database
- Messages decrypted on server when retrieving
- Per-conversation encryption keys (derived, not stored)
- Works with async messaging - no waiting for recipients
- No JavaScript required - works with HTMX/no-JS architecture

**How It Works with Your App Architecture:**
- **Asynchronous Communication:** User A sends message → Server encrypts → Stores encrypted → User B receives later (async)
- **Message Flow:** A sends plaintext → Server encrypts → Database stores encrypted → B requests → Server decrypts → B sees plaintext
- **Real-time UI Updates:** Works with existing HTMX/SSE - server decrypts before sending to client
- **Notification System:** Server can read message content (for notifications) - sends: "New message from [Sender] about [Ad]: [Message preview]. View at: [link]"
- **Perfect for Async:** No waiting for recipient to set up encryption - messages can be sent immediately

## System Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          SERVER-SIDE ENCRYPTION SYSTEM                      │
└─────────────────────────────────────────────────────────────────────────────┘

┌──────────────────────┐                                    ┌──────────────────────┐
│   USER A (Browser)  │                                    │   USER B (Browser)  │
│   (Buyer/Seller)    │                                    │   (Buyer/Seller)    │
│                      │                                    │                      │
│  ┌────────────────┐ │                                    │  ┌────────────────┐ │
│  │  Types Message │ │                                    │  │  Views Messages │ │
│  │  "How much?"   │ │                                    │  │  "How much?"    │ │
│  └──────┬─────────┘ │                                    │  └──────▲──────────┘ │
│         │            │                                    │         │            │
│         │            │                                    │         │            │
│  ASYNC: A and B don't communicate directly                │         │            │
│  They both use Server as intermediary/relay               │         │            │
│         │            │                                    │         │            │
└─────────┼────────────┘                                    └─────────┼────────────┘
          │                                                           │
          │  [1] POST /messages/501/send                             │
          │  { message: "How much?" }                                │
          │───────────────────────────────────────────────────────────┼──────────┐
          │                                                           │          │
          │                                                           │          │
┌─────────▼───────────────────────────────────────────────────────────▼──────────▼──┐
│                              SERVER (Go Backend)                                  │
│                                                                                   │
│  ┌──────────────────────────────────────────────────────────────────────────┐   │
│  │                    Message Handlers                                      │   │
│  │                                                                           │   │
│  │  • HandleSendMessage()  - Receives plaintext, encrypts, stores          │   │
│  │  • HandleExpandConversation() - Retrieves encrypted, decrypts, returns  │   │
│  │  • HandleSSE() - Real-time updates (decrypts before sending)            │   │
│  │                                                                           │   │
│  │  Server acts as ENCRYPTION LAYER:                                        │   │
│  │  - Receives plaintext from A → Encrypts → Stores encrypted             │   │
│  │  - B requests messages → Retrieves encrypted → Decrypts → Returns plaintext│
│  │  - Real-time: SSE decrypts before pushing to client                     │   │
│  │                                                                           │   │
│  │  Note: Server CAN decrypt messages (not true E2EE, but encrypted at rest)│   │
│  └──────────────────────────────────────────────────────────────────────────┘   │
│                                    │                                              │
│                                    │                                              │
│  ┌─────────────────────────────────▼──────────────────────────────────────┐   │
│  │                         Encryption Package                           │   │
│  │                                                                       │   │
│  │  • EncryptMessage(conversationID, plaintext) → encrypted bytes     │   │
│  │  • DecryptMessage(conversationID, encrypted bytes) → plaintext     │   │
│  │  • DeriveKey(conversationID) → encryption key (HKDF)               │   │
│  │                                                                       │   │
│  │  Algorithm: AES-256-GCM                                              │   │
│  │  Key Derivation: HKDF-SHA256(conversationID, master_key)            │   │
│  └───────────────────────────────────────────────────────────────────────┘   │
│                                    │                                              │
└────────────────────────────────────┼──────────────────────────────────────────────┘
                                     │
                                     │
┌────────────────────────────────────▼──────────────────────────────────────────────┐
│                              DATABASE (SQLite)                                     │
│                                                                                    │
│  ┌──────────────────────────────────────────────────────────────────────────┐    │
│  │                        Message Table                                      │    │
│  │  ┌────┬──────────────┬──────────┬──────────────────┬──────────┐         │    │
│  │  │ id │ conversation │ sender   │ encrypted_content │  nonce   │         │    │
│  │  ├────┼──────────────┼──────────┼──────────────────┼──────────┤         │    │
│  │  │ 1  │     501      │   101    │  [ciphertext]    │ [12 bytes]│         │    │
│  │  │ 2  │     501      │   102    │  [ciphertext]    │ [12 bytes]│         │    │
│  │  └────┴──────────────┴──────────┴──────────────────┴──────────┘         │    │
│  │                                                                           │    │
│  │  Note: Server stores encrypted bytes - protects against database breach  │    │
│  │  Master key NOT stored in database (in config/env)                       │    │
│  └──────────────────────────────────────────────────────────────────────────┘    │
└────────────────────────────────────────────────────────────────────────────────────┘
```

## Components Required

### 1. Encryption Package
- `encryption/encryption.go` - Generic AES-256-GCM encryption/decryption
- Supports multiple data types with separate key derivation
- Per-data-type key derivation using HKDF
- Master key from config (not stored in database)
- `messaging/encryption.go` - Message-specific convenience wrappers

### 2. Database Schema
- Replace `content` TEXT field with `encrypted_content` BLOB field
- Add `nonce` BLOB field (12 bytes for AES-GCM)
- Add `encryption_version` INTEGER field

### 3. Message Handlers
- Encrypt before storing in `AddMessage()`
- Decrypt when retrieving in `GetMessages()`
- Update all message retrieval paths

### 4. Configuration
- Add master encryption key to config
- Store in environment variable (like JWT secret)

## Implementation Details

### Part 1: Database Schema Changes

Update `schema.sql`:

```sql
-- Update Message table to use encryption
CREATE TABLE Message (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    conversation_id INTEGER NOT NULL REFERENCES Conversation(id),
    sender_id INTEGER NOT NULL REFERENCES User(id),
    encrypted_content BLOB NOT NULL,  -- Changed from content TEXT
    nonce BLOB NOT NULL,              -- For AES-GCM (12 bytes)
    encryption_version INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    read_at DATETIME
);
CREATE INDEX idx_message_conversation_id ON Message(conversation_id);
CREATE INDEX idx_message_sender_id ON Message(sender_id);
CREATE INDEX idx_message_created_at ON Message(created_at);
CREATE INDEX idx_message_read_at ON Message(read_at);

-- Update Conversation table to track notification state
-- Add these columns to existing Conversation table:
ALTER TABLE Conversation ADD COLUMN user1_first_msg_notified INTEGER DEFAULT 0;
ALTER TABLE Conversation ADD COLUMN user2_first_msg_notified INTEGER DEFAULT 0;
ALTER TABLE Conversation ADD COLUMN pinged_by_user_id INTEGER REFERENCES User(id);
ALTER TABLE Conversation ADD COLUMN pinged_at DATETIME;

-- Add user setting for summary notifications
-- Add to existing User table:
ALTER TABLE User ADD COLUMN summary_notifications_enabled INTEGER DEFAULT 0;
ALTER TABLE User ADD COLUMN summary_notifications_frequency TEXT DEFAULT 'daily';
```

**Note:** Since you'll rerun `rebuild_db`, just update `schema.sql` directly. No migration needed.

### Part 2: Package Structure

```
encryption/
  └── encryption.go    # Generic encryption functions (EncryptGeneric, DecryptGeneric)

messaging/
  └── encryption.go    # Message-specific convenience wrappers
```

### Part 4: Configuration

#### `config/config.go` - Add Encryption Key

```go
package config

import (
	"crypto/rand"
	"encoding/base64"
	"os"
)

var (
	// ... existing config ...
	
	// MessageEncryptionKey is the master key for encrypting messages
	// Must be 32 bytes (256 bits) for AES-256
	MessageEncryptionKey []byte
)

func init() {
	// Load encryption key from environment variable
	keyStr := os.Getenv("MESSAGE_ENCRYPTION_KEY")
	if keyStr == "" {
		// Generate a new key (for development only - should be set in production)
		logger.Warn("MESSAGE_ENCRYPTION_KEY not set, generating temporary key")
		key := make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			logger.Fatal("Failed to generate encryption key", "error", err)
		}
		MessageEncryptionKey = key
		logger.Info("Generated temporary encryption key. Set MESSAGE_ENCRYPTION_KEY in production!")
	} else {
		// Decode base64 key from environment
		key, err := base64.StdEncoding.DecodeString(keyStr)
		if err != nil {
			logger.Fatal("Invalid MESSAGE_ENCRYPTION_KEY format", "error", err)
		}
		if len(key) != 32 {
			logger.Fatal("MESSAGE_ENCRYPTION_KEY must be 32 bytes (256 bits)")
		}
		MessageEncryptionKey = key
	}
}
```

**Production Setup:**
```bash
# Generate a secure key
openssl rand -base64 32

# Set in environment
export MESSAGE_ENCRYPTION_KEY="<generated-key>"
```

### Part 5: Encryption Package

#### `encryption/encryption.go` - Generic Encryption Implementation

```go
package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/hkdf"
	"github.com/parts-pile/site/config"
)

const (
	EncryptionVersion1 = 1
	NonceSize         = 12 // For AES-GCM
	KeySize           = 32 // 256 bits
)

// EncryptGeneric encrypts any plaintext data using a data type and ID
// Returns encrypted content (base64) and nonce (base64)
func EncryptGeneric(dataType DataType, id int, plaintext string, masterKey []byte) (encryptedContent, nonce string, err error) {
	// Derive key for this data type and ID
	key, err := deriveKey(dataType, id, masterKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to derive key: %w", err)
	}

	// Create AES-GCM cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", "", fmt.Errorf("failed to create cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate random nonce
	nonceBytes := make([]byte, NonceSize)
	if _, err := rand.Read(nonceBytes); err != nil {
		return "", "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt
	plaintextBytes := []byte(plaintext)
	ciphertext := aesGCM.Seal(nil, nonceBytes, plaintextBytes, nil)

	// Encode as base64 for storage
	encryptedContent = base64.StdEncoding.EncodeToString(ciphertext)
	nonce = base64.StdEncoding.EncodeToString(nonceBytes)

	return encryptedContent, nonce, nil
}

// DecryptGeneric decrypts any encrypted data using a data type and ID
func DecryptGeneric(dataType DataType, id int, encryptedContent, nonce string, masterKey []byte) (plaintext string, err error) {
	// Derive key for this data type and ID
	key, err := deriveKey(dataType, id, masterKey)
	if err != nil {
		return "", fmt.Errorf("failed to derive key: %w", err)
	}

	// Decode from base64
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedContent)
	if err != nil {
		return "", fmt.Errorf("invalid encrypted content: %w", err)
	}

	nonceBytes, err := base64.StdEncoding.DecodeString(nonce)
	if err != nil {
		return "", fmt.Errorf("invalid nonce: %w", err)
	}

	// Create AES-GCM cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Decrypt
	plaintextBytes, err := aesGCM.Open(nil, nonceBytes, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decryption failed: %w", err)
	}

	return string(plaintextBytes), nil
}

// deriveKey derives an encryption key for a specific data type and ID
// Uses HKDF-SHA256 to derive key from data type, ID, and master key
// Keys are derived on-the-fly, never stored
func deriveKey(dataType DataType, id int, masterKey []byte) ([]byte, error) {
	if len(masterKey) != KeySize {
		return nil, fmt.Errorf("invalid master key size: expected %d, got %d", KeySize, len(masterKey))
	}

	// Convert ID to bytes for HKDF salt
	idBytes := []byte(fmt.Sprintf("%s-%d", dataType, id))

	// Derive key using HKDF-SHA256 with data type as context
	hkdf := hkdf.New(sha256.New, masterKey, idBytes, []byte(fmt.Sprintf("%s-encryption-v1", dataType)))
	
	key := make([]byte, KeySize)
	if _, err := hkdf.Read(key); err != nil {
		return nil, fmt.Errorf("failed to derive key: %w", err)
	}

	return key, nil
}

// GetMasterKey returns the master encryption key from config
func GetMasterKey() []byte {
	return config.MessageEncryptionKey
}

// VerifyEncryptedData verifies that encrypted data is valid
func VerifyEncryptedData(encryptedContent, nonce string) error {
	if encryptedContent == "" {
		return fmt.Errorf("encrypted content is empty")
	}
	if nonce == "" {
		return fmt.Errorf("nonce is empty")
	}
	
	// Verify base64 encoding
	if _, err := base64.StdEncoding.DecodeString(encryptedContent); err != nil {
		return fmt.Errorf("invalid encrypted content encoding: %w", err)
	}
	if _, err := base64.StdEncoding.DecodeString(nonce); err != nil {
		return fmt.Errorf("invalid nonce encoding: %w", err)
	}
	
	return nil
}
```

#### `messaging/encryption.go` - Message-Specific Wrapper

```go
package messaging

import (
	"github.com/parts-pile/site/encryption"
)

const (
	EncryptionVersion1 = encryption.EncryptionVersion1
)

// EncryptMessage encrypts a message for a conversation (convenience wrapper)
// Returns encrypted content (base64) and nonce (base64)
func EncryptMessage(conversationID int, plaintext string) (encryptedContent, nonce string, err error) {
	masterKey := encryption.GetMasterKey()
	return encryption.EncryptGeneric(encryption.DataTypeMessage, conversationID, plaintext, masterKey)
}

// DecryptMessage decrypts an encrypted message for a conversation (convenience wrapper)
func DecryptMessage(conversationID int, encryptedContent, nonce string) (plaintext string, err error) {
	masterKey := encryption.GetMasterKey()
	return encryption.DecryptGeneric(encryption.DataTypeMessage, conversationID, encryptedContent, nonce, masterKey)
}

// VerifyEncryptedMessage verifies that an encrypted message is valid (convenience wrapper)
func VerifyEncryptedMessage(encryptedContent, nonce string) error {
	return encryption.VerifyEncryptedData(encryptedContent, nonce)
}
```

### Part 6: Update Message Functions

#### Updated `messaging/messaging.go` - Encrypt/Decrypt Messages

```go
package messaging

import (
	"database/sql"
	"fmt"
	"time"
	
	"github.com/parts-pile/site/db"
	"github.com/parts-pile/site/logger"
	"github.com/parts-pile/site/encryption"
)

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
		INSERT INTO Message (conversation_id, sender_id, encrypted_content, nonce, encryption_version, created_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		RETURNING id
	`, conversationID, senderID, encryptedContent, nonce, EncryptionVersion1).Scan(&msgID)
	if err != nil {
		return 0, false, err
	}
	messageID = int(msgID)

	// Get conversation to check notification state
	// Note: We need to query within transaction to get current state
	var user1ID, user2ID int
	var user1FirstMsgNotified, user2FirstMsgNotified bool
	err = tx.QueryRow(`
		SELECT user1_id, user2_id, user1_first_msg_notified, user2_first_msg_notified
		FROM Conversation
		WHERE id = ?
	`, conversationID).Scan(&user1ID, &user2ID, &user1FirstMsgNotified, &user2FirstMsgNotified)
	if err != nil {
		return 0, false, err
	}

	// Determine if this is sender's first message
	var isFirstMessage bool
	var notifyField string
	if user1ID == senderID {
		isFirstMessage = !user1FirstMsgNotified
		notifyField = "user1_first_msg_notified"
	} else {
		isFirstMessage = !user2FirstMsgNotified
		notifyField = "user2_first_msg_notified"
	}

	// Mark first message as notified if this is the first
	if isFirstMessage {
		_, err = tx.Exec(fmt.Sprintf(`UPDATE Conversation SET %s = 1 WHERE id = ?`, notifyField), conversationID)
		if err != nil {
			return 0, false, err
		}
	}

	// Mark conversation as unread for recipient
	if user1ID == senderID {
		_, err = tx.Exec(`UPDATE Conversation SET user2_read = FALSE WHERE id = ?`, conversationID)
	} else {
		_, err = tx.Exec(`UPDATE Conversation SET user1_read = FALSE WHERE id = ?`, conversationID)
	}
	if err != nil {
		return 0, false, err
	}

	// Update conversation timestamp
	_, err = tx.Exec(`UPDATE Conversation SET updated_at = CURRENT_TIMESTAMP WHERE id = ?`, conversationID)
	if err != nil {
		return 0, false, err
	}

	if err := tx.Commit(); err != nil {
		return 0, false, err
	}

	// Return whether notification should be sent (only for first message)
	return messageID, isFirstMessage, nil
}

// GetMessages retrieves messages for a conversation (with decryption)
func GetMessages(conversationID int) ([]Message, error) {
	rows, err := db.Query(`
		SELECT id, conversation_id, sender_id, encrypted_content, nonce, encryption_version, 
		       created_at, read_at
		FROM Message
		WHERE conversation_id = ?
		ORDER BY created_at ASC
	`, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		var encryptedContent, nonce string
		var encryptionVersion int64
		var createdAt, readAt string

		err := rows.Scan(&msg.ID, &msg.ConversationID, &msg.SenderID, 
			&encryptedContent, &nonce, &encryptionVersion, &createdAt, &readAt)
		if err != nil {
			return nil, err
		}

		// Decrypt message
		plaintext, err := DecryptMessage(conversationID, encryptedContent, nonce)
		if err != nil {
			// Log error but continue (might be corrupted data)
			logger.Error("Failed to decrypt message", "error", err, "message_id", msg.ID)
			msg.Content = "[Unable to decrypt message]"
		} else {
			msg.Content = plaintext
		}

		msg.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
		if readAt != "" {
			readAtTime, _ := time.Parse(time.RFC3339Nano, readAt)
			msg.ReadAt = &readAtTime
		}

		messages = append(messages, msg)
	}

	return messages, nil
}
```

### Part 7: Notification Handlers

#### Updated `handlers/messaging.go` - Notification Logic

```go
// HandleSendMessage handles sending a new message
func HandleSendMessage(c *fiber.Ctx) error {
	// ... existing validation ...

	// Add message (with encryption)
	messageID, shouldNotify, err := messaging.AddMessage(conversationID, userID, content)
	if err != nil {
		return render(c, ui.ErrorPage(500, "Failed to send message", userID, userName, csrfToken))
	}

	// Determine recipient ID
	recipientID := conversation.User1ID
	if userID == conversation.User1ID {
		recipientID = conversation.User2ID
	}

	// Send notification ONLY if this is the first message from sender
	if shouldNotify {
		notificationService, err := notification.NewNotificationService()
		if err != nil {
			logger.Error("Failed to create notification service", "error", err)
		} else {
			go func() {
				// Server can decrypt for notification (server-side encryption)
				// Get encrypted message from database to decrypt
				var encryptedContent, nonce string
				err := db.QueryRow(`
					SELECT encrypted_content, nonce
					FROM Message
					WHERE id = ?
				`, messageID).Scan(&encryptedContent, &nonce)
				if err != nil {
					logger.Error("Failed to get message for notification", "error", err)
					return
				}

				plaintext, err := messaging.DecryptMessage(conversationID, encryptedContent, nonce)
				if err != nil {
					logger.Error("Failed to decrypt for notification", "error", err)
					plaintext = "[New message]" // Fallback
				}
				err = notificationService.NotifyNewMessage(conversationID, userID, recipientID, plaintext)
				if err != nil {
					logger.Error("Failed to send notification", "error", err)
				}
			}()
		}
	}

	// ... rest of handler ...
}

// HandlePing handles ping requests
func HandlePing(c *fiber.Ctx) error {
	userID := local.GetUserID(c)
	conversationID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).SendString("Invalid conversation ID")
	}

	// Verify user is part of conversation
	conv, err := messaging.GetConversationWithDetails(conversationID)
	if err != nil {
		return c.Status(404).SendString("Conversation not found")
	}

	if conv.User1ID != userID && conv.User2ID != userID {
		return c.Status(403).SendString("Access denied")
	}

	// Check if ping is already active (prevent abuse)
	if conv.PingedByUserID != nil && *conv.PingedByUserID == userID {
		return c.Status(400).SendString("Ping already sent. Wait for recipient to read conversation.")
	}

	// Send ping
	err = messaging.HandlePing(conversationID, userID)
	if err != nil {
		return c.Status(500).SendString("Failed to send ping")
	}

	// Return success (HTMX can update UI to show ping sent)
	return c.SendString("Ping sent")
}
```

#### Updated `messaging/messaging.go` - Ping and Read Logic

```go
// Conversation struct needs new fields (add to existing struct)
// User1FirstMsgNotified   bool      `json:"user1_first_msg_notified" db:"user1_first_msg_notified"`
// User2FirstMsgNotified   bool      `json:"user2_first_msg_notified" db:"user2_first_msg_notified"`
// PingedByUserID          *int      `json:"pinged_by_user_id,omitempty" db:"pinged_by_user_id"`
// PingedAt                *time.Time `json:"pinged_at,omitempty" db:"pinged_at"`

// HandlePing sends a ping notification to the other participant
func HandlePing(conversationID, userID int) error {
	conv, err := GetConversationByID(conversationID)
	if err != nil {
		return err
	}

	// Determine recipient
	var recipientID int
	if conv.User1ID == userID {
		recipientID = conv.User2ID
	} else {
		recipientID = conv.User1ID
	}

	// Update ping state
	_, err = db.Exec(`
		UPDATE Conversation
		SET pinged_by_user_id = ?, pinged_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, userID, conversationID)
	if err != nil {
		return err
	}

	// Send ping notification (async)
	go func() {
		notificationService, err := notification.NewNotificationService()
		if err != nil {
			logger.Error("Failed to create notification service for ping", "error", err)
			return
		}

		err = notificationService.NotifyPing(conversationID, userID, recipientID)
		if err != nil {
			logger.Error("Failed to send ping notification", "error", err)
		}
	}()

	return nil
}

// MarkConversationAsRead resets ping state when conversation is read
func MarkConversationAsRead(conversationID, userID int) error {
	// Check if this user was pinged (get current state)
	var pingedByUserID sql.NullInt64
	err := db.QueryRow(`
		SELECT pinged_by_user_id
		FROM Conversation
		WHERE id = ?
	`, conversationID).Scan(&pingedByUserID)
	if err != nil {
		return err
	}

	// If other user pinged this user, reset ping state
	if pingedByUserID.Valid && int(pingedByUserID.Int64) != userID {
		_, err = db.Exec(`
			UPDATE Conversation
			SET pinged_by_user_id = NULL, pinged_at = NULL
			WHERE id = ?
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
		_, err = db.Exec(`UPDATE Conversation SET user1_read = TRUE WHERE id = ?`, conversationID)
	} else {
		_, err = db.Exec(`UPDATE Conversation SET user2_read = TRUE WHERE id = ?`, conversationID)
	}
	return err
}
```

#### Updated `notification/notification.go` - Ping Notifications

```go
// NotifyPing sends a ping notification
func (n *NotificationService) NotifyPing(conversationID int, senderID, recipientID int) error {
	conversation, err := messaging.GetConversationByID(conversationID)
	if err != nil {
		return fmt.Errorf("failed to get conversation: %w", err)
	}

	sender, _, found := user.GetUserByID(senderID)
	if !found {
		return fmt.Errorf("sender not found")
	}

	recipient, _, found := user.GetUserByID(recipientID)
	if !found {
		return fmt.Errorf("recipient not found")
	}

	ad, err := getAdDetails(conversation.AdID)
	if err != nil {
		return fmt.Errorf("failed to get ad details: %w", err)
	}

	// Ping notification message
	message := fmt.Sprintf("%s pinged you about '%s'. View at: %s/messages/%d",
		sender.Name, ad.Title, config.BaseURL, conversationID)

	switch recipient.NotificationMethod {
	case user.NotificationMethodSMS:
		return sms.SendMessage(recipient.Phone, message)
	case user.NotificationMethodEmail:
		if recipient.EmailAddress == nil {
			return fmt.Errorf("recipient has email notifications enabled but no email address")
		}
		return n.emailService.SendPingEmail(*recipient.EmailAddress, sender.Name, ad.Title, conversationID)
	default:
		return nil
	}
}

// NotifySummary sends a summary notification of unread conversations
func (n *NotificationService) NotifySummary(userID int, conversations []messaging.Conversation) error {
	recipient, _, found := user.GetUserByID(userID)
	if !found {
		return fmt.Errorf("recipient not found")
	}

	if len(conversations) == 0 {
		return nil // No conversations to summarize
	}

	message := fmt.Sprintf("You have new messages in %d conversation(s). View at: %s/messages",
		len(conversations), config.BaseURL)

	switch recipient.NotificationMethod {
	case user.NotificationMethodSMS:
		return sms.SendMessage(recipient.Phone, message)
	case user.NotificationMethodEmail:
		if recipient.EmailAddress == nil {
			return fmt.Errorf("recipient has email notifications enabled but no email address")
		}
		return n.emailService.SendSummaryEmail(*recipient.EmailAddress, len(conversations))
	default:
		return nil
	}
}
```

### Part 8: Summary Notifications

#### Summary Notification Job

```go
// SendSummaryNotifications sends summary notifications to users who opted in
func SendSummaryNotifications() error {
	// Get all users with summary notifications enabled
	rows, err := db.Query(`
		SELECT id, name, notification_method, phone, email_address, summary_notifications_frequency
		FROM User
		WHERE summary_notifications_enabled = 1
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var u struct {
			ID                      int
			Name                    string
			NotificationMethod      string
			Phone                   string
			EmailAddress            *string
			SummaryNotificationsFrequency string
		}
		// ... scan user ...

		// Get conversations with unread messages (beyond first message)
		conversations, err := getConversationsWithUnreadMessages(u.ID)
		if err != nil {
			continue
		}

		if len(conversations) == 0 {
			continue // No unread messages
		}

		// Send summary notification
		notificationService, err := notification.NewNotificationService()
		if err != nil {
			continue
		}

		err = notificationService.NotifySummary(u.ID, conversations)
		if err != nil {
			logger.Error("Failed to send summary notification", "userID", u.ID, "error", err)
		}
	}

	return nil
}

// getConversationsWithUnreadMessages gets conversations with unread messages
// (excluding first messages from each participant - those already notified)
func getConversationsWithUnreadMessages(userID int) ([]messaging.Conversation, error) {
	// Get conversations where user has unread messages
	// AND where first messages from both participants have already been sent
	// (so we only include conversations with messages beyond the first exchange)
	rows, err := db.Query(`
		SELECT DISTINCT c.id
		FROM Conversation c
		JOIN Message m ON m.conversation_id = c.id
		WHERE (c.user1_id = ? OR c.user2_id = ?)
		AND m.read_at IS NULL
		AND m.sender_id != ?
		AND c.user1_first_msg_notified = 1
		AND c.user2_first_msg_notified = 1
		ORDER BY c.updated_at DESC
	`, userID, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conversationIDs []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			continue
		}
		conversationIDs = append(conversationIDs, id)
	}

	// Get full conversation details
	var conversations []messaging.Conversation
	for _, id := range conversationIDs {
		conv, err := messaging.GetConversationByID(id)
		if err != nil {
			continue
		}
		conversations = append(conversations, conv)
	}

	return conversations, nil
}

// Run as a scheduled job (daily, weekly, etc.)
// Can be added to main.go or run as separate cron job
```

### Part 9: UI Updates

#### Update `ui/messaging.go` - Add Ping Button

```go
// Add ping button to conversation UI
// Update ExpandedConversation to include ping button
func PingButton(conversationID int, currentUserID int, pingedByUserID *int) g.Node {
	// Check if ping is already active (prevent showing button if user already pinged)
	if pingedByUserID != nil && *pingedByUserID == currentUserID {
		// Ping already sent - show disabled state
		return Button(
			Type("button"),
			Class("px-3 py-1 text-sm bg-gray-300 text-gray-600 rounded"),
			Disabled(),
			g.Text("Ping Sent"),
		)
	}

	// Show active ping button
	return Form(
		Method("POST"),
		Action(fmt.Sprintf("/messages/%d/ping", conversationID)),
		hx.Post(fmt.Sprintf("/messages/%d/ping", conversationID)),
		hx.Target(fmt.Sprintf("#conversation-%d", conversationID)),
		hx.Swap("outerHTML"),
		Button(
			Type("submit"),
			Class("px-3 py-1 text-sm bg-blue-500 text-white rounded hover:bg-blue-600"),
			g.Text("Ping"),
		),
	)
}

// Update ExpandedConversation to include ping button
// Add PingButton to the header section alongside the close button
```

#### Update `ui/setting.go` - Add Summary Notification Settings

```go
// Add summary notification settings to user settings page
func SummaryNotificationSettings(userID int, enabled bool, frequency string) gomponents.Node {
	return Div(
		Class("mb-4"),
		H3(Class("text-lg font-semibold mb-2"), Text("Summary Notifications")),
		P(Class("text-sm text-gray-600 mb-4"),
			Text("Receive periodic summaries of new messages in your conversations."),
		),
		Form(
			Method("POST"),
			Action("/settings/notifications"),
			HxPost("/settings/notifications"),
			HxTarget("#settings-content"),
			HxSwap("outerHTML"),
			Div(
				Class("mb-4"),
				Label(
					Class("flex items-center"),
					Input(
						Type("checkbox"),
						Name("summary_notifications_enabled"),
						Value("1"),
						Checked(enabled),
					),
					Span(Class("ml-2"), Text("Enable summary notifications")),
				),
			),
			Div(
				Class("mb-4"),
				Label(
					Class("block mb-2"),
					Text("Frequency:"),
				),
				Select(
					Name("summary_notifications_frequency"),
					Option(Value("daily"), Selected(frequency == "daily"), Text("Daily")),
					Option(Value("weekly"), Selected(frequency == "weekly"), Text("Weekly")),
				),
			),
			Button(
				Type("submit"),
				Class("px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600"),
				Text("Save"),
			),
		),
	)
}
```

### Part 10: Update schema.sql

Since you'll rerun `rebuild_db`, just update the `Message` table definition in `schema.sql`:

```sql
CREATE TABLE Message (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    conversation_id INTEGER NOT NULL REFERENCES Conversation(id),
    sender_id INTEGER NOT NULL REFERENCES User(id),
    encrypted_content BLOB NOT NULL,
    nonce BLOB NOT NULL,
    encryption_version INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    read_at DATETIME
);
CREATE INDEX idx_message_conversation_id ON Message(conversation_id);
CREATE INDEX idx_message_sender_id ON Message(sender_id);
CREATE INDEX idx_message_created_at ON Message(created_at);
CREATE INDEX idx_message_read_at ON Message(read_at);
```

**No migration needed** - just update schema and rerun `rebuild_db`.

## Security Considerations

### What This Protects Against

✅ **Database Breaches:** Messages encrypted at rest - attacker with database access cannot read messages without master key
✅ **Backup Theft:** Encrypted backups require master key to decrypt
✅ **SQL Injection:** Even if attacker can read database, messages are encrypted

### What This Doesn't Protect Against

❌ **Server Compromise:** Server can decrypt messages (has master key)
❌ **Server Logs:** If server logs plaintext, it's visible
❌ **Server Admins:** Anyone with server access can decrypt
❌ **True E2EE:** Not end-to-end - server sees plaintext

### Security Best Practices

1. **Master Key Management:**
   - Store in environment variable (not in code)
   - Use strong random key (32 bytes)
   - Rotate periodically (requires re-encrypting all messages)
   - Never commit to version control

2. **Key Derivation:**
   - Use HKDF-SHA256 (cryptographically secure)
   - Per-conversation keys (isolates conversations)
   - Keys never stored (derived on-the-fly)

3. **Nonce Management:**
   - Random nonce per message (prevents replay attacks)
   - Never reuse nonces
   - Store with encrypted content

4. **Access Control:**
   - Limit server access (who can decrypt)
   - Audit server access logs
   - Use HTTPS for transmission

## Comparison with Option 2 (True E2EE)

| Feature | Option 1 (Server-Side) | Option 2 (True E2EE) |
|---------|----------------------|---------------------|
| **Protection against DB breach** | ✅ Yes | ✅ Yes |
| **Protection against server compromise** | ❌ No | ✅ Yes |
| **Works with async messaging** | ✅ Yes (immediate) | ❌ No (requires recipient setup) |
| **JavaScript required** | ❌ No | ✅ Yes |
| **Complexity** | Low (2-3 days) | High (2-3 weeks) |
| **Message preview in notifications** | ✅ Yes | ❌ No |
| **Server can read messages** | ✅ Yes | ❌ No |
| **Multi-device support** | ✅ Simple | ⚠️ Complex |

## Why Option 1 is Better for Your Use Case

1. **Async Messaging:** Works immediately - no waiting for recipient to set up encryption
2. **No JavaScript:** Maintains your HTMX/no-JS preference
3. **Simpler:** 2-3 days vs 2-3 weeks implementation
4. **Message Previews:** Server can include message content in notifications
5. **Database Protection:** Protects against the most common threat (database breaches)

## Implementation Steps

1. **Add encryption key to config** (1 hour)
   - Add `MESSAGE_ENCRYPTION_KEY` to environment
   - Update `config/config.go`

2. **Create encryption package** (3 hours)
   - Create `encryption/encryption.go` with generic functions
   - Create `messaging/encryption.go` with message wrappers
   - Test encryption/decryption

3. **Update database schema** (30 minutes)
   - Update `schema.sql` with new Message table structure
   - Rerun `rebuild_db` to apply changes

4. **Update message functions** (4 hours)
   - Modify `AddMessage()` to encrypt
   - Modify `GetMessages()` to decrypt
   - Update all message retrieval paths

5. **Testing** (4 hours)
   - Test sending messages
   - Test receiving messages
   - Test real-time updates (SSE)
   - Test notifications
   - Verify encryption/decryption

**Total Estimated Time:** 2-3 days (no migration needed, includes notification features)

## Testing

```go
// Test encryption/decryption
func TestEncryption(t *testing.T) {
	conversationID := 123
	plaintext := "Hello, how much for the part?"

	// Encrypt
	encrypted, nonce, err := messaging.EncryptMessage(conversationID, plaintext)
	require.NoError(t, err)
	require.NotEmpty(t, encrypted)
	require.NotEmpty(t, nonce)

	// Decrypt
	decrypted, err := messaging.DecryptMessage(conversationID, encrypted, nonce)
	require.NoError(t, err)
	require.Equal(t, plaintext, decrypted)

	// Verify different conversations have different keys
	encrypted2, _, _ := messaging.EncryptMessage(456, plaintext)
	require.NotEqual(t, encrypted, encrypted2) // Different keys = different ciphertext
}
```

## Production Deployment

1. **Generate master key:**
   ```bash
   openssl rand -base64 32
   ```

2. **Set environment variable:**
   ```bash
   export MESSAGE_ENCRYPTION_KEY="<generated-key>"
   ```

3. **Update schema.sql** with new Message table structure

4. **Rerun rebuild_db:**
   ```bash
   go run cmd/rebuild_db/main.go
   ```

5. **Verify encryption:**
   - Send test message - verify it's encrypted in database
   - Receive test message - verify it decrypts correctly
   - Check database - messages should have `encrypted_content` and `nonce` populated

6. **Monitor:**
   - Check logs for decryption errors
   - Monitor performance (encryption adds ~1ms per message)

## Rollback Plan

If issues arise:

1. **Add feature flag** to enable/disable encryption
2. **Fallback to plaintext** if decryption fails (for debugging)
3. **Revert schema.sql** and rerun `rebuild_db` if needed

## Future Enhancements

1. **Key Rotation:** Implement key rotation without downtime
2. **Forward Secrecy:** Rotate keys periodically (more complex)
3. **Hybrid Approach:** Start with server-side, add E2EE for active conversations
4. **Performance:** Cache derived keys (with expiration)

---

## Summary

Option 1 (Server-Side Encryption) provides:
- ✅ Protection against database breaches
- ✅ Works with async messaging (no waiting)
- ✅ No JavaScript required
- ✅ Simple implementation (2-3 days)
- ✅ Message previews in notifications
- ⚠️ Server can decrypt (not true E2EE, but encrypted at rest)

This is the recommended approach for your use case.

