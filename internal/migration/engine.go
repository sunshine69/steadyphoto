package migration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"steadyphoto/internal/database"
	"steadyphoto/internal/domain"
	"steadyphoto/internal/storage"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Engine orchestrates the migration from Immich to SteadyPhoto
type Engine struct {
	config     MigrationConfig
	immich     *ImmichClient
	db         *sqlx.DB
	userRepo   *database.PostgresUserRepository
	mediaRepo  *database.PostgresMediaRepository
	storageSvc *storage.StorageService
	userMap    *UserMapping

	downloadedFiles map[string]bool // Track files already downloaded to disk (dedup at storage level)
	mu              sync.Mutex
}

// NewEngine creates a new migration engine
func NewEngine(config MigrationConfig) (*Engine, error) {
	// Connect to database
	db, err := sqlx.Connect("postgres", config.DATABASE_URL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Initialize repositories
	userRepo := database.NewPostgresUserRepository(db)
	mediaRepo := database.NewPostgresMediaRepository(db)

	// Determine storage base directory
	storageBase := os.Getenv("STORAGE_BASE_DIR")
	if storageBase == "" {
		storageBase = "./storage"
	}
	storageSvc := storage.NewStorageService(storageBase)

	return &Engine{
		config:          config,
		immich:          NewImmichClient(config.ImmichURL, config.ImmichAPIKey),
		db:              db,
		userRepo:        userRepo,
		mediaRepo:       mediaRepo,
		storageSvc:      storageSvc,
		userMap:         NewUserMapping(),
		downloadedFiles: make(map[string]bool),
	}, nil
}

// Run executes the full migration process
func (e *Engine) Run(ctx context.Context) error {
	fmt.Println("=== Immich to SteadyPhoto Migration ===")
	fmt.Printf("Immich URL: %s\n", e.config.ImmichURL)
	if e.config.DryRun {
		fmt.Println("⚠️  DRY RUN MODE - No changes will be made")
	} else {
		fmt.Println("Running migration...")
	}

	// Step 1: Fetch all assets from Immich
	fmt.Println("\n[1/4] Fetching assets from Immich...")
	assets, err := e.immich.GetAssets(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch assets: %w", err)
	}

	fmt.Printf("  Found %d assets in Immich\n", len(assets))

	// Step 2: Build user mapping (Immich user ID -> SteadyPhoto user UUID)
	fmt.Println("\n[2/4] Building user mappings...")
	if err := e.buildUserMapping(ctx, assets); err != nil {
		return fmt.Errorf("failed to build user mapping: %w", err)
	}

	// Step 3: Migrate assets
	fmt.Println("\n[3/4] Migrating media files...")
	progress := NewProgressTracker(len(assets))

	if e.config.Parallel > 1 {
		err = e.migrateParallel(ctx, assets, progress)
	} else {
		err = e.migrateSequential(ctx, assets, progress)
	}

	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	// Print final status
	progress.FinalReport()

	// Step 4: Summary
	fmt.Println("\n[4/4] Migration complete!")
	fmt.Printf("  Output directory: %s\n", e.config.OutputDir)

	return nil
}

// buildUserMapping maps Immich user IDs to SteadyPhoto users
func (e *Engine) buildUserMapping(ctx context.Context, assets []*ImmichAsset) error {
	// Collect unique Immich user IDs from assets
	immichUserIDs := make(map[string]bool)
	for _, asset := range assets {
		if asset.UserID != "" {
			immichUserIDs[asset.UserID] = true
		}
	}

	// If a specific email is requested, only map that user
	if e.config.UserEmail != "" {
		user, err := e.userRepo.GetByEmail(ctx, e.config.UserEmail)
		if err != nil {
			return fmt.Errorf("user %s not found: %w", e.config.UserEmail, err)
		}

		// Check if any assets belong to this user (by matching Immich ID somehow)
		// For now, we'll map all assets to this user since we can't directly match
		fmt.Printf("  Migrating as user: %s (%s)\n", e.config.UserEmail, user.ID)

		// Map a placeholder - in practice you'd need to know which Immich users correspond
		// For single-user migration, all assets go to this user
		for immichID := range immichUserIDs {
			e.userMap.Set(immichID, user.ID)
		}

		if len(immichUserIDs) == 0 {
			// No assets with user IDs - just use the specified user
			fmt.Println("  Note: Assets don't have Immich user IDs. All will be assigned to the specified user.")
		}
		return nil
	}

	// For multi-user migration, we need to figure out which SteadyPhoto users exist
	users, err := e.userRepo.ListUsers(ctx, "active")
	if err != nil {
		return fmt.Errorf("failed to list users: %w", err)
	}

	fmt.Printf("  Found %d active user(s) in SteadyPhoto\n", len(users))

	if len(immichUserIDs) > len(users) && len(users) > 0 {
		fmt.Println("  ⚠️  More Immich users than SteadyPhoto users. All extra users will be mapped to the first SteadyPhoto user.")
	}

	// Simple mapping: assign all Immich users to the first SteadyPhoto user if no direct match
	if len(users) > 0 {
		defaultUser := users[0].ID
		for immichID := range immichUserIDs {
			e.userMap.Set(immichID, defaultUser)
		}
		fmt.Printf("  Mapped all Immich users to SteadyPhoto user: %s\n", users[0].Email)
	} else if len(immichUserIDs) > 0 {
		return fmt.Errorf("no active users found in SteadyPhoto. Please create at least one user first.")
	}

	return nil
}

// migrateSequential migrates assets one by one (single-threaded)
func (e *Engine) migrateSequential(ctx context.Context, assets []*ImmichAsset, progress *ProgressTracker) error {
	for i, asset := range assets {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		fmt.Printf("\n[%d/%d] Processing: %s\n", i+1, len(assets), asset.OriginalFileName)

		if err := e.migrateAsset(ctx, asset); err != nil {
			progress.Update(0, 0, 1)
			fmt.Printf("  ❌ Error: %v\n", err)
			continue
		}

		progress.Update(1, 0, 0)
		if (i+1)%50 == 0 || i+1 == len(assets) {
			progress.PrintStatus()
		}
	}

	return nil
}

// migrateParallel migrates assets using multiple workers
func (e *Engine) migrateParallel(ctx context.Context, assets []*ImmichAsset, progress *ProgressTracker) error {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, e.config.Parallel)

	for i, asset := range assets {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		wg.Add(1)
		go func(idx int, a *ImmichAsset) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			fmt.Printf("\n[%d/%d] Processing: %s\n", idx+1, len(assets), a.OriginalFileName)

			if err := e.migrateAsset(ctx, a); err != nil {
				progress.Update(0, 0, 1)
				fmt.Printf("  ❌ Error: %v\n", err)
				return
			}

			progress.Update(1, 0, 0)
		}(i, asset)
	}

	wg.Wait()
	return nil
}

