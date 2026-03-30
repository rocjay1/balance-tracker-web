package scheduler

import (
	"context"
	"log/slog"
	"time"

	"github.com/rocjay1/balance-tracker-web/backend/internal/alerts"
	"github.com/rocjay1/balance-tracker-web/backend/internal/config"
	"github.com/rocjay1/balance-tracker-web/backend/internal/mailer"
	"github.com/rocjay1/balance-tracker-web/backend/internal/store"
)

// StartAlertScheduler runs CheckAndSendAlerts daily at 7:00 AM in the configured timezone.
// It blocks indefinitely and should be called in a goroutine.
func StartAlertScheduler(ctx context.Context, s *store.Store, cfg *config.Config, m *mailer.Mailer) {
	loc, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		slog.Warn("Failed to load timezone, defaulting to UTC", "timezone", cfg.Timezone, "error", err)
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

	slog.Info("Alert scheduler started", "timezone", loc.String(), "next_run", nextRun)

	for {
		select {
		case <-ctx.Done():
			slog.Info("Alert scheduler stopping")
			return
		case <-timer.C:
			slog.Info("Running daily alert check")
			alerts.CheckAndSendAlerts(ctx, s, cfg, m, time.Time{}, false)

			// Reset for next day
			timer.Reset(24 * time.Hour)
		}
	}
}
