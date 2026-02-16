package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/roccodavino/balance-tracker-web/backend/internal/calculator"
	"github.com/roccodavino/balance-tracker-web/backend/internal/config"
	"github.com/roccodavino/balance-tracker-web/backend/internal/csv"
	"github.com/roccodavino/balance-tracker-web/backend/internal/store"
)

type Server struct {
	Store  *store.Store
	Config *config.Config
}

func NewServer(s *store.Store, cfg *config.Config) *Server {
	return &Server{
		Store:  s,
		Config: cfg,
	}
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
