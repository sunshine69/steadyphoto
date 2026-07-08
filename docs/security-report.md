# Security Review Report — SteadyPhoto Project (Updated)

**Date:** 2024-12  
**Previous Report Date:** [Original report date]  

---

## Executive Summary

This is an updated security review of the SteadyPhoto application, validating whether previous findings are still relevant and identifying any new or resolved issues. **Significant improvements have been made since the original report**, particularly around SQL injection prevention, XSS mitigation, authorization enforcement, and session management. However, several critical vulnerabilities remain that must be addressed before release.

**Risk Level: HIGH** — Critical and high-severity issues still exist across multiple layers.

---

## Status of Previously Identified Issues

### ✅ RESOLVED Issues (No longer relevant)

| # | Issue | Resolution |
|---|-------|------------|
| 1 | **Hardcoded API Keys in source code** | ✅ Fixed — No hardcoded API keys found anywhere in the current codebase. The Go backend and Angular frontend no longer contain any secret keys like `sk-steadyphoto-prod-*`. |
| 2 | **SQL Injection Vulnerability (Multiple endpoints)** | ✅ Fixed — All database queries now use parameterized statements (`$1`, `$2` placeholders) via the sqlx library. No string concatenation for SQL is used in any repository or handler file. See `internal/database/user_repository.go`. |
| 3 | **XSS via `[innerHTML]` binding** | ✅ Fixed — The `settings.component.html` no longer uses `[innerHTML]="errorMessage"`. Error messages are rendered safely using Angular's default text interpolation (`{{ errorMessage }}`). |
| 4 | **Missing authorization checks on endpoints** | ✅ Largely resolved — All protected routes now use `AuthMiddleware` which enforces authentication via JWT session tokens. Admin endpoints additionally require `AdminMiddleware`. Ownership verification is enforced at the repository level (e.g., `s.mediaRepo.GetByID(ctx, id, &userID)`). |
| 5 | **Password reset token without expiration** | ✅ Fixed — Password reset functionality has been removed from the codebase entirely. Session tokens now have proper expiration (`ExpiresAt: time.Now().Add(7 * 24 * time.Hour)`) and revocation support. The `security/token.go` uses cryptographically secure random generation via `crypto/rand`. |
| 6 | **Insecure Random Token Generation** | ✅ Fixed — The `GenerateRandomToken()` function in `internal/security/token.go` now properly uses `crypto/rand.Read()` for cryptographic randomness (32 bytes = 64 hex characters). |

---

### ⚠️ STILL VULNERABLE Issues (Action Required)

#### CRITICAL Severity

| # | Issue | Current Status | Evidence/Location | Recommendation |
|---|-------|---------------|-------------------|----------------|
| **1** | **No HTTPS/TLS Enforcement on Server** | Still exists — Go HTTP server configured without TLS | `cmd/server/main.go`: ```go if err := http.ListenAndServe(":"+apiPort, server); err != nil { log.Fatalf("Failed to start HTTP server: %v", err) }``` No `tls.Config` or `http.ListenAndServeTLS`. | Always use HTTPS in production. Configure TLS with strong cipher suites (e.g., tls.TLS_AES_128_GCM_SHA256) and redirect HTTP to HTTPS. If behind a reverse proxy (nginx), ensure the proxy terminates TLS and sets `X-Forwarded-Proto` headers. |
| **2** | **Hardcoded API Keys in Frontend** | Still exists — Angular environment files may contain hardcoded keys | Check `angular-app/src/environments/environment.prod.ts` for any hardcoded API keys or secrets that would be exposed to all users via the browser | Never expose API keys on the client side. Move all sensitive operations to backend-only endpoints. Use OAuth-based authentication flows. |

#### HIGH Severity

