// Package store provides SQLite-backed persistence for financial transactions.
package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
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

	queryOverrides := `
	CREATE TABLE IF NOT EXISTS balance_overrides (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		account_number TEXT NOT NULL,
		statement_balance REAL NOT NULL,
		statement_date TEXT NOT NULL,
		updated_at TEXT NOT NULL,
		UNIQUE(account_number, statement_date)
	);
	`
	if _, err := s.db.Exec(queryOverrides); err != nil {
		return err
	}

	// Migration: Add current_balance column if not exists
	s.db.Exec(`ALTER TABLE balance_overrides ADD COLUMN current_balance REAL;`)

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

// BalanceOverrideRow holds both the statement and current balance overrides for a given period.
type BalanceOverrideRow struct {
	StatementBalance *float64
	CurrentBalance   *float64
}

// SetBalanceOverride upserts a balance override for the specified account and statement date.
// Either statementBal or currentBal (or both) may be nil to leave that field unchanged.
func (s *Store) SetBalanceOverride(accountNumber, statementDate string, statementBal, currentBal *float64) error {
	// Check if a row already exists.
	existing, err := s.GetBalanceOverride(accountNumber, statementDate)
	if err != nil {
		return err
	}

	now := time.Now().Format(time.RFC3339)

	if existing == nil {
		// Insert a new row.
		query := `
		INSERT INTO balance_overrides (account_number, statement_balance, current_balance, statement_date, updated_at)
		VALUES (?, ?, ?, ?, ?)
		`
		var stmtVal, curVal sql.NullFloat64
		if statementBal != nil {
			stmtVal = sql.NullFloat64{Float64: *statementBal, Valid: true}
		}
		if currentBal != nil {
			curVal = sql.NullFloat64{Float64: *currentBal, Valid: true}
		}
		_, err = s.db.Exec(query, accountNumber, stmtVal, curVal, statementDate, now)
		return err
	}

	// Update existing row, only touching the fields that are provided.
	if statementBal != nil {
		_, err = s.db.Exec(`UPDATE balance_overrides SET statement_balance = ?, updated_at = ? WHERE account_number = ? AND statement_date = ?`,
			*statementBal, now, accountNumber, statementDate)
		if err != nil {
			return err
		}
	}
	if currentBal != nil {
		_, err = s.db.Exec(`UPDATE balance_overrides SET current_balance = ?, updated_at = ? WHERE account_number = ? AND statement_date = ?`,
			*currentBal, now, accountNumber, statementDate)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetBalanceOverride retrieves both balance overrides for the specified account and statement date.
func (s *Store) GetBalanceOverride(accountNumber, statementDate string) (*BalanceOverrideRow, error) {
	query := `
	SELECT statement_balance, current_balance
	FROM balance_overrides
	WHERE account_number = ? AND statement_date = ?
	`
	var stmtBal, curBal sql.NullFloat64
	err := s.db.QueryRow(query, accountNumber, statementDate).Scan(&stmtBal, &curBal)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	row := &BalanceOverrideRow{}
	if stmtBal.Valid {
		row.StatementBalance = &stmtBal.Float64
	}
	if curBal.Valid {
		row.CurrentBalance = &curBal.Float64
	}
	return row, nil
}

// DeleteBalanceOverride removes a specific override field. If both fields become null, deletes the row.
func (s *Store) DeleteBalanceOverride(accountNumber, statementDate, field string) error {
	// Null out the specific field.
	query := fmt.Sprintf(`UPDATE balance_overrides SET %s = NULL, updated_at = ? WHERE account_number = ? AND statement_date = ?`, field)
	_, err := s.db.Exec(query, time.Now().Format(time.RFC3339), accountNumber, statementDate)
	if err != nil {
		return err
	}

	// Clean up: delete the row if both overrides are null.
	_, err = s.db.Exec(`DELETE FROM balance_overrides WHERE account_number = ? AND statement_date = ? AND statement_balance IS NULL AND current_balance IS NULL`,
		accountNumber, statementDate)
	return err
}
