package main

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

func allowCors(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

func main() {
	// 2. Load Config & Store
	dbPath := "finance.db"
	if envDB := os.Getenv("DB_PATH"); envDB != "" {
		dbPath = envDB
	}

	store, err := store.New(dbPath)
	if err != nil {
		log.Fatalf("Error opening store: %v", err)
	}
	defer store.Close()

	// Load config for cards
	cfgPath := "config.yaml"
	if envCfg := os.Getenv("CONFIG_PATH"); envCfg != "" {
		cfgPath = envCfg
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	// 2a. Create ServeMux
	mux := http.NewServeMux()

	// 3. Register Routes
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	mux.HandleFunc("/api/status", allowCors(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
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

		var statuses []CardStatus
		for _, card := range cfg.Cards {
			res, err := calculator.CalculatePayment(store, card, time.Now())
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
	}))

	mux.HandleFunc("/api/upload", allowCors(func(w http.ResponseWriter, r *http.Request) {
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
		// Ideally refactor csv.Parse to take io.Reader, but for now we save.
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

		if err := store.SyncTransactions(transactions); err != nil {
			http.Error(w, fmt.Sprintf("Error syncing transactions: %v", err), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": fmt.Sprintf("Processed %d transactions", len(transactions)),
			"count":   len(transactions),
		})
	}))

	// 3. Create Server
	srv := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// 4. Initialize Mailer
	mail := mailer.New(cfg.SMTP, cfg.SMTP.Password)

	// 5. Start Alert Scheduler
	go startAlertScheduler(store, cfg, mail)

	log.Println("Server starting on :8080...")
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func startAlertScheduler(s *store.Store, cfg *config.Config, m *mailer.Mailer) {
	// Calculate time until next 7:00 AM
	now := time.Now()
	nextRun := time.Date(now.Year(), now.Month(), now.Day(), 7, 0, 0, 0, now.Location())
	if now.After(nextRun) {
		nextRun = nextRun.Add(24 * time.Hour)
	}

	timer := time.NewTimer(time.Until(nextRun))
	defer timer.Stop()

	log.Printf("Alert scheduler started. Next run at: %v", nextRun)

	for {
		<-timer.C

		log.Println("Running daily alert check...")
		alerts.CheckAndSendAlerts(s, cfg, m)

		// Reset for next day
		timer.Reset(24 * time.Hour)
	}
}
