package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"steadyphoto/internal/database"
	"steadyphoto/internal/domain"
	"steadyphoto/internal/scanner"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// immichAsset mirrors the Immich API response for an asset
type immichAsset struct {
	ID               string      `json:"id"`
	DeviceAssetID    string      `json:"deviceAssetId"`
	UserID           string      `json:"userId"`
	Type             string      `json:"type"`
	OriginalPath     string      `json:"originalPath"`
	OriginalFileName string      `json:"originalFileName"`
	OriginalMimeType string      `json:"originalMimeType"`
	Width            int         `json:"width"`
	Height           int         `json:"height"`
	Duration         *float64    `json:"duration"`
	ExifInfo         *exifInfo   `json:"exifInfo"`
	FileCreatedAt    *time.Time  `json:"fileCreatedAt"`
	FileModifiedAt   *time.Time  `json:"fileModifiedAt"`
	LocalDateTime    *time.Time  `json:"localDateTime"`
	UpdatedAt        *time.Time  `json:"updatedAt"`
	HasSidecar       bool        `json:"hasSidecar"`
	Tags             []immichTag `json:"tags"`
}

// exifInfo mirrors Immich EXIF data
type exifInfo struct {
	Model     string   `json:"model"`
	LensModel *string  `json:"lensModel"`
	ISO       *int     `json:"iso"`
	City      *string  `json:"city"`
	State     *string  `json:"state"`
	Country   *string  `json:"country"`
}

// immichTag mirrors Immich tag data
type immichTag struct {
	Name string `json:"name"`
}

// immichAPIResponse is the paginated response from Immich API
type immichAPIResponse struct {
	Items []immichAsset `json:"items"`
	Page  int           `json:"page"`
	Total int           `json:"total"`
	Size  int           `json:"size"`
	Limit int           `json:"limit"`
}

// immichClient handles communication with the Immich API
type immichClient struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

func newImmichClient(baseURL, apiKey string) *immichClient {
	return &immichClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		client:  &http.Client{Timeout: 10 * time.Minute},
	}
}

func (c *immichClient) headers() map[string]string {
	return map[string]string{
		"x-api-key": c.apiKey,
		"Accept":    "application/json",
	}
}

// fetchAssets retrieves all assets from Immich with pagination
func (c *immichClient) fetchAssets(ctx context.Context) ([]immichAsset, error) {
	var allAssets []immichAsset
	page := 1
	limit := 1000

	for {
		select {
		case <-ctx.Done():
			return allAssets, ctx.Err()
		default:
		}

		url := fmt.Sprintf("%s/assets?size=%d&page=%d", c.baseURL, limit, page)
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return allAssets, fmt.Errorf("failed to create request: %w", err)
		}

		for k, v := range c.headers() {
			req.Header.Set(k, v)
		}

		resp, err := c.client.Do(req)
		if err != nil {
			return allAssets, fmt.Errorf("failed to fetch assets page %d: %w", page, err)
		}

		var body []byte
		body, err = io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return allAssets, fmt.Errorf("failed to read response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return allAssets, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
		}

		var result immichAPIResponse
		if err := json.Unmarshal(body, &result); err != nil {
			return allAssets, fmt.Errorf("failed to decode response: %w", err)
		}

		allAssets = append(allAssets, result.Items...)

		if len(result.Items) < limit || page >= (result.Total+limit-1)/limit {
			break
		}
		page++

		if page%10 == 0 {
			fmt.Printf("  Fetched %d assets so far...\n", len(allAssets))
		}
	}

	return allAssets, nil
}

// downloadAsset downloads the original media file from Immich
func (c *immichClient) downloadAsset(ctx context.Context, assetID string) ([]byte, error) {
	url := fmt.Sprintf("%s/assets/%s/original", c.baseURL, assetID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create download request: %w", err)
	}

	for k, v := range c.headers() {
		req.Header.Set(k, v)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download failed for asset %s: %w", assetID, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read asset data: %w", err)
	}

	return data, nil
}

