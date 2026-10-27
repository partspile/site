package sms

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/partspile/site/internal/db"
)

// Notification represents a queued SMS notification
type Notification struct {
	ID              int
	RecipientUserID int
	ConversationID  int
	CreatedAt       time.Time
	ProcessedAt     *time.Time
	Status          string
}

// QueueEntry represents a queue entry for admin display
type QueueEntry struct {
	ID              int
	RecipientUserID int
	ConversationID  int
	AdTitle         string
	Status          string
	CreatedAt       time.Time
	ProcessedAt     *time.Time
}

// QueueStats represents queue statistics
type QueueStats struct {
	Pending    int
	Processed  int
	Suppressed int
}

// EnqueueNotification adds a notification to the queue
func EnqueueNotification(recipientUserID, conversationID int) error {
	_, err := db.Exec(`
		INSERT INTO sms_notification_queue (recipient_user_id, conversation_id, status, created_at)
		VALUES ($1, $2, 'pending', CURRENT_TIMESTAMP)
	`, recipientUserID, conversationID)
	if err != nil {
		return fmt.Errorf("failed to enqueue notification: %w", err)
	}
	return nil
}

// DequeuePendingNotifications retrieves pending notifications
func DequeuePendingNotifications(limit int) ([]Notification, error) {
	query := `
		SELECT id, recipient_user_id, conversation_id, created_at, processed_at, status
		FROM sms_notification_queue
		WHERE status = 'pending'
		ORDER BY created_at ASC
		LIMIT $1
	`
	var notifications []Notification
	rows, err := db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending notifications: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var n Notification
		var processedAt sql.NullTime
		err := rows.Scan(
			&n.ID,
			&n.RecipientUserID,
			&n.ConversationID,
			&n.CreatedAt,
			&processedAt,
			&n.Status,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan notification: %w", err)
		}
		if processedAt.Valid {
			n.ProcessedAt = &processedAt.Time
		}
		notifications = append(notifications, n)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating notifications: %w", err)
	}

	return notifications, nil
}

// MarkProcessed marks a notification as processed
func MarkProcessed(id int) error {
	_, err := db.Exec(`
		UPDATE sms_notification_queue
		SET status = 'processed', processed_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`, id)
	if err != nil {
		return fmt.Errorf("failed to mark notification as processed: %w", err)
	}
	return nil
}

// MarkSuppressed marks a notification as suppressed
func MarkSuppressed(id int) error {
	_, err := db.Exec(`
		UPDATE sms_notification_queue
		SET status = 'suppressed', processed_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`, id)
	if err != nil {
		return fmt.Errorf("failed to mark notification as suppressed: %w", err)
	}
	return nil
}

// CleanupOldRecords deletes processed/suppressed records older than retention period
func CleanupOldRecords(retentionHours int) error {
	_, err := db.Exec(`
		DELETE FROM sms_notification_queue
		WHERE status IN ('processed', 'suppressed')
		AND processed_at < NOW() - ($1 * INTERVAL '1 hour')
	`, retentionHours)
	if err != nil {
		return fmt.Errorf("failed to cleanup old records: %w", err)
	}
	return nil
}

// GetQueueStats returns queue statistics
func GetQueueStats() (QueueStats, error) {
	var stats QueueStats
	query := `
		SELECT 
			COALESCE(SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END), 0) as pending,
			COALESCE(SUM(CASE WHEN status = 'processed' THEN 1 ELSE 0 END), 0) as processed,
			COALESCE(SUM(CASE WHEN status = 'suppressed' THEN 1 ELSE 0 END), 0) as suppressed
		FROM sms_notification_queue
	`
	err := db.QueryRow(query).Scan(&stats.Pending, &stats.Processed, &stats.Suppressed)
	if err != nil {
		return QueueStats{}, fmt.Errorf("failed to get queue stats: %w", err)
	}
	return stats, nil
}

// GetQueueEntries retrieves queue entries for admin display
func GetQueueEntries(status string, limit, offset int) ([]QueueEntry, error) {
	var query string
	var args []interface{}

	if status == "" || status == "all" {
		query = `
			SELECT 
				q.id,
				q.recipient_user_id,
				q.conversation_id,
				a.title as ad_title,
				q.status,
				q.created_at,
				q.processed_at
			FROM sms_notification_queue q
			JOIN conversations c ON q.conversation_id = c.id
			JOIN ads a ON c.ad_id = a.id
			ORDER BY q.created_at DESC
			LIMIT $1 OFFSET $2
		`
		args = []interface{}{limit, offset}
	} else {
		query = `
			SELECT 
				q.id,
				q.recipient_user_id,
				q.conversation_id,
				a.title as ad_title,
				q.status,
				q.created_at,
				q.processed_at
			FROM sms_notification_queue q
			JOIN conversations c ON q.conversation_id = c.id
			JOIN ads a ON c.ad_id = a.id
			WHERE q.status = $1
			ORDER BY q.created_at DESC
			LIMIT $2 OFFSET $3
		`
		args = []interface{}{status, limit, offset}
	}

	var entries []QueueEntry
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query queue entries: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var e QueueEntry
		var processedAt sql.NullTime
		err := rows.Scan(
			&e.ID,
			&e.RecipientUserID,
			&e.ConversationID,
			&e.AdTitle,
			&e.Status,
			&e.CreatedAt,
			&processedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan queue entry: %w", err)
		}
		if processedAt.Valid {
			e.ProcessedAt = &processedAt.Time
		}
		entries = append(entries, e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating queue entries: %w", err)
	}

	return entries, nil
}
