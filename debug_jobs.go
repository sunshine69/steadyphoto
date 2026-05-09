package main

import (
	"fmt"
	"log"
	"os"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL not set")
	}

	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer db.Close()

	type JobRow struct {
		ID        string `db:"id"`
		Type      string `db:"job_type"`
		Status    string `db:"status"`
		PhotoID   string `db:"media_id"`
		CreatedAt string `db:"created_at"`
		Error     string `db:"error_message"`
	}

	var rows []JobRow
	// Using raw query to avoid any struct mapping issues during debugging
	err = db.Select(&rows, "SELECT id, job_type, status, media_id, created_at, error_message FROM jobs")
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}

	fmt.Println("--- JOBS IN DATABASE ---")
	if len(rows) == 0 {
		fmt.Println("No jobs found.")
		return
	}

	for _, r := range rows {
		fmt.Printf("ID: %s | Type: %s | Status: %s | PhotoID: %s | Created: %s | Error: %s\n",
			r.ID, r.Type, r.Status, r.PhotoID, r.CreatedAt, r.Error)
	}
	fmt.Println("------------------------")
}
