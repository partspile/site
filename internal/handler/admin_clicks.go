package handler

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/partspile/site/internal/ad"
	"github.com/partspile/site/internal/logger"
	"github.com/partspile/site/internal/ui"
)

func AdminClicksHandler(c *fiber.Ctx) error {
	data, err := clickAdminData()
	if err != nil {
		logger.Error("Failed to load click admin data", "error", err)
		return showError(c, "Failed to load click data")
	}
	return render(c, ui.AdminDashboardContainerWithClicks("clicks", data))
}

func clickAdminData() (ui.ClickAdminData, error) {
	summary, err := ad.GetClickSummary()
	if err != nil {
		return ui.ClickAdminData{}, err
	}
	topAds, err := ad.GetTopAdsByClicks(20)
	if err != nil {
		return ui.ClickAdminData{}, err
	}
	topImages, err := ad.GetTopImagesByClicks(20)
	if err != nil {
		return ui.ClickAdminData{}, err
	}
	return ui.ClickAdminData{
		UsersWithClicks: summary.UsersWithClicks,
		AdsClicked:      summary.AdsClicked,
		AdDetailViews:   summary.AdDetailViews,
		ImageNavClicks:  summary.ImageNavClicks,
		ActiveLast7Days: summary.ActiveLast7Days,
		TopAds:          clickAdRows(topAds),
		TopImages:       clickImageRows(topImages),
	}, nil
}

func clickAdRows(rows []ad.TopAdClick) []ui.ClickAdRow {
	out := make([]ui.ClickAdRow, len(rows))
	for i, r := range rows {
		out[i] = ui.ClickAdRow{
			AdID:         r.AdID,
			Title:        r.Title,
			CategoryName: r.CategoryName,
			UserCount:    r.UserCount,
			AdViews:      r.AdViews,
			ImageClicks:  r.ImageClicks,
			LastActivity: formatClickTime(r.LastActivity),
		}
	}
	return out
}

func clickImageRows(rows []ad.TopImageClick) []ui.ClickImageRow {
	out := make([]ui.ClickImageRow, len(rows))
	for i, r := range rows {
		out[i] = ui.ClickImageRow{
			AdID:       r.AdID,
			Title:      r.Title,
			ImageIndex: r.ImageIndex,
			UserCount:  r.UserCount,
			Clicks:     r.Clicks,
			LastClick:  formatClickTime(r.LastClickedAt),
		}
	}
	return out
}

func formatClickTime(t *time.Time) string {
	if t == nil {
		return "—"
	}
	now := time.Now()
	d := now.Sub(t.In(now.Location()))
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	default:
		return t.Format("Jan 2, 2006")
	}
}
