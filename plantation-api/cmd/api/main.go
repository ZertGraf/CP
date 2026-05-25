package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"plantation-api/internal/handler"
	"plantation-api/internal/middleware"
	"plantation-api/internal/simulation"
	"plantation-api/internal/storage"
	"plantation-api/internal/ws"
)

func main() {
	dsn := env("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/plantation?sslmode=disable")
	secret := env("JWT_SECRET", "super-secret-key")
	port := env("PORT", "8080")

	store := storage.New(dsn)
	hub := ws.NewHub()

	// seed test data if database is empty
	if err := store.Seed(context.Background()); err != nil {
		log.Printf("warning: seed failed: %v", err)
	}

	authH := handler.NewAuthHandler(store, secret)
	sectorH := handler.NewSectorHandler(store)
	plantH := handler.NewPlantHandler(store)
	waterH := handler.NewWateringHandler(store, hub)
	fileH := handler.NewFileHandler(store)
	notifH := handler.NewNotificationHandler(store)
	reportH := handler.NewReportHandler(store)

	// start simulation engine
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	engine := simulation.NewEngine(store, hub)
	engine.Start(ctx)

	r := chi.NewRouter()

	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:*", "http://127.0.0.1:*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	// websocket
	r.Get("/ws", hub.HandleConnect)

	// public
	r.Post("/api/auth/register", authH.Register)
	r.Post("/api/auth/login", authH.Login)

	// protected
	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(secret))

		// sectors — read for everyone
		r.Get("/api/sectors", sectorH.List)
		r.Get("/api/sectors/{id}", sectorH.Get)
		r.Get("/api/sectors/my", sectorH.ListMy)

		// agronomist only — sector management
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireRole("agronomist"))
			r.Post("/api/sectors", sectorH.Create)
			r.Put("/api/sectors/{id}", sectorH.Update)
			r.Delete("/api/sectors/{id}", sectorH.Delete)
			r.Put("/api/sectors/{id}/assign", sectorH.Assign)
			r.Delete("/api/sectors/{id}/assign", sectorH.Unassign)
			r.Get("/api/export/sectors", fileH.ExportSectors)
			r.Post("/api/import/sectors", fileH.ImportSectors)
		})

		// plants
		r.Get("/api/plants", plantH.List)
		r.Post("/api/plants", plantH.Create)
		r.Delete("/api/plants/{id}", plantH.Delete)

		// watering
		r.Post("/api/water", waterH.Water)
		r.Get("/api/water/stats/{sectorId}", waterH.Stats)

		// notifications
		r.Get("/api/notifications", notifH.List)
		r.Put("/api/notifications/{id}/read", notifH.MarkRead)

		// reports & telemetry
		r.Get("/api/telemetry/{sectorId}", reportH.Telemetry)
		r.Get("/api/reports/{sectorId}", reportH.Summary)
	})

	// graceful shutdown
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		log.Println("shutting down...")
		cancel()
		os.Exit(0)
	}()

	log.Printf("starting on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
