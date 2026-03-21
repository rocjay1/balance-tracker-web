// Package store provides SQLite-backed persistence for financial transactions.
package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite" // Pure Go SQLite driver
)

// Store wraps a SQLite database for storing and querying transactions.
type Store struct {
	db *sql.DB
}

// New opens or creates a SQLite database at dbPath, runs migrations, and returns a Store.
func New(dbPath string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("Failed to create db directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("Failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("Failed to ping database: %w", err)
	}

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("Failed to migrate database: %w", err)
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
		return fmt.Errorf("Failed to begin db transaction: %w", err)
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
			return fmt.Errorf("Failed to delete existing transactions for %s: %w", key.Name, err)
		}

		// Insert new
		insQuery := `
		INSERT OR IGNORE INTO transactions (date, account_name, institution_name, account_number, amount, description, category, ignored, hash)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`
		for _, t := range group {
			if _, err := tx.Exec(insQuery, t.Date, t.AccountName, t.InstitutionName, t.AccountNumber, t.Amount, t.Description, t.Category, t.Ignored, t.Hash); err != nil {
				return fmt.Errorf("Failed to insert transaction %s: %w", t.Hash, err)
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

	// Create or update balance_overrides table with statement_balance as nullable
	_, err := s.db.Exec(`
	CREATE TABLE IF NOT EXISTS balance_overrides (
		account_number TEXT NOT NULL,
		statement_balance REAL,
		current_balance REAL,
		statement_date TEXT NOT NULL,
		updated_at TEXT NOT NULL,
		UNIQUE(account_number, statement_date)
	);
	`)
	if err != nil {
		return err
	}

	// Handle the transition from the old schema where statement_balance was NOT NULL
	var sqlStmt string
	err = s.db.QueryRow(`SELECT sql FROM sqlite_master WHERE type='table' AND name='balance_overrides'`).Scan(&sqlStmt)
	if err == nil && strings.Contains(sqlStmt, "statement_balance REAL NOT NULL") {
		// Ensure current_balance exists before we copy data, just in case
		// This is a safeguard; the CREATE TABLE IF NOT EXISTS above should handle it for new tables.
		// For existing tables that need migration, this ensures the column is present before copying.
		s.db.Exec(`ALTER TABLE balance_overrides ADD COLUMN current_balance REAL;`)

		migrationSQL := `
		CREATE TABLE balance_overrides_v2 (
			account_number TEXT NOT NULL,
			statement_balance REAL,
			current_balance REAL,
			statement_date TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			UNIQUE(account_number, statement_date)
		);
		INSERT INTO balance_overrides_v2 (account_number, statement_balance, current_balance, statement_date, updated_at)
		SELECT account_number, statement_balance, current_balance, statement_date, updated_at FROM balance_overrides;
		DROP TABLE balance_overrides;
		ALTER TABLE balance_overrides_v2 RENAME TO balance_overrides;
		`
		if _, err := s.db.Exec(migrationSQL); err != nil {
			return fmt.Errorf("failed to drop NOT NULL constraint on statement_balance: %w", err)
		}
	} else if err == nil && !strings.Contains(sqlStmt, "current_balance") {
		// Fallback for an intermediate state where NOT NULL was removed but current_balance is missing
		s.db.Exec(`ALTER TABLE balance_overrides ADD COLUMN current_balance REAL;`)
	}

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

// GetTransactions returns a list of transactions matching the given criteria.
// Empty strings for any parameter mean "no filter" for that field.
func (s *Store) GetTransactions(accountName, accountNumber, dateFrom, dateTo string) ([]Transaction, error) {
	query := `
	SELECT date, account_name, institution_name, account_number, amount, description, category, ignored, hash
	FROM transactions
	WHERE 1=1
	`
	var args []any

	if accountName != "" {
		query += " AND (account_name = ? OR institution_name = ?)"
		args = append(args, accountName, accountName)
	}
	if accountNumber != "" {
		query += " AND account_number = ?"
		args = append(args, accountNumber)
	}
	if dateFrom != "" {
		query += " AND date >= ?"
		args = append(args, dateFrom)
	}
	if dateTo != "" {
		query += " AND date <= ?"
		args = append(args, dateTo)
	}

	query += " ORDER BY date DESC, id DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txs []Transaction
	for rows.Next() {
		var t Transaction
		if err := rows.Scan(&t.Date, &t.AccountName, &t.InstitutionName, &t.AccountNumber, &t.Amount, &t.Description, &t.Category, &t.Ignored, &t.Hash); err != nil {
			return nil, err
		}
		txs = append(txs, t)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if txs == nil {
		txs = make([]Transaction, 0)
	}

	return txs, nil
}

// SetBalanceOverride upserts a statement balance override for the specified account and statement date.
func (s *Store) SetBalanceOverride(accountNumber, statementDate string, balance float64) error {
	query := `
	INSERT INTO balance_overrides (account_number, statement_balance, statement_date, updated_at)
	VALUES (?, ?, ?, ?)
	ON CONFLICT(account_number, statement_date) DO UPDATE SET 
		statement_balance = excluded.statement_balance, 
		updated_at = excluded.updated_at
	`
	now := time.Now().Format(time.RFC3339)
	_, err := s.db.Exec(query, accountNumber, balance, statementDate, now)
	return err
}

// GetBalanceOverride retrieves the statement balance override for the specified account and statement date.
func (s *Store) GetBalanceOverride(accountNumber, statementDate string) (*float64, error) {
	query := `
	SELECT statement_balance
	FROM balance_overrides
	WHERE account_number = ? AND statement_date = ?
	`
	var bal sql.NullFloat64
	err := s.db.QueryRow(query, accountNumber, statementDate).Scan(&bal)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	// bal could be logically NULL if they deleted statement_balance but kept current_balance.
	// however, since we reverted current_balance, any row should have a statement balance.
	if !bal.Valid {
		return nil, nil
	}
	return &bal.Float64, nil
}

// DeleteBalanceOverride removes the override for the given account/date.
func (s *Store) DeleteBalanceOverride(accountNumber, statementDate string) error {
	query := `
	DELETE FROM balance_overrides
	WHERE account_number = ? AND statement_date = ?
	`
	_, err := s.db.Exec(query, accountNumber, statementDate)
	return err
}