// downloadSidecar downloads the XMP sidecar file if available
func (c *immichClient) downloadSidecar(ctx context.Context, assetID string) ([]byte, error) {
	url := fmt.Sprintf("%s/assets/%s/sidecar", c.baseURL, assetID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create sidecar request: %w", err)
	}

	for k, v := range c.headers() {
		req.Header.Set(k, v)
	}

	resp, err := c.client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil, nil // No sidecar available
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read sidecar data: %w", err)
	}

	return data, nil
}

// migrationStats tracks migration progress (similar to scanner's imported/skipped counters)
type migrationStats struct {
	mu        sync.Mutex
	imported  int
	skipped   int
	errors    int
	startTime time.Time
}

func newMigrationStats() *migrationStats {
	return &migrationStats{startTime: time.Now()}
}

func (s *migrationStats) recordImported() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.imported++
}

func (s *migrationStats) recordSkipped(reason string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.skipped++
	log.Printf("[MIGRATE SKIP] %s", reason)
}

func (s *migrationStats) recordError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.errors++
	log.Printf("[MIGRATE ERROR] %v", err)
}

func (s *migrationStats) printStatus(total int, email string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	processed := s.imported + s.skipped
	pct := float64(processed) / float64(total) * 100
	elapsed := time.Since(s.startTime).Round(time.Second)

	fmt.Printf("  Progress: %d/%d (%.1f%%) | Imported: %d | Skipped: %d | Errors: %d | Elapsed: %s\n",
		processed, total, pct, s.imported, s.skipped, s.errors, elapsed)
}

func (s *migrationStats) printFinalReport(total int) {
	fmt.Println("\n=== Migration Report ===")
	fmt.Printf("Total assets processed: %d\n", total)
	fmt.Printf("Successfully imported:  %d\n", s.imported)
	fmt.Printf("Skipped (duplicate):    %d\n", s.skipped)
	fmt.Printf("Errors:                 %d\n", s.errors)
	fmt.Printf("Duration:               %s\n", time.Since(s.startTime).Round(time.Second))

	if s.errors > 0 {
		fmt.Println("\n⚠️  Migration completed with errors. Check logs for details.")
	} else {
		fmt.Println("\n✅ Migration completed successfully!")
	}
}

