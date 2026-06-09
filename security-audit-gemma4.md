# Security Audit Report - SteadyPhoto Project
**Date:** 2024-05-23
**Auditor:** Gemma 4
**Scope:** Backend (Go), Frontend (Angular), and Android Client (Kotlin/Go-mobile)

## 1. Executive Summary
The SteadyPhoto project implements a multi-tenant media management system. The architecture focuses on data isolation via `user_id` paths and SHA256-based deduplication. While the core logic for ownership enforcement is present, several high-risk areas require immediate attention, particularly regarding **Path Traversal**, **Resource Exhaustion (DoS)**, and **Input Validation**.

---

## 2. Critical Vulnerabilities & Risks

### 2.1 Path Traversal (High Risk)
**Observation:** The system generates storage paths using:
`dir := fmt.Sprintf("storage/%s/%s", userID, time.Now().Format("2006/01/02"))`
`destPath := filepath.Join(dir, filename)`

**Risk:** If the `filename` is taken directly from the `multipart/form-data` upload request without sanitization, an attacker could provide a filename like `../../etc/passwd` or `../../root/.ssh/authorized_keys`.
- **Impact:** Arbitrary file write/overwrite on the server filesystem.
- **Recommendation:** 
    - Use `filepath.Base(filename)` to strip all directory information from the uploaded filename.
    - Validate that the resulting path remains within the intended `storage/{user_id}` root.

### 2.2 Denial of Service (DoS) via Resource Exhaustion (Medium-High Risk)
**Observation:** The system implements `MAX_UPLOAD_SIZE` (default 10MB). However, it's mentioned that this is checked via `Content-Length`.
**Risk:** 
1. **Content-Length Spoofing:** Attackers can send a small `Content-Length` but stream gigabytes of data. If the server doesn't use a `LimitedReader`, it may exhaust disk space or memory.
2. **Hash Computation Cost:** Computing SHA256 for every upload is CPU intensive. An attacker could flood the server with unique large files to spike CPU usage.
- **Recommendation:**
    - Use `http.MaxBytesReader` in the Go handler to strictly enforce size limits regardless of the header.
    - Implement rate limiting per `user_id` to prevent CPU exhaustion from hashing.

### 2.3 Logical Deletion vs. Physical Storage (Medium Risk)
**Observation:** The project uses "Logical Only" deletion (removing DB records but keeping files on disk).
**Risk:**
- **Storage Exhaustion:** The server will eventually run out of disk space as users "delete" and "re-upload" media.
- **Data Privacy/GDPR:** "Right to Erasure" is not fully implemented. Data remains on disk indefinitely.
- **Recommendation:** Implement a background cleanup worker (Garbage Collector) that deletes orphaned files (files in `storage/` that have no corresponding entry in the `media` table).

### 2.4 Insecure Direct Object Reference (IDOR) (Medium Risk)
**Observation:** The system uses UUIDs for `albums` and `media`, which mitigates guessing. However, the report mentions "ownership verification" in the API specs.
**Risk:** If any endpoint (e.g., `DELETE /api/albums/{id}/media/{media_id}`) fails to verify that *both* the album and the media item belong to the authenticated `user_id`, a user could potentially delete someone else's media by guessing/obtaining a UUID.
- **Recommendation:** Ensure every database query for a specific resource includes the `user_id` in the `WHERE` clause:
  `DELETE FROM album_media WHERE album_id = ? AND media_id = ? AND user_id = ?`

---

## 3. Android Client Security Analysis

### 3.1 Gomobile Binding Security
**Observation:** The Android client uses Gomobile to call Go functions like `MediaHasher(filePath string)`.
**Risk:** If the `filePath` passed from Kotlin to Go is not validated, it could potentially lead to unauthorized file access on the Android device (though restricted by Android's sandbox).
- **Recommendation:** Validate that the `filePath` belongs to the expected MediaStore directories before passing it to the native Go layer.

### 3.2 Token Storage
**Observation:** The Android client implements Login/Auth.
**Risk:** If JWTs are stored in `SharedPreferences` in plain text, they are vulnerable on rooted devices.
- **Recommendation:** Use **EncryptedSharedPreferences** (Jetpack Security library) to store authentication tokens.

### 3.3 Permission Over-privilege
**Observation:** The app requests `READ_MEDIA_IMAGES` and `READ_MEDIA_VIDEO`.
**Risk:** While necessary for the app's function, the app should implement "Least Privilege".
- **Recommendation:** Ensure the app clearly explains *why* these permissions are needed in the UI (which is partially implemented in `PermissionHelper`).

---

## 4. Architectural Recommendations

### 4.1 Metadata Extraction (Future Phase)
**Warning:** The planned use of `ffprobe` or `ffmpeg` for video metadata extraction is a significant security risk.
- **Risk:** Video files can be crafted to trigger buffer overflows or command injection in `ffmpeg`.
- **Recommendation:** 
    - Run `ffmpeg`/`ffprobe` in a restricted sandbox (e.g., a separate low-privilege container or using `seccomp`).
    - Never pass user-supplied filenames directly as arguments to shell commands; use `exec.Command` with a slice of arguments.

### 4.2 Authentication & IAM
**Observation:** The system uses JWT/Session-based auth.
**Recommendation:**
- Ensure JWTs have a short expiration time.
- Implement Refresh Tokens to maintain sessions without long-lived access tokens.
- Implement a "Revocation List" (Blacklist) for logged-out tokens.

---

## 5. Summary Checklist for Developers

| Feature | Status | Action Required |
| :--- | :--- | :--- |
| File Uploads | ⚠️ Warning | Use `filepath.Base()` and `http.MaxBytesReader`. |
| Deletion | ⚠️ Warning | Implement physical file cleanup worker. |
| Permissions | ✅ Good | Scoped storage is used. |
| Android Auth | ⚠️ Warning | Migrate to `EncryptedSharedPreferences`. |
| Admin API | ⚠️ Warning | Strictly verify `role == 'admin'` on all `/api/admin/*` endpoints. |
| Metadata | 🚧 Planned | Sandbox `ffmpeg` execution. |
