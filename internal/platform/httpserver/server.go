package httpserver

import (
	"context"
	"net/http"
	"time"

	"github.com/QahNaaRlN/grace/internal/accounts"
	"github.com/QahNaaRlN/grace/internal/analytics"
	"github.com/QahNaaRlN/grace/internal/assistant"
	"github.com/QahNaaRlN/grace/internal/auth"
	"github.com/QahNaaRlN/grace/internal/categories"
	"github.com/QahNaaRlN/grace/internal/financialevents"
	"github.com/QahNaaRlN/grace/internal/imports"
	"github.com/QahNaaRlN/grace/internal/obligations"
	"github.com/QahNaaRlN/grace/internal/platform/config"
	"github.com/QahNaaRlN/grace/internal/platform/web"
	"github.com/QahNaaRlN/grace/internal/receipts"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrServerClosed = http.ErrServerClosed

func New(cfg config.Config, pool *pgxpool.Pool) *http.Server {
	router := chi.NewRouter()
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORSOrigins,
		AllowedMethods:   []string{"GET", "POST", "DELETE", "PATCH", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "Idempotency-Key", "X-CSRF-Token"},
		AllowCredentials: true,
	}))

	accountsRepository := accounts.NewRepository(pool)
	analyticsRepository := analytics.NewRepository(pool)
	obligationsRepository := obligations.NewRepository(pool)
	receiptsRepository := receipts.NewRepository(pool)
	accountsHandler := accounts.NewHandler(accountsRepository)
	categoriesHandler := categories.NewHandler(categories.NewRepository(pool))
	financialEventsHandler := financialevents.NewHandler(financialevents.NewRepository(pool))
	receiptsHandler := receipts.NewHandler(receiptsRepository)
	obligationsHandler := obligations.NewHandler(obligationsRepository)
	analyticsHandler := analytics.NewHandler(analyticsRepository)
	assistantRouter := assistant.NewRouter(nil, nil)
	if cfg.AssistantRoutingEnabled && cfg.TypeSafeAPIKey != "" {
		provider, err := assistant.NewTypeSafeProvider(cfg.TypeSafeAPIKey, &http.Client{})
		if err == nil {
			executor := assistant.NewExecutor(accountsRepository, analyticsRepository, obligationsRepository, receiptsRepository)
			assistantRouter = assistant.NewRouter(provider, executor)
		}
	}
	assistantHandler := assistant.NewHandler(assistantRouter)
	var identityProvider auth.Provider
	if cfg.AuthMode == "telegram" {
		identityProvider = auth.NewTelegramProvider(cfg.TelegramClientID, cfg.TelegramClientSecret, cfg.PublicURL+"/api/auth/callback")
	}
	authRepository := auth.NewRepository(pool)
	authHandler := auth.NewHandler(authRepository, identityProvider, cfg.AuthMode, cfg.PublicURL)
	sender := auth.NewSMTPSender(auth.SMTPSettings{Host: cfg.SMTPHost, Port: cfg.SMTPPort, Username: cfg.SMTPUsername, Password: cfg.SMTPPassword, From: cfg.SMTPFrom, Security: cfg.SMTPSecurity})
	emailHandler := auth.NewEmailHandler(authRepository, sender, cfg.AuthMode, cfg.PublicURL, cfg.EmailBudget)
	profileHandler := auth.NewProfileHandler(authRepository)
	router.Route("/api/auth", func(router chi.Router) {
		router.Get("/methods", emailHandler.Methods)
		router.Post("/email/request", emailHandler.Request)
		router.Post("/email/verify", emailHandler.Verify)
		router.Post("/login", authHandler.Login)
		router.Get("/callback", authHandler.Callback)
		router.Group(func(router chi.Router) {
			router.Use(authHandler.Require)
			router.Get("/me", authHandler.Me)
			router.Get("/profile", profileHandler.Get)
			router.Post("/profile", profileHandler.Update)
			router.Post("/profile/email/request", emailHandler.RequestLink)
			router.Post("/profile/email/verify", emailHandler.VerifyLink)
			router.Get("/profile/avatar", profileHandler.GetAvatar)
			router.Post("/profile/avatar", profileHandler.UploadAvatar)
			router.Delete("/profile/avatar", profileHandler.DeleteAvatar)
			router.Post("/logout", authHandler.Logout)
			router.Post("/revoke-sessions", authHandler.RevokeSessions)
			router.Post("/invitations", authHandler.Invite)
			router.Post("/users/{userId}/disable", func(w http.ResponseWriter, r *http.Request) {
				r.SetPathValue("userId", chi.URLParam(r, "userId"))
				authHandler.Disable(w, r)
			})
		})
	})

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
		web.WriteJSON(w, http.StatusOK, map[string]string{"service": "grace", "version": "0.6.0-dev"})
	})
	router.Route("/api/imports", func(router chi.Router) {
		router.Use(authHandler.Require)
		imports.NewHandler(imports.NewRepository(pool)).Routes(router)
	})
	router.Route("/api/v1", func(router chi.Router) {
		router.Use(authHandler.Require)
		router.Get("/accounts", accountsHandler.List)
		router.Post("/accounts", accountsHandler.Create)
		router.Get("/categories", categoriesHandler.List)
		router.Get("/financial-events", financialEventsHandler.List)
		router.Post("/financial-events/purchases", financialEventsHandler.CreatePurchase)
		router.Post("/financial-events/incomes", financialEventsHandler.CreateIncome)
		router.Post("/financial-events/transfers", financialEventsHandler.CreateTransfer)
		router.Post("/financial-events/refunds", financialEventsHandler.CreateRefund)
		router.Post("/financial-events/debt-payments", financialEventsHandler.CreateDebtPayment)
		router.Get("/receipts", receiptsHandler.List)
		router.Get("/receipts/{receiptId}", receiptsHandler.Get)
		router.Post("/receipts", receiptsHandler.Create)
		router.Get("/obligations", obligationsHandler.List)
		router.Get("/obligations/{obligationId}", obligationsHandler.Get)
		router.Post("/obligations", obligationsHandler.Create)
		router.Post("/obligations/{obligationId}/scheduled-payments/{paymentId}/settlements", obligationsHandler.CreateSettlement)
		router.Get("/analytics/period-summary", analyticsHandler.PeriodSummary)
		router.Get("/analytics/cash-position", analyticsHandler.CashPosition)
		router.Post("/assistant/query", assistantHandler.Query)
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
