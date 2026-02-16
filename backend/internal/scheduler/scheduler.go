package scheduler

import (
	"log"
	"time"

	"github.com/roccodavino/balance-tracker-web/backend/internal/alerts"
	"github.com/roccodavino/balance-tracker-web/backend/internal/config"
	"github.com/roccodavino/balance-tracker-web/backend/internal/mailer"
	"github.com/roccodavino/balance-tracker-web/backend/internal/store"
)

func StartAlertScheduler(s *store.Store, cfg *config.Config, m *mailer.Mailer) {
	// Calculate time until next 7:00 AM
	now := time.Now()
	nextRun := time.Date(now.Year(), now.Month(), now.Day(), 7, 0, 0, 0, now.Location())
	if now.After(nextRun) {
		nextRun = nextRun.Add(24 * time.Hour)
	}

	timer := time.NewTimer(time.Until(nextRun))
	defer timer.Stop()

	log.Printf("Alert scheduler started. Next run at: %v", nextRun)

	for {
		<-timer.C

		log.Println("Running daily alert check...")
		alerts.CheckAndSendAlerts(s, cfg, m)

		// Reset for next day
		timer.Reset(24 * time.Hour)
	}
}