// migrateAsset handles the migration of a single Immich asset to SteadyPhoto
func (e *Engine) migrateAsset(ctx context.Context, asset *ImmichAsset) error {
	// Get target user
	userID, ok := e.userMap.Get(asset.UserID)
	if !ok {
		return fmt.Errorf("no SteadyPhoto user mapped for Immich user %s", asset.UserID)
	}

	// Determine media type and extension
	mediaType := DetermineMediaType(asset.OriginalMimeType)
	ext := ExtractExtension(asset.OriginalFileName, asset.OriginalMimeType)
	if ext == "" {
		ext = ".bin" // Fallback
	}

	// Build the original filename for storage
	filename := asset.OriginalFileName
	if filename == "" {
		filename = fmt.Sprintf("%s%s", asset.ID, ext)
	}

	// Download the file from Immich
	fmt.Printf("  Downloading: %s (%s)\n", filename, asset.OriginalMimeType)
	fileData, err := e.immich.DownloadAsset(ctx, asset.ID)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	// Compute hash for deduplication
	hash := ComputeSHA256(fileData)
	fmt.Printf("  SHA256: %s\n", hash[:16]+"...")

	// Check if file already exists in SteadyPhoto (deduplication)
	existingMedia, err := e.mediaRepo.GetByHash(ctx, hash)
	if err == nil && existingMedia != nil {
		fmt.Printf("  ⏭️  Skipping duplicate (hash match: %s)\n", hash[:16])
		return fmt.Errorf("duplicate") // Special error to indicate skip
	}

	// Determine the storage path based on capture date
	var captureDate time.Time
	if asset.FileCreatedAt != nil {
		captureDate = *asset.FileCreatedAt
	} else if asset.LocalDateTime != nil {
		captureDate = *asset.LocalDateTime
	} else {
		captureDate = time.Now()
	}

	datePath := captureDate.Format("2006/01/02")
	relPath := filepath.Join(datePath, filename)

	if e.config.DryRun {
		fmt.Printf("  [DRY RUN] Would save to: %s/%s\n", e.config.OutputDir, relPath)
		fmt.Printf("  [DRY RUN] Would create DB record for user %s\n", userID)
		return nil
	}

	// Ensure storage directory exists
	if err := e.storageSvc.EnsureDir(userID, relPath); err != nil {
		return fmt.Errorf("failed to create storage dir: %w", err)
	}

	// Write file to disk
	fullPath := filepath.Join(e.config.OutputDir, e.storageSvc.GetUserRelativePath(userID, relPath))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(fullPath, fileData, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	fmt.Printf("  Saved: %s\n", fullPath)

	// Build metadata from Immich EXIF data
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

	// Build tags from Immich tags
	var tags string
	if len(asset.Tags) > 0 {
		tagNames := make([]string, len(asset.Tags))
		for i, tag := range asset.Tags {
			tagNames[i] = tag.Name
		}
		tags = fmt.Sprintf("immich:%s", strings.Join(tagNames, ":"))
	}

	// Create media record in SteadyPhoto database
	now := time.Now()
	media := &domain.Media{
		ID:          uuid.New(),
		UserID:      userID,
		Path:        relPath,
		Filename:    filename,
		Hash:        hash,
		SizeBytes:   int64(len(fileData)),
		Width:       asset.Width,
		Height:      asset.Height,
		CapturedAt:  captureDate,
		MediaType:   domain.MediaType(mediaType),
		Metadata:    metadata,
		CreatedAt:   now,
		UpdatedAt:   now,
		Tags:        tags,
	}

	if err := e.mediaRepo.Create(ctx, media); err != nil {
		return fmt.Errorf("failed to create DB record: %w", err)
	}

	fmt.Printf("  Created DB record: %s (type: %s)\n", media.ID, mediaType)

	// Download sidecar if available
	if asset.HasSidecar {
		sidecarData, err := e.immich.DownloadSidecar(ctx, asset.ID)
		if err == nil && len(sidecarData) > 0 {
			sidecarPath := fullPath + ".xmp"
			if err := os.WriteFile(sidecarPath, sidecarData, 0644); err != nil {
				fmt.Printf("  ⚠️  Failed to save sidecar: %v\n", err)
			} else {
				fmt.Printf("  Saved sidecar: %s.xmp\n", fullPath)
			}
		}
	}

	return nil
}
