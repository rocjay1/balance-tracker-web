package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/roccodavino/balance-tracker-web/backend/internal/api"
	"github.com/roccodavino/balance-tracker-web/backend/internal/config"
	"github.com/roccodavino/balance-tracker-web/backend/internal/mailer"
	"github.com/roccodavino/balance-tracker-web/backend/internal/middleware"
	"github.com/roccodavino/balance-tracker-web/backend/internal/scheduler"
	"github.com/roccodavino/balance-tracker-web/backend/internal/store"
)

func main() {
	// Configure structured JSON logging
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	// 1. Load Store
	dbPath := "finance.db"
	if envDB := os.Getenv("DB_PATH"); envDB != "" {
		dbPath = envDB
	}

	store, err := store.New(dbPath)
	if err != nil {
		log.Fatalf("Error opening store: %v", err)
	}
	defer store.Close()

	// 2. Load Config
	cfgPath := "config.yaml"
	if envCfg := os.Getenv("CONFIG_PATH"); envCfg != "" {
		cfgPath = envCfg
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	// 3. Initialize Mailer
	mail := mailer.New(cfg.SMTP, cfg.SMTP.Password)

	// 4. Initialize Server Handlers
	srvHandler := api.NewServer(store, cfg, mail)
	mux := http.NewServeMux()

	// Register Routes
	mux.HandleFunc("/api/health", srvHandler.HealthHandler)
	mux.HandleFunc("/api/status", middleware.AllowCors(srvHandler.StatusHandler))
	mux.HandleFunc("/api/upload", middleware.AllowCors(srvHandler.UploadHandler))
	mux.HandleFunc("/api/alerts/test", middleware.AllowCors(srvHandler.TestAlertHandler))

	// 5. Create HTTP Server with request logging
	srv := &http.Server{
		Addr:         ":8080",
		Handler:      middleware.RequestLogger(mux),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// 6. Start Alert Scheduler
	go scheduler.StartAlertScheduler(store, cfg, mail)

	slog.Info("server starting", "addr", ":8080")
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
