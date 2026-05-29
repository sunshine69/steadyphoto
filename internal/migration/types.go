package migration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ImmichAsset represents an asset from the Immich API
type ImmichAsset struct {
	ID              string    `json:"id"`
	DeviceAssetID   string    `json:"deviceAssetId"`
	UserID          string    `json:"userId"`
	LibraryID       *string   `json:"libraryId"`
	Type            string    `json:"type"`
	OriginalPath    string    `json:"originalPath"`
	OriginalFileName string   `json:"originalFileName"`
	OriginalMimeType string  `json:"originalMimeType"`
	Resized         bool      `json:"resized"`
	Thumbhash       *string   `json:"thumbhash"`
	Width           int       `json:"width"`
	Height          int       `json:"height"`
	Duration        *float64  `json:"duration"`
	ExifInfo        *ExifInfo `json:"exifInfo"`
	FileCreatedAt   *time.Time `json:"fileCreatedAt"`
	FileModifiedAt  *time.Time `json:"fileModifiedAt"`
	LocalDateTime   *time.Time `json:"localDateTime"`
	UpdatedAt       *time.Time `json:"updatedAt"`
	IsFavorite      bool      `json:"isFavorite"`
	IsArchived      bool      `json:"isArchived"`
	IsTrashed       bool      `json:"isTrashed"`
	HasSidecar      bool      `json:"hasSidecar"`
	Tags            []Tag     `json:"tags"`
	// Additional fields that may be present but not always
	Stacked *bool `json:"stacked"`
}

// ExifInfo contains EXIF metadata from Immich
type ExifInfo struct {
	Model          string    `json:"model"`
	LensModel      *string   `json:"lensModel"`
	FNumber        *float64  `json:"fNumber"`
	FocalLength    *float64  `json:"focalLength"`
	ISO            *int      `json:"iso"`
	ExposureTime   *string   `json:"exposureTime"`
	Latitude       *float64  `json:"latitude"`
	Longitude      *float64  `json:"longitude"`
	City           *string   `json:"city"`
	State          *string   `json:"state"`
	Country        *string   `json:"country"`
	StartTime      *float64  `json:"startTime"`
}

// Tag represents a tag from Immich
type Tag struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

// ImmichClient provides methods to interact with the Immich API
type ImmichClient struct {
	BaseURL      string
	APIKey       string
	Client       *http.Client
	pageSize     int
	currentPage  int
	totalPages   int
	hasMore      bool
}

// NewImmichClient creates a new Immich API client
func NewImmichClient(baseURL, apiKey string) *ImmichClient {
	return &ImmichClient{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		APIKey:     apiKey,
		Client:     &http.Client{Timeout: 10 * time.Minute},
		pageSize:   1000,
		currentPage: 0,
	}
}

// Headers returns the HTTP headers needed for Immich API requests
func (c *ImmichClient) Headers() http.Header {
	return http.Header{
		"x-api-key": []string{c.APIKey},
		"Accept":    []string{"application/json"},
	}
}

