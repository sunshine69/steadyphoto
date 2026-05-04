package ai

import (
	"context"

	"steadyphoto/internal/domain"
)

// FaceDetectionResult contains the results of a face detection operation.
type FaceDetectionResult struct {
	BoundingBox domain.BoundingBox
	Embedding   []float32
}

// FaceDetector defines the interface for AI-based face detection.
// This allows us to swap between different models (e.g., MTCNN, RetinaFace) 
// and different runtimes (e.g., ONNX, TensorFlow, or a Mock).
type FaceDetector interface {
	// DetectFaces finds all faces in an image and returns their bounding boxes and embeddings.
	DetectFaces(ctx context.Context, imagePath string) ([]FaceDetectionResult, error)
}
