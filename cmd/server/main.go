package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"

	"steadyphoto/internal/api"
	"steadyphoto/internal/database"
	"steadyphoto/internal/storage"
	"steadyphoto/internal/utils"

	"github.com/robfig/cron"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

const (
	dbURLKey           = "DATABASE_URL"
	apiPortKey         = "API_PORT"
	defaultAPIPort     = "8081"
	storageRootEnv     = "STORAGE_ROOT"
	defaultStorageRoot = "./storage"
	thumbRootEnv       = "THUMBNAIL_ROOT"
	defaultThumbRoot   = "./storage/.thumbnails"

	tlsCertKey = "TLS_CERT" // Path to TLS certificate file (PEM) - env var fallback
	tlsKeyKey  = "TLS_KEY"  // Path to TLS private key file (PEM) - env var fallback

	workerCronTabKey = "WORKER_CRON_TAB" // Cron schedule for worker execution
	defaultCronTab   = "0 0 */1 * *"     // Default: run hourly
)

var (
	version   string // Will hold the version number
	buildTime string // Will hold the build time
)

func printVersionBuildInfo() {
	fmt.Printf("Version: %s\nBuild time: %s\n", version, buildTime)
}

// startWorkerScheduler starts the cron scheduler that periodically runs the worker
func startWorkerScheduler() {
	cronTab := os.Getenv(workerCronTabKey)
	if cronTab == "" {
		cronTab = defaultCronTab
	}

	log.Printf("Starting worker scheduler with cron tab: %s", cronTab)

	c := cron.New()

	// Add the worker execution job
	c.AddFunc(cronTab, func() {
		log.Println("Running worker job...")
		go runWorker()
	})

	// Start the scheduler
	c.Start()
	log.Println("Worker scheduler started")
}

