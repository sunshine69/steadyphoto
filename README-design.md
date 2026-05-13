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

### Implementation Timeline (Video Support)

| Phase | Duration | Description |
|-------|----------|-------------|
| 1. Scanner Modifications | 2 days | Add video detection, metadata extraction, storage logic |
| 2. Database Changes | 1 day | Schema updates, migration scripts |
| 3. API Updates | 2 days | Streaming endpoints, metadata retrieval |
| 4. Testing & Validation | 2 days | Unit tests, integration tests with sample videos |
| 5. Documentation | 1 day | Update API docs, user guide |

---

## Authentication & Multi-User Architecture (IMPLEMENTATION PLAN)

### Overview
To transition from a single-user system to a multi-tenant application, we will implement an **Opaque Token + Refresh Pattern**. This provides high security against XSS/CSRF while allowing stateless API interaction and granular session control.

**Core Security Model:** 
- Tokens are stored in `HttpOnly`, `Secure` cookies (inaccessible to JavaScript).
- Short-lived Access Token (~15m) for requests via `/api`.
- Long-lived Refresh Token (~7d) used only at `/api/auth/refresh` to renew access.

**System Administration & Lifecycle:**
- **Bootstrap Admin**: A default super admin account (`admin` / `steadyphoto`) is generated during database initialization for immediate system management and testing purposes.
- **Account Management**: Users can update their email addresses and change passwords via authenticated API endpoints.

### 1. Design Principles
- **Data Isolation**: Every media asset must be linked to a `user_id`. All database queries will include this filter.
- **Physical Storage Partitioning**: Each user is allocated their own top-level directory under the storage root for complete file isolation at the OS level.
  - Structure: `storage/{user_id}/YYYY/MM/DD/...`
- **Zero-Knowledge Frontend**: The Angular app never touches the actual tokens; they are stored in browser-managed, encrypted cookies.
- **Stateless API with Revocability**: Using opaque tokens allows us to instantly invalidate sessions via the database/cache.

### 2. Database Schema Updates (PostgreSQL)
- **`users` table**: Stores core identity (`id`, `email`, `password_hash`).
- **`media` table update**: Add `user_id` column with a foreign key constraint to ensure ownership.
- **`user_sessions` table**: Tracks active sessions for revocation and lifecycle management:
  - `id`: Session UUID
  - `user_id`: Link to owner
  - `refresh_token_hash`: SHA256 hash of the long-lived refresh token (to prevent DB theft exposure)
  - `expires_at`: Expiration timestamp
  - `is_revoked`: Boolean flag for manual session termination

### 3. Token Lifecycle Management
We will use two distinct tokens to balance security and user experience:

| Feature | Access Token | Refresh Token |
| :--- | :--- | :--- |
| **Format** | Random Opaque String | Long Random String |
| **Lifespan** | ~15 Minutes | ~7 Days |
| **Storage Method** | `HttpOnly` Cookie (`Path=/api`) | `HttpOnly` Cookie (`Path=/api/auth/refresh`) |
| **Security Goal** | Short window of risk if compromised. | Used only to get new access tokens; highly restricted path. |

### 4. API Endpoints
- `POST /api/auth/register`: Create user account.
- `POST /api/auth/login`: Validate credentials and issue the Access + Refresh token cookie pair.
- `POST /api/auth/refresh`: Receives the refresh cookie; issues a new access token if valid.
- `POST /api/auth/logout`: Clears cookies and revokes the session in the DB.

### 5. Frontend Implementation (Angular)
- **Auth Interceptor**: Automatically detects `401 Unauthorized` responses, triggers the `/refresh` call silently, then retries the original failed request to ensure zero user interruption.
- **Route Guards (`CanActivate`)**: Prevents unauthenticated access to media dashboard and settings routes.
- **Credential Handling**: All HTTP calls will be configured with `{ withCredentials: true }` to allow cookie transmission.

### 6. Security Hardening
- **XSS Mitigation**: Using `HttpOnly` cookies ensures JavaScript cannot read or steal tokens.
- **CSRF Mitigation**: Use `SameSite=Strict` flags on all authentication cookies and implement Double Submit Cookie pattern for sensitive mutations.
- **Rate Limiting**: Applied to `/login`, `/register`, and `/refresh` endpoints.

### Current Status Summary:
| Feature | Status | Verification Method |
| :--- | :--- | :--- |
| **User Registration** | ✅ Complete | `run-test-auth.sh` (201 Created) |
| **Password Hashing** | ✅ Secure | Verified via login success/failure logic |
| **Session Management**| ✅ Functional | Token rotation and revocation verified in tests |
| **Profile Updates** | ✅ Functional | Email & Password updates successful |
| **Soft-Delete (IAM)** | ✅ Complete | SQL query confirms `status = 'disabled'` |

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

