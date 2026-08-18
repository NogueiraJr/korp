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

	"korp/faturamento/internal/client"
	"korp/faturamento/internal/config"
	"korp/faturamento/internal/db"
	"korp/faturamento/internal/handlers"
	"korp/faturamento/internal/middleware"
	"korp/faturamento/internal/repository"
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

	invoiceRepo := repository.NewInvoiceRepository(database.Pool)
	activityRepo := repository.NewActivityRepository(database.Pool)
	estoqueClient := client.NewEstoqueClient(cfg.EstoqueURL, cfg.InternalToken)

	logsHandler := &handlers.LogsHandler{Repo: activityRepo}
	invoiceHandler := &handlers.InvoiceHandler{Repo: invoiceRepo, Estoque: estoqueClient, Logs: logsHandler}
	authHandler := &handlers.AuthHandler{
		Username:  cfg.AdminUsername,
		Password:  cfg.AdminPassword,
		JWTSecret: cfg.JWTSecret,
		TokenTTL:  time.Duration(cfg.TokenTTLHours) * time.Hour,
		Logs:      logsHandler,
	}

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
		w.Write([]byte(`{"status":"ok","service":"faturamento"}`))
	})

	r.Post("/api/auth/login", authHandler.Login)

	r.Route("/api/invoices", func(r chi.Router) {
		r.Use(middleware.Auth(cfg.JWTSecret))
		r.Post("/", invoiceHandler.Create)
		r.Get("/", invoiceHandler.List)
		r.Get("/{id}", invoiceHandler.GetByID)
		r.Post("/{id}/print", invoiceHandler.Print)
	})

	r.Route("/api/logs", func(r chi.Router) {
		r.Use(middleware.Auth(cfg.JWTSecret))
		r.Get("/", logsHandler.List)
	})

	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	go func() {
		log.Printf("faturamento service listening on :%s", cfg.Port)
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
	log.Println("faturamento service stopped")
}