package transporthttp

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/app"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/auth"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/http/handlers"
	appmiddleware "github.com/mugiew/justqiuv2-rewrite/apps/api/internal/http/middleware"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/integrations/nexusggr"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/integrations/qris"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/modules/balances"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/modules/banks"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/modules/callmanagement"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/modules/catalog"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/modules/dashboard"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/modules/nexusggrtopup"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/modules/nexusplayers"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/modules/players"
	qrismodule "github.com/mugiew/justqiuv2-rewrite/apps/api/internal/modules/qris"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/modules/tokos"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/modules/transactions"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/modules/users"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/modules/withdrawals"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/security"
)

func NewRouter(runtime *app.Runtime) http.Handler {
	router := chi.NewRouter()

	router.Use(chimiddleware.RequestID)
	router.Use(chimiddleware.RealIP)
	router.Use(chimiddleware.Recoverer)
	router.Use(chimiddleware.Timeout(60 * time.Second))
	router.Use(appmiddleware.SecurityHeaders)
	router.Use(appmiddleware.RequestLogger(runtime.Logger))
	router.Use(appmiddleware.Metrics())

	healthHandler := handlers.NewHealthHandler(runtime)
	authRepository := auth.NewRepository(runtime.DB)
	stringCipher := security.NewStringCipher(runtime.Config.Session.TokenEncryptionKey)
	authService := auth.NewService(authRepository, stringCipher)
	authHandler := handlers.NewAuthHandler(runtime.SessionManager, authService)
	balanceRepository := balances.NewRepository(runtime.DB)
	balanceService := balances.NewService(balanceRepository)
	playerRepository := players.NewRepository(runtime.DB)
	playerService := players.NewService(playerRepository)
	nexusggrClient := nexusggr.NewClient(runtime.Config.Integrations.NexusGGR)
	qrisClient := qris.NewClient(runtime.Config.Integrations.QRIS)
	bankService := banks.NewService(runtime.DB, qrisClient)
	callManagementService := callmanagement.NewService(runtime.DB, nexusggrClient)
	catalogService := catalog.NewService(runtime.CacheRedis, nexusggrClient)
	dashboardService := dashboard.NewService(runtime.DB, runtime.CacheRedis, qrisClient, nexusggrClient, runtime.Config.App.Timezone)
	nexusPlayerService := nexusplayers.NewService(runtime.DB, playerRepository, nexusggrClient)
	tokoRepository := tokos.NewRepository(runtime.DB)
	tokoService := tokos.NewService(runtime.DB, tokoRepository)
	transactionService := transactions.NewService(runtime.DB)
	nexusggrTopupService := nexusggrtopup.NewService(runtime.DB, qrisClient)
	userService := users.NewService(runtime.DB)
	withdrawalService := withdrawals.NewService(runtime.DB, runtime.Redis, qrisClient)
	qrisService := qrismodule.NewService(runtime.DB, qrisClient, runtime.Queue, runtime.Logger)
	tokoAPIHandler := handlers.NewTokoAPIHandler(balanceService, catalogService)
	backofficeCatalogHandler := handlers.NewBackofficeCatalogHandler(catalogService)
	backofficeBanksHandler := handlers.NewBackofficeBanksHandler(bankService)
	backofficeCallManagementHandler := handlers.NewBackofficeCallManagementHandler(callManagementService)
	backofficeDashboardHandler := handlers.NewBackofficeDashboardHandler(dashboardService)
	backofficeTokosHandler := handlers.NewBackofficeTokosHandler(tokoService)
	backofficePlayersHandler := handlers.NewBackofficePlayersHandler(playerService, nexusggrClient)
	backofficeNexusggrTopupHandler := handlers.NewBackofficeNexusggrTopupHandler(nexusggrTopupService)
	backofficeUsersHandler := handlers.NewBackofficeUsersHandler(userService)
	backofficeWithdrawalHandler := handlers.NewBackofficeWithdrawalHandler(withdrawalService)
	legacyPlayerAPIHandler := handlers.NewLegacyPlayerAPIHandler(balanceService, playerService, nexusggrClient)
	legacyCallAPIHandler := handlers.NewLegacyCallAPIHandler(playerService, nexusggrClient)
	legacyPlayerMutationAPIHandler := handlers.NewLegacyPlayerMutationAPIHandler(nexusPlayerService)
	legacyQrisAPIHandler := handlers.NewLegacyQrisAPIHandler(qrisService)
	backofficeTransactionsHandler := handlers.NewBackofficeTransactionsHandler(transactionService)
	webhookHandler := handlers.NewWebhookHandler(runtime.Config, runtime.Queue)
	csrfMiddleware := security.NewCSRFMiddleware(runtime.Config)

	router.Get("/health/live", healthHandler.Live)
	router.Get("/health/ready", healthHandler.Ready)
	if runtime.Config.Observability.PrometheusEnabled {
		router.Handle("/metrics", promhttp.Handler())
	}

	router.Route("/api/webhook", func(r chi.Router) {
		r.Post("/qris", webhookHandler.QRIS)
		r.Post("/disbursement", webhookHandler.Disbursement)
	})

	router.Route("/backoffice/api", func(r chi.Router) {
		r.Use(runtime.SessionManager.LoadAndSave)
		r.Use(csrfMiddleware)

		r.Route("/auth", func(r chi.Router) {
			r.Get("/bootstrap", authHandler.Bootstrap)
			r.Post("/login", authHandler.Login)
			r.Post("/register", authHandler.Register)
			r.Post("/mfa/login/verify", authHandler.VerifyLoginMFA)

			r.Group(func(r chi.Router) {
				r.Use(auth.BackofficeSessionAuth(runtime.SessionManager, authService))
				r.Get("/me", authHandler.Me)
				r.Get("/mfa", authHandler.MFAStatus)
				r.Post("/mfa/setup", authHandler.BeginMFASetup)
				r.Post("/mfa/confirm", authHandler.ConfirmMFASetup)
				r.Post("/mfa/disable", authHandler.DisableMFA)
				r.Post("/logout", authHandler.Logout)
			})
		})

		r.Group(func(r chi.Router) {
			r.Use(auth.BackofficeSessionAuth(runtime.SessionManager, authService))
			r.Get("/dashboard/overview", backofficeDashboardHandler.Overview)
			r.Get("/dashboard/operational-pulse", backofficeDashboardHandler.OperationalPulse)
			r.Get("/catalog/providers", backofficeCatalogHandler.Providers)
			r.Get("/catalog/games", backofficeCatalogHandler.Games)
			r.Get("/call-management/bootstrap", backofficeCallManagementHandler.Bootstrap)
			r.Get("/call-management/active-players", backofficeCallManagementHandler.ActivePlayers)
			r.Post("/call-management/call-list", backofficeCallManagementHandler.CallList)
			r.Post("/call-management/apply", backofficeCallManagementHandler.Apply)
			r.Post("/call-management/history", backofficeCallManagementHandler.History)
			r.Post("/call-management/cancel", backofficeCallManagementHandler.Cancel)
			r.Post("/call-management/control-rtp", backofficeCallManagementHandler.ControlRTP)
			r.Post("/call-management/control-users-rtp", backofficeCallManagementHandler.ControlUsersRTP)
			r.Get("/nexusggr-topup/bootstrap", backofficeNexusggrTopupHandler.Bootstrap)
			r.Post("/nexusggr-topup/generate", backofficeNexusggrTopupHandler.Generate)
			r.Post("/nexusggr-topup/check-status", backofficeNexusggrTopupHandler.CheckStatus)
			r.Get("/banks", backofficeBanksHandler.List)
			r.Post("/banks", backofficeBanksHandler.Create)
			r.Patch("/banks/{bankID}", backofficeBanksHandler.Update)
			r.Post("/banks/inquiry", backofficeBanksHandler.Inquiry)
			r.Get("/withdrawal/bootstrap", backofficeWithdrawalHandler.Bootstrap)
			r.Post("/withdrawal/inquiry", backofficeWithdrawalHandler.Inquiry)
			r.Post("/withdrawal/submit", backofficeWithdrawalHandler.Submit)
			r.Get("/users", backofficeUsersHandler.List)
			r.Post("/users", backofficeUsersHandler.Create)
			r.Get("/users/{userID}", backofficeUsersHandler.Detail)
			r.Patch("/users/{userID}", backofficeUsersHandler.Update)
			r.Get("/tokos", backofficeTokosHandler.List)
			r.Post("/tokos", backofficeTokosHandler.Create)
			r.Get("/tokos/{tokoID}", backofficeTokosHandler.Detail)
			r.Patch("/tokos/{tokoID}", backofficeTokosHandler.Update)
			r.Post("/tokos/{tokoID}/regenerate-token", backofficeTokosHandler.RegenerateToken)
			r.Get("/players", backofficePlayersHandler.List)
			r.Get("/players/{playerID}/money-info", backofficePlayersHandler.MoneyInfo)
			r.Get("/transactions", backofficeTransactionsHandler.List)
			r.Get("/transactions/export", backofficeTransactionsHandler.Export)
			r.Get("/transactions/{transactionID}", backofficeTransactionsHandler.Detail)
		})
	})

	router.Route("/api/v1", func(r chi.Router) {
		r.Use(auth.TokoBearerAuth(authService))
		r.Get("/providers", tokoAPIHandler.ProviderList)
		r.Post("/games", tokoAPIHandler.GameList)
		r.Post("/games/v2", tokoAPIHandler.GameListV2)
		r.Post("/user/create", legacyPlayerMutationAPIHandler.UserCreate)
		r.Post("/user/deposit", legacyPlayerMutationAPIHandler.UserDeposit)
		r.Post("/user/withdraw", legacyPlayerMutationAPIHandler.UserWithdraw)
		r.Post("/user/withdraw-reset", legacyPlayerMutationAPIHandler.UserWithdrawReset)
		r.Post("/transfer/status", legacyPlayerMutationAPIHandler.TransferStatus)
		r.Post("/generate", legacyQrisAPIHandler.Generate)
		r.Post("/check-status", legacyQrisAPIHandler.CheckStatus)
		r.Post("/game/launch", legacyPlayerAPIHandler.GameLaunch)
		r.Post("/money/info", legacyPlayerAPIHandler.MoneyInfo)
		r.Post("/game/log", legacyPlayerAPIHandler.GameLog)
		r.Get("/call/players", legacyCallAPIHandler.CallPlayers)
		r.Post("/call/list", legacyCallAPIHandler.CallList)
		r.Post("/call/apply", legacyCallAPIHandler.CallApply)
		r.Post("/call/history", legacyCallAPIHandler.CallHistory)
		r.Post("/call/cancel", legacyCallAPIHandler.CallCancel)
		r.Post("/control/rtp", legacyCallAPIHandler.ControlRtp)
		r.Post("/control/users-rtp", legacyCallAPIHandler.ControlUsersRtp)
		r.Get("/balance", tokoAPIHandler.Balance)
		r.Post("/merchant-active", tokoAPIHandler.MerchantActive)
	})

	return router
}
