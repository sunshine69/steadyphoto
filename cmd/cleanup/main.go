package main

import (
	"encoding/json"
	"fmt"
	"github.com/jbrodriguez/mlog"
	"os"
	"path/filepath"
	"strings"
)

// Session represents an upload session stored in .sessions.json
type Session struct {
	ID             string   `json:"uploadId"`
	Filename       string   `json:"fileName"`
	FileSize       int64    `json:"fileSize"`
	UploadedChunks []int    `json:"uploadedChunks"`
	TotalChunks    int      `json:"totalChunks"`
	CreatedAt      string   `json:"createdAt"`
	UserID         string   `json:"-"` // Not serialized
}

func main() {
	storageRoot := "./storage"
	uploadTempDir := filepath.Join(storageRoot, ".upload-temp")

	if _, err := os.Stat(uploadTempDir); os.IsNotExist(err) {
		mlog.Info("Upload temp directory does not exist - nothing to clean up")
		return
	}

	// Load active sessions from .sessions.json
	activeSessions := loadSessions(uploadTempDir)
	
	mlog.Info("Found %d active sessions", len(activeSessions))
	
	if len(activeSessions) == 0 {
		mlog.Info("No active sessions found - safe to clean all temp files")
	}
	
	// List all temp files
	entries, err := os.ReadDir(uploadTempDir)
	if err != nil {
		mlog.Fatalf("Failed to read upload temp directory: %v", err)
	}
	
	var tempFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".tmp") {
			tempFiles = append(tempFiles, entry.Name())
		}
	}
	
	mlog.Info("Found %d temp files", len(tempFiles))
	
	// Separate files by session
	filesBySession := make(map[string][]string)
	orphans := []string{}
	
	for _, file := range tempFiles {
		sessionID := extractSessionID(file)
		if sessionID == "" {
			orphans = append(orphans, file)
			continue
		}
		
		if _, isActive := activeSessions[sessionID]; isActive {
			filesBySession[sessionID] = append(filesBySession[sessionID], file)
		} else {
			orphans = append(orphans, file)
		}
	}
	
	// Report status
	mlog.Info("\n=== Status Report ===")
	
	for sessionID, files := range filesBySession {
		session := activeSessions[sessionID]
		mlog.Info("Session %s (active): %d/%d chunks uploaded", sessionID, len(session.UploadedChunks), session.TotalChunks)
		for _, f := range files {
			mlog.Info("  - %s", f)
		}
	}
	
	if len(orphans) > 0 {
		mlog.Info("\nOrphaned files (no active session): %d", len(orphans))
		for _, f := range orphans {
			mlog.Info("  - %s", f)
		}
	}
	
	// Ask for confirmation before removing orphaned files
	if len(orphans) > 0 {
		fmt.Println("\nRemove orphaned temp files? (y/N)")
		var response string
		fmt.Scanln(&response)
		
		if strings.ToLower(response) == "y" {
			for _, file := range orphans {
				path := filepath.Join(uploadTempDir, file)
				if err := os.Remove(path); err != nil {
					mlog.Info("Failed to remove %s: %v", file, err)
				} else {
					mlog.Info("Removed %s", file)
				}
			}
		}
	}
	
	mlog.Info("\n=== Cleanup Complete ===")
}

// loadSessions reads .sessions.json and returns active sessions
func loadSessions(uploadTempDir string) map[string]*Session {
	sessionsFile := filepath.Join(uploadTempDir, ".sessions.json")
	
	data, err := os.ReadFile(sessionsFile)
	if err != nil {
		if os.IsNotExist(err) {
			mlog.Info("No sessions file found")
			return make(map[string]*Session)
		}
		mlog.Fatalf("Failed to read sessions file: %v", err)
	}
	
	var sessions map[string]*Session
	if err := json.Unmarshal(data, &sessions); err != nil {
		mlog.Fatalf("Failed to parse sessions file: %v", err)
	}
	
	return sessions
}

// extractSessionID extracts the session ID from a chunk filename
// Format: <sessionID>_<chunkIndex>.tmp
func extractSessionID(filename string) string {
	// Remove .tmp extension
	base := strings.TrimSuffix(filename, ".tmp")
	
	// Split by underscore to get session ID (everything before the last underscore)
	parts := strings.Split(base, "_")
	if len(parts) < 2 {
		return ""
	}
	
	// Session ID is everything except the last part (chunk index)
	return strings.Join(parts[:len(parts)-1], "_")
}
