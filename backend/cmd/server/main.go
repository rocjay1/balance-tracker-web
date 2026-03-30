package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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

	ctx := context.Background()

	// Load Config (Prefer DB, fallback to config.yaml)
	cfg, err := store.GetConfig(ctx)
	if err != nil || len(cfg.Cards) == 0 {
		slog.Info("No config in database, attempting to load from config.yaml")
		cfgPath := "config.yaml"
		if envCfg := os.Getenv("CONFIG_PATH"); envCfg != "" {
			cfgPath = envCfg
		}
		
		cfgYaml, yamlErr := config.Load(cfgPath)
		if yamlErr == nil {
			slog.Info("Migrating config from yaml to database")
			if err := store.SaveConfig(ctx, cfgYaml); err != nil {
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

	mux.HandleFunc("GET /api/health", srvHandler.HealthHandler)
	mux.HandleFunc("GET /api/status", srvHandler.StatusHandler)
	mux.HandleFunc("GET /api/transactions", srvHandler.TransactionsHandler)
	mux.HandleFunc("POST /api/upload", srvHandler.UploadHandler)
	mux.HandleFunc("POST /api/alerts/test", srvHandler.TestAlertHandler)
	mux.HandleFunc("PUT /api/overrides/{account_number}", srvHandler.OverrideHandler)
	mux.HandleFunc("DELETE /api/overrides/{account_number}", srvHandler.OverrideHandler)
	mux.HandleFunc("GET /api/config", srvHandler.ConfigHandler)
	mux.HandleFunc("POST /api/config", srvHandler.ConfigHandler)
	mux.HandleFunc("POST /api/cards", srvHandler.CardHandler)
	mux.HandleFunc("DELETE /api/cards/{id}", srvHandler.CardHandler)

	// Middleware stack
	stack := middleware.Chain(
		middleware.RequestLogger,
		middleware.AllowCors,
	)

	// Create HTTP Server
	srv := &http.Server{
		Addr:         ":8080",
		Handler:      stack(mux),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Root context for cancellation
	rootCtx, cancel := context.WithCancel(context.Background())

	// Start Alert Scheduler
	go scheduler.StartAlertScheduler(rootCtx, store, cfg, mail)

	go func() {
		slog.Info("Server starting", "addr", ":8080")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("Graceful shutdown initiated")

	// Stop scheduler
	cancel()

	// Shutdown HTTP server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("Server shutdown failed", "error", err)
	} else {
		slog.Info("Server exited gracefully")
	}
}
