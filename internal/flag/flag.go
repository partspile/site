package flag

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/db"
	"github.com/partspile/site/internal/encryption"
	"github.com/partspile/site/internal/journal"
)

var ErrFlagNotFound = errors.New("flag not found")
var ErrMaxFlagsReached = errors.New("user has reached maximum outstanding flags")
var ErrFlagAlreadyRaised = errors.New("a flag has already been raised on this conversation")

// RaiseFlag raises a flag on a conversation, making it public.
// If the inquirer raises it, flag_raiser_id is the inquirer (bound to the ad).
// If the owner raises it, flag_raiser_id is the owner (bound to the inquirer).
func RaiseFlag(userID, conversationID int, reason string) error {
	if !ValidReason(reason) {
		return ErrInvalidReason
	}

	conv, err := getConversationForFlag(conversationID)
	if err != nil {
		return fmt.Errorf("failed to get conversation: %w", err)
	}

	if conv.OwnerID != userID && conv.InquirerID != userID {
		return fmt.Errorf("only conversation participants can raise flags")
	}

	count, err := GetUserFlagCount(userID)
	if err != nil {
		return fmt.Errorf("failed to check flag count: %w", err)
	}
	if count >= config.MaxOutstandingFlags {
		return ErrMaxFlagsReached
	}

	flagCount, err := GetFlagCountForConversation(conversationID)
	if err != nil {
		return fmt.Errorf("failed to check existing flags: %w", err)
	}
	if flagCount > 0 {
		return ErrFlagAlreadyRaised
	}

	now := time.Now().UTC()
	plain, err := encryption.Open(conversationID, conv.Journal, config.DBEncryptionKey)
	if err != nil {
		return fmt.Errorf("open journal: %w", err)
	}
	newJournal := journal.AppendFlag(plain, journal.FlagRaised, userID,
		reason, now, time.UTC)
	sealed, err := encryption.Seal(conversationID, newJournal, config.DBEncryptionKey)
	if err != nil {
		return fmt.Errorf("seal journal: %w", err)
	}

	_, err = db.Exec(`
		UPDATE conversations
		SET flag_raiser_id = $1, flag_raised_at = $2,
			journal = $3, updated_at = $2
		WHERE id = $4
	`, userID, now, sealed, conversationID)
	if err != nil {
		return fmt.Errorf("failed to throw flag: %w", err)
	}

	if userID == conv.InquirerID {
		if err := SyncAdFlagCount(conv.AdID); err != nil {
			return fmt.Errorf("sync ad flag count: %w", err)
		}
	}

	return nil
}

func getConversationForFlag(conversationID int) (struct {
	OwnerID    int
	InquirerID int
	AdID       int
	Journal    string
}, error) {
	var conv struct {
		OwnerID    int
		InquirerID int
		AdID       int
		Journal    string
	}
	err := db.QueryRow(`
		SELECT owner_id, inquirer_id, ad_id, journal
		FROM conversations
		WHERE id = $1
	`, conversationID).Scan(
		&conv.OwnerID, &conv.InquirerID, &conv.AdID, &conv.Journal,
	)
	if err != nil {
		return conv, fmt.Errorf("failed to get conversation: %w", err)
	}
	plain, err := encryption.Open(conversationID, conv.Journal, config.DBEncryptionKey)
	if err != nil {
		return conv, fmt.Errorf("open journal: %w", err)
	}
	conv.Journal = plain
	return conv, nil
}

// LowerFlag removes a flag from a conversation
func LowerFlag(userID, conversationID int) error {
	var flagRaiserID sql.NullInt64
	var inquirerID, adID int
	var j string
	err := db.QueryRow(`
		SELECT flag_raiser_id, inquirer_id, ad_id, journal
		FROM conversations
		WHERE id = $1
	`, conversationID).Scan(&flagRaiserID, &inquirerID, &adID, &j)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrFlagNotFound
		}
		return fmt.Errorf("failed to get conversation: %w", err)
	}

	if !flagRaiserID.Valid {
		return ErrFlagNotFound
	}

	if int(flagRaiserID.Int64) != userID {
		return fmt.Errorf("only the person who raised the flag can lower it")
	}

	adBound := int(flagRaiserID.Int64) == inquirerID

	now := time.Now().UTC()
	plain, err := encryption.Open(conversationID, j, config.DBEncryptionKey)
	if err != nil {
		return fmt.Errorf("open journal: %w", err)
	}
	newJournal := journal.AppendFlag(plain, journal.FlagLowered, userID, "",
		now, time.UTC)
	sealed, err := encryption.Seal(conversationID, newJournal, config.DBEncryptionKey)
	if err != nil {
		return fmt.Errorf("seal journal: %w", err)
	}

	_, err = db.Exec(`
		UPDATE conversations
		SET flag_raiser_id = NULL, flag_raised_at = NULL,
			journal = $1, updated_at = $2
		WHERE id = $3
	`, sealed, now, conversationID)
	if err != nil {
		return fmt.Errorf("failed to remove flag: %w", err)
	}

	if adBound {
		if err := SyncAdFlagCount(adID); err != nil {
			return fmt.Errorf("sync ad flag count: %w", err)
		}
	}

	return nil
}

