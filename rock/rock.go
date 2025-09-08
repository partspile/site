package rock

import (
	"fmt"
	"time"

	"github.com/parts-pile/site/db"
	"github.com/parts-pile/site/messaging"
)

// UserRock represents a user's rock inventory
type UserRock struct {
	ID        int       `json:"id" db:"id"`
	UserID    int       `json:"user_id" db:"user_id"`
	RockCount int       `json:"rock_count" db:"rock_count"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// AdRock represents a rock thrown at an ad
type AdRock struct {
	ID             int        `json:"id" db:"id"`
	AdID           int        `json:"ad_id" db:"ad_id"`
	ThrowerID      int        `json:"thrower_id" db:"thrower_id"`
	ConversationID int        `json:"conversation_id" db:"conversation_id"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	ResolvedAt     *time.Time `json:"resolved_at,omitempty" db:"resolved_at"`
	ResolvedBy     *int       `json:"resolved_by,omitempty" db:"resolved_by"`
	// Runtime fields
	ThrowerName  string                  `json:"thrower_name,omitempty" db:"thrower_name"`
	AdTitle      string                  `json:"ad_title,omitempty" db:"ad_title"`
	Conversation *messaging.Conversation `json:"conversation,omitempty"`
}

// InitializeUserRocks creates a new rock inventory for a user with 3 rocks
func InitializeUserRocks(userID int) error {
	_, err := db.Exec(`INSERT INTO UserRock (user_id, rock_count) VALUES (?, 3)`, userID)
	return err
}

// GetUserRocks retrieves a user's rock inventory
func GetUserRocks(userID int) (UserRock, error) {
	row := db.QueryRow(`SELECT id, user_id, rock_count, created_at, updated_at FROM UserRock WHERE user_id = ?`, userID)
	var ur UserRock
	var createdAt, updatedAt string
	err := row.Scan(&ur.ID, &ur.UserID, &ur.RockCount, &createdAt, &updatedAt)
	if err != nil {
		return UserRock{}, err
	}

	ur.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	ur.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)
	return ur, nil
}

// CanThrowRock checks if a user can throw a rock
func CanThrowRock(userID int) (bool, error) {
	rocks, err := GetUserRocks(userID)
	if err != nil {
		return false, err
	}
	return rocks.RockCount > 0, nil
}

// ThrowRock throws a rock at an ad, creating a conversation and reducing rock count
func ThrowRock(userID, adID int, initialMessage string) error {
	// Check if user can throw rock
	canThrow, err := CanThrowRock(userID)
	if err != nil {
		return err
	}
	if !canThrow {
		return fmt.Errorf("user has no rocks available")
	}

	// Get ad owner
	var adOwnerID int
	err = db.QueryRow(`SELECT user_id FROM Ad WHERE id = ?`, adID).Scan(&adOwnerID)
	if err != nil {
		return err
	}

	// Can't throw rock at your own ad
	if userID == adOwnerID {
		return fmt.Errorf("cannot throw rock at your own ad")
	}

	// Check if rock already exists for this user and ad
	var existingRock int
	err = db.QueryRow(`SELECT id FROM AdRock WHERE thrower_id = ? AND ad_id = ? AND resolved_at IS NULL`, userID, adID).Scan(&existingRock)
	if err == nil {
		return fmt.Errorf("rock already thrown at this ad")
	}

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Create conversation
	convID, err := messaging.CreateConversationWithTx(tx, userID, adOwnerID, adID)
	if err != nil {
		return err
	}

	// Add initial message
	_, err = messaging.AddMessageWithTx(tx, convID, userID, initialMessage)
	if err != nil {
		return err
	}

	// Create rock record
	_, err = tx.Exec(`INSERT INTO AdRock (ad_id, thrower_id, conversation_id) VALUES (?, ?, ?)`, adID, userID, convID)
	if err != nil {
		return err
	}

	// Reduce rock count
	_, err = tx.Exec(`UPDATE UserRock SET rock_count = rock_count - 1, updated_at = CURRENT_TIMESTAMP WHERE user_id = ?`, userID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// GetAdRocks retrieves all rocks for an ad
func GetAdRocks(adID int) ([]AdRock, error) {
	var rocks []AdRock
	err := db.Select(&rocks, `
		SELECT ar.id, ar.ad_id, ar.thrower_id, ar.conversation_id, ar.created_at, ar.resolved_at, ar.resolved_by,
		       u.name as thrower_name, a.title as ad_title
		FROM AdRock ar
		JOIN User u ON ar.thrower_id = u.id
		JOIN Ad a ON ar.ad_id = a.id
		WHERE ar.ad_id = ?
		ORDER BY ar.created_at DESC
	`, adID)
	return rocks, err
}

// GetUserThrownRocks retrieves all rocks thrown by a user
func GetUserThrownRocks(userID int) ([]AdRock, error) {
	var rocks []AdRock
	err := db.Select(&rocks, `
		SELECT ar.id, ar.ad_id, ar.thrower_id, ar.conversation_id, ar.created_at, ar.resolved_at, ar.resolved_by,
		       u.name as thrower_name, a.title as ad_title
		FROM AdRock ar
		JOIN User u ON ar.thrower_id = u.id
		JOIN Ad a ON ar.ad_id = a.id
		WHERE ar.thrower_id = ?
		ORDER BY ar.created_at DESC
	`, userID)
	return rocks, err
}

// ResolveRock resolves a rock dispute and returns the rock to the thrower
func ResolveRock(rockID, resolvedByUserID int) error {
	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Get rock details
	var throwerID int
	err = tx.QueryRow(`SELECT thrower_id FROM AdRock WHERE id = ? AND resolved_at IS NULL`, rockID).Scan(&throwerID)
	if err != nil {
		return err
	}

	// Mark rock as resolved
	_, err = tx.Exec(`UPDATE AdRock SET resolved_at = CURRENT_TIMESTAMP, resolved_by = ? WHERE id = ?`, resolvedByUserID, rockID)
	if err != nil {
		return err
	}

	// Return rock to thrower
	_, err = tx.Exec(`UPDATE UserRock SET rock_count = rock_count + 1, updated_at = CURRENT_TIMESTAMP WHERE user_id = ?`, throwerID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// GetAdRockCount returns the number of unresolved rocks for an ad
func GetAdRockCount(adID int) (int, error) {
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM AdRock WHERE ad_id = ? AND resolved_at IS NULL`, adID).Scan(&count)
	return count, err
}
