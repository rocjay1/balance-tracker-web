// Package api provides HTTP handlers for the balance tracker API.
package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/rocjay1/balance-tracker-web/backend/internal/alerts"
	"github.com/rocjay1/balance-tracker-web/backend/internal/calculator"
	"github.com/rocjay1/balance-tracker-web/backend/internal/config"
	"github.com/rocjay1/balance-tracker-web/backend/internal/csv"
	"github.com/rocjay1/balance-tracker-web/backend/internal/mailer"
	"github.com/rocjay1/balance-tracker-web/backend/internal/store"
)

// Server holds dependencies for the API handlers.
type Server struct {
	store  *store.Store
	config *config.Config
	mailer *mailer.Mailer
}

// NewServer creates a Server with the given dependencies.
func NewServer(s *store.Store, cfg *config.Config, m *mailer.Mailer) *Server {
	return &Server{
		store:  s,
		config: cfg,
		mailer: m,
	}
}

// TestAlertHandler triggers a manual alert check. Accepts optional "date" (YYYY-MM-DD)
// and "force" query parameters.
func (s *Server) TestAlertHandler(w http.ResponseWriter, r *http.Request) {
	dateStr := r.URL.Query().Get("date")
	var refTime time.Time
	if dateStr != "" {
		loc, err := time.LoadLocation(s.config.Timezone)
		if err != nil {
			slog.Error("Error loading timezone, defaulting to UTC", "timezone", s.config.Timezone, "error", err)
			loc = time.UTC
		}
		parsedDate, err := time.ParseInLocation("2006-01-02", dateStr, loc)
		if err != nil {
			http.Error(w, "Invalid date format. Use YYYY-MM-DD", http.StatusBadRequest)
			return
		}
		refTime = parsedDate
		slog.Info("Manual alert check triggered for specific date", "date", dateStr, "timezone", loc.String())
	} else {
		slog.Info("Manual alert check triggered via API")
	}

	force := r.URL.Query().Get("force") == "true"
	alerts.CheckAndSendAlerts(s.store, s.config, s.mailer, refTime, force)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Alert check triggered"))
}

// HealthHandler returns a simple 200 OK for liveness probes.
func (s *Server) HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// CardStatus represents the computed financial status of a single credit card.
type CardStatus struct {
	CardName         string  `json:"card_name"`
	AccountNumber    string  `json:"account_number"`
	StatementBalance float64 `json:"statement_balance"`
	CurrentBalance   float64 `json:"current_balance"`
	ProjectedBalance float64 `json:"projected_balance"`
	TargetBalance    float64 `json:"target_balance"`
	PaymentNeeded    float64 `json:"payment_needed"`
	DueDate          string  `json:"due_date"`
}

// StatusHandler returns the computed financial status for all configured cards.
func (s *Server) StatusHandler(w http.ResponseWriter, r *http.Request) {
	var statuses []CardStatus
	for _, card := range s.config.Cards {
		res, err := calculator.CalculatePayment(s.store, card, time.Now())
		if err != nil {
			slog.Error("Error calculating payment for card", "card", card.Name, "error", err)
			continue
		}
		statuses = append(statuses, CardStatus{
			CardName:         card.Name,
			AccountNumber:    card.AccountNumber,
			StatementBalance: res.StatementBalance,
			CurrentBalance:   res.CurrentBalance,
			ProjectedBalance: res.ProjectedBalance,
			TargetBalance:    res.TargetBalance,
			PaymentNeeded:    res.PaymentNeeded,
			DueDate:          res.DueDate.Format("2006-01-02"),
		})
	}

	slog.Info("Status check complete", "cards", len(statuses))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(statuses)
}

// UploadHandler accepts a CSV file upload, parses it, and syncs the transactions to the store.
func (s *Server) UploadHandler(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20) // 10MB

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Invalid file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Save to temp file to parse (since csv.Parse takes a path)
	tempFile, err := os.CreateTemp("", "upload-*.csv")
	if err != nil {
		http.Error(w, "Failed to create temp file", http.StatusInternalServerError)
		return
	}
	defer os.Remove(tempFile.Name())

	if _, err := io.Copy(tempFile, file); err != nil {
		http.Error(w, "Failed to write temp file", http.StatusInternalServerError)
		return
	}
	tempFile.Close()

	transactions, err := csv.Parse(tempFile.Name())
	if err != nil {
		http.Error(w, fmt.Sprintf("Error parsing CSV: %v", err), http.StatusBadRequest)
		return
	}

	if err := s.store.SyncTransactions(transactions); err != nil {
		http.Error(w, fmt.Sprintf("Error syncing transactions: %v", err), http.StatusInternalServerError)
		return
	}

	slog.Info("upload complete", "transactions", len(transactions))

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"message": fmt.Sprintf("Processed %d transactions", len(transactions)),
		"count":   len(transactions),
	})
}
