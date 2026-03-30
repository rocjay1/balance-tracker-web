// Package store provides SQLite-backed persistence for financial transactions.
package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/rocjay1/balance-tracker-web/backend/internal/config"
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
func (s *Store) SyncTransactions(ctx context.Context, txs []Transaction) error {
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
	tx, err := s.db.BeginTx(ctx, nil)
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
		if _, err := tx.ExecContext(ctx, delQuery, key.Name, key.Number, minDate, maxDate); err != nil {
			return fmt.Errorf("Failed to delete existing transactions for %s: %w", key.Name, err)
		}

		// Insert new
		insQuery := `
		INSERT OR IGNORE INTO transactions (date, account_name, institution_name, account_number, amount, description, category, ignored, hash)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`
		for _, t := range group {
			if _, err := tx.ExecContext(ctx, insQuery, t.Date, t.AccountName, t.InstitutionName, t.AccountNumber, t.Amount, t.Description, t.Category, t.Ignored, t.Hash); err != nil {
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

	// Migration: Add import_name to cards if not exists
	alterCardsQuery := `ALTER TABLE cards ADD COLUMN import_name TEXT DEFAULT '';`
	s.db.Exec(alterCardsQuery) // Ignore error if column exists

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

	// New tables for configuration
	_, err = s.db.Exec(`
	CREATE TABLE IF NOT EXISTS cards (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		account_number TEXT NOT NULL,
		credit_limit REAL NOT NULL,
		statement_day INTEGER NOT NULL,
		due_day INTEGER NOT NULL,
		starting_balance REAL DEFAULT 0.0,
		starting_date TEXT,
		statement_grace_days INTEGER DEFAULT 0,
		UNIQUE(name, account_number)
	);

	CREATE TABLE IF NOT EXISTS subscribers (
		email TEXT PRIMARY KEY
	);

	CREATE TABLE IF NOT EXISTS app_settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);
	`)
	if err != nil {
		return fmt.Errorf("failed to create config tables: %w", err)
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
func (s *Store) GetBalance(ctx context.Context, name string, accountNumber string, untilDate string) (float64, error) {
	return s.getBalanceInternal(ctx, name, accountNumber, "", untilDate)
}

// GetBalanceSince returns sum of amounts > fromDate AND <= untilDate.
func (s *Store) GetBalanceSince(ctx context.Context, name string, accountNumber string, fromDate string, untilDate string) (float64, error) {
	return s.getBalanceInternal(ctx, name, accountNumber, fromDate, untilDate)
}

func (s *Store) getBalanceInternal(ctx context.Context, name, accountNumber, fromDate, untilDate string) (float64, error) {
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
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&balance)
	if err != nil {
		return 0, err
	}
	return balance, nil
}

// GetTransactions returns a list of transactions matching the given criteria.
// Empty strings for any parameter mean "no filter" for that field.
func (s *Store) GetTransactions(ctx context.Context, accountName, accountNumber, dateFrom, dateTo string) ([]Transaction, error) {
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

	rows, err := s.db.QueryContext(ctx, query, args...)
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
func (s *Store) SetBalanceOverride(ctx context.Context, accountNumber, statementDate string, balance float64) error {
	query := `
	INSERT INTO balance_overrides (account_number, statement_balance, statement_date, updated_at)
	VALUES (?, ?, ?, ?)
	ON CONFLICT(account_number, statement_date) DO UPDATE SET 
		statement_balance = excluded.statement_balance, 
		updated_at = excluded.updated_at
	`
	now := time.Now().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, query, accountNumber, balance, statementDate, now)
	return err
}

// GetBalanceOverride retrieves the statement balance override for the specified account and statement date.
func (s *Store) GetBalanceOverride(ctx context.Context, accountNumber, statementDate string) (*float64, error) {
	query := `
	SELECT statement_balance
	FROM balance_overrides
	WHERE account_number = ? AND statement_date = ?
	`
	var bal sql.NullFloat64
	err := s.db.QueryRowContext(ctx, query, accountNumber, statementDate).Scan(&bal)
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
func (s *Store) DeleteBalanceOverride(ctx context.Context, accountNumber, statementDate string) error {
	query := `
	DELETE FROM balance_overrides
	WHERE account_number = ? AND statement_date = ?
	`
	_, err := s.db.ExecContext(ctx, query, accountNumber, statementDate)
	return err
}

// GetConfig fetches the complete configuration from the database.
func (s *Store) GetConfig(ctx context.Context) (*config.Config, error) {
	cfg := &config.Config{}

	// Fetch subscribers
	rows, err := s.db.QueryContext(ctx, "SELECT email FROM subscribers")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch subscribers: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var email string
		if err := rows.Scan(&email); err != nil {
			return nil, err
		}
		cfg.Subscribers = append(cfg.Subscribers, email)
	}

	// Fetch cards
	rows, err = s.db.QueryContext(ctx, "SELECT id, name, import_name, account_number, credit_limit, statement_day, due_day, starting_balance, starting_date, statement_grace_days FROM cards")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch cards: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var c config.CardConfig
		if err := rows.Scan(&c.ID, &c.Name, &c.ImportName, &c.AccountNumber, &c.Limit, &c.StatementDay, &c.DueDay, &c.StartingBalance, &c.StartingDate, &c.StatementGraceDays); err != nil {
			return nil, err
		}
		cfg.Cards = append(cfg.Cards, c)
	}

	// Fetch global settings
	rows, err = s.db.QueryContext(ctx, "SELECT key, value FROM app_settings")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch app_settings: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var key, val string
		if err := rows.Scan(&key, &val); err != nil {
			return nil, err
		}
		switch key {
		case "alert_days_before_due":
			cfg.AlertDaysBeforeDue, _ = strconv.Atoi(val)
		case "timezone":
			cfg.Timezone = val
		case "smtp":
			if err := json.Unmarshal([]byte(val), &cfg.SMTP); err != nil {
				return nil, fmt.Errorf("failed to unmarshal SMTP config: %w", err)
			}
		}
	}

	// Defaults if not in DB
	if cfg.Timezone == "" {
		cfg.Timezone = "America/Chicago"
	}
	if cfg.AlertDaysBeforeDue == 0 {
		cfg.AlertDaysBeforeDue = 3
	}

	return cfg, nil
}

// SaveConfig performs a complete configuration sync to the database.
func (s *Store) SaveConfig(ctx context.Context, cfg *config.Config) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Clear existing
	if _, err := tx.ExecContext(ctx, "DELETE FROM subscribers"); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM cards"); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM app_settings"); err != nil {
		return err
	}

	// 2. Insert subscribers
	for _, email := range cfg.Subscribers {
		if _, err := tx.ExecContext(ctx, "INSERT INTO subscribers (email) VALUES (?)", email); err != nil {
			return err
		}
	}

	// 3. Insert cards
	for _, c := range cfg.Cards {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO cards (name, import_name, account_number, credit_limit, statement_day, due_day, starting_balance, starting_date, statement_grace_days) 
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, c.Name, c.ImportName, c.AccountNumber, c.Limit, c.StatementDay, c.DueDay, c.StartingBalance, c.StartingDate, c.StatementGraceDays); err != nil {
			return err
		}
	}

	// 4. Insert global settings
	settings := map[string]string{
		"alert_days_before_due": strconv.Itoa(cfg.AlertDaysBeforeDue),
		"timezone":              cfg.Timezone,
	}
	smtpJSON, _ := json.Marshal(cfg.SMTP)
	settings["smtp"] = string(smtpJSON)

	for k, v := range settings {
		if _, err := tx.ExecContext(ctx, "INSERT INTO app_settings (key, value) VALUES (?, ?)", k, v); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// SaveCard upserts a card configuration.
func (s *Store) SaveCard(ctx context.Context, c config.CardConfig) error {
	if c.ID > 0 {
		_, err := s.db.ExecContext(ctx, `
			UPDATE cards SET 
				name = ?, 
				import_name = ?,
				account_number = ?, 
				credit_limit = ?, 
				statement_day = ?, 
				due_day = ?, 
				starting_balance = ?, 
				starting_date = ?, 
				statement_grace_days = ? 
			WHERE id = ?
		`, c.Name, c.ImportName, c.AccountNumber, c.Limit, c.StatementDay, c.DueDay, c.StartingBalance, c.StartingDate, c.StatementGraceDays, c.ID)
		return err
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO cards (name, import_name, account_number, credit_limit, statement_day, due_day, starting_balance, starting_date, statement_grace_days)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(name, account_number) DO UPDATE SET
			import_name = excluded.import_name,
			credit_limit = excluded.credit_limit,
			statement_day = excluded.statement_day,
			due_day = excluded.due_day,
			starting_balance = excluded.starting_balance,
			starting_date = excluded.starting_date,
			statement_grace_days = excluded.statement_grace_days
	`, c.Name, c.ImportName, c.AccountNumber, c.Limit, c.StatementDay, c.DueDay, c.StartingBalance, c.StartingDate, c.StatementGraceDays)
	return err
}

// DeleteCard removes a card from tracking.
func (s *Store) DeleteCard(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM cards WHERE id = ?", id)
	return err
}