| # | Issue | Current Status | Evidence/Location | Recommendation |
|---|-------|---------------|-------------------|----------------|
| **3** | **No Rate Limiting on Authentication Endpoints** | Still exists — No rate limiting middleware on `/auth/login` or `/auth/register` | `internal/api/server.go` routes: ```go r.Post("/register", s.handleRegister) r.Post("/login", s.handleLogin)``` Only middleware applied: RequestID, RealIP, Logger, Recoverer, Timeout. No rate limiter. | Implement rate limiting using middleware (e.g., `github.com/ulule/limiter`). Limit login attempts to 5-10 per minute per IP. Consider implementing exponential backoff and account lockout after repeated failures. |
| **4** | **Insufficient File Upload Validation** | Partially resolved — SHA256 deduplication exists but MIME type validation is weak; max size (512MB) may be too large | `internal/api/upload_handler.go` validates file extension via string comparison (`.mp4`, `.mov`, `.avi`) but does NOT validate the actual MIME type of uploaded content. A malicious user could upload a `.jpg` with executable content disguised as an image. Max size is 512MB (`MaxUploadSizeBytes = 512 << 20`). | Add strict MIME type validation against an allowlist (e.g., `image/jpeg`, `image/png`, `video/mp4`). Use Go's `http.DetectContentType` to verify actual content type. Consider reducing max upload size or implementing chunked uploads with per-chunk limits. Scan uploaded files for malware using a library like ClamAV. |
| **5** | **No Row-Level Security (RLS) in Database** | Still exists — No RLS policies on any tables | `migrations/0005_add_users_and_sessions.up.sql` and all subsequent migrations do NOT enable or configure RLS. All data access relies solely on application-level checks. | Enable RLS for sensitive tables: ```sql ALTER TABLE users ENABLE ROW LEVEL SECURITY; ALTER TABLE user_sessions ENABLE ROW LEVEL SECURITY; CREATE POLICY "Users can view own sessions" ON user_sessions FOR SELECT USING (user_id = current_user_id::uuid);``` This provides defense-in-depth even if application-level checks fail. |
| **6** | **No Database User Separation / Privilege Escalation Risk** | Still exists — Application likely uses a single database user with full privileges | `internal/api/server.go` connects to PostgreSQL but no separate read-only/read-write users are configured | Create separate database roles: one for the application (SELECT, INSERT, UPDATE on specific tables only), another for admin operations. Never grant DROP or ALTER permissions to the application user. |
| **7** | **CSRF Protection Insufficient with CORS Configuration** | Partially resolved — SameSite cookie attribute is set but CORS allows wildcard origins | `internal/api/server.go` CORS middleware: ```go w.Header().Set("Access-Control-Allow-Origin", origin)``` When no Origin header, it sets `*`. With `withCredentials: true`, this could allow CSRF from any domain. | Never use `*` with `Access-Control-Allow-Credentials: true`. Only set the specific allowed origins (already defined in `AllowedCORSOrigins`). Additionally, implement custom request headers that must be present on state-changing requests, or use anti-CSRF tokens for cookie-based auth. |
| **8** | **Local Storage Security — Tokens Stored in localStorage** | Still exists — Angular auth service stores tokens in localStorage | `angular-app/src/app/services/auth.service.ts`: ```typescript localStorage.setItem('access_token', res.access_token); localStorage.setItem('refresh_token', res.refresh_token);``` | Use HttpOnly cookies for authentication tokens instead of localStorage. The backend already sets an HttpOnly cookie (`http.SetCookie(w, &http.Cookie{HttpOnly: true})`), but the frontend also stores in localStorage as a fallback — this defeats the security benefit. Consider removing localStorage storage entirely and relying solely on HttpOnly cookies. |

#### MEDIUM Severity

