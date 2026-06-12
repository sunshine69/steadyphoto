package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	baseURL = "https://media.kaykraft.org/api"
	apiKey  = "hlMQkHq4oYqnTRHGnlSsRxTOZgfniW82BoeLvsWqg"
	outDir  = "/mnt/portdata/tmp/immich-migrate" // temp dir for download/upload/delete
)

// SearchResponse mirrors Immich search API response
type SearchResponse struct {
	Assets struct {
		Items []Asset `json:"items"`
	} `json:"assets"`
}

// Asset mirrors Immich asset data (minimal fields we need)
type Asset struct {
	ID               string `json:"id"`
	OriginalFileName string `json:"originalFileName"`
}

// steadyPhotoLoginRequest is the login request body for SteadyPhoto API
type steadyPhotoLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// steadyPhotoLoginResponse is the login response from SteadyPhoto API
type steadyPhotoLoginResponse struct {
	AccessToken string `json:"access_token"`
	UserID      string `json:"user_id,omitempty"`
	Role        string `json:"role,omitempty"`
	Status      string `json:"status,omitempty"`
}

// immichClient handles communication with the Immich API (using /api/ prefix, NOT /api/v1/)
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

// searchAssets retrieves all assets from Immich using the POST /api/search/metadata endpoint
func (c *immichClient) searchAssets(ctx context.Context, page int) ([]Asset, error) {
	body := fmt.Sprintf(`{"page":%d,"size":1000}`, page)

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/search/metadata", strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create search request: %w", err)
	}

	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search failed for page %d: %w", page, err)
	}
	defer resp.Body.Close()

	var result SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %w", err)
	}

	return result.Assets.Items, nil
}

// downloadAsset downloads the original media file from Immich and saves it locally
func (c *immichClient) downloadAsset(ctx context.Context, assetID string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/assets/"+assetID+"/original", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create download request: %w", err)
	}

	req.Header.Set("x-api-key", c.apiKey)

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

// steadyPhotoClient handles communication with the SteadyPhoto API
type steadyPhotoClient struct {
	baseURL   string
	authToken string // Bearer token from login
	client    *http.Client
}

func newSteadyPhotoClient(baseURL string) *steadyPhotoClient {
	return &steadyPhotoClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 10 * time.Minute},
	}
}

// login authenticates with the SteadyPhoto API and stores the auth token
func (c *steadyPhotoClient) login(email, password string) (*steadyPhotoLoginResponse, error) {
	url := fmt.Sprintf("%s/api/v1/auth/login", c.baseURL)
	body, _ := json.Marshal(steadyPhotoLoginRequest{Email: email, Password: password})

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create login request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("login failed: %w", err)
	}
	defer resp.Body.Close()

	var responseBody []byte
	responseBody, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read login response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("login failed (status %d): %s", resp.StatusCode, string(responseBody))
	}

	var result steadyPhotoLoginResponse
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return nil, fmt.Errorf("failed to decode login response: %w", err)
	}

	c.authToken = result.AccessToken
	log.Printf("[AUTH] Logged in as user ID: %s (role: %s)", result.UserID, result.Role)
	return &result, nil
}

