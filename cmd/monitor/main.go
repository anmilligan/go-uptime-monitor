package main

import (
	"Uptime_Monitor/internal/checker"
	"Uptime_Monitor/internal/notifier"
	"Uptime_Monitor/internal/repository/postgres"
	"Uptime_Monitor/internal/scheduler"
	"Uptime_Monitor/internal/transport/rest"
	"Uptime_Monitor/internal/usecase"
	"Uptime_Monitor/internal/worker"
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	telegramtransport "Uptime_Monitor/internal/transport/telegram"

	"github.com/go-chi/chi/v5"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx := context.Background()

	databaseURL := os.Getenv("DATABASE_URL")

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		panic(err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		panic(err)
	}

	fmt.Println("database connected")

	targetRepo := postgres.NewTargetRepository(pool)
	checkRepo := postgres.NewCheckRepository(pool)
	incidentRepo := postgres.NewIncidentRepository(pool)
	alertRepo := postgres.NewAlertRepository(pool)
	userRepo := postgres.NewUserRepository(pool)

	telegramToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if telegramToken == "" {
		panic("TELEGRAM_BOT_TOKEN is not set")
	}
	telegramAPI, err := tgbotapi.NewBotAPIWithClient(
		telegramToken,
		tgbotapi.APIEndpoint,
		&http.Client{
			Timeout: 35 * time.Second,
		},
	)
	if err != nil {
		panic("failed to initialize Telegram API")
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

	router := chi.NewRouter()
	router.Get("/healthz", healthHandler.Health)
	router.Get("/readyz", healthHandler.Ready)
	router.Post("/api/v1/monitors", targetHandler.Create)
	router.Get("/api/v1/monitors/{id}", targetHandler.GetByID)
	router.Get("/api/v1/monitors", targetHandler.List)
	router.Delete("/api/v1/monitors/{id}", targetHandler.Delete)
	router.Patch("/api/v1/monitors/{id}", targetHandler.Update)
	router.Get("/api/v1/monitors/{id}/checks", targetHandler.GetChecks)
	router.Get("/api/v1/monitors/{id}/incidents", targetHandler.GetIncidents)

	fmt.Println("server started on :8080")

	if err := http.ListenAndServe(":8080", router); err != nil {
		panic(err)
	}
}
