package main

import (
	"testing"

	"github.com/partspile/site/internal/db"
	"github.com/partspile/site/internal/message"
)

func TestIntegrationOpenConversationReadMarking(t *testing.T) {
	var adID int
	err := db.QueryRow(`INSERT INTO ads (category_id, title, description, user_id,
		 expires_at)
		VALUES ($1, 'Conversation test ad', 'desc', $2,
		        NOW() + INTERVAL '3 months') RETURNING id`,
		integrationCarsCategory, integrationTestUserID).Scan(&adID)
	if err != nil {
		t.Fatal(err)
	}

	var convID int
	err = db.QueryRow(`INSERT INTO conversations (ad_id, owner_id, inquirer_id, inquirer_has_unread, rock_thrower_id, rock_thrown_at)
		VALUES ($1, $2, $3, 1, $3, CURRENT_TIMESTAMP) RETURNING id`,
		adID, integrationTestUserID, integrationInquirerUserID).Scan(&convID)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("denies non-participant even when rocked", func(t *testing.T) {
		_, marked, err := message.OpenConversation(convID, 5)
		if err != message.ErrModalAccess {
			t.Fatalf("err = %v, want ErrModalAccess", err)
		}
		if marked {
			t.Fatal("non-participant should not mark conversation read")
		}
		var unread int
		if err := db.QueryRow(
			`SELECT inquirer_has_unread FROM conversations WHERE id = $1`, convID,
		).Scan(&unread); err != nil {
			t.Fatal(err)
		}
		if unread != 1 {
			t.Fatalf("inquirer_has_unread = %d, want 1", unread)
		}
	})

	t.Run("marks read for participant", func(t *testing.T) {
		if _, err := db.Exec(
			`UPDATE conversations SET inquirer_has_unread = 1 WHERE id = $1`, convID,
		); err != nil {
			t.Fatal(err)
		}
		_, marked, err := message.OpenConversation(convID, integrationInquirerUserID)
		if err != nil {
			t.Fatal(err)
		}
		if !marked {
			t.Fatal("participant should mark conversation read")
		}
		var unread int
		if err := db.QueryRow(
			`SELECT inquirer_has_unread FROM conversations WHERE id = $1`, convID,
		).Scan(&unread); err != nil {
			t.Fatal(err)
		}
		if unread != 0 {
			t.Fatalf("inquirer_has_unread = %d, want 0", unread)
		}
	})
}
