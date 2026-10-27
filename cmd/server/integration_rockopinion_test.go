package main

import (
	"testing"
	"time"

	"github.com/partspile/site/internal/db"
	"github.com/partspile/site/internal/message"
	"github.com/partspile/site/internal/rockopinion"
)

func setupIntegrationRockOpinion(t *testing.T) (message.Conversation, int) {
	t.Helper()
	var adID int
	err := db.QueryRow(`INSERT INTO ads (category_id, title, description, user_id,
		 expires_at)
		VALUES ($1, 'Rock opinion test ad', 'desc', $2,
		        NOW() + INTERVAL '3 months') RETURNING id`,
		integrationCarsCategory, integrationTestUserID).Scan(&adID)
	if err != nil {
		t.Fatal(err)
	}

	var convID int
	err = db.QueryRow(`INSERT INTO conversations (ad_id, owner_id, inquirer_id, rock_thrower_id, rock_thrown_at)
		VALUES ($1, $2, $3, $3, CURRENT_TIMESTAMP) RETURNING id`,
		adID, integrationTestUserID, integrationInquirerUserID).Scan(&convID)
	if err != nil {
		t.Fatal(err)
	}

	conv, err := message.GetConversationByID(convID)
	if err != nil {
		t.Fatal(err)
	}
	return conv, convID
}

func insertIntegrationRockOpinion(t *testing.T, convID int) {
	t.Helper()
	_, err := db.Exec(`INSERT INTO rock_opinions (
		conversation_id, summary, assessment, assessment_detail,
		resolution, reasoning
	) VALUES ($1, 'Cached summary text.', 5,
		'Both sides acted reasonably.', 'Keep talking.',
		'No policy violation.')`, convID)
	if err != nil {
		t.Fatal(err)
	}
}

func TestIntegrationRockOpinionDB(t *testing.T) {
	conv, convID := setupIntegrationRockOpinion(t)
	loc := time.UTC

	t.Run("returns cached opinion", func(t *testing.T) {
		insertIntegrationRockOpinion(t, convID)

		op, err := rockopinion.GetOrGenerate(conv, loc)
		if err != nil {
			t.Fatal(err)
		}
		if op.Summary != "Cached summary text." {
			t.Fatalf("summary = %q", op.Summary)
		}
		if op.Assessment != 5 {
			t.Fatalf("assessment = %d", op.Assessment)
		}
	})

	t.Run("invalidate conversation", func(t *testing.T) {
		if err := rockopinion.Invalidate(convID); err != nil {
			t.Fatal(err)
		}
		var count int
		if err := db.QueryRow(
			`SELECT COUNT(*) FROM rock_opinions WHERE conversation_id = $1`, convID,
		).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatal("expected cache cleared")
		}
	})

	t.Run("invalidate for ad", func(t *testing.T) {
		insertIntegrationRockOpinion(t, convID)

		if err := rockopinion.InvalidateForAd(conv.AdID); err != nil {
			t.Fatal(err)
		}
		var count int
		if err := db.QueryRow(
			`SELECT COUNT(*) FROM rock_opinions WHERE conversation_id = $1`, convID,
		).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatal("expected cache cleared for ad")
		}
	})
}
