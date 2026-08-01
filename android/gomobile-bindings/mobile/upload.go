package mobile

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// UploadResult represents the result of an upload operation.
type UploadResult struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// ChunkUploadStatus represents the status of a chunked upload session.
type ChunkUploadStatus struct {
	SessionID   string    `json:"uploadId"`
	Filename    string    `json:"fileName"`
	FileSize    int64     `json:"fileSize"`
	UploadedIdx string    `json:"uploadedChunks,omitempty"` // internal: comma-separated chunk indices as string (gobind-compatible)
	TotalChunks int       `json:"totalChunks"`
	IsComplete  bool      `json:"isComplete"`
}

// UploadedChunkIndices returns the list of uploaded chunk indices as a comma-separated string.
func (s ChunkUploadStatus) UploadedChunkIndices() string {
	return s.UploadedIdx
}

// UploadProgress represents progress of a single upload.
type UploadProgress struct {
	SessionID      string  `json:"uploadId,omitempty"`
	Filename       string  `json:"fileName"`
	TotalBytes     int64   `json:"totalBytes"`
	UploadedBytes  int64   `json:"uploadedBytes"`
	Status         string  `json:"status"` // "uploading", "success", "error"
	Message        string  `json:"message,omitempty"`
}

var (
	httpClient = &http.Client{
		Timeout: 15 * time.Minute, // Long timeout for large file uploads on mobile networks
	}
	userAgent = "immich-android/1.0"
)

// UploadSingleFile uploads a single file to the server via Go's http package.
// This is much more reliable than OkHttp on Android because Go handles network
// interruptions gracefully and doesn't have the "Stream Closed" issue.
func UploadSingleFile(filePath string, uploadURL string, authToken string) (*UploadResult, error) {
	// Open the file for reading
	file, err := os.Open(filePath)
	if err != nil {
		return &UploadResult{Success: false, Message: fmt.Sprintf("Failed to open file: %v", err)}, err
	}
	defer file.Close()

	// Get file info for size and name
	info, err := file.Stat()
	if err != nil {
		return &UploadResult{Success: false, Message: fmt.Sprintf("Failed to stat file: %v", err)}, err
	}

	fileName := filepath.Base(filePath)
	fileSize := info.Size()

	// Create multipart form body
	bodyBuffer := &bytes.Buffer{}
	writer := multipart.NewWriter(bodyBuffer)

	// Add the filename field
	err = writer.WriteField("fileName", fileName)
	if err != nil {
		return &UploadResult{Success: false, Message: fmt.Sprintf("Failed to write fileName field: %v", err)}, err
	}

	// Add the fileSize field
	err = writer.WriteField("fileSize", fmt.Sprintf("%d", fileSize))
	if err != nil {
		return &UploadResult{Success: false, Message: fmt.Sprintf("Failed to write fileSize field: %v", err)}, err
	}

	// Create a part for the file itself with streaming upload
	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		return &UploadResult{Success: false, Message: fmt.Sprintf("Failed to create form file part: %v", err)}, err
	}

	// Stream copy from file to the multipart body - this is efficient and won't load entire file into memory
	_, err = io.Copy(part, file)
	if err != nil {
		return &UploadResult{Success: false, Message: fmt.Sprintf("Failed to stream file data: %v", err)}, err
	}

	err = writer.Close()
	if err != nil {
		return &UploadResult{Success: false, Message: fmt.Sprintf("Failed to close multipart writer: %v", err)}, err
	}

	// Create HTTP request with retry logic
	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		if attempt > 1 {
			time.Sleep(time.Duration(attempt) * time.Second) // Exponential backoff: 1s, 2s, then continue
		}

		req, err := http.NewRequest("POST", uploadURL, bodyBuffer)
		if err != nil {
			lastErr = fmt.Errorf("failed to create request: %v", err)
			continue
		}

		// Set headers - this must be done after creating the multipart writer since it sets Content-Type
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("User-Agent", userAgent)

		// Add auth token if provided
		if authToken != "" {
			req.Header.Set("Authorization", "Bearer "+authToken)
		}

		// Execute request with timeout handling
		resp, err := httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed on attempt %d: %v", attempt, err)

			// Check if error is due to network interruption (common cause of "Stream Closed")
			if strings.Contains(err.Error(), "connection reset by peer") ||
				strings.Contains(err.Error(), "broken pipe") ||
				strings.Contains(err.Error(), "stream closed") {
				// These are retryable errors - Go handles them better than OkHttp
				continue
			}

			return &UploadResult{Success: false, Message: lastErr.Error()}, lastErr
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
			// Read and parse response body to check for success/failure
			var result map[string]interface{}
			bodyBytes, _ := io.ReadAll(resp.Body)

			err = json.Unmarshal(bodyBytes, &result)
			if err != nil {
				return &UploadResult{Success: false, Message: fmt.Sprintf("Failed to parse response: %v", err)}, err
			}

			// Check if upload was successful or if it's a duplicate
			if uploaded, ok := result["uploaded"]; ok && len(uploaded.([]interface{})) > 0 {
				return &UploadResult{Success: true}, nil
			} else if skipped, ok := result["skipped_duplicates"]; ok && len(skipped.([]interface{})) > 0 {
				return &UploadResult{Success: false, Message: "Duplicate file detected"}, fmt.Errorf("duplicate file")
			}

			return &UploadResult{Success: true}, nil
		} else if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			lastErr = fmt.Errorf("authentication failed (HTTP %d)", resp.StatusCode)
			return &UploadResult{Success: false, Message: lastErr.Error()}, lastErr
		}

		// For other error codes, try retrying if it's a server-side issue
		if attempt < 3 && (resp.StatusCode == http.StatusServiceUnavailable ||
			resp.StatusCode == http.StatusBadGateway ||
			resp.StatusCode >= 500) {
			lastErr = fmt.Errorf("server error %d on attempt %d", resp.StatusCode, attempt)
			continue
		}

		return &UploadResult{Success: false, Message: fmt.Sprintf("HTTP %d", resp.StatusCode)},
			fmt.Errorf("upload failed with HTTP status %d", resp.StatusCode)
	}

	lastErr = fmt.Errorf("failed after 3 attempts")
	return &UploadResult{Success: false, Message: lastErr.Error()}, lastErr
}

