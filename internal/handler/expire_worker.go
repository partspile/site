package handler

import (
	"time"

	"github.com/partspile/site/internal/ad"
	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/logger"
)

// StartAdExpireWorker pauses active ads past the auto-expire age.
func StartAdExpireWorker() {
	logger.Info("Starting ad expire worker", "component", "AdExpire")
	go runAdExpireWorker()
}

func runAdExpireWorker() {
	ticker := time.NewTicker(config.AdExpireWorkerInterval)
	defer ticker.Stop()

	expireDueAds()
	for range ticker.C {
		expireDueAds()
	}
}

func expireDueAds() {
	due, err := ad.ListAdsDueToExpire()
	if err != nil {
		logger.Error("Failed to list ads due to expire",
			"error", err, "component", "AdExpire")
		return
	}
	for _, row := range due {
		if err := pauseAdWithSideEffects(row.ID, row.UserID); err != nil {
			logger.Error("Failed to expire ad",
				"error", err, "adID", row.ID, "component", "AdExpire")
			continue
		}
		logger.Info("Auto-expired ad",
			"adID", row.ID, "component", "AdExpire")
	}
}
