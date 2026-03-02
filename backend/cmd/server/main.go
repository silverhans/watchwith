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
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/watchwith/watchwith/internal/handler"
	"github.com/watchwith/watchwith/internal/room"
	"github.com/watchwith/watchwith/internal/store"
	"github.com/watchwith/watchwith/internal/ws"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// PostgreSQL (optional — runs in-memory without it)
	var db *store.DB
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		var err error
		db, err = store.NewDB(dbURL)
		if err != nil {
			log.Printf("WARNING: Failed to connect to PostgreSQL: %v", err)
			log.Println("Running in memory-only mode")
		} else {
			if err := db.RunMigrations(); err != nil {
				log.Printf("WARNING: Migration failed: %v", err)
			}
			defer db.Close()
		}
	} else {
		log.Println("DATABASE_URL not set — running in memory-only mode")
	}

	hub := ws.NewHub()
	go hub.Run()

	manager := room.NewManager(hub, db)
	h := handler.New(manager, hub)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173", "http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	r.Get("/api/health", h.Health)
	r.Post("/api/rooms", h.CreateRoom)
	r.Get("/api/rooms/{slug}", h.GetRoom)
	r.Get("/ws/{slug}", h.HandleWebSocket)

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Server starting on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Println("Server stopped")
}
