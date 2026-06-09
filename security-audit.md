# Security Audit Report - SteadyPhoto

## 1. Executive Summary
This report outlines the findings of a thorough security code analysis conducted on the SteadyPhoto backend and Android sync client design. The system implements several strong security patterns, such as multi-tenancy via user-scoped storage and bcrypt password hashing. However, several critical vulnerabilities related to path traversal, resource exhaustion (DoS), and insufficient input validation were identified.

**Severity Levels:**
*   🔴 **CRITICAL**: Immediate risk of data breach or complete system takeover.
*   🟠 **HIGH**: Significant risk of unauthorized access or service disruption.
*   🟡 **MEDIUM**: Potential for exploitation under specific conditions.
*   🔵 **LOW**: Minor security improvements recommended.

---

## 2. Detailed Findings

### [CRITICAL] Path Traversal in `StorageService` (File Access Vulnerability)
**Location:** `internal/storage/service.go`, specifically `GetAbsolutePath` and `ResolvePath`.

**Description:**
The `GetAbsolutePath` function uses `filepath.Join(s.baseDir, relativePath)` to resolve paths. While it attempts to "clean" the input using `filepath.Clean(relativePath)`, this is insufficient to prevent path traversal if a malicious user provides a path containing sequences like `../../`. An attacker could potentially craft a request (e.g., via an upload or by manipulating media metadata) that resolves to files outside of the intended `storage/` directory, such as `/etc/passwd` or system configuration files.

**Impact:**
An attacker can read, overwrite, or delete any file on the host filesystem that the server process has permissions to access. This is a full system compromise risk.

**Recommendation:**
Validate that the resolved absolute path still resides within `s.baseDir`. Use a check like:
```go
if !strings.HasPrefix(fullPath, s.baseDir) {
    return "", errors.New("path traversal attempt detected")
}
```

---

### [HIGH] Unbounded Resource Consumption in Upload Handlers (Denial of Service)
**Location:** `internal/api/upload_handler.go` and `internal/api/upload_handler_single.go`.

**Description:**
1.  **Multipart Form Memory**: In `HandleSingleFileUpload`, the code calls `r.ParseMultipartForm(1 << 30)` (1GB). While this allows large uploads, if many users upload simultaneously, the server will quickly run out of RAM, leading to a crash or OOM killer intervention.
2.  **Temp File Exhaustion**: Both handlers create temporary files (`os.CreateTemp`). There is no mechanism in the code to automatically clean up these files if an upload is abandoned mid-stream (e.g., client disconnects). An attacker could repeatedly start large uploads and then disconnect, filling up the disk with `.tmp` files.

**Impact:**
Denial of Service (DoS) via memory exhaustion or disk space depletion.

**Recommendation:**
*   Reduce the in-memory buffer for `ParseMultipartForm`.
*   Implement a background worker/cron job to periodically clean up old temporary upload files and abandoned sessions from `.upload-temp`.

---

### [HIGH] Insecure Cookie Configuration (Session Hijacking Risk)
**Location:** `internal/api/auth_handler.go`, inside `handleLogin`.

**Description:**
The `access_token` cookie is set with `Secure: false`. 

```go
http.SetCookie(w, &http.Cookie{
    Name:     "access_token",
    Value:    session.ID.String(),
    Path:     "/",
    HttpOnly: true,
    Secure:   false, // <--- VULNERABILITY
    SameSite: http.SameSiteLaxMode,
})
```

**Impact:**
In a production environment using HTTPS (which is mandatory), an attacker could perform a Man-in-the-Middle (MitM) attack to intercept the session cookie if it is transmitted over any part of the connection that falls back to HTTP or via other insecure side channels.

**Recommendation:**
Always set `Secure: true` in production environments. Use environment variables to toggle this based on whether the server is running locally (`http`) or in production (`https`).

---

### [MEDIUM] Missing MIME Type Validation (File Upload Security)
**Location:** `internal/api/upload_handler.go`.

**Description:**
The current implementation relies heavily on file extensions (e.g., `.mp4`, `.jpg`) to determine the media type and allow uploads. An attacker could upload a malicious script (e.g., `shell.php` or a large binary) renamed as `image.jpg`. While the system might not execute it, it's stored on disk, which could be exploited later if another part of the system (or a misconfigured web server) serves these files directly.

**Impact:**
Storage of malicious payloads that can lead to Remote Code Execution (RCE) if the storage directory is ever exposed via a direct-access web server or used by another process.

**Recommendation:**
Use `http.DetectContentType` on the first 512 bytes of the uploaded file content to verify the actual MIME type, rather than trusting the user-provided extension/filename.

---

### [MEDIUM] Information Disclosure in Error Messages (User Enumeration)
**Location:** Various handlers (`handleLogin`, `handleRegister`).

**Description:**
While some parts of `handleLogin` use generic messages ("Invalid credentials"), other errors might inadvertently leak information about the existence or state of a user. For example, if registration explicitly returns "Email already registered", it allows an attacker to probe which emails are in the system (User Enumeration).

**Impact:**
Privacy violation and facilitating targeted attacks against known users.

**Recommendation:**
Standardize error responses for authentication-related actions so they do not reveal whether a specific email address exists or is currently disabled/pending.

---

## 3. Summary of Actions Required

| Task ID | Severity | Category | Description |
| :--- | :--- | :--- | :--- |
| SEC-01 | 🔴 CRITICAL | Path Traversal | Implement strict prefix checking in `StorageService`. |
| SEC-02 | 🟠 HIGH | DoS (Memory) | Lower the memory limit for multipart parsing. |
| SEC-03 | 🟠 HIGH | DoS (Disk) | Add a cleanup task for `.upload-temp` and abandoned sessions. |
| SEC-04 | 🟠 HIGH | Session Security | Ensure `Secure: true` is set on session cookies in production. |
| SEC-05 | 🟡 MEDIUM | File Uploads | Implement content-based MIME type sniffing. |
| SEC-06 | 🟡 MEDIUM | Info Disclosure | Sanitize error messages to prevent user enumeration. |

---
*End of Report*