// SyncAdFlagCount sets ads.flag_count from ad-bound flags on conversations.
func SyncAdFlagCount(adID int) error {
	_, err := db.Exec(`
		UPDATE ads
		SET flag_count = (
			SELECT COUNT(*)::int
			FROM conversations c
			WHERE c.ad_id = $1
			  AND c.flag_raiser_id IS NOT NULL
			  AND c.flag_raiser_id = c.inquirer_id
		)
		WHERE id = $1
	`, adID)
	if err != nil {
		return fmt.Errorf("update ads.flag_count: %w", err)
	}
	return nil
}

// GetPublicConversationsForAd returns public conversation IDs for an ad
// Only returns conversations with flags bound to the ad (flag_raiser_id = inquirer_id)
func GetPublicConversationsForAd(adID int) ([]int, error) {
	query := `
		SELECT c.id
		FROM conversations c
		WHERE c.ad_id = $1 AND c.flag_raiser_id IS NOT NULL AND c.flag_raiser_id = c.inquirer_id
		ORDER BY c.updated_at DESC
	`
	var conversationIDs []int
	err := db.Select(&conversationIDs, query, adID)
	if err != nil {
		return nil, fmt.Errorf("failed to get public conversations for ad: %w", err)
	}
	return conversationIDs, nil
}

// GetPublicConversationIDByOrdinal returns the conversation ID at the given ordinal position (0-based)
// for public conversations with flags bound to the ad (flag_raiser_id = inquirer_id), ordered by latest activity DESC
func GetPublicConversationIDByOrdinal(adID int, ordinal int) (int, error) {
	query := `
		SELECT c.id
		FROM conversations c
		WHERE c.ad_id = $1 AND c.flag_raiser_id IS NOT NULL AND c.flag_raiser_id = c.inquirer_id
		ORDER BY c.updated_at DESC
		LIMIT 1 OFFSET $2
	`
	var conversationID int
	err := db.QueryRow(query, adID, ordinal).Scan(&conversationID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("conversation not found at ordinal %d for ad %d", ordinal, adID)
		}
		return 0, fmt.Errorf("failed to get public conversation by ordinal: %w", err)
	}
	return conversationID, nil
}

// GetUserFlagCount returns the count of outstanding flags thrown by a user
func GetUserFlagCount(userID int) (int, error) {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*)
		FROM conversations
		WHERE flag_raiser_id = $1
	`, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get user flag count: %w", err)
	}
	return count, nil
}

// GetFlagCountForConversation returns whether a flag exists for a conversation
func GetFlagCountForConversation(conversationID int) (int, error) {
	var flagRaiserID sql.NullInt64
	err := db.QueryRow(`
		SELECT flag_raiser_id
		FROM conversations
		WHERE id = $1
	`, conversationID).Scan(&flagRaiserID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("conversation not found: %w", err)
		}
		return 0, fmt.Errorf("failed to get flag count for conversation: %w", err)
	}
	if !flagRaiserID.Valid {
		return 0, nil
	}
	return 1, nil
}

// HasUserRaisedFlag checks if a user has thrown a flag at a conversation
func HasUserRaisedFlag(userID, conversationID int) (bool, error) {
	var flagRaiserID sql.NullInt64
	err := db.QueryRow(`
		SELECT flag_raiser_id
		FROM conversations
		WHERE id = $1
	`, conversationID).Scan(&flagRaiserID)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, fmt.Errorf("conversation not found: %w", err)
		}
		return false, fmt.Errorf("failed to check if user threw flag: %w", err)
	}
	return flagRaiserID.Valid && int(flagRaiserID.Int64) == userID, nil
}

// GetConversationIDsForUserFlags returns conversation IDs for flags bound to a user
// (where flag_raiser_id = owner_id, meaning owner threw flag at inquirer)
func GetConversationIDsForUserFlags(userID int) ([]int, error) {
	query := `
		SELECT id
		FROM conversations
		WHERE flag_raiser_id IS NOT NULL AND flag_raiser_id = owner_id AND inquirer_id = $1
		ORDER BY flag_raised_at DESC
	`
	var conversationIDs []int
	err := db.Select(&conversationIDs, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation IDs for user flags: %w", err)
	}
	return conversationIDs, nil
}

// GetFlagCountForUser returns the count of flags bound to a user
// (flags where owner threw at this user)
func GetFlagCountForUser(userID int) (int, error) {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*)
		FROM conversations
		WHERE flag_raiser_id IS NOT NULL AND flag_raiser_id = owner_id AND inquirer_id = $1
	`, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get flag count for user: %w", err)
	}
	return count, nil
}