// GetAssets fetches all assets from Immich with pagination
func (c *ImmichClient) GetAssets(ctx context.Context) ([]*ImmichAsset, error) {
	var allAssets []*ImmichAsset
	page := 1

	for {
		select {
		case <-ctx.Done():
			return allAssets, ctx.Err()
		default:
		}

		url := fmt.Sprintf("%s/assets?size=%d&page=%d", c.BaseURL, c.pageSize, page)
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return allAssets, fmt.Errorf("failed to create request: %w", err)
		}

		for k, v := range c.Headers() {
			req.Header.Set(k, v[0])
		}

		resp, err := c.Client.Do(req)
		if err != nil {
			return allAssets, fmt.Errorf("failed to fetch assets page %d: %w", page, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body := make([]byte, 1024)
			n, _ := resp.Body.Read(body)
			return allAssets, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body[:n]))
		}

		var result struct {
			Items  []*ImmichAsset `json:"items"`
			Page   int            `json:"page"`
			Total  int            `json:"total"`
			Size   int            `json:"size"`
			Limit  int            `json:"limit"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return allAssets, fmt.Errorf("failed to decode response: %w", err)
		}

		allAssets = append(allAssets, result.Items...)

		// Check if we have more pages
		if len(result.Items) < c.pageSize || page >= result.Total/c.pageSize+1 {
			break
		}
		page++

		if page%10 == 0 {
			fmt.Printf("  Fetched %d assets so far...\n", len(allAssets))
		}
	}

	return allAssets, nil
}

// DownloadAsset downloads the original media file from Immich
func (c *ImmichClient) DownloadAsset(ctx context.Context, assetID string) ([]byte, error) {
	url := fmt.Sprintf("%s/assets/%s/original", c.BaseURL, assetID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create download request: %w", err)
	}

	for k, v := range c.Headers() {
		req.Header.Set(k, v[0])
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download asset %s: %w", assetID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body := make([]byte, 1024)
		n, _ := resp.Body.Read(body)
		return nil, fmt.Errorf("download error (status %d): %s", resp.StatusCode, string(body[:n]))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read asset data: %w", err)
	}

	return data, nil
}

// DownloadSidecar downloads the XMP sidecar file if it exists
func (c *ImmichClient) DownloadSidecar(ctx context.Context, assetID string) ([]byte, error) {
	url := fmt.Sprintf("%s/assets/%s/sidecar", c.BaseURL, assetID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create sidecar request: %w", err)
	}

	for k, v := range c.Headers() {
		req.Header.Set(k, v[0])
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download sidecar for asset %s: %w", assetID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil // No sidecar available
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read sidecar data: %w", err)
	}

	return data, nil
}

// ComputeSHA256 computes the SHA256 hash of the given data
func ComputeSHA256(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// DetermineMediaType determines if the asset is a photo or video based on MIME type
func DetermineMediaType(mimeType string) string {
	if strings.HasPrefix(mimeType, "video/") {
		return "video"
	}
	return "photo"
}

// ExtractExtension extracts file extension from filename or mime type
func ExtractExtension(filename, mimeType string) string {
	// Try to get extension from filename first
	ext := filepath.Ext(filename)
	if ext != "" {
		return strings.ToLower(ext)
	}

	// Fallback to MIME type mapping
	switch mimeType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "video/mp4":
		return ".mp4"
	case "video/quicktime":
		return ".mov"
	case "video/x-msvideo":
		return ".avi"
	default:
		return ""
	}
}

// ProgressTracker tracks migration progress
type ProgressTracker struct {
	mu             sync.Mutex
	total          int
	downloaded     int
	skipped        int
	errors         int
	startTime      time.Time
}

// NewProgressTracker creates a new progress tracker
func NewProgressTracker(total int) *ProgressTracker {
	return &ProgressTracker{
		total:     total,
		startTime: time.Now(),
	}
}

// Update updates the progress with counts of downloaded, skipped, and errored items
func (p *ProgressTracker) Update(downloaded, skipped, errors int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.downloaded += downloaded
	p.skipped += skipped
	p.errors += errors
}

// PrintStatus prints the current migration status
func (p *ProgressTracker) PrintStatus() {
	p.mu.Lock()
	defer p.mu.Unlock()

	elapsed := time.Since(p.startTime).Round(time.Second)
	remaining := "unknown"
	if p.downloaded+p.skipped > 0 {
		rate := float64(p.downloaded + p.skipped) / elapsed.Seconds()
		left := float64(p.total-p.downloaded-p.skipped) / rate
		remaining = fmt.Sprintf("~%s", time.Duration(left*float64(time.Second)).Round(time.Second))
	}

	fmt.Printf("  Progress: %d/%d (%.1f%%) | Downloaded: %d | Skipped: %d | Errors: %d | Elapsed: %s | ETA: %s\n",
		p.downloaded+p.skipped, p.total,
		float64(p.downloaded+p.skipped)/float64(p.total)*100,
		p.downloaded, p.skipped, p.errors,
		elapsed, remaining)
}

// FinalReport prints the final migration report
func (p *ProgressTracker) FinalReport() {
	fmt.Println("\n=== Migration Report ===")
	fmt.Printf("Total assets processed: %d\n", p.total)
	fmt.Printf("Successfully migrated:  %d\n", p.downloaded)
	fmt.Printf("Skipped (duplicate):    %d\n", p.skipped)
	fmt.Printf("Errors:                 %d\n", p.errors)
	fmt.Printf("Duration:               %s\n", time.Since(p.startTime).Round(time.Second))
	if p.errors > 0 {
		fmt.Println("\n⚠️  Migration completed with errors. Check logs for details.")
	} else {
		fmt.Println("\n✅ Migration completed successfully!")
	}
}

// UserMapping maps Immich user IDs to SteadyPhoto user IDs
type UserMapping struct {
	mu         sync.RWMutex
	immichToSteady map[string]uuid.UUID
}

// NewUserMapping creates a new user mapping instance
func NewUserMapping() *UserMapping {
	return &UserMapping{
		immichToSteady: make(map[string]uuid.UUID),
	}
}

// Set maps an Immich user ID to a SteadyPhoto user UUID
func (m *UserMapping) Set(immichID string, steadyUUID uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.immichToSteady[immichID] = steadyUUID
}

// Get returns the SteadyPhoto user UUID for an Immich user ID
func (m *UserMapping) Get(immichID string) (uuid.UUID, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	id, ok := m.immichToSteady[immichID]
	return id, ok
}

// MigrationConfig holds the configuration for migration
type MigrationConfig struct {
	ImmichURL      string
	ImmichAPIKey   string
	DATABASE_URL   string
	UserEmail      string // Which user's data to migrate (empty = all users)
	DryRun         bool
	Parallel       int    // Number of parallel download workers (default: 3)
	OutputDir      string // For dry-run, where files would be saved
}

// Validate checks if the configuration is valid
func (c *MigrationConfig) Validate() error {
	if c.ImmichURL == "" {
		return fmt.Errorf("Immich URL is required")
	}
	if c.ImmichAPIKey == "" {
		return fmt.Errorf("Immich API key is required")
	}
	if c.DATABASE_URL == "" {
		return fmt.Errorf("DATABASE_URL environment variable must be set")
	}
	if c.Parallel <= 0 {
		c.Parallel = 3
	}
	if c.OutputDir == "" {
		c.OutputDir = "./immich-migration-output"
	}
	return nil
}