func main() {
	sourceURL := flag.String("url", "", "Immich server URL (e.g., https://immich.example.com)")
	apiKey := flag.String("api-key", "", "Immich API key with asset.read and asset.download permissions")
	email := flag.String("email", "", "Email of the user who will own these imports")
	storageDir := flag.String("storage", "", "Root directory for organized storage (default: ./storage)")
	dbURL := flag.String("db", "", "PostgreSQL connection URL")
	dryRun := flag.Bool("dry-run", false, "Preview migration without making changes")
	parallel := flag.Int("parallel", 1, "Number of parallel workers (default: 1 for sequential)")
	flag.Parse()

	if *sourceURL == "" || *apiKey == "" {
		fmt.Println("Usage: immich-migrate -url <immich-url> -api-key <key> -email <user@example.com> [-storage <dir>] [-db <db_url>] [-dry-run] [-parallel <n>]")
		os.Exit(1)
	}

	// Priority: 1. Command line flag, 2. Environment variable (same pattern as scanner)
	finalDBURL := *dbURL
	if finalDBURL == "" {
		finalDBURL = os.Getenv("DATABASE_URL")
	}

	if finalDBURL == "" {
		log.Fatal("Database URL must be provided via -db flag or DATABASE_URL environment variable")
	}

	finalStorageDir := *storageDir
	if finalStorageDir == "" {
		finalStorageDir = "./storage"
	}

	// 1. Connect to DB (same pattern as scanner)
	db, err := sqlx.Connect("postgres", finalDBURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// 2. Initialize repositories (reusing scanner's repo pattern)
	repo := database.NewPostgresMediaRepository(db)
	jobRepo := database.NewPostgresJobRepository(db)

	// 3. Look up user by email to get their ID for ownership attribution (same as scanner)
	var userID uuid.UUID
	err = db.Get(&userID, "SELECT id FROM users WHERE email = $1 AND status != 'disabled'", *email)
	if err != nil {
		log.Fatalf("Failed to find an active user with email %s: %v", *email, err)
	}

	fmt.Printf("=== Immich to SteadyPhoto Migration ===\n")
	fmt.Printf("Immich URL:   %s\n", *sourceURL)
	fmt.Printf("Target User:  %s (ID: %s)\n", *email, userID)
	fmt.Printf("Storage Root: %s\n", finalStorageDir)

	if *dryRun {
		fmt.Println("\n⚠️  DRY RUN MODE - No changes will be made")
	} else {
		fmt.Println("\nStarting migration...")
	}

	// Create scanner instance for file operations (reusing scanner's copy/hash logic)
	scannerInstance := scanner.NewMediaScanner(repo, jobRepo, finalStorageDir)

	// 4. Connect to Immich API and fetch all assets
	fmt.Println("\n[1/3] Fetching assets from Immich...")
	client := newImmichClient(*sourceURL, *apiKey)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	assets, err := client.fetchAssets(ctx)
	if err != nil {
		log.Fatalf("Failed to fetch assets: %v", err)
	}

	fmt.Printf("  Found %d assets in Immich\n", len(assets))

	// 5. Migrate each asset (reusing scanner's hash/check/create pattern)
	fmt.Println("\n[2/3] Migrating media files...")
	stats := newMigrationStats()

	if *parallel > 1 {
		migrateParallel(ctx, client, repo, jobRepo, scannerInstance, assets, &userID, finalStorageDir, *dryRun, stats, *parallel)
	} else {
		migrateSequential(ctx, client, repo, jobRepo, scannerInstance, assets, &userID, finalStorageDir, *dryRun, stats)
	}

	stats.printFinalReport(len(assets))

	// 6. Summary (same pattern as scanner's "Scan completed successfully")
	fmt.Printf("\n[3/3] Migration complete! Imported: %d, Skipped: %d\n", stats.imported, stats.skipped)
}

// migrateSequential processes assets one by one (like scanner walks directory)
func migrateSequential(
	ctx context.Context,
	client *immichClient,
	repo domain.MediaRepository,
	jobRepo domain.JobRepository,
	scannerInstance *scanner.MediaScanner,
	assets []immichAsset,
	userID *uuid.UUID,
	storageRoot string,
	dryRun bool,
	stats *migrationStats,
) {
	for i, asset := range assets {
		select {
		case <-ctx.Done():
			log.Printf("Migration cancelled by user")
			return
		default:
		}

		fmt.Printf("\n[%d/%d] Processing: %s\n", i+1, len(assets), asset.OriginalFileName)

		if err := migrateAsset(ctx, client, repo, jobRepo, scannerInstance, &asset, userID, storageRoot, dryRun); err != nil {
			stats.recordError(err)
			continue
		}

		stats.recordImported()

		if (i+1)%50 == 0 || i+1 == len(assets) {
			stats.printStatus(len(assets), "")
		}
	}
}

// migrateParallel processes assets with multiple workers
func migrateParallel(
	ctx context.Context,
	client *immichClient,
	repo domain.MediaRepository,
	jobRepo domain.JobRepository,
	scannerInstance *scanner.MediaScanner,
	assets []immichAsset,
	userID *uuid.UUID,
	storageRoot string,
	dryRun bool,
	stats *migrationStats,
	parallel int,
) {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, parallel)

	for i, asset := range assets {
		select {
		case <-ctx.Done():
			log.Printf("Migration cancelled by user")
			return
		default:
		}

		wg.Add(1)
		go func(idx int, a immichAsset) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			fmt.Printf("\n[%d/%d] Processing: %s\n", idx+1, len(assets), a.OriginalFileName)

			if err := migrateAsset(ctx, client, repo, jobRepo, scannerInstance, &a, userID, storageRoot, dryRun); err != nil {
				stats.recordError(err)
				return
			}

			stats.recordImported()
		}(i, asset)
	}

	wg.Wait()
}