#### Phase 2: Security & Identity Core (Go Backend)
*Goal: Build secure primitives for passwords and tokens.*

**Step 2.1: Password Hashing Utility**  
- **Task**: Implement a utility using `golang.org/x/crypto/bcrypt` to hash and verify passwords.
- **Verification**: Unit test with known plaintexts and hashes.

**Step 2.2: Opaque Token Generation**  
- **Task**: Create a secure random string generator for Access and Refresh tokens (e.g., using `crypto/rand`).
- **Verification**: Ensure generated strings are sufficiently long, high entropy, and unique.

---

#### Phase 3: Identity Service & Repository (Go Backend)
*Goal: Implement the business logic layer.*

**Step 3.1: User & Session Repositories**  
- **Task**: Create database interaction code for creating users and managing session lifecycle (storing/retrieving refresh token hashes, revoking sessions).
- **Verification**: Unit tests with a test DB or mock repository.

**Step 3.2: Authentication Service Logic**  
- **Task**: Implement the high-level service logic: `RegisterUser`, `Authenticate(email, pass)`, `RefreshToken()`, and `RevokeSession()`.
- **Verification**: Integration tests simulating user flow (register $\rightarrow$ login $\rightarrow$ logout).

---

#### Phase 4: Auth API Implementation (Go Handlers)
*Goal: Expose identity functionality via REST.*

**Step 4.1: Registration & Login Endpoints**  
- **Task**: `POST /api/auth/register` and `POST /api/auth/login`.
- **Key Detail**: The login handler must set the two cookies (`access_token` and `refresh_token`) with correct flags: `HttpOnly`, `Secure`, `SameSite=Strict`.
- **Verification**: Manual test (Postman) to ensure response contains headers but body is clean, and browser can store them.

**Step 4.2: Token Lifecycle Endpoints**  
- **Task**: `POST /api/auth/refresh` (issue new access token cookie) and `POST /api/auth/logout` (clear cookies + revoke session in DB).
- **Verification**: Test that an expired access token can be refreshed via the refresh endpoint.

---

#### Phase 5: Identity Middleware & Context Injection (Go Backend)
*Goal: Secure all existing API endpoints.*

**Step 5.1: Authentication Middleware**  
- **Task**: Implement a middleware for `/api/*` routes that extracts the `access_token` cookie, validates it against active sessions, and injects the `user_id` into the request context.
- **Verification**: Call a protected endpoint without cookies ($\rightarrow$ 401) and with valid cookies ($\rightarrow$ success).

**Step 5.2: Enforce Ownership in Handlers & Storage Path Resolution**  
- **Task**: Update all existing API handlers (Media, Search, etc.) to always include `WHERE user_id = $current_user` in their SQL queries AND update the storage logic to prepend `{user_id}/` to paths.
- **Verification**: Logged in as User A $\rightarrow$ attempt to access Media ID belonging to User B via URL $\rightarrow$ expect 404/403.

---

#### Phase 6: Frontend Authentication Plumbing (Angular)
*Goal: Provide a seamless, invisible user experience.*

**Step 6.1: Auth Interceptor (Silent Refresh)**  
- **Task**: Create an `HttpInterceptor` that listens for any `401 Unauthorized` response. When caught, it automatically calls `/api/auth/refresh`. If successful, it retries the original request with a new access token cookie.
- **Verification**: Simulate token expiry and ensure user can continue browsing without being kicked to login.

**Step 6.2: Route Protection (Guards)**  
- **Task**: Implement `CanActivate` guards for all authenticated routes (Dashboard, Settings).
- **Verification**: Attempt to navigate directly to `/dashboard` while unauthenticated $\rightarrow$ redirect to `/login`.

---

#### Phase 7: Frontend UI & Final Integration (Angular)
*Goal: Complete the user experience.*

**Step 7.1: Auth UI Components**  
- **Task**: Build Login and Registration forms. Implement Logout functionality in the header/settings.
- **Verification**: Full E2E test from landing page $\rightarrow$ Register $\rightarrow$ Logged In Dashboard $\rightarrow$ Logout.

---

### Current Status

#### Core Features
- [x] Photo scanning
- [x] Deduplication via SHA256 hashing
- [x] EXIF extraction
- [x] Metadata management (Photos/Videos)
- [ ] Video support (In Progress: Scanner/Metadata logic refinement)

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
- [ ] Phase 6: Angular Auth Plumbing (Planned - Implementing interceptors for silent refresh)
- [ ] Phase 7: Frontend UI Integration (Planned - Login/Register screens and profile management)