// CreateChunkedUploadSession creates a new chunked upload session on the server.
func CreateChunkedUploadSession(uploadURL string, authToken string, fileName string, fileSize int64, totalChunks int) (*ChunkUploadStatus, error) {
	var lastErr error

	for attempt := 1; attempt <= 3; attempt++ {
		if attempt > 1 {
			time.Sleep(time.Duration(attempt) * time.Second) // Exponential backoff: 1s, 2s, then continue
		}

		var bodyBuffer bytes.Buffer

		// Build form data for creating the session
		writer := multipart.NewWriter(&bodyBuffer)

		err := writer.WriteField("fileName", fileName)
		if err != nil {
			lastErr = fmt.Errorf("failed to write fileName field: %v", err)
			continue
		}

		err = writer.WriteField("fileSize", fmt.Sprintf("%d", fileSize))
		if err != nil {
			lastErr = fmt.Errorf("failed to write fileSize field: %v", err)
			continue
		}

		err = writer.WriteField("totalChunks", fmt.Sprintf("%d", totalChunks))
		if err != nil {
			lastErr = fmt.Errorf("failed to write totalChunks field: %v", err)
			continue
		}

		writer.Close()

		req, err := http.NewRequest("POST", uploadURL+"/api/v1/media/upload/chunked", &bodyBuffer)
		if err != nil {
			lastErr = fmt.Errorf("failed to create request: %v", err)
			continue
		}

		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("User-Agent", userAgent)
		if authToken != "" {
			req.Header.Set("Authorization", "Bearer "+authToken)
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("failed to create session on attempt %d: %v", attempt, err)

			// Check if error is due to network interruption
			if strings.Contains(err.Error(), "connection reset by peer") ||
				strings.Contains(err.Error(), "broken pipe") ||
				strings.Contains(err.Error(), "stream closed") {
				// These are retryable errors - Go handles them better than OkHttp
				continue
			}

			return nil, lastErr
		}

		// Don't defer close here - we need to read the response body first
		bodyBytes, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("failed to read response on attempt %d: %v", attempt, err)
			if strings.Contains(err.Error(), "connection reset by peer") ||
				strings.Contains(err.Error(), "broken pipe") ||
				strings.Contains(err.Error(), "stream closed") {
				// These are retryable errors
				continue
			}
			return nil, lastErr
		}

		if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
			var result map[string]interface{}
			err = json.Unmarshal(bodyBytes, &result)
			if err != nil {
				lastErr = fmt.Errorf("failed to parse response: %v", err)
				continue
			}

			sessionID, ok := result["uploadId"].(string)
			if !ok || sessionID == "" {
				lastErr = fmt.Errorf("no upload ID returned from server")
				continue
			}

			return &ChunkUploadStatus{
				SessionID:   sessionID,
				Filename:    fileName,
				FileSize:    fileSize,
				TotalChunks: totalChunks,
			}, nil
		} else if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			lastErr = fmt.Errorf("authentication failed (HTTP %d)", resp.StatusCode)
			return nil, lastErr
		}

		// For other error codes, try retrying if it's a server-side issue
		if attempt < 3 && (resp.StatusCode == http.StatusServiceUnavailable ||
			resp.StatusCode == http.StatusBadGateway ||
			resp.StatusCode >= 500) {
			lastErr = fmt.Errorf("server error %d on attempt %d", resp.StatusCode, attempt)
			continue
		}

		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	lastErr = fmt.Errorf("failed after 3 attempts")
	return nil, lastErr
}