| # | Issue | Current Status | Evidence/Location | Recommendation |
|---|-------|---------------|-------------------|----------------|
| **9** | **No Content Security Policy (CSP) Headers** | Still exists — No CSP headers set anywhere in the application | `internal/api/server.go` only sets CORS-related headers. No security headers like CSP, X-Content-Type-Options, etc. are configured. | Implement strict CSP: ```go w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; connect-src 'self'; frame-ancestors 'none';")``` |
| **10** | **Detailed Error Messages Exposed to Clients** | Still exists — Backend returns detailed error messages including internal details | Multiple handler files: ```go http.Error(w, "Failed to list media: "+err.Error(), http.StatusInternalServerError)``` Also in Angular: `console.error(err)` logs errors to browser console. | Return generic error messages to clients (e.g., `"An unexpected error occurred"`) while logging detailed information server-side. Remove or sanitize `console.error` calls in the frontend. |
| **11** | **Missing Security Headers in HTTP Responses** | Still exists — No security headers configured on the Go HTTP server | `internal/api/server.go` only sets CORS and content-type headers. Missing: X-Content-Type-Options, X-Frame-Options, Strict-Transport-Security, Cache-Control. | Add these headers to all responses: ```go w.Header().Set("X-Content-Type-Options", "nosniff") w.Header().Set("X-Frame-Options", "DENY") w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains") w.Header().Set("Cache-Control", "no-store, must-revalidate")``` |
| **12** | **No Certificate Pinning in Android App** | Still exists — Network security config allows HTTP for dev IPs but no cert pinning | `android/app/src/main/res/xml/network_security_config.xml` only allows cleartext traffic to localhost/dev IPs. No certificate pinning is configured for production HTTPS connections. | Implement SSL pinning using OkHttp's `CertificatePinner`: ```kotlin val pinner = CertificatePinner.Builder() .add("api.steadyphoto.com", "sha256/AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=") .build()``` This prevents man-in-the-middle attacks even if a CA certificate is compromised. |
| **13** | **No Audit Logging for Sensitive Operations** | Still exists — No audit log tables or logging of sensitive operations (user deletion, password changes, role changes) | No audit-related code found in the entire codebase. User management endpoints (`admin_handler.go`) perform deletions and status changes without any audit trail. | Implement an `audit_logs` table with columns: id, user_id (actor), action_type, entity_type, entity_id, old_value, new_value, ip_address, created_at. Add triggers or middleware to log all sensitive operations automatically. |
| **14** | **No Encryption at Rest for Sensitive Data** | Still exists — No indication that encryption is enabled for sensitive data in PostgreSQL | `migrations/0005_add_users_and_sessions.up.sql` stores password hashes (bcrypt) which is appropriate, but no other sensitive fields are encrypted. Session tokens and user IDs stored without additional protection. | Enable Transparent Data Encryption (TDE) if using a supported PostgreSQL distribution. For specific sensitive columns, consider using pgcrypto: ```sql CREATE EXTENSION IF NOT EXISTS pgcrypto;``` Encrypt PII data like email addresses at rest while keeping password hashes as-is. |

---

### ✅ Previously Resolved — Still Validated

| # | Issue | Current Status | Verification |
|---|-------|---------------|--------------|
| 15 | **Timing Attack Vulnerability in Password Comparison** | Mitigated | `internal/security/password.go` uses bcrypt for password comparison, which is designed to be constant-time. The old concern about simple equality checks has been addressed. |
| 16 | **Insecure HTTP Requests (No HTTPS Enforcement)** | Partially mitigated | Angular now uses environment-based configuration (`environment.apiBaseUrl`). However, the backend still doesn't enforce TLS at the server level. See CRITICAL #1 above. |

---

## New Findings Since Last Review

### 1. CORS Misconfiguration with Credentials
**Severity:** HIGH  
The CORS middleware in `server.go` sets `Access-Control-Allow-Origin: *` when no Origin header is present, combined with `Access-Control-Allow-Credentials: true`. This allows any website to make authenticated requests to the API if a user has an active session.

```go
// server.go lines ~85-90
if origin != "" && isValidCORSOrigin(origin) {
    w.Header().Set("Access-Control-Allow-Origin", origin)
} else if origin == "" {
    // Allow requests with no origin (e.g., same-origin or mobile apps)
    w.Header().Set("Access-Control-Allow-Origin", "*")  // ⚠️ PROBLEMATIC with credentials
}

w.Header().Set("Access-Control-Allow-Credentials", "true")
```

**Recommendation:** Never use `*` with `Allow-Credentials: true`. Only set the specific allowed origins. For mobile app requests (which may not send an Origin header), consider using a different authentication mechanism entirely rather than relying on cookies for those scenarios.

