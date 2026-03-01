// Package store provides SQLite-backed persistence for financial transactions.
package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite" // Pure Go SQLite driver
)

// Store wraps a SQLite database for storing and querying transactions.
type Store struct {
	db *sql.DB
}

// New opens or creates a SQLite database at dbPath, runs migrations, and returns a Store.
func New(dbPath string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create db directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return s, nil
}

// SyncTransactions atomically replaces transactions for the given account within the date range of the new transactions.
// This handles updates (Pending -> Posted) and deletions (Pre-auths dropping off).
func (s *Store) SyncTransactions(txs []Transaction) error {
	if len(txs) == 0 {
		return nil
	}

	// Group by Account
	type accountKey struct {
		Name   string
		Number string
	}
	grouped := make(map[accountKey][]Transaction)

	for _, t := range txs {
		k := accountKey{Name: t.AccountName, Number: t.AccountNumber}
		grouped[k] = append(grouped[k], t)
	}

	// Process each group in a DB transaction
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin db transaction: %w", err)
	}
	defer tx.Rollback() // Rollback if not committed

	for key, group := range grouped {
		// Find min/max date
		minDate := group[0].Date
		maxDate := group[0].Date
		for _, t := range group {
			if t.Date < minDate {
				minDate = t.Date
			}
			if t.Date > maxDate {
				maxDate = t.Date
			}
		}

		// Delete existing in range
		// We delete based on AccountName and AccountNumber as that's how we group
		delQuery := `
		DELETE FROM transactions 
		WHERE account_name = ? AND account_number = ? AND date >= ? AND date <= ?
		`
		if _, err := tx.Exec(delQuery, key.Name, key.Number, minDate, maxDate); err != nil {
			return fmt.Errorf("failed to delete existing transactions for %s: %w", key.Name, err)
		}

		// Insert new
		insQuery := `
		INSERT OR IGNORE INTO transactions (date, account_name, institution_name, account_number, amount, description, category, ignored, hash)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`
		for _, t := range group {
			if _, err := tx.Exec(insQuery, t.Date, t.AccountName, t.InstitutionName, t.AccountNumber, t.Amount, t.Description, t.Category, t.Ignored, t.Hash); err != nil {
				return fmt.Errorf("failed to insert transaction %s: %w", t.Hash, err)
			}
		}
	}

	return tx.Commit()
}

func (s *Store) migrate() error {
	query := `
	CREATE TABLE IF NOT EXISTS transactions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		date TEXT NOT NULL,
		account_name TEXT NOT NULL,
		account_number TEXT NOT NULL,
		amount REAL NOT NULL,
		description TEXT NOT NULL,
		category TEXT,
		ignored BOOLEAN DEFAULT 0,
		hash TEXT UNIQUE NOT NULL
	);
	`
	if _, err := s.db.Exec(query); err != nil {
		return err
	}

	// Migration: Add institution_name if not exists
	// SQLite doesn't have "IF NOT EXISTS" for columns, so we try and ignore error
	alterQuery := `ALTER TABLE transactions ADD COLUMN institution_name TEXT DEFAULT '';`
	s.db.Exec(alterQuery) // Ignore error if column exists

	return nil
}

// Close closes the underlying database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// Transaction represents a single financial transaction record.
type Transaction struct {
	Date            string
	AccountName     string
	InstitutionName string
	AccountNumber   string
	Amount          float64
	Description     string
	Category        string
	Ignored         bool
	Hash            string
}

// AddTransaction inserts a single transaction, ignoring duplicates by hash.
func (s *Store) AddTransaction(t Transaction) error {
	query := `
	INSERT OR IGNORE INTO transactions (date, account_name, institution_name, account_number, amount, description, category, ignored, hash)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.Exec(query, t.Date, t.AccountName, t.InstitutionName, t.AccountNumber, t.Amount, t.Description, t.Category, t.Ignored, t.Hash)
	return err
}

// GetBalance returns the sum of amounts for the given account up to (and including) the given date.
// Date should be in "YYYY-MM-DD" format.
func (s *Store) GetBalance(name string, accountNumber string, untilDate string) (float64, error) {
	return s.getBalanceInternal(name, accountNumber, "", untilDate)
}

// GetBalanceSince returns sum of amounts > fromDate AND <= untilDate.
func (s *Store) GetBalanceSince(name string, accountNumber string, fromDate string, untilDate string) (float64, error) {
	return s.getBalanceInternal(name, accountNumber, fromDate, untilDate)
}

func (s *Store) getBalanceInternal(name, accountNumber, fromDate, untilDate string) (float64, error) {
	query := `
	SELECT COALESCE(SUM(amount), 0)
	FROM transactions
	WHERE (account_name = ? OR institution_name = ?) AND ignored = 0
	`
	args := []any{name, name}

	if accountNumber != "" {
		query += " AND account_number = ?"
		args = append(args, accountNumber)
	}

	if fromDate != "" {
		query += " AND date > ?"
		args = append(args, fromDate)
	}

	if untilDate != "" {
		query += " AND date <= ?"
		args = append(args, untilDate)
	}

	var balance float64
	err := s.db.QueryRow(query, args...).Scan(&balance)
	if err != nil {
		return 0, err
	}
	return balance, nil
}
