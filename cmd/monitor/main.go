package main

import (
	"Uptime_Monitor/internal/checker"
	"Uptime_Monitor/internal/metrics"
	"Uptime_Monitor/internal/notifier"
	"Uptime_Monitor/internal/repository/postgres"
	"Uptime_Monitor/internal/scheduler"
	"Uptime_Monitor/internal/transport/rest"
	"Uptime_Monitor/internal/usecase"
	"Uptime_Monitor/internal/worker"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	telegramtransport "Uptime_Monitor/internal/transport/telegram"

	"github.com/go-chi/chi/v5"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {

	logger := slog.New(
		slog.NewJSONHandler(
			os.Stdout,
			nil,
		),
	)

	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	databaseURL := os.Getenv("DATABASE_URL")

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		slog.Error(
			"failed to create database pool",
			"error", err,
		)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		slog.Error(
			"failed to connect to database",
			"error", err,
		)
		os.Exit(1)
	}

	slog.Info("database connected")

	targetRepo := postgres.NewTargetRepository(pool)
	checkRepo := postgres.NewCheckRepository(pool)
	incidentRepo := postgres.NewIncidentRepository(pool)
	alertRepo := postgres.NewAlertRepository(pool)
	userRepo := postgres.NewUserRepository(pool)
	metricsRepo := postgres.NewMetricsRepository(pool)

	telegramToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if telegramToken == "" {
		slog.Error("TELEGRAM_BOT_TOKEN is not set")
		os.Exit(1)
	}
	telegramAPI, err := tgbotapi.NewBotAPIWithClient(
		telegramToken,
		tgbotapi.APIEndpoint,
		&http.Client{
			Timeout: 35 * time.Second,
		},
	)
	if err != nil {
		slog.Error(
			"failed to initialize Telegram API",
			"error", err,
		)
		os.Exit(1)
	}

	telegramNotifier := notifier.NewTelegramNotifier(telegramAPI)

	registerUserUseCase := usecase.NewRegisterTelegramUserUseCase(userRepo)
	addTargetUseCase := usecase.NewAddTargetUseCase(targetRepo)
	getTargetUseCase := usecase.NewGetTargetUseCase(targetRepo)
	listTargetsUseCase := usecase.NewListTargetsUseCase(targetRepo)
	updateTargetUseCase := usecase.NewUpdateTargetUseCase(targetRepo)
	deleteTargetUseCase := usecase.NewDeleteTargetUseCase(targetRepo)
	pauseTargetUseCase := usecase.NewPauseTargetUseCase(targetRepo)
	resumeTargetUseCase := usecase.NewResumeTargetUseCase(targetRepo)
	getLastCheckUseCase := usecase.NewGetLastCheckUseCase(checkRepo)
	getCheckHistoryUseCase := usecase.NewGetCheckHistoryUseCase(checkRepo)
	getOpenIncidentUseCase := usecase.NewGetOpenIncidentUseCase(incidentRepo)
	listIncidentsUseCase := usecase.NewListIncidentsUseCase(incidentRepo)
	calculateUptimeUseCase := usecase.NewCalculateUptimeUseCase(incidentRepo)
	createAlertUseCase := usecase.NewCreateAlertUseCase(alertRepo, userRepo, telegramNotifier)
	processCheckUseCase := usecase.NewProcessCheckUseCase(checkRepo, incidentRepo, createAlertUseCase)
	cleanupCheckHistoryUseCase := usecase.NewCleanupCheckHistoryUseCase(checkRepo, 30*24*time.Hour)

	// Telegram transport.
	telegramBot := telegramtransport.NewBot(
		telegramAPI,
		registerUserUseCase,
		addTargetUseCase,
		listTargetsUseCase,
		getLastCheckUseCase,
		getOpenIncidentUseCase,
		calculateUptimeUseCase,
		listIncidentsUseCase,
		deleteTargetUseCase,
		pauseTargetUseCase,
		resumeTargetUseCase,
		telegramNotifier,
	)

	go telegramBot.Run(ctx)

	// REST transport.
	targetHandler := rest.NewTargetHandler(
		addTargetUseCase,
		getTargetUseCase,
		listTargetsUseCase,
		deleteTargetUseCase,
		updateTargetUseCase,
		getLastCheckUseCase,
		getOpenIncidentUseCase,
		calculateUptimeUseCase,
		getCheckHistoryUseCase,
		listIncidentsUseCase,
	)

	healthHandler := rest.NewHealthHandler(pool)

	httpChecker := checker.NewHTTPChecker()
	tcpChecker := checker.NewTCPChecker()

	targetChecker := checker.NewTargetChecker(httpChecker, tcpChecker)

	workerPool := worker.NewPool(
		4,
		targetChecker,
		processCheckUseCase,
		3,
	)

	workerPool.Start(ctx)

	monitorScheduler := scheduler.NewScheduler(targetRepo, workerPool)

	go monitorScheduler.Run(ctx)

	cleanupScheduler := scheduler.NewCleanupScheduler(cleanupCheckHistoryUseCase, 24*time.Hour)

	go cleanupScheduler.Run(ctx)

	router := chi.NewRouter()
	router.Get("/healthz", healthHandler.Health)
	router.Get("/readyz", healthHandler.Ready)
	router.Handle("/metrics", metrics.Handler(metricsRepo, calculateUptimeUseCase))

	router.Post("/api/v1/monitors", targetHandler.Create)
	router.Get("/api/v1/monitors/{id}", targetHandler.GetByID)
	router.Get("/api/v1/monitors", targetHandler.List)
	router.Delete("/api/v1/monitors/{id}", targetHandler.Delete)
	router.Patch("/api/v1/monitors/{id}", targetHandler.Update)
	router.Get("/api/v1/monitors/{id}/checks", targetHandler.GetChecks)
	router.Get("/api/v1/monitors/{id}/incidents", targetHandler.GetIncidents)

	server := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	go func() {
		slog.Info(
			"HTTP server started",
			"address", server.Addr,
		)

		if err := server.ListenAndServe(); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {

			slog.Error(
				"HTTP server failed",
				"error", err,
			)

			stop()
		}
	}()

	<-ctx.Done()

	slog.Info("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error(
			"HTTP server shutdown failed",
			"error", err,
		)
	} else {
		slog.Info("HTTP server stopped")
	}

	slog.Info("waiting for workers to finish")

	workerShutdownCtx, workerCancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer workerCancel()

	if err := workerPool.Wait(workerShutdownCtx); err != nil {
		slog.Warn(
			"workers did not stop before shutdown timeout",
			"error", err,
		)
	} else {
		slog.Info("workers stopped")
	}
}