// UploadChunk uploads a single chunk of a file.
func UploadChunk(chunkData []byte, uploadURL string, authToken string, sessionID string, chunkIndex int, fileName string) (*UploadResult, error) {
	var lastErr error

	for attempt := 1; attempt <= 3; attempt++ {
		if attempt > 1 {
			time.Sleep(time.Duration(attempt) * time.Second) // Exponential backoff: 1s, 2s, then continue
		}

		var bodyBuffer bytes.Buffer
		writer := multipart.NewWriter(&bodyBuffer)

		// Add metadata fields
		err := writer.WriteField("uploadId", sessionID)
		if err != nil {
			lastErr = fmt.Errorf("failed to write uploadId field: %v", err)
			continue
		}

		err = writer.WriteField("chunkIndex", fmt.Sprintf("%d", chunkIndex))
		if err != nil {
			lastErr = fmt.Errorf("failed to write chunkIndex field: %v", err)
			continue
		}

		err = writer.WriteField("fileName", fileName)
		if err != nil {
			lastErr = fmt.Errorf("failed to write fileName field: %v", err)
			continue
		}

		// Add the chunk data as a form file part
		part, err := writer.CreateFormFile("chunk", fmt.Sprintf("chunk_%d_%s", chunkIndex, filepath.Base(fileName)))
		if err != nil {
			lastErr = fmt.Errorf("failed to create form file part: %v", err)
			continue
		}

		_, err = io.Copy(part, bytes.NewReader(chunkData))
		if err != nil {
			lastErr = fmt.Errorf("failed to write chunk data: %v", err)
			continue
		}

		writer.Close()

		req, err := http.NewRequest("POST", uploadURL+"/api/v1/media/upload/chunk", &bodyBuffer)
		if err != nil {
			lastErr = fmt.Errorf("failed to create request: %v", err)
			continue
		}

		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("User-Agent", userAgent)
		if authToken != "" {
			req.Header.Set("Authorization", "Bearer "+authToken)
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("chunk upload failed on attempt %d: %v", attempt, err)

			// Check if error is due to network interruption
			if strings.Contains(err.Error(), "connection reset by peer") ||
				strings.Contains(err.Error(), "broken pipe") ||
				strings.Contains(err.Error(), "stream closed") {
				// These are retryable errors - Go handles them better than OkHttp
				continue
			}

			return &UploadResult{Success: false}, lastErr
		}

		// Don't defer close here - we need to read the response body first
		bodyBytes, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("failed to read response on attempt %d: %v", attempt, err)
			if strings.Contains(err.Error(), "connection reset by peer") ||
				strings.Contains(err.Error(), "broken pipe") ||
				strings.Contains(err.Error(), "stream closed") {
				// These are retryable errors
				continue
			}
			return &UploadResult{Success: false}, lastErr
		}

		var result map[string]interface{}
		err = json.Unmarshal(bodyBytes, &result)
		if err != nil {
			lastErr = fmt.Errorf("failed to parse response: %v", err)
			continue
		}

		success, ok := result["success"].(bool)
		if !ok || !success {
			message, _ := result["message"].(string)
			return &UploadResult{Success: false, Message: message}, fmt.Errorf("chunk upload failed")
		}

		return &UploadResult{Success: true}, nil
	}

	lastErr = fmt.Errorf("failed after 3 attempts")
	return &UploadResult{Success: false}, lastErr
}