// runWorker executes the worker at /worker.exe and logs output
func runWorker() {
	var cmd *exec.Cmd

	if _, err := os.Stat("/tmp/worker.lock"); err == nil {
		return
	} else {
		if err := os.WriteFile("/tmp/worker.lock", []byte("worker running"), 0o777); err != nil {
			println("[EEROR] writting lock file")
		} else {
			defer os.RemoveAll("/tmp/worker.lock")
		}
	}

	// Use platform-specific path for worker
	for _, commandStr := range []string{"/app/worker"} {
		cmd = exec.Command(commandStr)

		// Capture stdout and stderr
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			log.Printf("Error creating worker stdout pipe: %v", err)
			return
		}

		stderr, err := cmd.StderrPipe()
		if err != nil {
			log.Printf("Error creating worker stderr pipe: %v", err)
			return
		}

		// Start the command
		if err := cmd.Start(); err != nil {
			log.Printf("Error starting worker: %v", err)
			return
		}

		// Read stdout and log it
		go func() {
			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				log.Printf("[WORKER STDOUT] %s", scanner.Text())
			}
		}()

		// Read stderr and log it
		go func() {
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				log.Printf("[WORKER STDERR] %s", scanner.Text())
			}
		}()

		// Wait for the command to complete
		err = cmd.Wait()
		if err != nil {
			log.Printf("Worker completed with error: %v", err)
		} else {
			log.Println("Worker completed successfully")
		}
	}
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		printVersionBuildInfo()
		os.Exit(0)
	}

	// Command-line flags (override env vars if supplied)
	tlsCertPath := flag.String("tls-cert", "", "(CLI override) Path to TLS certificate file (PEM). If provided along with -tls-key, starts HTTPS server.")
	tlsKeyPath := flag.String("tls-key", "", "(CLI override) Path to TLS private key file (PEM). If provided along with -tls-cert, starts HTTPS server.")

	// Show help if requested
	h := flag.Bool("h", false, "Show this help message and exit")
	flag.Parse()

	if *h {
		printHelp()
		return
	}

	// Get configuration from environment variables (fallback)
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

	// TLS: CLI flags override env vars if both are provided; otherwise use whichever is available
	var certPath, keyPath string
	if *tlsCertPath != "" || os.Getenv(tlsCertKey) != "" {
		certPath = *tlsCertPath // CLI flag takes precedence over env var
	} else {
		certPath = os.Getenv(tlsCertKey)
	}

	if *tlsKeyPath != "" || os.Getenv(tlsKeyKey) != "" {
		keyPath = *tlsKeyPath // CLI flag takes precedence over env var
	} else {
		keyPath = os.Getenv(tlsKeyKey)
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
	shareRepo := database.NewPostgresShareRepository(db)
	mediaShareRepo := database.NewPostgresMediaShareRepository(db)
	albumShareRepo := database.NewPostgresAlbumShareRepository(db)
	publicShareRepo := database.NewPostgresPublicShareRepository(db)
	publicAccessRepo := database.NewPostgresPublicShareAccessRepository(db)

	// Seed the initial admin user (idempotent - updates if email already exists, creates if not)
	ctx := context.Background()
	if err := utils.CreateAdminUser(ctx, userRepo); err != nil {
		log.Fatalf("Failed to create seed admin user: %v", err)
	}

	// Set up storage service (single parameter: baseDir)
	storageService := storage.NewStorageService(storageRoot, thumbRoot)

	// Set up job repository for background job processing
	jobRepo := database.NewPostgresJobRepository(db)

	// Create API server
	server := api.NewServer(mediaRepo, albumRepo, userRepo, sessionRepo, storageService, thumbRoot, shareRepo, mediaShareRepo, albumShareRepo, publicShareRepo, publicAccessRepo, jobRepo)

	addr := ":" + apiPort
	log.Printf("Starting SteadyPhoto API on port %s", addr)

	// Start worker scheduler
	startWorkerScheduler()

	if certPath != "" && keyPath != "" {
		log.Printf("[SECURITY] Starting HTTPS server with TLS (cert=%s, key=%s)", certPath, keyPath)
		if err := http.ListenAndServeTLS(addr, certPath, keyPath, server); err != nil {
			log.Fatalf("Failed to start HTTPS server: %v", err)
		}
	} else {
		// No TLS configured — plain HTTP. This is fine when running behind a reverse proxy (nginx/caddy) that handles TLS termination.
		if certPath == "" || keyPath == "" {
			log.Printf("[WARN] TLS not configured (-tls-cert/-tls-key flags or %s/%s env vars). Serving over plain HTTP. Use if behind a reverse proxy with TLS termination.", tlsCertKey, tlsKeyKey)
		}
		if err := http.ListenAndServe(addr, server); err != nil {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
	}
}

func printHelp() {
	fmt.Println(`SteadyPhoto API Server - Configuration Help
=============================================

Environment Variables (fallback):
  DATABASE_URL      PostgreSQL connection string (required). E.g., postgres://user:pass@host:5432/dbname?sslmode=disable
  API_PORT          Port to listen on (default: 8081)
  STORAGE_ROOT      Base directory for media storage (default: ./storage)
  THUMBNAIL_ROOT    Directory for thumbnails (default: ./storage/.thumbnails)
  WORKER_CRON_TAB   Cron schedule for worker execution (default: "0 * * * *" - hourly)
  TLS_CERT          Path to TLS certificate file (PEM) — used if -tls-cert is not provided
  TLS_KEY           Path to TLS private key file (PEM) — used if -tls-key is not provided

  Admin User Seeding (optional):
  ADMIN_EMAIL       Email for the initial admin user. If set, an admin user will be created/updated on startup.
  ADMIN_PASSWORD    Password for the initial admin user. Must be set alongside ADMIN_EMAIL.

Command-Line Flags (override env vars):
  -h, -help         Show this help message and exit
  -tls-cert=path    Path to TLS certificate file (PEM). If both -tls-cert and -tls-key are set, starts HTTPS server.
                    Overrides $TLS_CERT if both CLI flag and env var are provided.
  -tls-key=path     Path to TLS private key file (PEM). If both -tls-cert and -tls-key are set, starts HTTPS server.
                    Overrides $TLS_KEY if both CLI flag and env var are provided.

Sub-commands:
  version           Show version and build information.

Worker Scheduler:
  The server includes a cron scheduler that periodically runs the worker program.
  The cron schedule is controlled by the WORKER_CRON_TAB environment variable.
  Default schedule: "0 * * * *" (every hour at minute 0).

  Examples of valid cron schedules:
    "0 * * * *"       - Run at the top of every hour
    "*/15 * * * *"    - Run every 15 minutes
    "0 */2 * * *"     - Run every 2 hours
    "0 0 * * *"       - Run daily at midnight
    "0 0 * * 0"       - Run weekly on Sunday at midnight

  The worker is executed as a background goroutine, so it doesn't block the HTTP server.
  Worker output is logged with [WORKER STDOUT] and [WORKER STDERR] prefixes.

HTTPS/TLS Modes:
  - Direct HTTPS: Set either environment variables ($TLS_CERT + $TLS_KEY) OR command-line flags (-tls-cert + -tls-key).
    Both must be provided for the same mode (env or CLI) to start an HTTPS server.
  - Plain HTTP (reverse proxy): Don't set TLS config at all. The server listens on plain HTTP, and a reverse proxy
    (nginx/caddy) handles TLS termination in front of it. This is the recommended deployment pattern.

Examples:
  # Direct HTTPS via CLI flags:
  ./server -tls-cert=/etc/ssl/certs/steadyphoto.pem -tls-key=/etc/ssl/private/steadyphoto.key

  # Plain HTTP behind nginx (no TLS config needed):
  DATABASE_URL=postgres://user:pass@localhost:5432/steadyphoto ./server

  # HTTPS via environment variables:
  export TLS_CERT=/etc/ssl/certs/steadyphoto.pem
  export TLS_KEY=/etc/ssl/private/steadyphoto.key
  DATABASE_URL=postgres://user:pass@localhost:5432/steadyphoto ./server

  # Seed admin user on startup (requires PostgreSQL connection):
  DATABASE_URL=postgres://user:pass@localhost:5432/steadyphoto ADMIN_EMAIL=admin@example.com ADMIN_PASSWORD=secret123 ./server

  # Run worker every 15 minutes:
  WORKER_CRON_TAB="*/15 * * * *" DATABASE_URL=postgres://user:pass@localhost:5432/steadyphoto ./server`)
}
