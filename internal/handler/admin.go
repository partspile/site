package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/partspile/site/internal/logger"
	"github.com/partspile/site/internal/service/sms"
	"github.com/partspile/site/internal/ui"
)

func AdminDashboardHandler(c *fiber.Ctx) error {
	stats, entries, err := loadSMSQueueUI(c)
	if err != nil {
		return err
	}
	return renderPage(c, "Admin Dashboard",
		ui.AdminDashboardPage(stats, entries))
}

func AdminTabHandler(c *fiber.Ctx) error {
	tabID := c.Params("tab")
	switch tabID {
	case "sms-queue":
		return AdminSMSQueueHandler(c)
	case "embeddings":
		return AdminEmbeddingsHandler(c)
	case "clicks":
		return AdminClicksHandler(c)
	case "settings":
		return render(c, ui.AdminDashboardContainer("settings"))
	default:
		return fiber.NewError(fiber.StatusBadRequest, "Invalid tab")
	}
}

func AdminSMSQueueHandler(c *fiber.Ctx) error {
	stats, entries, err := loadSMSQueueUI(c)
	if err != nil {
		return err
	}
	return render(c, ui.AdminDashboardContainerWithQueue(
		"sms-queue", stats, entries,
	))
}

func loadSMSQueueUI(c *fiber.Ctx) (ui.QueueStats, []ui.SMSQueueEntry, error) {
	status := c.Query("status", "all")
	stats, err := sms.GetQueueStats()
	if err != nil {
		logger.Error("Failed to get queue stats", "error", err)
		return ui.QueueStats{}, nil,
			showError(c, "Failed to load queue statistics")
	}

	queueEntries, err := sms.GetQueueEntries(status, 50, 0)
	if err != nil {
		logger.Error("Failed to get queue entries", "error", err)
		return ui.QueueStats{}, nil,
			showError(c, "Failed to load queue entries")
	}

	inputs := make([]ui.SMSQueueEntryInput, len(queueEntries))
	for i, entry := range queueEntries {
		inputs[i] = ui.SMSQueueEntryInput{
			ID:          entry.ID,
			AdTitle:     entry.AdTitle,
			Status:      entry.Status,
			CreatedAt:   entry.CreatedAt,
			ProcessedAt: entry.ProcessedAt,
		}
	}

	uiStats := ui.QueueStats{
		Pending:    stats.Pending,
		Processed:  stats.Processed,
		Suppressed: stats.Suppressed,
	}
	return uiStats, ui.SMSQueueEntriesFrom(inputs), nil
}
