package ui

import (
	"bytes"
	"strings"
	"testing"
	"time"

	g "maragu.dev/gomponents"
)

func TestFormatExpiresIn(t *testing.T) {
	// Fixed midday UTC so calendar-day math does not depend on wall clock
	// (AddDate+hour near midnight UTC used to flip "1 day" → "2 days" in CI).
	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		at   time.Time
		want string
	}{
		{"soon_minutes", now.Add(30 * time.Minute), "Expires soon"},
		{"soon_hours", now.Add(5 * time.Hour), "Expires soon"},
		{"one_day", now.AddDate(0, 0, 1), "Expires in 1 day"},
		{"days", now.AddDate(0, 0, 5), "Expires in 5 days"},
		{"one_month", now.AddDate(0, 1, 0), "Expires in 1 month"},
		{"one_month_days", now.AddDate(0, 1, 12), "Expires in 1 month 12 days"},
		{"months", now.AddDate(0, 2, 0), "Expires in 2 months"},
		{"months_days", now.AddDate(0, 2, 12), "Expires in 2 months 12 days"},
		// Fresh AdExpireMonths grant is AddDate(0, 12, 0); must not pick up
		// leftover days from 30-day-month math.
		{"initial_grant", now.AddDate(0, 12, 0), "Expires in 12 months"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatExpiresInAt(tt.at, now)
			if got != tt.want {
				t.Fatalf("formatExpiresIn(%v) = %q, want %q", tt.at, got, tt.want)
			}
		})
	}
}

func TestCalendarMonthsDays(t *testing.T) {
	loc := time.UTC
	from := time.Date(2026, 7, 25, 11, 26, 0, 0, loc)
	tests := []struct {
		name       string
		to         time.Time
		wantMonths int
		wantDays   int
	}{
		{"exact_3mo", from.AddDate(0, 3, 0), 3, 0},
		{"3mo_plus_1d", from.AddDate(0, 3, 1), 3, 1},
		{"under_3mo", from.AddDate(0, 3, -1), 2, 29}, // Sep has 30 days
		{"same_day", from, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, d := calendarMonthsDays(from, tt.to)
			if m != tt.wantMonths || d != tt.wantDays {
				t.Fatalf("calendarMonthsDays = %d mo %d d, want %d mo %d d",
					m, d, tt.wantMonths, tt.wantDays)
			}
		})
	}
}

func TestAdExpireToolbarVisibility(t *testing.T) {
	created := time.Now().UTC()
	detail := AdDetail{
		ID:        1,
		OwnerID:   42,
		Title:     "Test Ad",
		Active:    true,
		CreatedAt: created,
		ExpiresAt: created.AddDate(0, 12, 0),
	}

	t.Run("owner_active", func(t *testing.T) {
		html := renderAdNodes(t, Ad(detail, 42, "csrf"))
		if !strings.Contains(html, "Expires in") {
			t.Fatal("expected expire countdown for owner")
		}
		if strings.Contains(html, `/auth/ad/1/renew`) {
			t.Fatal("did not expect Renew for fresh 12-month expires_at")
		}
		if !strings.Contains(html, "/images/post_add.svg") {
			t.Fatal("expected new-ad icon for owner")
		}
		if !strings.Contains(html, "/images/copy.svg") {
			t.Fatal("expected copy-ad icon for owner")
		}
		if !strings.Contains(html, `/auth/ad/1/copy`) {
			t.Fatal("expected copy-ad link for owner")
		}
	})

	t.Run("owner_renew_when_due", func(t *testing.T) {
		due := detail
		due.ExpiresAt = created.AddDate(0, 0, 20)
		html := renderAdNodes(t, Ad(due, 42, "csrf"))
		if !strings.Contains(html, "Renew") {
			t.Fatal("expected Renew when <= 1 month remains")
		}
		if !strings.Contains(html, `/auth/ad/1/renew`) {
			t.Fatal("expected renew endpoint for owner")
		}
	})

	t.Run("non_owner", func(t *testing.T) {
		html := renderAdNodes(t, Ad(detail, 99, "csrf"))
		if strings.Contains(html, "Expires in") {
			t.Fatal("did not expect expire countdown for non-owner")
		}
		if strings.Contains(html, "/images/post_add.svg") {
			t.Fatal("did not expect new-ad icon for non-owner")
		}
		if strings.Contains(html, "/images/copy.svg") {
			t.Fatal("did not expect copy-ad icon for non-owner")
		}
	})

	t.Run("owner_inactive", func(t *testing.T) {
		paused := detail
		paused.Active = false
		paused.Inactive = true
		html := renderAdNodes(t, Ad(paused, 42, "csrf"))
		if strings.Contains(html, "Expires in") {
			t.Fatal("did not expect expire countdown for paused ad")
		}
		if strings.Contains(html, "/images/post_add.svg") {
			t.Fatal("did not expect new-ad icon for paused ad")
		}
		if strings.Contains(html, "/images/copy.svg") {
			t.Fatal("did not expect copy-ad icon for paused ad")
		}
		if strings.Contains(html, "/images/bookmark") {
			t.Fatal("did not expect bookmark icon for paused ad")
		}
		if strings.Contains(html, "/images/share.svg") {
			t.Fatal("did not expect share icon for paused ad")
		}
		if !strings.Contains(html, "/images/restore.svg") {
			t.Fatal("expected restore icon for paused ad owner")
		}
	})

	t.Run("deleted_hides_bookmark_share", func(t *testing.T) {
		deleted := detail
		deleted.Active = false
		deleted.Deleted = true
		html := renderAdNodes(t, Ad(deleted, 42, "csrf"))
		if strings.Contains(html, "/images/bookmark") {
			t.Fatal("did not expect bookmark icon for deleted ad")
		}
		if strings.Contains(html, "/images/share.svg") {
			t.Fatal("did not expect share icon for deleted ad")
		}
	})

	t.Run("active_shows_bookmark_share", func(t *testing.T) {
		html := renderAdNodes(t, Ad(detail, 42, "csrf"))
		if !strings.Contains(html, "/images/bookmark") {
			t.Fatal("expected bookmark icon for active ad when logged in")
		}
		if !strings.Contains(html, "/images/share.svg") {
			t.Fatal("expected share icon for active ad")
		}
	})
}

func renderAdNodes(t *testing.T, nodes []g.Node) string {
	t.Helper()
	var buf bytes.Buffer
	for _, n := range nodes {
		if err := n.Render(&buf); err != nil {
			t.Fatalf("render: %v", err)
		}
	}
	return buf.String()
}