// migrateAsset handles the migration of a single Immich asset to SteadyPhoto
// Reuses scanner's hash/check/create pattern but fetches from Immich API instead of local disk
func migrateAsset(
	ctx context.Context,
	client *immichClient,
	repo domain.MediaRepository,
	jobRepo domain.JobRepository,
	scannerInstance *scanner.MediaScanner,
	asset *immichAsset,
	userID *uuid.UUID,
	storageRoot string,
	dryRun bool,
) error {
	// Download the file from Immich (instead of reading from local disk like scanner does)
	fmt.Printf("  Downloading: %s (%s)\n", asset.OriginalFileName, asset.OriginalMimeType)
	fileData, err := client.downloadAsset(ctx, asset.ID)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	// Calculate hash from downloaded data (same logic as scanner.calculateHash but on bytes)
	hash := calculateHashFromBytes(fileData)
	fmt.Printf("  SHA256: %s\n", hash[:16]+"...")

	// Check for duplicates using scanner's repo.GetByHash pattern (same dedup logic)
	existing, _ := repo.GetByHash(ctx, hash)
	if existing != nil {
		log.Printf("[MIGRATE SKIP] Duplicate detected (hash: %s)", hash[:16])
		return fmt.Errorf("duplicate") // Special error to indicate skip
	}

	// Determine media type (same logic as scanner.processFile)
	mediaType := domain.MediaTypePhoto
	ext := strings.ToLower(filepath.Ext(asset.OriginalFileName))
	if ext == ".mp4" || ext == ".mov" || ext == ".avi" || ext == ".mkv" || ext == ".webm" || ext == ".flv" {
		mediaType = domain.MediaTypeVideo
	}

	// Determine capture date (same as scanner uses info.ModTime())
	var capturedAt time.Time
	if asset.FileCreatedAt != nil {
		capturedAt = *asset.FileCreatedAt
	} else if asset.LocalDateTime != nil {
		capturedAt = *asset.LocalDateTime
	} else {
		capturedAt = time.Now()
	}

	// Build metadata from Immich EXIF data (scanner leaves this empty)
	metadata := make(domain.Metadata)
	if asset.ExifInfo != nil {
		if asset.ExifInfo.Model != "" {
			metadata["camera"] = asset.ExifInfo.Model
		}
		if asset.ExifInfo.LensModel != nil && *asset.ExifInfo.LensModel != "" {
			metadata["lens"] = *asset.ExifInfo.LensModel
		}
		if asset.ExifInfo.ISO != nil {
			metadata["iso"] = fmt.Sprintf("%d", *asset.ExifInfo.ISO)
		}
		if asset.ExifInfo.City != nil && *asset.ExifInfo.City != "" {
			metadata["city"] = *asset.ExifInfo.City
		}
		if asset.ExifInfo.State != nil && *asset.ExifInfo.State != "" {
			metadata["state"] = *asset.ExifInfo.State
		}
		if asset.ExifInfo.Country != nil && *asset.ExifInfo.Country != "" {
			metadata["country"] = *asset.ExifInfo.Country
		}
	}

	// Build tags from Immich tags (scanner has no tags)
	var tags string
	if len(asset.Tags) > 0 {
		tagNames := make([]string, len(asset.Tags))
		for i, tag := range asset.Tags {
			tagNames[i] = tag.Name
		}
		tags = fmt.Sprintf("immich:%s", strings.Join(tagNames, ":"))
	}

	if dryRun {
		fmt.Printf("  [DRY RUN] Would save: %s/%s/%s/%s\n",
			userID.String(), capturedAt.Format("2006/01/02"), userID.String(), asset.OriginalFileName)
		return nil
	}

	// Create media struct (same pattern as scanner.processFile)
	media := &domain.Media{
		ID:         uuid.New(),
		Hash:       hash,
		Filename:   asset.OriginalFileName,
		SizeBytes:  int64(len(fileData)),
		CreatedAt:  time.Now(),
		CapturedAt: capturedAt,
		MediaType:  mediaType,
		Metadata:   metadata,
		UserID:     *userID, // Assign ownership at creation (same as scanner)
		Tags:       tags,
	}

	// Copy file to storage using scanner's copyToStorage pattern
	relPath, err := copyToStorage(fileData, media, storageRoot)
	if err != nil {
		return fmt.Errorf("storage copy failed: %w", err)
	}
	media.Path = relPath

	// Create DB record (same as scanner.repo.Create)
	if err := repo.Create(ctx, media); err != nil {
		return fmt.Errorf("db creation failed: %w", err)
	}

	fmt.Printf("  Saved: %s\n", relPath)
	fmt.Printf("  Created DB record: %s (type: %s)\n", media.ID, mediaType)

	// Create thumbnail job (same pattern as scanner creates Job after import)
	job := &domain.Job{
		ID:        uuid.New(),
		Type:      domain.JobTypeThumbnail,
		Status:    domain.JobStatusPending,
		MediaID:   media.ID,
		UserID:    *userID, // Important to pass user context to jobs too! (same comment as scanner)
		CreatedAt: time.Now(),
	}
	if err := jobRepo.Create(ctx, job); err != nil {
		log.Printf("[WARN] Media created but job failed for %s: %v", media.ID, err)
	}

	// Download sidecar if available (Immich-specific feature)
	if asset.HasSidecar {
		sidecarData, err := client.downloadSidecar(ctx, asset.ID)
		if err == nil && len(sidecarData) > 0 {
			fullPath := filepath.Join(storageRoot, relPath)
			sidecarPath := fullPath + ".xmp"
			if err := os.WriteFile(sidecarPath, sidecarData, 0644); err != nil {
				log.Printf("[WARN] Failed to save sidecar for %s: %v", asset.OriginalFileName, err)
			} else {
				fmt.Printf("  Saved sidecar: %s.xmp\n", fullPath)
			}
		}
	}

	return nil
}

