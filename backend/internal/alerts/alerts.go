package alerts

import (
	"fmt"
	"log"
	"time"

	"github.com/roccodavino/balance-tracker-web/backend/internal/calculator"
	"github.com/roccodavino/balance-tracker-web/backend/internal/config"
	"github.com/roccodavino/balance-tracker-web/backend/internal/mailer"
	"github.com/roccodavino/balance-tracker-web/backend/internal/store"
)

// CheckAndSendAlerts checks if any card has a payment due in exactly 3 days.
func CheckAndSendAlerts(s *store.Store, cfg *config.Config, m *mailer.Mailer) {
	now := time.Now()
	// Target date is 3 days from now
	targetDate := now.AddDate(0, 0, 3)
	
	// Normalize to start of day for comparison if needed, but calculator returns a full Time.
	// Actually better: Check if DueDate is strictly "Same Day" as targetDate.
	
	for _, card := range cfg.Cards {
		res, err := calculator.CalculatePayment(s, card, now)
		if err != nil {
			log.Printf("Error calculating payment for %s in alert check: %v", card.Name, err)
			continue
		}

		// Check if Res.DueDate is essentially "3 days from today" date-wise
	    if isSameDay(res.DueDate, targetDate) && res.PaymentNeeded > 1.0 { // Alert only if > $1 needed? Avoiding spam for cents.
			log.Printf("Alert: Payment due for %s on %s. Amount: %.2f", card.Name, res.DueDate.Format("2006-01-02"), res.PaymentNeeded)
			
			subject := fmt.Sprintf("Payment Alert: %s Due Soon", card.Name)
			body := fmt.Sprintf("Reminder: A payment of $%.2f is needed for %s by %s to maintain target utilization.\n\nCurrent Balance: $%.2f\nTarget Balance: $%.2f", 
				res.PaymentNeeded, card.Name, res.DueDate.Format("Jan 02"), res.CurrentBalance, res.TargetBalance)

			if err := m.Send(cfg.Subscribers, subject, body); err != nil {
				log.Printf("Failed to send alert email for %s: %v", card.Name, err)
			} else {
				log.Printf("Sent alert email for %s", card.Name)
			}
		}
	}
}

func isSameDay(t1, t2 time.Time) bool {
	return t1.Year() == t2.Year() && t1.Month() == t2.Month() && t1.Day() == t2.Day()
}
