package alerts

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/rocjay1/balance-tracker-web/backend/internal/calculator"
	"github.com/rocjay1/balance-tracker-web/backend/internal/config"
	"github.com/rocjay1/balance-tracker-web/backend/internal/mailer"
	"github.com/rocjay1/balance-tracker-web/backend/internal/store"
)

// CheckAndSendAlerts checks if any card has a payment due in AlertDaysBeforeDue days or fewer.
func CheckAndSendAlerts(ctx context.Context, s *store.Store, cfg *config.Config, m *mailer.Mailer, refTime time.Time, force bool) {
	loc, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		slog.Warn("Failed to load timezone, defaulting to UTC", "timezone", cfg.Timezone, "error", err)
		loc = time.UTC
	}

	if refTime.IsZero() {
		refTime = time.Now()
	}
	now := refTime.In(loc)
	slog.Info("Starting daily alert check", "now", now)

	for _, card := range cfg.Cards {
		res, err := calculator.CalculatePayment(ctx, s, card, now)
		if err != nil {
			slog.Error("Error calculating payment in alert check", "card", card.Name, "error", err)
			continue
		}

		dueDate := res.DueDate.In(loc)
		alertThreshold := time.Duration(cfg.AlertDaysBeforeDue) * 24 * time.Hour

		if dueDate.Sub(now) <= alertThreshold && res.PaymentNeeded > 1.0 {
			slog.Info("Alert: Payment due", "card", card.Name, "due_date", res.DueDate.Format("2006-01-02"), "amount", res.PaymentNeeded)

			subject := fmt.Sprintf("Payment Alert: %s Due Soon", card.Name)
			body := fmt.Sprintf("Reminder: A payment of $%.2f is needed for %s by %s to maintain target utilization.\n\nCurrent Balance: $%.2f\nTarget Balance: $%.2f",
				res.PaymentNeeded, card.Name, res.DueDate.Format("Jan 02"), res.CurrentBalance, res.TargetBalance)

			if err := m.Send(cfg.Subscribers, subject, body); err != nil {
				slog.Error("Failed to send alert email", "card", card.Name, "error", err)
			} else {
				slog.Info("Sent alert email", "card", card.Name)
			}
		}
	}
}
