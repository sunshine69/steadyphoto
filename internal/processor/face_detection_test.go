package processor

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"steadyphoto/internal/ai"
	"steadyphoto/internal/domain"
)

// MockFaceRepository is a mock for domain.FaceRepository
type MockFaceRepository struct {
	mock.Mock
}

func (m *MockFaceRepository) Create(ctx context.Context, face *domain.Face) error {
	args := m.Called(ctx, face)
	return args.Error(0)
}

func (m *MockFaceRepository) GetByMediaID(ctx context.Context, photoID uuid.UUID) ([]*domain.Face, error) {
	args := m.Called(ctx, photoID)
	return args.Get(0).([]*domain.Face), args.Error(1)
}

func (m *MockFaceRepository) DeleteByMediaID(ctx context.Context, photoID uuid.UUID) error {
	args := m.Called(ctx, photoID)
	return args.Error(0)
}

// MockMediaRepository is a mock for domain.MediaRepository
type MockMediaRepository struct {
	mock.Mock
}

func (m *MockMediaRepository) Create(ctx context.Context, photo *domain.Media) error {
	args := m.Called(ctx, photo)
	return args.Error(0)
}

func (m *MockMediaRepository) GetByID(ctx context.Context, id uuid.UUID, userID *uuid.UUID) (*domain.Media, error) {
	args := m.Called(ctx, id, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Media), args.Error(1)
}

func (m *MockMediaRepository) GetByHash(ctx context.Context, hash string) (*domain.Media, error) {
	args := m.Called(ctx, hash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Media), args.Error(1)
}

func (m *MockMediaRepository) Update(ctx context.Context, photo *domain.Media) error {
	args := m.Called(ctx, photo)
	return args.Error(0)
}

func (m *MockMediaRepository) Delete(ctx context.Context, id uuid.UUID, userID *uuid.UUID) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

func (m *MockMediaRepository) DeleteByMediaID(ctx context.Context, photoID uuid.UUID) error {
	args := m.Called(ctx, photoID)
	return args.Error(0)
}

func (m *MockMediaRepository) List(ctx context.Context, limit, offset int, userID *uuid.UUID) ([]*domain.Media, int, error) {
	args := m.Called(ctx, limit, offset, userID)
	return args.Get(0).([]*domain.Media), args.Int(1), args.Error(2)
}

func (m *MockMediaRepository) ListByType(ctx context.Context, mediaType domain.MediaType, limit, offset int, userID *uuid.UUID) ([]*domain.Media, int, error) {
	args := m.Called(ctx, mediaType, limit, offset, userID)
	return args.Get(0).([]*domain.Media), args.Int(1), args.Error(2)
}

func (m *MockMediaRepository) SearchByTags(ctx context.Context, tags string, userID *uuid.UUID) ([]*domain.Media, error) {
	args := m.Called(ctx, tags, userID)
	return args.Get(0).([]*domain.Media), args.Error(1)
}

func (m *MockMediaRepository) ListTrashed(ctx context.Context, limit, offset int, userID uuid.UUID) ([]*domain.Media, int, error) {
	args := m.Called(ctx, limit, offset, userID)
	return args.Get(0).([]*domain.Media), args.Int(1), args.Error(2)
}

func (m *MockMediaRepository) GetTrashedMedia(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*domain.Media, error) {
	args := m.Called(ctx, id, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Media), args.Error(1)
}

func (m *MockMediaRepository) RestoreMedia(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

func (m *MockMediaRepository) PermanentlyDeleteMedia(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}
func (m *MockMediaRepository) Search(ctx context.Context, query string, scope string, limit, offset int, userID *uuid.UUID) ([]*domain.Media, int, error) {
	args := m.Called(ctx, query, scope, limit, offset, userID)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*domain.Media), args.Int(1), args.Error(2)
}

// MockFaceDetector is a mock for ai.FaceDetector
type MockFaceDetector struct {
	mock.Mock
}

func (m *MockFaceDetector) DetectFaces(ctx context.Context, imagePath string) ([]ai.FaceDetectionResult, error) {
	args := m.Called(ctx, imagePath)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]ai.FaceDetectionResult), args.Error(1)
}

func TestFaceDetectionProcessor_ProcessJob(t *testing.T) {
	ctx := context.Background()
	photoID := uuid.New()
	photoPath := "2023/01/01/test.jpg"
	storageRoot := "/tmp/storage"

	photo := &domain.Media{
		ID:   photoID,
		Path: photoPath,
	}

	job := &domain.Job{
		ID:      uuid.New(),
		MediaID: photoID,
		Type:    domain.JobTypeFaceDetection,
	}

	mockDetector := new(MockFaceDetector)
	mockFaceRepo := new(MockFaceRepository)
	mockMediaRepo := new(MockMediaRepository)

	// Setup mock expectations
	expectedAbsPath := "/tmp/storage/2023/01/01/test.jpg"
	mockDetector.On("DetectFaces", ctx, expectedAbsPath).Return([]ai.FaceDetectionResult{
		{
			BoundingBox: domain.BoundingBox{X: 10, Y: 10, Width: 50, Height: 50},
			Embedding:   []float32{0.1, 0.2, 0.3},
		},
	}, nil)

	mockFaceRepo.On("Create", ctx, mock.AnythingOfType("*domain.Face")).Return(nil)

	processor := NewFaceDetectionProcessor(mockDetector, mockFaceRepo, mockMediaRepo, storageRoot)

	// Execute
	err := processor.ProcessJob(ctx, job, photo)

	// Assert
	assert.NoError(t, err)
	mockDetector.AssertExpectations(t)
	mockFaceRepo.AssertExpectations(t)
}

func TestFaceDetectionProcessor_ProcessJob_InvalidType(t *testing.T) {
	ctx := context.Background()
	photoID := uuid.New()
	photo := &domain.Media{ID: photoID}
	job := &domain.Job{Type: domain.JobTypeThumbnail} // Wrong type

	processor := NewFaceDetectionProcessor(nil, nil, nil, "/tmp")

	err := processor.ProcessJob(ctx, job, photo)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid job type")
}
