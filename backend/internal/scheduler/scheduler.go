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
	// Load configured timezone
	loc, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		log.Printf("Error loading timezone %s: %v. Defaulting to UTC.", cfg.Timezone, err)
		loc = time.UTC
	}

	// Calculate time until next 7:00 AM in the configured timezone
	now := time.Now().In(loc)
	nextRun := time.Date(now.Year(), now.Month(), now.Day(), 7, 0, 0, 0, loc)
	if now.After(nextRun) {
		nextRun = nextRun.Add(24 * time.Hour)
	}

	timer := time.NewTimer(time.Until(nextRun))
	defer timer.Stop()

	log.Printf("Alert scheduler started. Timezone: %s. Next run at: %v", loc, nextRun)

	for {
		<-timer.C

		log.Println("Running daily alert check...")
		alerts.CheckAndSendAlerts(s, cfg, m, time.Time{}, false)

		// Reset for next day
		timer.Reset(24 * time.Hour)
	}
}