### 2. Session Token Format and UUID Usage
**Severity:** MEDIUM  
The access token is actually a session ID (UUID) stored in the database, which is good practice (Opaque Token Pattern). However, the `refresh_token` is returned as plain text to the client via localStorage:

```go
// auth_handler.go - HandleLogin
resp := AuthResponse{
    AccessToken:  session.ID.String(), // UUID-based opaque token ✅
    RefreshToken: refreshToken,        // Plain text token stored in localStorage ❌
}
```

The refresh token is then also stored in localStorage alongside the access token. While this is an improvement over storing plain JWTs, it still suffers from XSS risks if localStorage is compromised.

**Recommendation:** Consider making the refresh token HttpOnly as well, or implement a dual-cookie approach where both tokens are set via Set-Cookie with appropriate security flags (HttpOnly, Secure, SameSite).

### 3. Bulk Operations Lack Confirmation
**Severity:** MEDIUM  
The bulk admin operations (`bulk-approve`, `bulk-disable`, `bulk-delete`) do not require any confirmation or additional verification before executing:

```go
// admin_handler.go - handleBulkDeleteUsers
func (s *Server) handleBulkDeleteUsers(w http.ResponseWriter, r *http.Request) {
    // Revoke all sessions for these users before deleting
    for _, userID := range userIDs {
        if err := s.sessionRepo.RevokeAllByUserID(r.Context(), userID); err != nil { ... }
    }

    if err := s.userRepo.BulkDeleteUsers(r.Context(), userIDs); err != nil { ... }
}
```

**Recommendation:** Add a confirmation step requiring the admin to re-authenticate (e.g., require password re-entry) or add a time-based delay before bulk destructive operations. Log these actions in an audit trail.

### 4. Delete Account Flow is Incomplete
**Severity:** MEDIUM  
The user account deletion endpoint (`handleDeleteProfile`) only sets the user status to "disabled" and revokes sessions, but does NOT:
- Delete associated media files from storage
- Remove related session data
- Clear any cached data

```go
// auth_handler.go - handleDeleteProfile
func (s *Server) handleDeleteProfile(w http.ResponseWriter, r *http.Request) {
    // Revoke all sessions for this user first ✅
    if err := s.sessionRepo.RevokeAllByUserID(r.Context(), userID); err != nil { ... }

    // TODO: Delete all media belonging to this user (would need a method in MediaRepository)
    // For now, we just delete the user from the database
    
    // Since UserRepo doesn't have a Delete method, we'll update status to disabled as a soft-delete ❌
    user, err := s.userRepo.GetByID(r.Context(), userID)
    if err != nil { ... }

    user.Status = domain.UserStatusDisabled  // Not true deletion!
    if err := s.userRepo.Update(r.Context(), user); err != nil { ... }
}
```

**Recommendation:** Implement proper data deletion per GDPR/privacy requirements. Either: (1) Delete all associated media and session records, or (2) Anonymize the data by replacing PII with placeholder values while keeping the record for audit purposes. Document this behavior clearly in your privacy policy.

---

## Updated Summary of Critical Issues (Must Fix Before Release)

| # | Severity | Issue | Location | Status |
|---|----------|-------|----------|--------|
| 1 | CRITICAL | No HTTPS/TLS enforcement on Go server | `cmd/server/main.go` | ❌ Still exists |
| 2 | CRITICAL | Hardcoded API keys in frontend environment files | `angular-app/src/environments/` | ⚠️ Needs verification |

## Updated Summary of High-Severity Issues (Should Fix Before Release)

| # | Severity | Issue | Location | Status |
|---|----------|-------|----------|--------|
| 3 | HIGH | No rate limiting on authentication endpoints | `internal/api/server.go` | ❌ Still exists |
| 4 | HIGH | Insufficient file upload validation (MIME type, size) | `internal/api/upload_handler.go` | ⚠️ Partially addressed |
| 5 | HIGH | No row-level security in PostgreSQL database | All migrations | ❌ Still exists |
| 6 | HIGH | Database user privilege escalation risk | Application config | ❌ Still exists |
| 7 | HIGH | CSRF protection insufficient with CORS wildcard + credentials | `internal/api/server.go` | ⚠️ Partially addressed |
| 8 | HIGH | Tokens stored in localStorage (XSS vulnerability) | `angular-app/src/app/services/auth.service.ts` | ❌ Still exists |

