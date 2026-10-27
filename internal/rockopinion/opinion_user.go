package rockopinion

import (
	"fmt"
	"strings"
	"time"

	"github.com/partspile/site/internal/ad"
	"github.com/partspile/site/internal/message"
	"github.com/partspile/site/internal/rock"
)

type promptInput struct {
	AdTitle       string
	AdOriginal    string
	AdHistory     []ad.HistoryEntryDisplay
	FormalFacets  []string
	Tags          []string
	Messages      []message.Message
	OwnerID       int
	InquirerID    int
	RockThrowerID int
	RockThrownAt  time.Time
	Reason        string
	Tz            *time.Location
	ImageCount    int
}

func buildUserPrompt(in promptInput) string {
	var b strings.Builder
	b.WriteString("AD:\n")
	fmt.Fprintf(&b, "Title: %s\n", in.AdTitle)
	fmt.Fprintf(&b, "Original description: %s\n", in.AdOriginal)

	if len(in.AdHistory) > 0 {
		b.WriteString("\nEdit history (newest first):\n")
		for _, e := range in.AdHistory {
			fmt.Fprintf(&b, "  [%s] %s\n", e.Header, e.Body)
		}
	}

	if len(in.FormalFacets) > 0 {
		b.WriteString("\nFormal fields:\n")
		for _, line := range in.FormalFacets {
			fmt.Fprintf(&b, "  %s\n", line)
		}
	}

	if len(in.Tags) > 0 {
		b.WriteString("\nTags:\n")
		for _, t := range in.Tags {
			fmt.Fprintf(&b, "  %s\n", t)
		}
	}

	if in.ImageCount > 0 {
		fmt.Fprintf(&b,
			"\nListing images: %d image(s) attached to this request "+
				"for visual review.\n", in.ImageCount)
	}

	complaintAt := in.RockThrownAt
	if in.Tz != nil {
		complaintAt = complaintAt.In(in.Tz)
	}
	b.WriteString("\nDispute metadata:\n")
	if in.RockThrowerID == in.InquirerID {
		b.WriteString(
			"The inquirer filed a complaint about the ad with the owner.\n",
		)
	} else {
		b.WriteString(
			"The owner filed a complaint about the inquirer regarding this ad.\n",
		)
	}
	fmt.Fprintf(&b, "Complaint filed at: %s\n",
		complaintAt.Format(time.RFC3339))
	if in.Reason != "" {
		atAd := in.RockThrowerID == in.InquirerID
		label := rock.ReasonLabelForTarget(in.Reason, atAd)
		if label == "" {
			label = in.Reason
		}
		fmt.Fprintf(&b, "Complainant selected reason: %s (%s)\n",
			in.Reason, label)
	}

	if len(in.Messages) > 0 {
		b.WriteString("\nConversation:\n")
		for _, m := range in.Messages {
			sender := "Inquirer"
			if m.SenderID == in.OwnerID {
				sender = "Owner"
			}
			at := m.CreatedAt
			if in.Tz != nil {
				at = at.In(in.Tz)
			}
			fmt.Fprintf(&b, "[%s] %s: %s\n",
				at.Format(time.RFC3339), sender, m.Content)
		}
	} else {
		b.WriteString("\nConversation: (no messages)\n")
	}

	return b.String()
}