// uploadFile uploads a single file using the SteadyPhoto API endpoint
func (c *steadyPhotoClient) uploadFile(ctx context.Context, filename string, mimeType string, fileData []byte) (*map[string]interface{}, error) {
	url := fmt.Sprintf("%s/api/v1/media/upload/single", c.baseURL)

	// Build multipart form body
	// NOTE: SteadyPhoto's single-file upload handler reads fileName from a SEPARATE
	// regular form field via r.FormValue("fileName"), NOT from the Content-Disposition
	// header of the file part. We must write it as a separate field.
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	if err := writer.WriteField("fileName", filename); err != nil {
		return nil, fmt.Errorf("failed to write fileName form field: %w", err)
	}

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create form part: %w", err)
	}
	part.Write(fileData)

	err = writer.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close form: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create upload request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+c.authToken)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upload failed: %w", err)
	}
	defer resp.Body.Close()

	var responseBody []byte
	responseBody, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read upload response: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("authentication failed for upload - token may have expired: %s", string(responseBody))
	}

	var result map[string]interface{}
	if json.Unmarshal(responseBody, &result); err != nil {
		return nil, fmt.Errorf("failed to decode upload response (status %d): %w", resp.StatusCode, err)
	}

	// Check for duplicate
	if skippedDuplicates, ok := result["skipped_duplicates"].([]interface{}); ok && len(skippedDuplicates) > 0 {
		return nil, fmt.Errorf("duplicate") // Special error to indicate skip
	}

	// Check for errors in response
	if errors, ok := result["errors"].([]interface{}); ok && len(errors) > 0 {
		return nil, fmt.Errorf("upload returned errors: %v", errors)
	}

	uploaded, ok := result["uploaded"].([]interface{})
	if !ok || len(uploaded) == 0 {
		return nil, fmt.Errorf("unexpected upload response")
	}

	// Return the first uploaded item as a map
	uploadItem, ok := uploaded[0].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected uploaded item format")
	}

	return &uploadItem, nil
}

// migrationStats tracks migration progress
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
	log.Printf("[SKIP] %s", reason)
}

func (s *migrationStats) recordError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.errors++
	log.Printf("[ERROR] %v", err)
}

func (s *migrationStats) printFinalReport(total int) {
	elapsed := time.Since(s.startTime).Round(time.Second)

	fmt.Println("\n=== Migration Report ===")
	fmt.Printf("Total assets processed: %d\n", total)
	fmt.Printf("Successfully imported:  %d\n", s.imported)
	fmt.Printf("Skipped (duplicate):    %d\n", s.skipped)
	fmt.Printf("Errors:                 %d\n", s.errors)
	fmt.Printf("Duration:               %s\n", elapsed)

	if s.errors > 0 {
		fmt.Println("\n⚠️  Migration completed with errors. Check logs for details.")
	} else {
		fmt.Println("\n✅ Migration completed successfully!")
	}
}

