package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/rocjay1/balance-tracker-web/backend/internal/api"
	"github.com/rocjay1/balance-tracker-web/backend/internal/config"
	"github.com/rocjay1/balance-tracker-web/backend/internal/mailer"
	"github.com/rocjay1/balance-tracker-web/backend/internal/middleware"
	"github.com/rocjay1/balance-tracker-web/backend/internal/scheduler"
	"github.com/rocjay1/balance-tracker-web/backend/internal/store"
)

func main() {
	// Configure structured JSON logging
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	// Load Store
	dbPath := "finance.db"
	if envDB := os.Getenv("DB_PATH"); envDB != "" {
		dbPath = envDB
	}

	store, err := store.New(dbPath)
	if err != nil {
		slog.Error("Error opening store", "error", err)
		os.Exit(1)
	}
	defer store.Close()

	// Load Config (Prefer DB, fallback to config.yaml)
	cfg, err := store.GetConfig()
	if err != nil || len(cfg.Cards) == 0 {
		slog.Info("No config in database, attempting to load from config.yaml")
		cfgPath := "config.yaml"
		if envCfg := os.Getenv("CONFIG_PATH"); envCfg != "" {
			cfgPath = envCfg
		}
		
		cfgYaml, yamlErr := config.Load(cfgPath)
		if yamlErr == nil {
			slog.Info("Migrating config from yaml to database")
			if err := store.SaveConfig(cfgYaml); err != nil {
				slog.Error("Failed to migrate config to database", "error", err)
			} else {
				cfg = cfgYaml
			}
		} else if err != nil {
			// If we had a real DB error and no YAML, we can't continue
			slog.Error("Error loading config from database and config.yaml not found", "db_error", err, "yaml_error", yamlErr)
			os.Exit(1)
		} else if len(cfg.Cards) == 0 {
			slog.Error("No configuration found in database or config.yaml")
			os.Exit(1)
		}
	}

	// Initialize Mailer
	mail := mailer.New(cfg.SMTP, cfg.SMTP.Password)

	// Initialize Server Handlers
	srvHandler := api.NewServer(store, cfg, mail)
	mux := http.NewServeMux()

	mux.HandleFunc("/api/health", srvHandler.HealthHandler)
	mux.HandleFunc("GET /api/status", middleware.AllowCors(srvHandler.StatusHandler))
	mux.HandleFunc("GET /api/transactions", middleware.AllowCors(srvHandler.TransactionsHandler))
	mux.HandleFunc("POST /api/upload", middleware.AllowCors(srvHandler.UploadHandler))
	mux.HandleFunc("POST /api/alerts/test", middleware.AllowCors(srvHandler.TestAlertHandler))
	mux.HandleFunc("PUT /api/overrides/{account_number}", middleware.AllowCors(srvHandler.OverrideHandler))
	mux.HandleFunc("DELETE /api/overrides/{account_number}", middleware.AllowCors(srvHandler.OverrideHandler))
	mux.HandleFunc("OPTIONS /api/status", middleware.AllowCors(srvHandler.HealthHandler))
	mux.HandleFunc("OPTIONS /api/transactions", middleware.AllowCors(srvHandler.HealthHandler))
	mux.HandleFunc("OPTIONS /api/upload", middleware.AllowCors(srvHandler.HealthHandler))
	mux.HandleFunc("OPTIONS /api/alerts/test", middleware.AllowCors(srvHandler.HealthHandler))
	mux.HandleFunc("OPTIONS /api/overrides/{account_number}", middleware.AllowCors(srvHandler.HealthHandler))

	// Create HTTP Server with request logging
	srv := &http.Server{
		Addr:         ":8080",
		Handler:      middleware.RequestLogger(mux),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start Alert Scheduler
	go scheduler.StartAlertScheduler(store, cfg, mail)

	slog.Info("Server starting", "addr", ":8080")
	if err := srv.ListenAndServe(); err != nil {
		slog.Error("Server failed", "error", err)
		os.Exit(1)
	}
}
