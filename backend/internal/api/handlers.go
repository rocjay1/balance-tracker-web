package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
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
	ctx := r.Context()
	dateStr := r.URL.Query().Get("date")
	var refTime time.Time
	if dateStr != "" {
		loc, err := time.LoadLocation(s.config.Timezone)
		if err != nil {
			slog.Warn("Failed to load timezone, defaulting to UTC", "timezone", s.config.Timezone, "error", err)
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
	alerts.CheckAndSendAlerts(ctx, s.store, s.config, s.mailer, refTime, force)

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
	HasOverride      bool    `json:"has_override"`
}

// StatusHandler returns the computed financial status for all configured cards.
func (s *Server) StatusHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var statuses []CardStatus
	for _, card := range s.config.Cards {
		res, err := calculator.CalculatePayment(ctx, s.store, card, time.Now())
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
			HasOverride:      res.HasOverride,
		})
	}

	slog.Info("Status check complete", "cards_processed", len(statuses))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(statuses)
}

// UploadHandler accepts a CSV file upload, parses it, and syncs the transactions to the store.
func (s *Server) UploadHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20) // 10MB

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Invalid file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	transactions, err := csv.Parse(file)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error parsing CSV: %v", err), http.StatusBadRequest)
		return
	}

	if err := s.store.SyncTransactions(ctx, transactions); err != nil {
		http.Error(w, fmt.Sprintf("Error syncing transactions: %v", err), http.StatusInternalServerError)
		return
	}

	slog.Info("Upload complete", "transactions_processed", len(transactions))

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"message": fmt.Sprintf("Processed %d transactions", len(transactions)),
		"count":   len(transactions),
	})
}

// TransactionsHandler returns a list of transactions, optionally filtered by query parameters.
func (s *Server) TransactionsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()
	accountName := q.Get("account_name")
	accountNumber := q.Get("account_number")
	dateFrom := q.Get("date_from")
	dateTo := q.Get("date_to")

	if accountName != "" {
		for _, c := range s.config.Cards {
			if c.Name == accountName && c.ImportName != "" {
				accountName = c.ImportName
				break
			}
		}
	}

	txs, err := s.store.GetTransactions(ctx, accountName, accountNumber, dateFrom, dateTo)
	if err != nil {
		slog.Error("Error querying transactions", "error", err)
		http.Error(w, "Error querying transactions", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(txs); err != nil {
		slog.Error("Error encoding transactions", "error", err)
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
	}
}

// OverrideHandler handles saving and deleting statement balance overrides.
func (s *Server) OverrideHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	accountNumber := r.PathValue("account_number")
	if accountNumber == "" {
		http.Error(w, "Account number required", http.StatusBadRequest)
		return
	}

	var matchedCard *config.CardConfig
	for _, c := range s.config.Cards {
		if c.AccountNumber == accountNumber {
			matchedCard = &c
			break
		}
	}
	if matchedCard == nil {
		http.Error(w, "Card not found", http.StatusNotFound)
		return
	}

	stmtDate := calculator.GetStatementDate(*matchedCard, time.Now())
	stmtDateStr := stmtDate.Format("2006-01-02")

	switch r.Method {
	case http.MethodPut:
		var req struct {
			StatementBalance float64 `json:"statement_balance"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if err := s.store.SetBalanceOverride(ctx, accountNumber, stmtDateStr, req.StatementBalance); err != nil {
			slog.Error("Failed to set override", "error", err)
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)

	case http.MethodDelete:
		if err := s.store.DeleteBalanceOverride(ctx, accountNumber, stmtDateStr); err != nil {
			slog.Error("Failed to delete override", "error", err)
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// ConfigHandler returns the current application configuration or updates it.
func (s *Server) ConfigHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(s.config)

	case http.MethodPost:
		var newCfg config.Config
		if err := json.NewDecoder(r.Body).Decode(&newCfg); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if err := s.store.SaveConfig(ctx, &newCfg); err != nil {
			slog.Error("Failed to save config", "error", err)
			http.Error(w, "Failed to save config", http.StatusInternalServerError)
			return
		}

		s.config = &newCfg
		w.WriteHeader(http.StatusOK)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// CardHandler handles adding, updating, and removing cards.
func (s *Server) CardHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	switch r.Method {
	case http.MethodPost:
		var card config.CardConfig
		if err := json.NewDecoder(r.Body).Decode(&card); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if err := s.store.SaveCard(ctx, card); err != nil {
			slog.Error("Failed to save card", "error", err)
			http.Error(w, "Failed to save card", http.StatusInternalServerError)
			return
		}

		// Refresh in-memory config
		updatedCfg, err := s.store.GetConfig(ctx)
		if err == nil {
			s.config = updatedCfg
		}

		w.WriteHeader(http.StatusOK)

	case http.MethodDelete:
		idStr := r.PathValue("id")
		if idStr == "" {
			http.Error(w, "ID required", http.StatusBadRequest)
			return
		}

		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid ID", http.StatusBadRequest)
			return
		}

		if err := s.store.DeleteCard(ctx, id); err != nil {
			slog.Error("Failed to delete card", "error", err)
			http.Error(w, "Failed to delete card", http.StatusInternalServerError)
			return
		}

		// Refresh in-memory config
		updatedCfg, err := s.store.GetConfig(ctx)
		if err == nil {
			s.config = updatedCfg
		}

		w.WriteHeader(http.StatusOK)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
