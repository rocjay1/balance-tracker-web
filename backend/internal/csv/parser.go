package csv

import (
	"crypto/sha256"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/roccodavino/balance-tracker-web/backend/internal/store"
)

// Parse reads the CSV file and returns a list of transactions.
func Parse(path string) ([]store.Transaction, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open csv file: %w", err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	// Read header
	header, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read header: %w", err)
	}

	// Map header columns to indices
	colMap := make(map[string]int)
	for i, name := range header {
		colMap[name] = i
	}

	var transactions []store.Transaction
	lineNum := 1 // Header is 0, but usually 1-indexed for humans, so let's say data starts at line 2

	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error reading line %d: %w", lineNum, err)
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
			// Skip or log error? For now, let's skip invalid amounts but maybe warn.
			// Ideally we return error or log it.
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
		// This combination should be unique enough for a single transaction.
		// Note: If description changes (e.g. pending -> posted), this will be treated as new.
		// Detailed deduplication might require fuzzy matching, but hashing is the requested MVP approach.
		hashInput := fmt.Sprintf("%s|%s|%.2f|%s", t.Date, t.AccountNumber, t.Amount, t.Description)
		hash := sha256.Sum256([]byte(hashInput))
		t.Hash = fmt.Sprintf("%x", hash)

		transactions = append(transactions, t)
	}

	return transactions, nil
}
