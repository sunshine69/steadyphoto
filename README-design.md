# README-design.md

## Project Overview

SteadyPhoto is a comprehensive photo management system with AI-powered capabilities. The platform allows users to organize, search, and manage their media collections efficiently.


## Video Support in Scanner

### Overview

We are extending the scanner to support video files alongside existing photo support. This includes detection, metadata extraction, storage organization, and streaming capabilities.

### Technical Specifications

#### 1. Video Detection & Classification
- **Supported Formats**: MP4, MOV, AVI, MKV, WEBM, FLV
- **Detection Method**: File extension validation + MIME type verification using libvips
- **File Size Limits**: Maximum 5GB per video file to prevent memory issues
- **Deduplication**: SHA256 hash of file content for duplicate detection

#### 2. Metadata Extraction
- **Video Properties**:
  - Duration (seconds)
  - Resolution (width × height)
  - Bitrate (average and peak)
  - Codec (video and audio)
  - Creation/Modification timestamps
  - Frame rate
- **Storage**: JSONB column `video_metadata` in database
- **Thumbnail Generation**: Extract first frame or keyframe as thumbnail image

#### 3. Storage Organization
- **Directory Structure**:
  ```
  storage/
    YYYY/
      MM/
        DD/
          .videos/     # Video files
          .thumbnails/ # Video thumbnails
  ```
- **Path Resolution**: Use existing `StorageService` for consistent path translation
- **Naming Convention**: Maintain relative paths from database entries

#### 4. API Streaming Endpoints
- **Video Streaming**:
  - Leverage existing Range request support for efficient seeking
  - Content-Type headers: `video/mp4`, `video/webm`, etc.
  - Metadata endpoint: `/api/media/{id}/metadata` returns video-specific info
- **Thumbnail Endpoint**: `/api/media/{id}/thumbnail` serves extracted frame

#### 6. Frontend Integration Requirements
- **Media Type Display**: Show video icon alongside photo icons
- **Playback Controls**: Implement video player component with seek functionality
- **Metadata Display**: Show duration, resolution, and codec info
- **Thumbnail Preview**: Use extracted thumbnails for quick preview

#### 7. Error Handling & Validation
- **Corrupted Files**: Gracefully handle files that fail metadata extraction
- **Unsupported Codecs**: Log warnings but allow import with limited metadata
- **File Integrity**: Validate SHA256 hash against stored value

#### 8. Performance Considerations
- **Batch Processing**: Process videos in batches to avoid memory spikes
- **Caching**: Cache video metadata and thumbnails for faster retrieval
- **Async Operations**: Use background workers for large video processing

### Implementation Timeline

| Phase | Duration | Description |
|-------|----------|-------------|
| 1. Scanner Modifications | 2 days | Add video detection, metadata extraction, storage logic |
| 2. Database Changes | 1 day | Schema updates, migration scripts |
| 3. API Updates | 2 days | Streaming endpoints, metadata retrieval |
| 4. Testing & Validation | 2 days | Unit tests, integration tests with sample videos |
| 5. Documentation | 1 day | Update API docs, user guide |

### Testing Requirements

- **Unit Tests**: Video detection logic, metadata extraction accuracy
- **Integration Tests**: End-to-end video import workflow
- **Performance Tests**: Streaming performance with various video sizes
- **Compatibility Tests**: Multiple video formats and codecs

### Security Considerations

- **File Validation**: Strict MIME type checking to prevent injection attacks
- **Path Sanitization**: Ensure no directory traversal vulnerabilities
- **Access Control**: Maintain existing permission model for video files

## Current Status

- **Core Features**: Photo scanning, deduplication via SHA256 hashing, EXIF extraction, and metadata management
- **Storage Architecture**: Relative path-based storage with `storage/YYYY/MM/DD/` directory structure
- **API Layer**: RESTful API supporting Range requests for efficient file streaming
- **Database**: PostgreSQL with JSONB for metadata and pgvector for AI capabilities
- Front end using angular app. 
  - Images and Video display - done
  - Pagination with page number jump - done (two button Previous / Next and in the middle there is a input box and button for page jump 
  - Video playing when clikcing video - done with seekable support 
  - Image open - done and with full size display 
  - Image search by name only - done - search box at teh top.
