package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/jbrodriguez/mlog"

	"github.com/joho/godotenv"
)

func init() {
	mlog.Start(mlog.LevelInfo, "")
}

func getBaseURL() string {
	baseURL := os.Getenv("API_BASE_URL")
	if baseURL == "" {
		mlog.Fatal("API_BASE_URL environment variable is not set")
	}
	return baseURL
}

// LoginRequest represents the login request body.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthResponse represents the login response.
type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	UserID       string `json:"user_id"`
	Role         string `json:"role,omitempty"`
	Status       string `json:"status,omitempty"`
}

// ScannerClient provides HTTP client methods for the SteadyPhoto API.
type ScannerClient struct {
	baseURL string
	client  *http.Client
	token   string // JWT token after login
	userID  string // authenticated user ID
}

func NewScannerClient(baseURL, username, password string) (*ScannerClient, error) {
	c := &ScannerClient{
		baseURL: baseURL,
		client:  &http.Client{},
	}

	if err := c.login(username, password); err != nil {
		return nil, fmt.Errorf("failed to login: %w", err)
	}

	mlog.Info("Authenticated as user ID: %s", c.userID)
	return c, nil
}

// login authenticates with the API and stores the JWT token.
func (c *ScannerClient) login(username string, password string) error {
	loginURL := fmt.Sprintf("%s/api/v1/auth/login", c.baseURL)

	payload := LoginRequest{
		Email:    username,
		Password: password,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal login request: %w", err)
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, loginURL, io.NopCloser(strings.NewReader(string(body))))
	if err != nil {
		return fmt.Errorf("failed to create login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	responseBody, _ := io.ReadAll(resp.Body)

	var authResp AuthResponse
	if err := json.Unmarshal(responseBody, &authResp); err != nil {
		return fmt.Errorf("failed to parse login response (status=%d): %s - %w", resp.StatusCode, string(responseBody), err)
	}

	// Handle error from API if present in the JSON body or via status code
	if authResp.Status == "error" || resp.StatusCode >= 400 {
		return fmt.Errorf("login failed: status=%d, message=%s", resp.StatusCode, authResp.Status)
	}

	c.token = authResp.AccessToken
	c.userID = authResp.UserID
	mlog.Info("Login successful. Token: %s...", c.token[:min(20, len(c.token))])
	return nil
}

// addAuthHeader adds the Bearer token to the request headers for authentication.
func (c *ScannerClient) addAuthHeader(req *http.Request) {
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
}

// ScanAndUpload scans a source directory for media files and uploads them via the API.
func (c *ScannerClient) ScanAndUpload(ctx context.Context, sourceDir string) error {
	mlog.Info("Scanning directory: %s", sourceDir)

	mediaTypes := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true,
		".mp4": true, ".mov": true, ".avi": true,
	}

	var files []string
	err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			mlog.Info("Error walking path %s: %v", path, err)
			return nil // Continue scanning other paths
		}

		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if mediaTypes[ext] {
			files = append(files, path)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to walk source directory: %w", err)
	}

	mlog.Info("Found %d files in %s", len(files), sourceDir)

	uploaded := 0
	skipped := 0
	var errors []string

	for _, file := range files {
		if err := c.uploadFile(ctx, file); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", filepath.Base(file), err))
			mlog.Info("Error uploading %s: %v", file, err)
		} else {
			uploaded++
		}

		// Rate limit uploads to avoid overwhelming the server
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	mlog.Info("Scan complete - Uploaded: %d, Skipped: %d, Errors: %d", uploaded, skipped, len(errors))

	if len(errors) > 0 {
		jsonErrors, _ := json.MarshalIndent(errors, "", "  ")
		return fmt.Errorf("some uploads failed:\n%s", string(jsonErrors))
	}

	return nil
}

