# README-design.md

## Project Overview

SteadyPhoto is a comprehensive photo and video management system with AI-powered capabilities. The platform allows users to organize, search, and manage their media collections efficiently through a secure, multi-tenant architecture.

## Media Capabilities

### 1. Photo & Video Support
The system treats photos and videos as first-class citizens within the same unified pipeline:
- **Detection**: Automatic detection of supported formats (Images: JPG, PNG; Videos: MP4, MOV, AVI, etc.) during scanning.
- **Deduplication**: Content-based deduplication using SHA256 hashing to prevent redundant storage.
- **Metadata Extraction**: 
  - Images: EXIF data (camera model, timestamps, location).
  - Videos: Basic properties and duration via existing metadata parsing logic.

### 2. Seamless Streaming & Playback
The API is optimized for high-performance media consumption:
- **Range Request Support**: The backend supports HTTP Range requests out of the box, enabling seamless video seeking (scrubbing) in modern web players without downloading entire files first.
- **Unified Endpoint Architecture**: All media assets are served via a consistent streaming interface that respects user ownership and authentication.

### 3. Storage Organization
To maintain simplicity and performance, all media is stored using a clean, date-based hierarchy:
```
storage/
  {user_id}/         # Logical isolation per user
    YYYY/
      MM/
        DD/          # Actual file assets (Photos & Videos)
```
*Note: This structure ensures physical data isolation while keeping the directory tree predictable and easy to manage.*

## System Architecture

### Authentication & Security (IAM)
- **Multi-Tenancy**: Strict ownership enforcement; users can only access, view, or delete media linked to their unique `user_id`.
- **Identity Management**: JWT/Session-based authentication with secure password hashing.
- **Security Layers**: Integrated middleware for token validation and cross-origin resource sharing (CORS) protection.

### Frontend Integration
The Angular application provides a responsive dashboard featuring:
- **Unified Media View**: A single, seamless feed of both photos and videos.
- **Advanced Playback**: Native video players with seekable support integrated directly into the media detail view.
- **Search & Discovery**: Fast name-based search and tag-based filtering for rapid asset retrieval.

## Album Feature (Planned)

### Data Model
Albums are logical groupings of existing media assets via a many-to-many relationship to avoid file duplication.

#### `albums` Table
| Column | Type | Description |
| :--- | :--- | :--- |
| `id` | UUID | Primary Key |
| `user_id` | UUID | Foreign Key (owner) |
| `name` | VARCHAR | Album title |
| `description` | TEXT | Optional description |
| `created_at` | TIMESTAMP | Creation time |

#### `album_media` Table
| Column | Type | Description |
| :--- | :--- | :--- |
| `album_id` | UUID | Foreign Key (`albums.id`) |
| `media_id` | UUID/INT | Foreign Key to media asset |

### API Specifications
- **Album Management**: 
    - `POST /api/albums`: Create album.
    - `GET /api/albums`: List user's albums.
    - `PUT /api/albums/{id}`: Update metadata.
    - `DELETE /api/albums/{id}`: Remove album (does not delete media).
- **Media Association**:
    - `POST /api/albums/{id}/media`: Bulk add assets to an album. Requires ownership verification of all provided IDs.
    - `DELETE /api/albums/{id}/media/{media_id}`: Unlink asset from album.

### Frontend Requirements (Angular)
- **Sidebar**: Navigation link for "Albums".
- **Bulk Actions**: Selection mode in media grid to allow adding multiple items to an album at once.
- **Album View**: Dedicated route `/albums/:id` displaying a filtered view of the unified media feed based on album membership.

## Current Status

- ✅ **Core Features**: Photo/Video scanning, deduplication via SHA256 hashing.
- ✅ **Authentication**: Full Multi-user support (IAM) with secure login/registration.
- ✅ **Data Isolation**: Physical storage isolation and API ownership enforcement completed.
- ✅ **Streaming**: High-performance playback for both images and videos with seekable support.
- Implementation of Album Feature (Database, API, UI). - done 
- Presentation mode - completed 