func main() {
	steadyEmail := os.Getenv("STEADY_EMAIL")
	steadyPassword := os.Getenv("STEADY_PASSWORD")
	targetURL := os.Getenv("STEADY_URL")
	dryRun := len(os.Getenv("DRY_RUN")) > 0

	if steadyEmail == "" || steadyPassword == "" {
		fmt.Println("Usage: STEADY_EMAIL=... STEADY_PASSWORD=... [STEADY_URL=http://192.168.20.23:7071] go run main.go")
		os.Exit(1)
	}

	if targetURL == "" {
		targetURL = "http://192.168.20.23:7071"
	}

	fmt.Printf("=== Immich to SteadyPhoto Migration ===\n")
	fmt.Printf("Immich URL:        %s\n", baseURL)
	fmt.Printf("SteadyPhoto URL:   %s\n", targetURL)
	fmt.Printf("Target User:       %s\n", steadyEmail)

	if dryRun {
		fmt.Println("\n⚠️  DRY RUN MODE - No changes will be made")
	} else {
		fmt.Println("\nStarting migration... (files downloaded to temp dir, then uploaded and deleted)")
	}

	// Step 1: Authenticate with SteadyPhoto API
	var steadyClient *steadyPhotoClient
	if !dryRun {
		fmt.Println("\n[1/3] Authenticating with SteadyPhoto API...")
		steadyClient = newSteadyPhotoClient(targetURL)
		loginResp, err := steadyClient.login(steadyEmail, steadyPassword)
		if err != nil {
			log.Fatalf("Failed to authenticate: %v", err)
		}

		fmt.Printf("  Authenticated as user ID: %s (role: %s)\n", loginResp.UserID, loginResp.Role)
	} else {
		fmt.Println("\n[1/3] Skipping authentication (dry run)")
		steadyClient = nil
	}

	// Step 2: Fetch assets from Immich API using POST /api/search/metadata
	client := newImmichClient(baseURL, apiKey)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var allAssets []Asset
	page := 1

	for {
		select {
		case <-ctx.Done():
			log.Printf("Migration cancelled by user")
			return
		default:
		}

		fmt.Printf("\n[2/3] Fetching page %d...\n", page)
		assets, err := client.searchAssets(ctx, page)
		if err != nil {
			log.Fatalf("Failed to fetch assets: %v", err)
		}

		allAssets = append(allAssets, assets...)

		fmt.Printf("  Fetched %d assets (total so far: %d)\n", len(assets), len(allAssets))

		if len(assets) == 0 {
			break // No more pages
		}

		page++
	}

	fmt.Printf("\n  Total assets found: %d\n", len(allAssets))

	// Step 3: Migrate each asset - download, upload to SteadyPhoto, then delete temp file
	fmt.Println("\n[3/3] Migrating media files (download → upload → delete)...")
	stats := newMigrationStats()

	batchSize := 10 // Process in batches of 10 if possible

	for i, asset := range allAssets {
		select {
		case <-ctx.Done():
			log.Printf("Migration cancelled by user")
			return
		default:
		}

		fmt.Printf("\n[%d/%d] Processing: %s\n", i+1, len(allAssets), asset.OriginalFileName)

		if err := migrateAsset(ctx, client, steadyClient, asset, dryRun); err != nil {
			stats.recordError(err)
			continue
		}

		stats.recordImported()

		if (i+1)%batchSize == 0 || i+1 == len(allAssets) {
			fmt.Printf("  Progress: %d/%d | Imported: %d | Skipped: %d | Errors: %d\n",
				i+1, len(allAssets), stats.imported, stats.skipped, stats.errors)
		}
	}

	stats.printFinalReport(len(allAssets))

	fmt.Printf("\nMigration complete! Imported: %d, Skipped: %d\n", stats.imported, stats.skipped)
}

// migrateAsset handles the migration of a single Immich asset to SteadyPhoto.
// Downloads file from Immich → uploads to SteadyPhoto → deletes temp file.
func migrateAsset(
	ctx context.Context,
	immichClient *immichClient,
	steadyClient *steadyPhotoClient,
	asset Asset,
	dryRun bool,
) error {
	ext := filepath.Ext(asset.OriginalFileName)
	base := strings.TrimSuffix(asset.OriginalFileName, ext)

	localPath := filepath.Join(
		outDir,
		fmt.Sprintf("%s_%s%s",
			base,
			asset.ID[:8],
			ext,
		),
	)

	// Download the file from Immich (same as backup script - we need to fetch it first)
	fmt.Printf("  Downloading: %s (%s)\n", asset.OriginalFileName, ext)
	fileData, err := immichClient.downloadAsset(ctx, asset.ID)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	if dryRun {
		fmt.Printf("  [DRY RUN] Would upload via API and delete temp file\n")
		os.Remove(localPath) // Clean up any leftover temp file from previous run
		return nil
	}

	if steadyClient == nil {
		return fmt.Errorf("steadyPhoto client is nil - authentication required (or use DRY_RUN=true)")
	}

	// Upload the file using the SteadyPhoto API endpoint.
	fmt.Printf("  Uploading via API: %s\n", asset.OriginalFileName)
	uploadResp, err := steadyClient.uploadFile(ctx, asset.OriginalFileName, "", fileData)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate") {
			log.Printf("[SKIP] Duplicate detected for %s", asset.OriginalFileName)
			os.Remove(localPath)           // Clean up temp file
			return fmt.Errorf("duplicate") // Special error to indicate skip
		}
		return fmt.Errorf("upload via API failed: %w", err)
	}

	fmt.Printf("  Uploaded successfully - Media ID: %v\n", (*uploadResp)["id"])

	// Delete the temp file after successful upload to save storage
	os.Remove(localPath)
	fmt.Printf("  Temp file deleted: %s\n", localPath)

	return nil
}
