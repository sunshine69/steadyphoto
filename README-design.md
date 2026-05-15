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
   <user-id>/
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

---


### Implementation Roadmap (Atomic Steps)

#### Phase 1: Database Foundation (Migrations)
*Goal: Prepare the schema for identities and ownership.*

**Step 1.1: Create Identity Tables**  
- **Task**: New migration `0005_add_users_and_sessions.up.sql` creating `users` and `user_sessions`.
- **Details**: 
    - `users`: `id`, `email` (unique), `password_hash`, `created_at`.
    - `user_sessions`: `id`, `user_id`, `refresh_token_hash`, `expires_at`, `is_revoked`.
- **Verification**: Run migrations and confirm tables exist in PostgreSQL.

**Step 1.2: Implement Data Ownership & Storage Partitioning Logic**  
- **Task**: Update existing asset tables (`media`, `albums`) to include a nullable `user_id` column referencing `users(id)`.
- **Storage Change Requirement**: The backend must be updated so that all new files follow the pattern: `storage/{user_id}/YYYY/MM/DD/...`. 
- **Verification**: Verify schema change via SQL query.

---

### Current Status

#### Core Features
- [x] Photo scanning
- [x] Deduplication via SHA256 hashing
- [x] EXIF extraction
- [x] Metadata management (Photos/Videos)
- [x] Video support 

#### Storage Architecture
- [x] Relative path-based storage.
- [x] **Multi-user Partitioned Storage** (Implemented via `storage/{user_id}/...` architecture).

#### API Layer
- [x] RESTful API supporting Range requests for efficient file streaming.
- [x] JWT/Opaque Token Authentication and Ownership enforcement implemented across all media endpoints.

#### Frontend Status (Angular)
- [x] Images and Video display components.
- [x] Pagination with page number jump functionality.
- [x] Seekable video playback.
- [x] Full-size image viewing mode.
- [x] Image search by name.

#### Security & Multi-User Implementation Status (COMPLETED)
- [x] Phase 1: Database Foundation (Completed migrations and schema updates)
- [x] Phase 2: Identity Core (Implemented password hashing and token generation)
- [x] Phase 3: Service Layer (Implemented User/Session repositories and Auth service logic)
- [x] Phase 4: API Handlers (Implemented Register, Login, Refresh, Logout, and Profile update endpoints)
- [x] Phase 5: Middleware & Ownership Enforcement (Implemented Authentication middleware and enforced media ownership in handlers)
- [x] Phase 6: Angular Auth Plumbing (Planned - Implementing interceptors for silent refresh)
- [x] Phase 7: Frontend UI Integration (Planned - Login/Register screens and profile management)
