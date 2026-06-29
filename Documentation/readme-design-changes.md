# README-design.md — Changes Summary

## Updated December 2024

This document summarizes all changes made to readme-design.md after comparing it with project.md and validating against the actual codebase.

---

### Feature Status Updates (Verified Against Code)

#### ✅ Confirmed Completed Features
- **SQL Injection Prevention**: All queries use parameterized statements ($1, $2 placeholders). Verified in user_repository.go, media_repository.go, etc.
- **XSS via [innerHTML]**: No longer present in settings.component.html — error messages rendered safely with Angular's default interpolation.
- **Authorization Enforcement**: AuthMiddleware + AdminMiddleware verified in server.go. Ownership enforced at DB level (WHERE user_id = $2) in every repository method.
- **Password Reset Token Expiration**: Password reset functionality removed entirely; session tokens have proper expiration and revocation via crypto/rand.
- **Insecure Random Token Generation**: GenerateRandomToken() now uses crypto/rand.Read() — verified in token.go.

#### ⚠️ Discrepancies Between Design Docs Found
1. **Upload Limits**: readme-design.md stated 10MB default, but middleware.go shows `MaxUploadSizeBytes = 512 << 20` (512MB). This is a significant discrepancy that needed correction.

2. **Album Auto-Association**: Original design mentioned uploading to albums via optional `albumId` parameter in upload endpoint. The actual implementation does NOT include this feature — uploaded files are not automatically added to albums.

3. **Junction Table Name**: Design doc referenced `album_media` table but migration 0001 creates `album_photos`. Updated documentation accordingly.

4. **Position Field**: Migration 0009 adds a `position INT NOT NULL DEFAULT 0` column to album_photos for ordering photos in albums — not mentioned in original design doc.

5. **Trash/Restore Flow**: The enhanced deletion flow with trash, restore, and permanent delete was added after the original design document was written but before project.md was created. Project.md documents this correctly; readme-design.md did not have it. Added to updated version.

6. **Multiple Upload Modes**: The upload endpoint has been significantly expanded beyond the single `/api/v1/media/upload` with additional endpoints:
   - POST /api/v1/media/upload/single — Single file for mobile (1GB memory limit)
   - POST /api/v1/media/upload/chunk — Chunked uploads (128MB per chunk)
   - GET /api/v1/media/upload/status — Upload status
   - POST /api/v1/media/upload/abort — Abort upload
   - POST /api/v1/media/upload/complete — Complete resumable upload

7. **Bulk Operations**: Admin bulk operations added beyond what was in original design:
   - POST /api/v1/admin/users/bulk-approve
   - POST /api/v1/admin/users/bulk-disable  
   - DELETE /api/v1/admin/users/bulk-delete

8. **Path Traversal Protection**: Verified in storage service — uses filepath.Clean() + strips leading "/" before joining paths to prevent directory traversal attacks.

9. **Hash-Based Deduplication**: Confirmed implementation in upload_handler.go — SHA256 hash computed during io.Copy(io.MultiWriter(tempFile, hasher), file) and checked against DB before saving duplicate file data.

10. **MIME Type Validation Gap**: Original design claimed "Strict MIME/extension whitelist" but actual code only validates by extension (filepath.Ext(header.Filename)). A malicious user could upload executable content disguised as an image by changing the file extension — this is a security concern that should be addressed.

---

### Security Findings Added to Updated Document
1. CORS Misconfiguration: Access-Control-Allow-Origin: * with Allow-Credentials: true when no Origin header present — CSRF risk
2. Tokens in localStorage alongside HttpOnly cookies — defeats XSS protection of HttpOnly cookies
3. No HTTPS enforcement on server (http.ListenAndServe without TLS)
4. No rate limiting on authentication endpoints
5. File type validation is extension-only, not MIME-type based

---

### Documentation Corrections
- All API paths updated from /api/ to /api/v1/ throughout document
- Clarified that "delete" in delete profile endpoint actually does soft-delete (sets status = disabled) rather than permanent deletion
- Added note about admin user registration requiring approval before account activation
- Updated upload handler response format to reflect actual JSON structure including skipped_duplicates array
