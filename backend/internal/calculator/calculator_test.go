package calculator

import (
	"context"
	"testing"
	"time"

	"github.com/rocjay1/balance-tracker-web/backend/internal/config"
	"github.com/rocjay1/balance-tracker-web/backend/internal/store"
)

func TestCalculatePayment(t *testing.T) {
	// Setup in-memory store
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("Failed to create in-memory store: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	card := config.CardConfig{
		Name:          "Test Card",
		AccountNumber: "1234",
		Limit:         1000,
		StatementDay:  15,
		DueDay:        10,
	}

	// Reference time: March 10, 2026. StatementDay: 15.
	// Last Statement: Feb 15.
	refTime := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)

	// Add some transactions
	txs := []store.Transaction{
		{Date: "2026-02-10", AccountName: "Test Card", AccountNumber: "1234", Amount: 100, Description: "Before Feb 15", Hash: "h1"},
		{Date: "2026-02-20", AccountName: "Test Card", AccountNumber: "1234", Amount: 200, Description: "After Feb 15", Hash: "h2"},
		{Date: "2026-03-05", AccountName: "Test Card", AccountNumber: "1234", Amount: 50, Description: "In March", Hash: "h3"},
	}
	if err := s.SyncTransactions(ctx, txs); err != nil {
		t.Fatalf("Failed to sync transactions: %v", err)
	}

	res, err := CalculatePayment(ctx, s, card, refTime)
	if err != nil {
		t.Fatalf("CalculatePayment failed: %v", err)
	}

	// Expected:
	// Last Statement Date: Feb 15
	// Statement Balance: sum up to Feb 15 = 100
	// Current Balance (up to Mar 10): 100 + 200 + 50 = 350
	// Projected Balance: 350 - 100 = 250
	// Target Balance (10% of 1000): 100
	// Payment Needed: 250 - 100 = 150
	// Due Date: March 10 (today is the due date)

	if res.StatementBalance != 100 {
		t.Errorf("Expected statement balance 100, got %f", res.StatementBalance)
	}
	if res.CurrentBalance != 350 {
		t.Errorf("Expected current balance 350, got %f", res.CurrentBalance)
	}
	if res.PaymentNeeded != 150 {
		t.Errorf("Expected payment needed 150, got %f", res.PaymentNeeded)
	}
	
	expectedDueDate := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	if !res.DueDate.Equal(expectedDueDate) {
		t.Errorf("Expected due date %v, got %v", expectedDueDate, res.DueDate)
	}
}

func TestGetStatementDate(t *testing.T) {
	card := config.CardConfig{StatementDay: 15}

	tests := []struct {
		ref      time.Time
		expected time.Time
	}{
		{
			ref:      time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC),
			expected: time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			ref:      time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC),
			expected: time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			ref:      time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC),
			expected: time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tc := range tests {
		got := GetStatementDate(card, tc.ref)
		if !got.Equal(tc.expected) {
			t.Errorf("For %v, expected %v, got %v", tc.ref, tc.expected, got)
		}
	}
}
