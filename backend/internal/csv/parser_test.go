package csv

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	csvData := `Date,Account Name,Institution Name,Account Number,Amount,Description,Category,Ignored From
2026-03-01,Test Account,Bank A,1234,10.50,Grocery,Food,
2026-03-02,Test Account,Bank A,1234,25.00,Gas,Transport,Something
`

	reader := strings.NewReader(csvData)
	transactions, err := Parse(reader)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(transactions) != 2 {
		t.Fatalf("Expected 2 transactions, got %d", len(transactions))
	}

	t1 := transactions[0]
	if t1.Date != "2026-03-01" {
		t.Errorf("Expected date 2026-03-01, got %s", t1.Date)
	}
	if t1.Amount != 10.50 {
		t.Errorf("Expected amount 10.50, got %f", t1.Amount)
	}
	if t1.Ignored != false {
		t.Errorf("Expected ignored false, got %v", t1.Ignored)
	}
	if t1.Hash == "" {
		t.Error("Expected hash, got empty string")
	}

	t2 := transactions[1]
	if t2.Ignored != true {
		t.Errorf("Expected ignored true, got %v", t2.Ignored)
	}
}

func TestParse_InvalidAmount(t *testing.T) {
	csvData := `Date,Account Name,Institution Name,Account Number,Amount,Description,Category,Ignored From
2026-03-01,Test Account,Bank A,1234,INVALID,Grocery,Food,
`
	reader := strings.NewReader(csvData)
	transactions, err := Parse(reader)
	if err != nil {
		t.Fatalf("Parse should not fail on invalid amount line, but skip it: %v", err)
	}

	if len(transactions) != 0 {
		t.Errorf("Expected 0 transactions due to invalid amount, got %d", len(transactions))
	}
}
