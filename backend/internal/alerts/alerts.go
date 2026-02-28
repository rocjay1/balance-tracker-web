package alerts

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/roccodavino/balance-tracker-web/backend/internal/calculator"
	"github.com/roccodavino/balance-tracker-web/backend/internal/config"
	"github.com/roccodavino/balance-tracker-web/backend/internal/mailer"
	"github.com/roccodavino/balance-tracker-web/backend/internal/store"
)

// CheckAndSendAlerts checks if any card has a payment due in exactly 3 days.
func CheckAndSendAlerts(s *store.Store, cfg *config.Config, m *mailer.Mailer, refTime time.Time, force bool) {
	loc, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		slog.Error("Error loading timezone, defaulting to UTC", "timezone", cfg.Timezone, "error", err)
		loc = time.UTC
	}

	if refTime.IsZero() {
		refTime = time.Now()
	}
	now := refTime.In(loc)
	// Target date is configured days from now
	targetDate := now.AddDate(0, 0, cfg.AlertDaysBeforeDue)

	// Normalize to start of day for comparison if needed, but calculator returns a full Time.
	// Actually better: Check if DueDate is strictly "Same Day" as targetDate.
	slog.Info("Checking alerts", "now", now, "target_date", targetDate)

	for _, card := range cfg.Cards {
		res, err := calculator.CalculatePayment(s, card, now)
		if err != nil {
			slog.Error("Error calculating payment in alert check", "card", card.Name, "error", err)
			continue
		}

		isDueOnTargetDate := isSameDay(res.DueDate, targetDate)
		isPaymentNeeded := res.PaymentNeeded > 1.0

		// Check if Res.DueDate matches the calculated target alert date
		if isDueOnTargetDate && (isPaymentNeeded || force) { // Alert only if > $1 needed? Avoiding spam for cents.
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

func isSameDay(t1, t2 time.Time) bool {
	return t1.Year() == t2.Year() && t1.Month() == t2.Month() && t1.Day() == t2.Day()
}
