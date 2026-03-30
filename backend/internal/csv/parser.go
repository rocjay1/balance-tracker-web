// Package csv parses exported financial CSV files into transaction records.
package csv

import (
	"crypto/sha256"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/rocjay1/balance-tracker-web/backend/internal/store"
)

// Parse reads the CSV file from an io.Reader and returns a list of transactions.
func Parse(r io.Reader) ([]store.Transaction, error) {
	cr := csv.NewReader(r)
	// Read header
	header, err := cr.Read()
	if err != nil {
		return nil, fmt.Errorf("Failed to read header: %w", err)
	}

	// Map header columns to indices
	colMap := make(map[string]int)
	for i, name := range header {
		colMap[name] = i
	}

	var transactions []store.Transaction
	lineNum := 1

	for {
		record, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("Error reading line %d: %w", lineNum, err)
		}
		lineNum++

		// Helper to safely get value
		get := func(col string) string {
			if idx, ok := colMap[col]; ok && idx < len(record) {
				return strings.TrimSpace(record[idx])
			}
			return ""
		}

		amountStr := get("Amount")
		amount, err := strconv.ParseFloat(amountStr, 64)
		if err != nil {
			continue
		}

		ignored := get("Ignored From") != ""

		t := store.Transaction{
			Date:            get("Date"),
			AccountName:     get("Account Name"),
			InstitutionName: get("Institution Name"),
			AccountNumber:   get("Account Number"),
			Amount:          amount,
			Description:     get("Description"),
			Category:        get("Category"),
			Ignored:         ignored,
		}

		// Generate Hash
		// Hash = SHA256(Date + AccountNumber + Amount + Description)
		hashInput := fmt.Sprintf("%s|%s|%.2f|%s", t.Date, t.AccountNumber, t.Amount, t.Description)
		hash := sha256.Sum256([]byte(hashInput))
		t.Hash = fmt.Sprintf("%x", hash)

		transactions = append(transactions, t)
	}

	return transactions, nil
}
