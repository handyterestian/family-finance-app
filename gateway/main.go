package main

import (
	"log"
	"net/http"
	"os"

	"family-finance/gateway/grpcclient"
	"family-finance/gateway/handlers"
	"family-finance/gateway/middleware"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func main() {
	// Inisialisasi koneksi gRPC ke server
	if err := grpcclient.Init(); err != nil {
		log.Fatalf("[Gateway] Gagal koneksi ke gRPC server: %v", err)
	}
	defer grpcclient.Conn.Close()
	log.Println("[Gateway] Koneksi gRPC berhasil")

	r := chi.NewRouter()

	// ── Middleware global ─────────────────────────────────────
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{getenv("CORS_ORIGIN", "http://localhost:8080")},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// ── Health check ──────────────────────────────────────────
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// ── Auth routes (tidak perlu auth middleware) ─────────────
	r.Post("/auth/register", handlers.AuthRegister)
	r.Post("/auth/login", handlers.AuthLogin)

	// ── Protected routes ──────────────────────────────────────
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireAuth)

		r.Post("/auth/logout", handlers.AuthLogout)
		r.Get("/auth/me", handlers.AuthMe)

		// Categories
		r.Get("/categories", handlers.CategoryList)
		r.Post("/categories", handlers.CategoryCreate)
		r.Put("/categories/{name}", handlers.CategoryUpdate)
		r.Delete("/categories/{name}", handlers.CategoryDelete)

		// Wallets
		r.Get("/wallets", handlers.WalletList)
		r.Post("/wallets", handlers.WalletCreate)
		r.Post("/wallets/transfer", handlers.WalletTransfer)
		r.Put("/wallets/{id}", handlers.WalletUpdate)
		r.Delete("/wallets/{id}", handlers.WalletDelete)

		// Transactions
		r.Get("/transactions", handlers.TransactionList)
		r.Post("/transactions", handlers.TransactionCreate)
		r.Put("/transactions/{id}", handlers.TransactionUpdate)
		r.Delete("/transactions/{id}", handlers.TransactionDelete)

		// Budgets
		r.Get("/budgets/emergency", handlers.EmergencyFundGet)
		r.Put("/budgets/emergency", handlers.EmergencyFundSet)
		r.Post("/budgets/emergency/deposit", handlers.EmergencyFundDeposit)
		r.Get("/budgets/emergency/history", handlers.EmergencyFundHistory)
		r.Get("/budgets/recurring", handlers.RecurringBudgetList)
		r.Post("/budgets/recurring", handlers.RecurringBudgetSet)
		r.Post("/budgets/recurring/apply", handlers.RecurringBudgetApply)
		r.Delete("/budgets/recurring/{id}", handlers.RecurringBudgetDelete)
		r.Get("/budgets", handlers.BudgetList)
		r.Post("/budgets", handlers.BudgetSet)
		r.Put("/budgets/{id}", handlers.BudgetUpdate)
		r.Delete("/budgets/{id}", handlers.BudgetDelete)

		// Debts
		r.Get("/debts", handlers.DebtList)
		r.Post("/debts", handlers.DebtCreate)
		r.Put("/debts/{id}", handlers.DebtUpdate)
		r.Post("/debts/{id}/pay", handlers.DebtPay)
		r.Get("/debts/{id}/payments", handlers.DebtPaymentList)
		r.Delete("/debts/{id}", handlers.DebtDelete)

		// Savings
		r.Get("/savings", handlers.SavingList)
		r.Post("/savings", handlers.SavingCreate)
		r.Put("/savings/{id}", handlers.SavingUpdate)
		r.Post("/savings/{id}/deposit", handlers.SavingDeposit)
		r.Delete("/savings/{id}", handlers.SavingDelete)

		// Dashboard
		r.Get("/dashboard", handlers.DashboardGet)
		r.Post("/dashboard/seed", handlers.DashboardSeed)
		r.Delete("/dashboard/clear", handlers.DashboardClear)

		// Backup & Restore
		r.Get("/backup", handlers.BackupExport)
		r.Post("/restore", handlers.BackupRestore)
	})

	port := getenv("GATEWAY_PORT", "8000")
	log.Printf("[Gateway] HTTP berjalan di port %s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("[Gateway] ListenAndServe error: %v", err)
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