// GetConversationIDForUserFlagByOrdinal returns the conversation ID at the given ordinal position (0-based)
// for flags bound to a user (where flag_raiser_id = owner_id AND inquirer_id = userID), ordered by flag_raised_at DESC
func GetConversationIDForUserFlagByOrdinal(userID int, ordinal int) (int, error) {
	query := `
		SELECT id
		FROM conversations
		WHERE flag_raiser_id IS NOT NULL AND flag_raiser_id = owner_id AND inquirer_id = $1
		ORDER BY flag_raised_at DESC
		LIMIT 1 OFFSET $2
	`
	var conversationID int
	err := db.QueryRow(query, userID, ordinal).Scan(&conversationID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("conversation not found at ordinal %d for user %d", ordinal, userID)
		}
		return 0, fmt.Errorf("failed to get conversation ID for user flag by ordinal: %w", err)
	}
	return conversationID, nil
}

// LowerActiveForAd clears active flags on all conversations for an ad.
func LowerActiveForAd(adID int) error {
	rows, err := db.Query(`
		SELECT id, flag_raiser_id, journal
		FROM conversations
		WHERE ad_id = $1 AND flag_raiser_id IS NOT NULL
	`, adID)
	if err != nil {
		return fmt.Errorf("failed to list flags for ad: %w", err)
	}
	defer rows.Close()
	return lowerRows(rows)
}

// LowerActiveForUser clears flags thrown by or bound to the user.
func LowerActiveForUser(userID int) error {
	rows, err := db.Query(`
		SELECT id, flag_raiser_id, journal
		FROM conversations
		WHERE flag_raiser_id IS NOT NULL
		  AND (
			flag_raiser_id = $1
			OR (inquirer_id = $1 AND flag_raiser_id = owner_id)
		  )
	`, userID)
	if err != nil {
		return fmt.Errorf("failed to list flags for user: %w", err)
	}
	defer rows.Close()
	return lowerRows(rows)
}

func lowerRows(rows *sql.Rows) error {
	type flagRow struct {
		id      int
		thrower int
		journal string
	}
	var list []flagRow
	for rows.Next() {
		var r flagRow
		if err := rows.Scan(&r.id, &r.thrower, &r.journal); err != nil {
			return fmt.Errorf("failed to scan flag row: %w", err)
		}
		list = append(list, r)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, r := range list {
		if err := forceLower(r.id, r.thrower, r.journal); err != nil {
			return err
		}
	}
	return nil
}

func forceLower(conversationID, throwerID int, j string) error {
	now := time.Now().UTC()
	plain, err := encryption.Open(conversationID, j, config.DBEncryptionKey)
	if err != nil {
		return fmt.Errorf("open journal: %w", err)
	}
	newJournal := journal.AppendFlag(plain, journal.FlagLowered, throwerID, "",
		now, time.UTC)
	sealed, err := encryption.Seal(conversationID, newJournal, config.DBEncryptionKey)
	if err != nil {
		return fmt.Errorf("seal journal: %w", err)
	}
	_, err = db.Exec(`
		UPDATE conversations
		SET flag_raiser_id = NULL, flag_raised_at = NULL,
			journal = $1, updated_at = $2
		WHERE id = $3 AND flag_raiser_id IS NOT NULL
	`, sealed, now, conversationID)
	if err != nil {
		return fmt.Errorf("failed to force-lower flag: %w", err)
	}
	return nil
}