// GetChunkedUploadStatus checks the status of a chunked upload session.
func GetChunkedUploadStatus(uploadURL string, authToken string, sessionID string) (*ChunkUploadStatus, error) {
	var lastErr error

	for attempt := 1; attempt <= 3; attempt++ {
		if attempt > 1 {
			time.Sleep(time.Duration(attempt) * time.Second) // Exponential backoff: 1s, 2s, then continue
		}

		req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/media/upload/status?uploadId=%s", uploadURL, sessionID), nil)
		if err != nil {
			lastErr = fmt.Errorf("failed to create request: %v", err)
			continue
		}

		req.Header.Set("User-Agent", userAgent)
		req.Header.Set("Authorization", "Bearer "+authToken)

		resp, err := httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("status check failed on attempt %d: %v", attempt, err)

			// Check if error is due to network interruption
			if strings.Contains(err.Error(), "connection reset by peer") ||
				strings.Contains(err.Error(), "broken pipe") ||
				strings.Contains(err.Error(), "stream closed") {
				// These are retryable errors - Go handles them better than OkHttp
				continue
			}

			return nil, lastErr
		}

		// Don't defer close here - we need to read the response body first
		bodyBytes, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("failed to read response on attempt %d: %v", attempt, err)
			if strings.Contains(err.Error(), "connection reset by peer") ||
				strings.Contains(err.Error(), "broken pipe") ||
				strings.Contains(err.Error(), "stream closed") {
				// These are retryable errors
				continue
			}
			return nil, lastErr
		}

		var result map[string]interface{}
		err = json.Unmarshal(bodyBytes, &result)
		if err != nil {
			lastErr = fmt.Errorf("failed to parse response: %v", err)
			continue
		}

		chunks, ok := result["uploadedChunks"].([]interface{})
		var uploadedIdxStr string
		if ok {
			parts := make([]string, 0, len(chunks))
			for _, chunk := range chunks {
				if idx, isInt := chunk.(float64); isInt {
					parts = append(parts, fmt.Sprintf("%d", int(idx)))
				}
			}
			uploadedIdxStr = strings.Join(parts, ",")
		}

		totalChunks, _ := result["totalChunks"].(float64)
		fileSize, _ := result["fileSize"].(float64)
		isComplete, _ := result["isComplete"].(bool)

		return &ChunkUploadStatus{
			SessionID:   sessionID,
			TotalChunks: int(totalChunks),
			FileSize:    int64(fileSize),
			UploadedIdx: uploadedIdxStr,
			IsComplete:  isComplete,
		}, nil
	}

	lastErr = fmt.Errorf("failed after 3 attempts")
	return nil, lastErr
}

// CompleteChunkedUpload signals the server to assemble all chunks into the final file.
func CompleteChunkedUpload(uploadURL string, authToken string, sessionID string) (*UploadResult, error) {
	var lastErr error

	for attempt := 1; attempt <= 3; attempt++ {
		if attempt > 1 {
			time.Sleep(time.Duration(attempt) * time.Second) // Exponential backoff: 1s, 2s, then continue
		}

		var bodyBuffer bytes.Buffer
		writer := multipart.NewWriter(&bodyBuffer)

		err := writer.WriteField("uploadId", sessionID)
		if err != nil {
			lastErr = fmt.Errorf("failed to write uploadId field: %v", err)
			continue
		}

		writer.Close()

		req, err := http.NewRequest("POST", uploadURL+"/api/v1/media/upload/complete", &bodyBuffer)
		if err != nil {
			lastErr = fmt.Errorf("failed to create request: %v", err)
			continue
		}

		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("User-Agent", userAgent)
		if authToken != "" {
			req.Header.Set("Authorization", "Bearer "+authToken)
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("complete upload failed on attempt %d: %v", attempt, err)

			// Check if error is due to network interruption
			if strings.Contains(err.Error(), "connection reset by peer") ||
				strings.Contains(err.Error(), "broken pipe") ||
				strings.Contains(err.Error(), "stream closed") {
				// These are retryable errors - Go handles them better than OkHttp
				continue
			}

			return &UploadResult{Success: false}, lastErr
		}

		// Don't defer close here - we need to read the response body first
		bodyBytes, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("failed to read response on attempt %d: %v", attempt, err)
			if strings.Contains(err.Error(), "connection reset by peer") ||
				strings.Contains(err.Error(), "broken pipe") ||
				strings.Contains(err.Error(), "stream closed") {
				// These are retryable errors
				continue
			}
			return &UploadResult{Success: false}, lastErr
		}

		var result map[string]interface{}
		err = json.Unmarshal(bodyBytes, &result)
		if err != nil {
			lastErr = fmt.Errorf("failed to parse response: %v", err)
			continue
		}

		success, ok := result["success"].(bool)
		if !ok || !success {
			message, _ := result["message"].(string)
			return &UploadResult{Success: false, Message: message}, fmt.Errorf("complete upload failed")
		}

		return &UploadResult{Success: true}, nil
	}

	lastErr = fmt.Errorf("failed after 3 attempts")
	return &UploadResult{Success: false}, lastErr
}
