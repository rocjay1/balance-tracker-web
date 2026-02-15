package calculator

import (
	"fmt"
	"time"

	"github.com/roccodavino/balance-tracker-web/backend/internal/config"
	"github.com/roccodavino/balance-tracker-web/backend/internal/store"
)

type PaymentResult struct {
	CardName         string
	StatementBalance float64
	CurrentBalance   float64
	ProjectedBalance float64 // Current - Statement
	TargetBalance    float64 // Limit * 0.10
	PaymentNeeded    float64
	DueDate          time.Time
}

// CalculatePayment determines the payment needed for a card to maintain target utilization.
// refTime is mostly for testing, pass time.Now() normally.
func CalculatePayment(s *store.Store, card config.CardConfig, refTime time.Time) (*PaymentResult, error) {
	// 1. Determine dates
	// We need the *last* statement date to know what the statement balance is.
	// If StatementDay is 20, and today is Feb 12, last statement was Jan 20.
	// If today is Feb 21, last statement was Feb 20.
	
	year, month, _ := refTime.Date()
	
	// Construct potential statement date for this month
	thisMonthStatement := mkDate(year, month, card.StatementDay)
	
	var lastStatementDate time.Time
	if refTime.After(thisMonthStatement) || refTime.Equal(thisMonthStatement) {
		lastStatementDate = thisMonthStatement
	} else {
		// Go back to previous month
		lastStatementDate = thisMonthStatement.AddDate(0, -1, 0)
		// Handle month rolling edge cases (e.g. if StatementDay is 31 and prev month is Feb)
		// For simplicity, we assume StatementDay exists, or we clamp.
		// Detailed logic would use standard libs to normalize "Jan 31 - 1 month" -> "Dec 31" vs "Feb 28"
		// time.AddDate normalizes, so (March 31).AddDate(0, -1, 0) -> March 3? No, it's safer to reconstruct.
		// Let's rely on mkDate clamping if needed, or just basic day setting.
		// Actually time.Date normalizes: October 32 -> November 1. This isn't what we want for "last month".
		// Simple approach: Set day to 1, subtract month, then set day back to StatementDay (or max for that month).
		prevMonth := time.Date(year, month-1, 1, 0, 0, 0, 0, time.UTC)
		lastStatementDate = mkDate(prevMonth.Year(), prevMonth.Month(), card.StatementDay)
	}

	lastStatementStr := lastStatementDate.Format("2006-01-02")
	refDateStr := refTime.Format("2006-01-02")

	// 2. Query Balances
	// If StartingBalance is set, we need to calculate: StartingBalance + Sum(trans > StartingDate AND trans <= queryDate)
	
	getBalance := func(until string) (float64, error) {
		bal := 0.0
		fromDate := "0000-01-01" // Default start of time

		if card.StartingDate != "" {
			fromDate = card.StartingDate
			bal = card.StartingBalance
		}

		// We need a store method that supports a date range or "since"
		// Current GetBalance is "until".
		// Let's modify GetBalance or add GetBalanceSince.
		// Actually, Store.GetBalance is: WHERE account_name = ? AND date <= ?
		// If we want range: WHERE ... AND date > fromDate AND date <= until
		
		dbBal, err := s.GetBalanceSince(card.Name, card.AccountNumber, fromDate, until)
		if err != nil {
			return 0, err
		}
		return bal + dbBal, nil
	}

	statementBalance, err := getBalance(lastStatementStr)
	if err != nil {
		return nil, fmt.Errorf("failed to get statement balance: %w", err)
	}

	currentBalance, err := getBalance(refDateStr)
	if err != nil {
		return nil, fmt.Errorf("failed to get current balance: %w", err)
	}

	// 3. Logic
	projectedBalance := currentBalance - statementBalance
	targetBalance := card.Limit * 0.10
	paymentNeeded := projectedBalance - targetBalance

	if paymentNeeded < 0 {
		paymentNeeded = 0
	}

	// Calculate Due Date (usually based on Statement Day + gap, but strictly we have a "Due Day" config)
	// We need the Due Date *following* the reference time?
	// The prompt says: "3 days before a card's payment due date, I want..."
	// So we need to find the NEXT occurrence of DueDay.
	dueDate := mkDate(year, month, card.DueDay)
	if dueDate.Before(refTime) {
		dueDate = dueDate.AddDate(0, 1, 0)
		// re-clamp due date for next month
		dueDate = mkDate(dueDate.Year(), dueDate.Month(), card.DueDay)
	}

	return &PaymentResult{
		CardName:         card.Name,
		StatementBalance: statementBalance,
		CurrentBalance:   currentBalance,
		ProjectedBalance: projectedBalance,
		TargetBalance:    targetBalance,
		PaymentNeeded:    paymentNeeded,
		DueDate:          dueDate,
	}, nil
}

func mkDate(y int, m time.Month, d int) time.Time {
	// Handle invalid days (e.g. Feb 30) by clamping to end of month?
	// Simplest: Time.Date normalizes, so Feb 30 becomes Mar 2.
	// But credit cards usually stick to "last day" if day doesn't exist.
	// Let's implement clamping.
	t := time.Date(y, m+1, 0, 0, 0, 0, 0, time.UTC) // Last day of month m
	if d > t.Day() {
		d = t.Day()
	}
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
