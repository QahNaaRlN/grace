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
	"github.com/QahNaaRlN/grace/internal/commitments"
	"github.com/QahNaaRlN/grace/internal/creditcards"
	"github.com/QahNaaRlN/grace/internal/financialevents"
	"github.com/QahNaaRlN/grace/internal/imports"
	"github.com/QahNaaRlN/grace/internal/installments"
	"github.com/QahNaaRlN/grace/internal/loans"
	"github.com/QahNaaRlN/grace/internal/obligations"
	"github.com/QahNaaRlN/grace/internal/pdfextract"
	"github.com/QahNaaRlN/grace/internal/platform/config"
	"github.com/QahNaaRlN/grace/internal/platform/web"
	"github.com/QahNaaRlN/grace/internal/receiptdrafts"
	"github.com/QahNaaRlN/grace/internal/receipts"
	"github.com/QahNaaRlN/grace/internal/statementextract"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrServerClosed = http.ErrServerClosed

func New(cfg config.Config, pool *pgxpool.Pool) *http.Server {
	router := chi.NewRouter()
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORSOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
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
	aiHandler := assistant.NewAIHandler(assistant.AISettings{Enabled: cfg.AIEnabled, Key: cfg.OpenRouterAPIKey, AnalysisPolicy: cfg.AIAnalysisPolicy, ReceiptPolicy: cfg.AIReceiptPolicy, ModelGroups: cfg.AIModelGroups, PhotoDir: cfg.AIPhotoDir}, pool, analyticsRepository)
	var queryRunner assistant.QueryRunner = assistant.NewRouter(nil, nil)
	if cfg.AssistantRoutingEnabled {
		queryRunner = aiHandler.QuickQuery(assistant.NewExecutor(accountsRepository, analyticsRepository, obligationsRepository, receiptsRepository))
	}
	assistantHandler := assistant.NewHandler(queryRunner)
	var photoStore *receiptdrafts.PhotoStore
	if cfg.AIPhotoDir != "" {
		photoStore, _ = receiptdrafts.NewPhotoStore(cfg.AIPhotoDir)
	}
	aiHandler.SetReceiptStorageAvailable(photoStore != nil)
	draftHandler := receiptdrafts.NewHandler(pool, photoStore, aiHandler.ReceiptExtractor())
	if cfg.PDFExtractorSocket != "" {
		if helper, e := pdfextract.NewClient(cfg.PDFExtractorSocket); e == nil {
			draftHandler.SetPDFRenderer(helper)
		}
	}
	var identityProvider auth.Provider
	if cfg.AuthMode == "telegram" {
		identityProvider = auth.NewTelegramProvider(cfg.TelegramClientID, cfg.TelegramClientSecret, cfg.PublicURL+"/api/auth/callback")
	}
	authRepository := auth.NewRepository(pool)
	authHandler := auth.NewHandler(authRepository, identityProvider, cfg.AuthMode, cfg.PublicURL)
	sender := auth.NewSMTPSender(auth.SMTPSettings{Host: cfg.SMTPHost, Port: cfg.SMTPPort, Username: cfg.SMTPUsername, Password: cfg.SMTPPassword, From: cfg.SMTPFrom, Security: cfg.SMTPSecurity})
	emailHandler := auth.NewEmailHandler(authRepository, sender, cfg.AuthMode, cfg.PublicURL, cfg.EmailBudget)
	profileHandler := auth.NewProfileHandler(authRepository)
	passwordHandler := auth.NewPasswordHandler(authRepository, sender, cfg.AuthMode, cfg.PublicURL, cfg.EmailBudget)
	router.Route("/api/auth", func(router chi.Router) {
		router.Get("/methods", func(w http.ResponseWriter, r *http.Request) {
			if cfg.AuthMode == "password" {
				passwordHandler.Methods(w, r)
			} else {
				emailHandler.Methods(w, r)
			}
		})
		router.Post("/password/login", passwordHandler.Login)
		router.Post("/password/register", passwordHandler.Register)
		router.Post("/password/confirm", passwordHandler.Confirm)
		router.Post("/password/forgot", passwordHandler.Forgot)
		router.Post("/password/reset", passwordHandler.Reset)
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
			router.Post("/invitations", func(w http.ResponseWriter, r *http.Request) {
				if cfg.AuthMode == "password" {
					web.WriteError(w, 503, "invitation_unavailable", "Используйте приглашение для конкретной почты.")
					return
				}
				authHandler.Invite(w, r)
			})
			router.Post("/password/invitations", passwordHandler.Invite)
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
		repository := imports.NewRepository(pool)
		if cfg.PDFExtractorSocket != "" {
			if extractor, e := pdfextract.NewClient(cfg.PDFExtractorSocket); e == nil {
				repository.SetPDFExtractor(extractor)
			}
		}
		imports.NewHandler(repository).Routes(router)
		service := statementextract.NewService(pool, repository, nil)
		if cfg.AIEnabled && cfg.StatementAIEnabled {
			service.SetDocumentReasoner(aiHandler.DocumentReasoner())
		}
		statementextract.NewHandler(service).Routes(router)
	})
	router.Route("/api/assistant", func(router chi.Router) {
		router.Use(authHandler.Require)
		aiHandler.Routes(router)
	})
	router.Route("/api/receipt-drafts", func(r chi.Router) { r.Use(authHandler.Require); draftHandler.Routes(r) })
	router.Route("/api/v1", func(router chi.Router) {
		router.Use(authHandler.Require)
		router.Get("/accounts", accountsHandler.List)
		commitments.NewHandler(pool).Routes(router)
		creditcards.NewHandler(pool).Routes(router)
		loans.NewHandler(pool).Routes(router)
		installments.NewHandler(pool).Routes(router)
		router.Post("/accounts", accountsHandler.Create)
		router.Patch("/accounts/{accountId}", accountsHandler.Update)
		router.Post("/accounts/{accountId}/payment-instruments", accountsHandler.AddCard)
		router.Patch("/accounts/{accountId}/payment-instruments/{cardId}", accountsHandler.UpdateCard)
		router.Post("/accounts/{accountId}/reconciliations/preview", accountsHandler.PreviewReconciliation)
		router.Post("/accounts/{accountId}/reconciliations", accountsHandler.Reconcile)
		router.Get("/accounts/{accountId}/reconciliations", accountsHandler.Reconciliations)
		router.Get("/categories", categoriesHandler.List)
		router.Post("/categories", categoriesHandler.Create)
		router.Patch("/categories/{categoryId}", categoriesHandler.Rename)
		router.Delete("/categories/{categoryId}", categoriesHandler.Delete)
		router.Post("/categories/{categoryId}/delete-preview", categoriesHandler.DeletePreview)
		router.Put("/categories/{categoryId}/hidden", categoriesHandler.Hide)
		router.Delete("/categories/{categoryId}/hidden", categoriesHandler.Unhide)
		router.Get("/financial-events", financialEventsHandler.List)
		router.Patch("/financial-events/{eventId}", financialEventsHandler.Correct)
		router.Post("/financial-events/{eventId}/void", financialEventsHandler.Void)
		router.Get("/financial-events/{eventId}/history", financialEventsHandler.History)
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
		router.Post("/obligations/schedule-preview", obligationsHandler.PreviewSchedule)
		router.Post("/obligations/{obligationId}/scheduled-payments/{paymentId}/settlements", obligationsHandler.CreateSettlement)
		router.Get("/analytics/period-summary", analyticsHandler.PeriodSummary)
		router.Get("/analytics/statistics", analyticsHandler.Statistics)
		router.Get("/analytics/cash-position", analyticsHandler.CashPosition)
		router.Post("/assistant/query", assistantHandler.Query)
	})

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	cleanupCtx, stopCleanup := context.WithCancel(context.Background())
	server.RegisterOnShutdown(stopCleanup)
	go commitments.NewReminderWorker(pool, sender, authRepository, cfg.EmailBudget, cfg.PublicURL).Run(cleanupCtx)
	if photoStore != nil {
		go draftHandler.RunCleanup(cleanupCtx, pool)
	}
	return server
}
