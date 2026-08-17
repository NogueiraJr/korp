package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"korp/estoque/internal/ai"
	"korp/estoque/internal/config"
	"korp/estoque/internal/db"
	"korp/estoque/internal/handlers"
	"korp/estoque/internal/middleware"
	"korp/estoque/internal/repository"
)

func main() {
	cfg := config.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	database, err := db.Connect(ctx, cfg.DatabaseURL)
	cancel()
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer database.Close()

	migrateCtx, migrateCancel := context.WithTimeout(context.Background(), 15*time.Second)
	if err := database.Migrate(migrateCtx); err != nil {
		migrateCancel()
		log.Fatalf("migrations failed: %v", err)
	}
	migrateCancel()
	log.Println("migrations applied")

	productRepo := repository.NewProductRepository(database.Pool)
	activityRepo := repository.NewActivityRepository(database.Pool)
	aiService := ai.NewService(cfg.AIBaseURL, cfg.AIAPIKey, cfg.AIModel)
	productHandler := &handlers.ProductHandler{Repo: productRepo, Activity: activityRepo, AI: aiService}
	logsHandler := &handlers.LogsHandler{Repo: activityRepo}

	r := chi.NewRouter()
	r.Use(chimw.RealIP)
	r.Use(chimw.RequestID)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Logger)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:4200", "http://127.0.0.1:4200"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization", "Idempotency-Key", "X-Internal-Token"},
		AllowCredentials: false,
		MaxAge:           300,
	}))
	r.Use(chimw.Timeout(30 * time.Second))

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"estoque"}`))
	})

	r.Route("/api", func(r chi.Router) {
		r.Use(middleware.Auth(cfg.JWTSecret, cfg.InternalToken))

		r.Post("/products", productHandler.Create)
		r.Get("/products", productHandler.List)
		r.Post("/products/ai-description", productHandler.AIDescription)
		r.Post("/products/consume", productHandler.ConsumeStock)
		r.Get("/logs", logsHandler.List)
		r.Get("/products/{code}", productHandler.GetByCode)
		r.Put("/products/{code}", productHandler.Update)
	})

	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	go func() {
		log.Printf("estoque service listening on :%s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
	log.Println("estoque service stopped")
}