package ai

import (
	"context"
)

// NoopFaceDetector is a no-op implementation of FaceDetector for testing and development.
type NoopFaceDetector struct{}

func (d *NoopFaceDetector) DetectFaces(ctx context.Context, imagePath string) ([]FaceDetectionResult, error) {
	return []FaceDetectionResult{}, nil
}
