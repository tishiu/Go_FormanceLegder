package main

import (
	"Go_FormanceLegder/internal/auth"
	"Go_FormanceLegder/internal/config"
	"Go_FormanceLegder/internal/dashboard"
	"Go_FormanceLegder/internal/db"
	"Go_FormanceLegder/internal/ledger"
	"Go_FormanceLegder/internal/webhook"
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
)

func main() {
	ctx := context.Background()

	cfg := config.Load()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	workers := river.NewWorkers()
	river.AddWorker(workers, &webhook.Worker{DB: pool})

	riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Workers: workers,
	})
	if err != nil {
		log.Fatalf("failed to create river client: %v", err)
	}

	// Create ledger service with River client
	ledgerService := &ledger.Service{
		DB:          pool,
		RiverClient: riverClient,
	}

	ledgerHandler := &ledger.Handler{Service: ledgerService}

	authHandler := &dashboard.AuthHandler{DB: pool, Config: cfg}
	dashboardLedgerHandler := &dashboard.LedgerHandler{DB: pool}
	apiKeyHandler := &dashboard.APIKeyHandler{DB: pool, APIKeySecret: cfg.APIKeySecret}
	webhookHandler := &dashboard.WebhookHandler{DB: pool}

	apiKeyAuth := &auth.Middleware{DB: pool, APIKeySecret: cfg.APIKeySecret}

	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Dashboard Auth APIs (no auth required)
	mux.HandleFunc("/api/auth/register", authHandler.Register)
	mux.HandleFunc("/api/auth/login", authHandler.Login)
	mux.HandleFunc("/api/auth/me", authHandler.GetCurrentUser)

	// Dashboard Ledger Management APIs (JWT auth)
	mux.HandleFunc("/api/ledgers", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if r.URL.Query().Get("id") != "" {
				dashboardLedgerHandler.GetLedger(w, r)
			} else {
				dashboardLedgerHandler.ListLedgers(w, r)
			}
		case http.MethodPost:
			dashboardLedgerHandler.CreateLedger(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Dashboard API Key Management APIs (JWT auth)
	mux.HandleFunc("/api/ledgers/api-keys", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			apiKeyHandler.ListAPIKeys(w, r)
		case http.MethodPost:
			apiKeyHandler.CreateAPIKey(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/api-keys/revoke", apiKeyHandler.RevokeAPIKey)

	// Ledger APIs (API key auth)
	authWrap := func(handler http.HandlerFunc) http.Handler {
		return apiKeyAuth.AuthMiddleware(handler)
	}

	// Transaction APIs
	mux.Handle("/v1/transactions", authWrap(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			ledgerHandler.PostTransaction(w, r)
		case http.MethodGet:
			if r.URL.Query().Get("id") != "" {
				ledgerHandler.GetTransaction(w, r)
			} else {
				ledgerHandler.ListTransactions(w, r)
			}
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	// Account APIs
	mux.Handle("/v1/accounts", authWrap(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if r.URL.Query().Get("code") != "" {
				ledgerHandler.GetAccount(w, r)
			} else {
				ledgerHandler.ListAccounts(w, r)
			}
		case http.MethodPost:
			ledgerHandler.CreateAccount(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	// Event APIs
	mux.Handle("/v1/events", authWrap(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Query().Get("id") != "" {
			ledgerHandler.GetEvent(w, r)
		} else {
			ledgerHandler.ListEvents(w, r)
		}
	}))

	// Balance APIs
	mux.Handle("/v1/balance/summary", authWrap(ledgerHandler.GetBalanceSummary))
	mux.Handle("/v1/accounts/balance-history", authWrap(ledgerHandler.GetAccountBalanceHistory))

	// Webhook APIs (API key auth)
	mux.Handle("/v1/webhook-endpoints", authWrap(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			webhookHandler.ListWebhookEndpoints(w, r)
		case http.MethodPost:
			webhookHandler.CreateWebhookEndpoint(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))
	mux.Handle("/v1/webhook-deliveries", authWrap(webhookHandler.ListWebhookDeliveries))

	server := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: mux,
	}

	go func() {
		log.Printf("Server starting on port %s", cfg.ServerPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	log.Println("Shutting down server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server shutdown error: %v", err)
	}

	log.Println("Server stopped")
}
