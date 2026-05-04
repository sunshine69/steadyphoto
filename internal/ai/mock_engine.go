package ai

import (
	"context"

	"steadyphoto/internal/domain"
)

// MockFaceDetector implements FaceDetector for testing purposes.
type MockFaceDetector struct {
	ResultCount int
}

// NewMockFaceDetector creates a new MockFaceDetector.
func NewMockFaceDetector(count int) *MockFaceDetector {
	return &MockFaceDetector{
		ResultCount: count,
	}
}

// DetectFaces returns a predefined set of face detection results.
func (m *MockFaceDetector) DetectFaces(ctx context.Context, imagePath string) ([]FaceDetectionResult, error) {
	results := make([]FaceDetectionResult, m.ResultCount)
	for i := 0; i < m.ResultCount; i++ {
		results[i] = FaceDetectionResult{
			BoundingBox: domain.BoundingBox{
				X:      float64(i * 10),
				Y:      float64(i * 10),
				Width:  50.0,
				Height: 50.0,
			},
			Embedding: []float32{0.1, 0.2, 0.3},
		}
	}
	return results, nil
}
