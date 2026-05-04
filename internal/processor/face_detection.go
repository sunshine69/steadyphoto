package processor

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/google/uuid"
	"steadyphoto/internal/ai"
	"steadyphoto/internal/domain"
)

// FaceDetectionProcessor handles the orchestration of face detection jobs.
type FaceDetectionProcessor struct {
	detector    ai.FaceDetector
	faceRepo    domain.FaceRepository
	photoRepo   domain.PhotoRepository
	storageRoot string
}

// NewFaceDetectionProcessor creates a new instance of the processor.
func NewFaceDetectionProcessor(
	detector ai.FaceDetector,
	faceRepo domain.FaceRepository,
	photoRepo domain.PhotoRepository,
	storageRoot string,
) *FaceDetectionProcessor {
	return &FaceDetectionProcessor{
		detector:    detector,
		faceRepo:    faceRepo,
		photoRepo:   photoRepo,
		storageRoot: storageRoot,
	}
}

// ProcessJob takes a background job and performs the face detection.
func (p *FaceDetectionProcessor) ProcessJob(ctx context.Context, job *domain.Job, photo *domain.Photo) error {
	if job.Type != domain.JobTypeFaceDetection {
		return fmt.Errorf("invalid job type: %s", job.Type)
	}

	if photo.Path == "" {
		return fmt.Errorf("photo path is empty")
	}

	// 1. Resolve absolute path of the image
	imageAbsPath := filepath.Join(p.storageRoot, photo.Path)

	// 2. Run AI Detection
	results, err := p.detector.DetectFaces(ctx, imageAbsPath)
	if err != nil {
		return fmt.Errorf("face detection failed: %w", err)
	}

	// 3. Save results to database
	for _, res := range results {
		face := &domain.Face{
			ID:          uuid.New(),
			PhotoID:     photo.ID,
			BoundingBox: res.BoundingBox,
			Embedding:   res.Embedding,
		}

		if err := p.faceRepo.Create(ctx, face); err != nil {
			return fmt.Errorf("failed to save face to DB: %w", err)
		}
	}

	return nil
}
