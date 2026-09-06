package main

import (
	"Uptime_Monitor/internal/repository/postgres"
	"Uptime_Monitor/internal/transport/rest"
	"Uptime_Monitor/internal/usecase"
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
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
	addTargetUseCase := usecase.NewAddTargetUseCase(targetRepo)
	getTargetUseCase := usecase.NewGetTargetUseCase(targetRepo)
	listTargetsUseCase := usecase.NewListTargetsUseCase(targetRepo)
	deleteTargetUseCase := usecase.NewDeleteTargetUseCase(targetRepo)
	updateTargetUseCase := usecase.NewUpdateTargetUseCase(targetRepo)
	targetHandler := rest.NewTargetHandler(addTargetUseCase, getTargetUseCase, listTargetsUseCase, deleteTargetUseCase, updateTargetUseCase)

	router := chi.NewRouter()
	router.Post("/api/v1/monitors", targetHandler.Create)
	router.Get("/api/v1/monitors/{id}", targetHandler.GetByID)
	router.Get("/api/v1/monitors", targetHandler.List)
	router.Delete("/api/v1/monitors/{id}", targetHandler.Delete)
	router.Patch("/api/v1/monitors/{id}", targetHandler.Update)

	fmt.Println("server started on :8080")

	if err := http.ListenAndServe(":8080", router); err != nil {
		panic(err)
	}
}