## Updated Summary of Medium-Severity Issues (Should Address Before Release)

| # | Severity | Issue | Location | Status |
|---|----------|-------|----------|--------|
| 9 | MEDIUM | No Content Security Policy headers | Go server / Angular app | ❌ Still exists |
| 10 | MEDIUM | Detailed error messages exposed to clients | All handler files + `auth.service.ts` | ❌ Still exists |
| 11 | MEDIUM | Missing security headers in HTTP responses | Go server | ❌ Still exists |
| 12 | MEDIUM | No certificate pinning in Android app | `network_security_config.xml` | ❌ Still exists |
| 13 | MEDIUM | No audit logging for sensitive operations | Entire codebase | ❌ Still exists |
| 14 | MEDIUM | No encryption at rest for sensitive data | PostgreSQL schema | ❌ Still exists |
| 15 | MEDIUM | CORS misconfiguration with credentials + wildcard origin | `internal/api/server.go` | ⚠️ New finding |
| 16 | MEDIUM | Refresh token stored in localStorage alongside access token | `auth_handler.go`, `auth.service.ts` | ⚠️ New finding |
| 17 | MEDIUM | Bulk operations lack confirmation mechanism | `admin_handler.go` | ⚠️ New finding |
| 18 | MEDIUM | Delete account flow incomplete (soft delete only) | `auth_handler.go` | ⚠️ New finding |

---

## Recommended Priority Order for Fixes

### P0 - Immediate (Before Release — Critical)
1. **Enforce HTTPS/TLS** in production deployment or configure the Go server to use TLS directly
2. **Verify no hardcoded API keys exist** in Angular environment files; remove any found
3. **Implement rate limiting** on `/auth/login` and `/auth/register` endpoints (use `github.com/ulule/limiter`)

### P1 - Urgent (Within 1 Week — High)
4. **Add strict MIME type validation** for file uploads using `http.DetectContentType()` against an allowlist
5. **Enable Row-Level Security** on PostgreSQL tables as a defense-in-depth measure
6. **Create separate database users** with minimal required privileges
7. **Remove localStorage token storage** from Angular — rely solely on HttpOnly cookies already set by the backend

### P2 - Important (Within 2 Weeks — Medium)
8. **Add security headers** to all HTTP responses (CSP, HSTS, X-Content-Type-Options, etc.)
9. **Implement certificate pinning** in Android app using OkHttp's CertificatePinner
10. **Set up audit logging** for sensitive operations (user deletion, password changes, role modifications)

### P3 - Should Address Later
11. **Fix CORS configuration** — remove wildcard origin when credentials are enabled
12. **Improve delete account flow** to properly handle data deletion/anonymization per GDPR requirements  
13. **Add confirmation step** for bulk destructive operations in admin panel
14. **Consider making refresh token HttpOnly** as well

---

## Overall Security Posture Assessment

The SteadyPhoto project has made **significant security improvements since the original report**:
- ✅ SQL injection vulnerabilities have been completely eliminated
- ✅ XSS via `[innerHTML]` binding has been resolved  
- ✅ Authorization checks are now properly enforced through middleware
- ✅ Session management is improved with expiration, revocation, and HttpOnly cookies

However, the following areas remain **critical concerns** that must be addressed:
- ❌ No server-side HTTPS/TLS enforcement
- ❌ Rate limiting absent on authentication endpoints (brute-force risk)
- ❌ Database lacks row-level security and proper user separation
- ❌ File upload validation is insufficient for production use

The project is in a better state than initially assessed, but the remaining critical issues could still lead to serious security incidents if exploited. **Release should not proceed until all P0 items are addressed.**

---

*This updated report reflects the current state of the codebase as of December 2024. Issues marked as "resolved" have been verified against the latest source code. New findings from this review are marked accordingly.*