// uploadFile sends a single file to the API for upload using streaming to avoid
// buffering the entire file in memory. This is critical for large files (e.g. 2GB+).
func (c *ScannerClient) uploadFile(ctx context.Context, filePath string) error {
	mlog.Info("Uploading: %s", filePath)

	// Open the file
	fileHandle, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer fileHandle.Close()

	fileInfo, err := fileHandle.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	// Create a pipe for streaming: one goroutine writes multipart data to the pipe writer,
	// the HTTP client reads from the pipe reader and sends it directly to the network.
	// This ensures the file is never fully buffered in memory.
	pr, pw := io.Pipe()
	formWriter := multipart.NewWriter(pw)

	// Start a goroutine to write the multipart body to the pipe
	go func() {
		var writeErr error

		// Add the file field (required by the API)
		part, err := formWriter.CreateFormFile("file", fileInfo.Name())
		if err != nil {
			writeErr = fmt.Errorf("failed to create form file part: %w", err)
			pw.CloseWithError(writeErr)
			return
		}
		if _, err := io.Copy(part, fileHandle); err != nil {
			writeErr = fmt.Errorf("failed to copy file to multipart body: %w", err)
			pw.CloseWithError(writeErr)
			return
		}

		// Add metadata fields
		formWriter.WriteField("fileName", fileInfo.Name())
		formWriter.WriteField("fileCreatedAt", fileInfo.ModTime().Format("2006/01/02 15:04:05"))

		// Close the form writer to finalize the multipart body
		if err := formWriter.Close(); err != nil {
			writeErr = fmt.Errorf("failed to close multipart writer: %w", err)
			pw.CloseWithError(writeErr)
			return
		}

		// Signal that we're done writing
		pw.Close()
	}()

	// Create the POST request with the pipe as the body
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/media/upload/single", pr)
	if err != nil {
		pr.Close()
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set authentication header (Bearer token)
	c.addAuthHeader(req)

	// Set Content-Type with boundary from the multipart writer
	req.Header.Set("Content-Type", formWriter.FormDataContentType())

	resp, err := c.client.Do(req)
	if err != nil {
		pr.Close()
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	responseBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(responseBody))
	}

	mlog.Info("Upload successful for %s (status=%d)", filepath.Base(filePath), resp.StatusCode)
	return nil
}

func main() {
	if err := godotenv.Load(); err != nil {
		mlog.Info("Warning: Failed to load .env file: %v", err)
	}

	apiURL := ""

	var sourceDir string
	flag.StringVar(&sourceDir, "source", ".", "Directory to scan for media files")

	var username string
	flag.StringVar(&username, "u", "admin@steadyphoto.com", "Username/email for authentication (short)")
	flag.StringVar(&username, "email", "admin@steadyphoto.com", "Username/email for authentication")

	var password string
	flag.StringVar(&password, "p", "password", "Password for authentication (short)")
	flag.StringVar(&password, "password", "password", "Password for authentication")

	var apiURLFlag string
	flag.StringVar(&apiURLFlag, "api-url", "", "API base URL (overrides API_BASE_URL env var)")

	flag.Parse()

	// Use -api-url flag if provided, otherwise use getBaseURL() which reads from API_BASE_URL
	if apiURLFlag != "" {
		apiURL = apiURLFlag
	} else {
		apiURL = os.Getenv("API_BASE_URL")
		if apiURL == "" {
			mlog.Fatal("API_BASE_URL environment variable is not set and no -api-url flag provided")
		}
	}

	mlog.Info("Starting SteadyPhoto Scanner")
	mlog.Info("  API URL: %s", apiURL)
	mlog.Info("  Source:  %s", sourceDir)
	mlog.Info("  Username:   %s", username)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, err := NewScannerClient(apiURL, username, password)
	if err != nil {
		mlog.Fatalf("Failed to create scanner client: %v", err)
	}

	if err := client.ScanAndUpload(ctx, sourceDir); err != nil {
		mlog.Fatalf("Scan failed: %v", err)
	}

	mlog.Info("Done!")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
