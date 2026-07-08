package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

func getBaseURL() string {
	baseURL := os.Getenv("API_BASE_URL")
	if baseURL == "" {
		log.Fatal("API_BASE_URL environment variable is not set")
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

	log.Printf("Authenticated as user ID: %s", c.userID)
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

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, loginURL, bytes.NewReader(body))
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
	log.Printf("Login successful. Token: %s...", c.token[:min(20, len(c.token))])
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
	log.Printf("Scanning directory: %s", sourceDir)

	mediaTypes := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true,
		".mp4": true, ".mov": true, ".avi": true,
	}

	var files []string
	err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("Error walking path %s: %v", path, err)
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

	log.Printf("Found %d files in %s", len(files), sourceDir)

	uploaded := 0
	skipped := 0
	var errors []string

	for _, file := range files {
		if err := c.uploadFile(ctx, file); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", filepath.Base(file), err))
			log.Printf("Error uploading %s: %v", file, err)
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

	log.Printf("Scan complete - Uploaded: %d, Skipped: %d, Errors: %d", uploaded, skipped, len(errors))

	if len(errors) > 0 {
		jsonErrors, _ := json.MarshalIndent(errors, "", "  ")
		return fmt.Errorf("some uploads failed:\n%s", string(jsonErrors))
	}

	return nil
}

// uploadFile sends a single file to the API for upload.
func (c *ScannerClient) uploadFile(ctx context.Context, filePath string) error {
	log.Printf("Uploading: %s", filePath)

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

	// Create a multipart form body with the file data
	body := &bytes.Buffer{}
	formWriter := multipart.NewWriter(body)

	// Add the file field (required by the API)
	part, err := formWriter.CreateFormFile("file", fileInfo.Name())
	if err != nil {
		return fmt.Errorf("failed to create form file part: %w", err)
	}
	if _, err := io.Copy(part, fileHandle); err != nil {
		return fmt.Errorf("failed to write file data to multipart body: %w", err)
	}

	// Add optional fileName field (the API expects this for metadata)
	formWriter.WriteField("fileName", fileInfo.Name())

	// Close the form writer to finalize the multipart body
	if err := formWriter.Close(); err != nil {
		return fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Create the POST request with the correct endpoint (/api/v1/media/upload/single)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/media/upload/single", body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set authentication header (Bearer token)
	c.addAuthHeader(req)

	// Set Content-Type with boundary from the multipart writer
	req.Header.Set("Content-Type", formWriter.FormDataContentType())

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	responseBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(responseBody))
	}

	log.Printf("Upload successful for %s (status=%d)", filepath.Base(filePath), resp.StatusCode)
	return nil
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Failed to load .env file: %v", err)
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
			log.Fatal("API_BASE_URL environment variable is not set and no -api-url flag provided")
		}
	}

	log.Printf("Starting SteadyPhoto Scanner")
	log.Printf("  API URL: %s", apiURL)
	log.Printf("  Source:  %s", sourceDir)
	log.Printf("  Username:   %s", username)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, err := NewScannerClient(apiURL, username, password)
	if err != nil {
		log.Fatalf("Failed to create scanner client: %v", err)
	}

	if err := client.ScanAndUpload(ctx, sourceDir); err != nil {
		log.Fatalf("Scan failed: %v", err)
	}

	log.Println("Done!")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
