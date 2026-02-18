package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/roccodavino/balance-tracker-web/backend/internal/alerts"
	"github.com/roccodavino/balance-tracker-web/backend/internal/calculator"
	"github.com/roccodavino/balance-tracker-web/backend/internal/config"
	"github.com/roccodavino/balance-tracker-web/backend/internal/csv"
	"github.com/roccodavino/balance-tracker-web/backend/internal/mailer"
	"github.com/roccodavino/balance-tracker-web/backend/internal/store"
)

type Server struct {
	Store  *store.Store
	Config *config.Config
	Mailer *mailer.Mailer
}

func NewServer(s *store.Store, cfg *config.Config, m *mailer.Mailer) *Server {
	return &Server{
		Store:  s,
		Config: cfg,
		Mailer: m,
	}
}

func (s *Server) TestAlertHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	dateStr := r.URL.Query().Get("date")
	var refTime time.Time
	if dateStr != "" {
		loc, err := time.LoadLocation(s.Config.Timezone)
		if err != nil {
			log.Printf("Error loading timezone %s: %v. Defaulting to UTC.", s.Config.Timezone, err)
			loc = time.UTC
		}
		parsedDate, err := time.ParseInLocation("2006-01-02", dateStr, loc)
		if err != nil {
			http.Error(w, "Invalid date format. Use YYYY-MM-DD", http.StatusBadRequest)
			return
		}
		refTime = parsedDate
		log.Printf("Manual alert check triggered for specific date: %s (in %s)", dateStr, loc)
	} else {
		log.Println("Manual alert check triggered via API")
	}

	force := r.URL.Query().Get("force") == "true"
	alerts.CheckAndSendAlerts(s.Store, s.Config, s.Mailer, refTime, force)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Alert check triggered"))
}

func (s *Server) HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

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

func (s *Server) StatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var statuses []CardStatus
	for _, card := range s.Config.Cards {
		res, err := calculator.CalculatePayment(s.Store, card, time.Now())
		if err != nil {
			log.Printf("Error calculating for %s: %v", card.Name, err)
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(statuses)
}

func (s *Server) UploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// limit upload size
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
	defer os.Remove(tempFile.Name()) // clean up

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

	if err := s.Store.SyncTransactions(transactions); err != nil {
		http.Error(w, fmt.Sprintf("Error syncing transactions: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": fmt.Sprintf("Processed %d transactions", len(transactions)),
		"count":   len(transactions),
	})
}
