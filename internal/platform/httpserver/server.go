package httpserver

import (
	"context"
	"net/http"
	"time"

	"github.com/QahNaaRlN/grace/internal/accounts"
	"github.com/QahNaaRlN/grace/internal/categories"
	"github.com/QahNaaRlN/grace/internal/financialevents"
	"github.com/QahNaaRlN/grace/internal/platform/config"
	"github.com/QahNaaRlN/grace/internal/platform/web"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrServerClosed = http.ErrServerClosed

func New(cfg config.Config, pool *pgxpool.Pool) *http.Server {
	router := chi.NewRouter()
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins: cfg.CORSOrigins,
		AllowedMethods: []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type", "Idempotency-Key"},
	}))

	accountsHandler := accounts.NewHandler(accounts.NewRepository(pool))
	categoriesHandler := categories.NewHandler(categories.NewRepository(pool))
	financialEventsHandler := financialevents.NewHandler(financialevents.NewRepository(pool))

	router.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		web.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	router.Get("/ready", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := pool.Ping(ctx); err != nil {
			web.WriteError(w, http.StatusServiceUnavailable, "not_ready", "База данных недоступна.")
			return
		}
		web.WriteJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})
	router.Get("/api/meta", func(w http.ResponseWriter, _ *http.Request) {
		web.WriteJSON(w, http.StatusOK, map[string]string{"service": "grace", "version": "0.2.0-dev"})
	})
	router.Route("/api/v1", func(router chi.Router) {
		router.Get("/accounts", accountsHandler.List)
		router.Post("/accounts", accountsHandler.Create)
		router.Get("/categories", categoriesHandler.List)
		router.Get("/financial-events", financialEventsHandler.List)
		router.Post("/financial-events/purchases", financialEventsHandler.CreatePurchase)
		router.Post("/financial-events/incomes", financialEventsHandler.CreateIncome)
		router.Post("/financial-events/transfers", financialEventsHandler.CreateTransfer)
		router.Post("/financial-events/refunds", financialEventsHandler.CreateRefund)
		router.Post("/financial-events/debt-payments", financialEventsHandler.CreateDebtPayment)
	})

	return &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}