// calculateHashFromBytes computes SHA256 hash from byte slice (same logic as scanner.calculateHash)
func calculateHashFromBytes(data []byte) string {
	h := sha256.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// copyToStorage writes file data to disk using the same storage pattern as scanner.copyToStorage
func copyToStorage(fileData []byte, media *domain.Media, storageRoot string) (string, error) {
	// Create the relative directory structure including UserID for physical isolation
	relDir := filepath.Join(media.UserID.String(), media.CapturedAt.Format("2006/01/02"))

	// The actual physical directory on disk (same as scanner)
	fullDir := filepath.Join(storageRoot, relDir)

	if err := os.MkdirAll(fullDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create storage dir: %w", err)
	}

	destPath := filepath.Join(fullDir, media.Filename)

	// Collision handling if filename exists, append _1, _2...) (same as scanner)
	counter := 1
	for {
		if _, err := os.Stat(destPath); os.IsNotExist(err) {
			break
		}
		ext := filepath.Ext(media.Filename)
		nameOnly := strings.TrimSuffix(media.Filename, ext)
		destPath = filepath.Join(fullDir, fmt.Sprintf("%s_%d%s", nameOnly, counter, ext))
		counter++
	}

	if err := os.WriteFile(destPath, fileData, 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	// IMPORTANT: Return the path RELATIVE to the storage root (including userID) - same as scanner
	relPath := filepath.Join(relDir, filepath.Base(destPath))
	return relPath, nil
}
