package main

import (
	"log"
	"net/http"
	"os"

	"steadyphoto/internal/api"
	"steadyphoto/internal/database"
	"steadyphoto/internal/storage"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

const (
	dbURLKey         = "DATABASE_URL"
	apiPortKey       = "API_PORT"
	defaultAPIPort   = "8081"
	storageRootEnv   = "STORAGE_ROOT"
	defaultStorageRoot = "./storage"
	thumbRootEnv       = "THUMBNAIL_ROOT"
	defaultThumbRoot   = "./storage/.thumbnails"
)

func main() {
	dbURL := os.Getenv(dbURLKey)
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is not set")
	}

	apiPort := os.Getenv(apiPortKey)
	if apiPort == "" {
		apiPort = defaultAPIPort
	}

	storageRoot := os.Getenv(storageRootEnv)
	if storageRoot == "" {
		storageRoot = defaultStorageRoot
	}

	thumbRoot := os.Getenv(thumbRootEnv)
	if thumbRoot == "" {
		thumbRoot = defaultThumbRoot
	}

	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Ensure the connection pool is working
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	// Set up repositories
	mediaRepo := database.NewPostgresMediaRepository(db)
	albumRepo := database.NewPostgresAlbumRepository(db)
	userRepo := database.NewPostgresUserRepository(db)
	sessionRepo := database.NewPostgresSessionRepository(db)

	// Set up storage service (single parameter: baseDir)
	storageService := storage.NewStorageService(storageRoot)

	// Create API server
	server := api.NewServer(mediaRepo, albumRepo, userRepo, sessionRepo, storageService, thumbRoot)

	log.Printf("Starting SteadyPhoto API on port %s", apiPort)
	if err := http.ListenAndServe(":"+apiPort, server); err != nil {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}
}
