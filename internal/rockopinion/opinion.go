package rockopinion

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/partspile/site/internal/ad"
	"github.com/partspile/site/internal/db"
	"github.com/partspile/site/internal/logger"
	"github.com/partspile/site/internal/message"
	"github.com/partspile/site/internal/rock"
	"github.com/partspile/site/internal/service/grok"
	"github.com/partspile/site/internal/user"
)

var ErrUnavailable = errors.New("rock opinion unavailable")

// Opinion is a cached or freshly generated dispute assessment.
type Opinion struct {
	ConversationID   int       `db:"conversation_id"`
	GeneratedAt      time.Time `db:"generated_at"`
	Summary          string    `db:"summary"`
	Assessment       int       `db:"assessment"`
	AssessmentDetail string    `db:"assessment_detail"`
	Resolution       string    `db:"resolution"`
	Reasoning        string    `db:"reasoning"`
}

// GetOrGenerate returns a cached opinion or generates and stores one.
func GetOrGenerate(conv message.Conversation,
	tz *time.Location) (Opinion, error) {
	if conv.RockThrowerID == nil {
		return Opinion{}, fmt.Errorf("conversation has no rock")
	}

	cached, err := loadOpinion(conv.ID)
	if err == nil {
		return cached, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return Opinion{}, err
	}

	reason := message.LatestThrownReason(conv.Journal)
	op, err := generate(conv, *conv.RockThrowerID, reason, tz)
	if err != nil {
		return Opinion{}, err
	}
	if err := storeOpinion(op); err != nil {
		return Opinion{}, err
	}
	return loadOpinion(conv.ID)
}

// Preview generates an ephemeral opinion without storing or requiring a rock.
func Preview(conv message.Conversation, throwerID int, reason string,
	tz *time.Location) (Opinion, error) {
	if !rock.ValidReason(reason) {
		return Opinion{}, rock.ErrInvalidReason
	}
	return generate(conv, throwerID, reason, tz)
}

func loadOpinion(conversationID int) (Opinion, error) {
	var op Opinion
	err := db.QueryRow(`
		SELECT conversation_id, generated_at, summary, assessment,
			assessment_detail, resolution, reasoning
		FROM rock_opinions
		WHERE conversation_id = $1
	`, conversationID).Scan(
		&op.ConversationID,
		&op.GeneratedAt,
		&op.Summary,
		&op.Assessment,
		&op.AssessmentDetail,
		&op.Resolution,
		&op.Reasoning,
	)
	if err != nil {
		return Opinion{}, err
	}
	return op, nil
}

func storeOpinion(op Opinion) error {
	_, err := db.Exec(`
		INSERT INTO rock_opinions (
			conversation_id, generated_at, summary, assessment,
			assessment_detail, resolution, reasoning
		) VALUES ($1, CURRENT_TIMESTAMP, $2, $3, $4, $5, $6)
	`, op.ConversationID, op.Summary, op.Assessment,
		op.AssessmentDetail, op.Resolution, op.Reasoning)
	if err != nil {
		return fmt.Errorf("store rock opinion: %w", err)
	}
	return nil
}

// Invalidate removes the cached opinion for one conversation.
func Invalidate(conversationID int) error {
	_, err := db.Exec(
		`DELETE FROM rock_opinions WHERE conversation_id = $1`,
		conversationID,
	)
	if err != nil {
		return fmt.Errorf("invalidate rock opinion: %w", err)
	}
	return nil
}

// InvalidateForAd removes cached opinions for all conversations on an ad.
func InvalidateForAd(adID int) error {
	_, err := db.Exec(`
		DELETE FROM rock_opinions
		WHERE conversation_id IN (
			SELECT id FROM conversations WHERE ad_id = $1
		)
	`, adID)
	if err != nil {
		return fmt.Errorf("invalidate rock opinions for ad: %w", err)
	}
	return nil
}

func generate(conv message.Conversation, throwerID int, reason string,
	tz *time.Location) (Opinion, error) {
	a, err := ad.GetAd(0, conv.AdID, tz)
	if err != nil {
		return Opinion{}, fmt.Errorf("load ad: %w", err)
	}
	if err := ad.LoadTags(&a); err != nil {
		return Opinion{}, fmt.Errorf("load ad tags: %w", err)
	}

	category := ad.GetCategory(a.CategoryID)

	owner, err := user.GetByID(conv.OwnerID)
	if err != nil {
		return Opinion{}, fmt.Errorf("load owner: %w", err)
	}
	inquirer, err := user.GetByID(conv.InquirerID)
	if err != nil {
		return Opinion{}, fmt.Errorf("load inquirer: %w", err)
	}

	messages, err := listMessages(conv.ID, tz)
	if err != nil {
		return Opinion{}, err
	}

	redacted := make([]message.Message, len(messages))
	for i, m := range messages {
		redacted[i] = m
		content := RedactText(m.Content)
		content = RedactNames(content, owner.Name, inquirer.Name)
		redacted[i].Content = content
	}

	desc := ad.ParseDescriptionForDisplay(a.Description)
	tags := make([]string, len(a.Tags))
	for i, t := range a.Tags {
		tags[i] = t.PromptDisplay()
	}

	thrownAt := time.Now()
	if conv.RockThrownAt != nil {
		thrownAt = *conv.RockThrownAt
	}

	attachImages := shouldAttachImages(reason, redacted, a.ImageCount)
	promptImageCount := 0
	if attachImages {
		promptImageCount = a.ImageCount
	}

	userPrompt := buildUserPrompt(promptInput{
		AdTitle:       a.Title,
		AdOriginal:    desc.Original,
		AdHistory:     desc.History,
		FormalFacets:  ad.FormalFacetLines(category, a.Facets),
		Tags:          tags,
		Messages:      redacted,
		OwnerID:       conv.OwnerID,
		InquirerID:    conv.InquirerID,
		RockThrowerID: throwerID,
		RockThrownAt:  thrownAt,
		Reason:        reason,
		Tz:            tz,
		ImageCount:    promptImageCount,
	})

	parts := buildGrokParts(userPrompt, a.ID, a.ImageCount, attachImages)
	resp, err := grok.CallGrokConvParts(
		opinionSystemPrompt(), parts, rockOpinionConvID,
	)
	if err != nil {
		logger.Warn("rock opinion: grok call failed",
			"error", err, "conversationID", conv.ID)
		return Opinion{}, ErrUnavailable
	}

	op, err := parseOpinionResponse(resp)
	if err != nil {
		logger.Warn("rock opinion: parse failed",
			"error", err, "conversationID", conv.ID)
		return Opinion{}, ErrUnavailable
	}

	op.ConversationID = conv.ID
	return op, nil
}

func listMessages(conversationID int,
	tz *time.Location) ([]message.Message, error) {
	if conversationID == 0 {
		return nil, nil
	}
	conv, err := message.GetConversationByID(conversationID)
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}
	return message.MessagesFromJournal(conv.ID, conv.Journal, tz), nil
}

// AdFactLines returns description history lines for UI display.
func AdFactLines(a ad.Ad, tz *time.Location) []string {
	desc := ad.ParseDescriptionForDisplay(a.Description)
	var lines []string
	if desc.Original != "" {
		lines = append(lines, "Original: "+ad.DisplayDescription(desc.Original))
	}
	for _, e := range desc.History {
		line := e.Header
		if e.Body != "" {
			line += " — " + ad.DisplayDescription(e.Body)
		}
		lines = append(lines, line)
	}
	return lines
}
