package storage

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Service manages access to the physical storage on disk
type Service struct {
	rootPath string
}

// NewService creates a new storage service with a defined root directory
func NewService(rootPath string) (*Service, error) {
	// Ensure the path is absolute and cleaned
	absRoot, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path for root: %w", err)
	}

	return &Service{
		rootPath: absRoot,
	}, nil
}

// GetAbsolutePath converts a relative path from the database into a safe, absolute filesystem path
func (s *Service) GetAbsolutePath(relPath string) (string, error) {
	// 1. Clean the input path to prevent basic traversal attempts like /../
	relPath = filepath.Clean("/" + relPath)
	// Remove the leading slash to make it relative for filepath.Join
	relPath = strings.TrimPrefix(relPath, "/")

	// 2. Join with the root path
	absPath := filepath.Join(s.rootPath, relPath)

	// 3. Security Check: Ensure the resulting path is actually inside the rootPath
	// This prevents directory traversal attacks (e.g., relPath = "../../etc/passwd")
	rel, err := filepath.Rel(s.rootPath, absPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("security violation: attempted access outside storage root")
	}

	return absPath, nil
}

// Root returns the base directory for storage
func (s *Service) Root() string {
	return s.rootPath
}
