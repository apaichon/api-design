package main

import (
	"context"
	// "fmt"
	"log"
	"net/http"
	"path/filepath"

	"apidesign/config"
	"apidesign/internal/database"
	"apidesign/internal/middleware"

	"github.com/gorilla/mux"
)

func main() {
	absPath, err := filepath.Abs("./config/config.json") // Get absolute path
	// fmt.Printf("Path: %s", absPath)
	// Load configuration
	cfg, err := config.LoadConfig(absPath)
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	// Initialize database
	ctx := context.Background()
	db := database.NewPostgresDatabase()

	// fmt.Printf("\nDb: %s", cfg.DatabaseURL)
	err = db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}
	defer db.Close(ctx)

	// Setup router
	r := mux.NewRouter()
	middleware.SetupRoutes(r, db)

	// Start server
	log.Printf("Starting server on %s", cfg.Port)            // Updated to use port from config
	if err := http.ListenAndServe(cfg.Port, r); err != nil { // Updated to use port from config
		log.Fatalf("Error starting server: %v", err)
	}
}
