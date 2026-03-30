// Package calculator computes credit card payment amounts based on balances and targets.
package calculator

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/rocjay1/balance-tracker-web/backend/internal/config"
	"github.com/rocjay1/balance-tracker-web/backend/internal/store"
)

type PaymentResult struct {
	CardName         string
	StatementBalance float64
	CurrentBalance   float64
	ProjectedBalance float64 // Current - Statement
	TargetBalance    float64 // Limit * 0.10
	PaymentNeeded    float64
	DueDate          time.Time
	HasOverride      bool
}

// GetStatementDate returns the last statement cutoff date for the given reference time.
func GetStatementDate(card config.CardConfig, refTime time.Time) time.Time {
	year, month, _ := refTime.Date()
	thisMonthStatement := mkDate(year, month, card.StatementDay)

	var lastStatementDate time.Time
	if refTime.After(thisMonthStatement) || refTime.Equal(thisMonthStatement) {
		lastStatementDate = thisMonthStatement
	} else {
		lastStatementDate = thisMonthStatement.AddDate(0, -1, 0)
		prevMonth := time.Date(year, month-1, 1, 0, 0, 0, 0, time.UTC)
		lastStatementDate = mkDate(prevMonth.Year(), prevMonth.Month(), card.StatementDay)
	}

	return lastStatementDate.AddDate(0, 0, card.StatementGraceDays)
}

// CalculatePayment determines the payment needed for a card to maintain target utilization.
func CalculatePayment(ctx context.Context, s *store.Store, card config.CardConfig, refTime time.Time) (*PaymentResult, error) {
	// Determine dates
	year, month, _ := refTime.Date()
	lastStatementDate := GetStatementDate(card, refTime)
	lastStatementStr := lastStatementDate.Format("2006-01-02")
	refDateStr := refTime.Format("2006-01-02")

	// Query Balances
	// If StartingBalance is set, we need to calculate: StartingBalance + Sum(trans > StartingDate AND trans <= queryDate)
	getBalance := func(until string) (float64, error) {
		bal := 0.0
		fromDate := "0000-01-01" // Default start of time

		if card.StartingDate != "" {
			fromDate = card.StartingDate
			bal = card.StartingBalance
		}

		matchName := card.Name
		if card.ImportName != "" {
			matchName = card.ImportName
		}

		dbBal, err := s.GetBalanceSince(ctx, matchName, card.AccountNumber, fromDate, until)
		if err != nil {
			return 0, err
		}
		return bal + dbBal, nil
	}

	calculatedStatementBalance, err := getBalance(lastStatementStr)
	if err != nil {
		return nil, fmt.Errorf("Failed to get statement balance: %w", err)
	}

	statementBalance := calculatedStatementBalance
	hasOverride := false

	override, err := s.GetBalanceOverride(ctx, card.AccountNumber, lastStatementStr)
	if err != nil {
		slog.Error("Failed to check for balance override", "card", card.Name, "error", err)
	}

	if override != nil {
		statementBalance = *override
		hasOverride = true
	}

	currentBalance, err := getBalance(refDateStr)
	if err != nil {
		return nil, fmt.Errorf("Failed to get current balance: %w", err)
	}

	// Logic
	projectedBalance := currentBalance - statementBalance
	targetBalance := card.Limit * 0.10
	paymentNeeded := projectedBalance - targetBalance

	if paymentNeeded < 0 {
		paymentNeeded = 0
	}

	refDate := time.Date(year, month, refTime.Day(), 0, 0, 0, 0, time.UTC)
	dueDate := mkDate(year, month, card.DueDay)
	if dueDate.Before(refDate) {
		dueDate = dueDate.AddDate(0, 1, 0)
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
		HasOverride:      hasOverride,
	}, nil
}

func mkDate(y int, m time.Month, d int) time.Time {
	// Handle invalid days (e.g. Feb 30)
	// Simplest: Time.Date normalizes, so Feb 30 becomes Mar 2
	// But credit cards usually stick to "last day" if day doesn't exist
	t := time.Date(y, m+1, 0, 0, 0, 0, 0, time.UTC) // Last day of month m
	if d > t.Day() {
		d = t.Day()
	}
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
