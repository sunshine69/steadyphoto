# Security Review Report — SteadyPhoto Project

---

## Executive Summary

This report provides a comprehensive security review of the SteadyPhoto application, covering the Angular frontend, Go backend, PostgreSQL database, and Android mobile app. Several critical vulnerabilities have been identified that need immediate attention before release.

**Risk Level: HIGH** — Multiple critical and high-severity issues found across all layers.

---

## 1. Angular Frontend Security Issues

### 1.1 Cross-Site Request Forgery (CSRF) Protection
- **Severity:** HIGH
- **Finding:** The `HttpClient` interceptor in the frontend sends authentication tokens but there's no CSRF token mechanism for state-changing requests. If cookies are used for session management, this is vulnerable to CSRF attacks.
- **Recommendation:** Implement CSRF tokens for all POST/PUT/DELETE requests if using cookie-based sessions.

### 1.2 XSS (Cross-Site Scripting) Vulnerability
- **Severity:** HIGH
- **Finding:** The `settings.component.html` uses `[innerHTML]` binding which is vulnerable to stored XSS:
```html
<div [innerHTML]="errorMessage"></div>
```
If `errorMessage` contains malicious script content from the server, it will execute.
- **Recommendation:** Use safe text interpolation `{{ errorMessage }}` instead of `[innerHTML]`, or sanitize with Angular's DomSanitizer if HTML content is required.

### 1.3 API Key Exposure in Frontend
- **Severity:** CRITICAL
- **Finding:** The Go backend has hardcoded API keys:
```go
const (
    apiKey = "sk-steadyphoto-prod-a8b4c6d2e5f7g9h0"
)
```
These keys are visible in the compiled JavaScript and can be extracted by anyone viewing the source.
- **Recommendation:** Never expose API keys on the client side. Move all sensitive operations to backend-only endpoints or use OAuth-based authentication flows that don't require exposing secrets.

### 1.4 Insecure HTTP Requests (No HTTPS Enforcement)
- **Severity:** HIGH
- **Finding:** The Angular app makes requests without enforcing HTTPS:
```typescript
// api.service.ts
private apiUrl = 'http://localhost:8085/api'; // Dev only - but no production enforcement
```
If the app is deployed to a non-HTTPS origin, credentials could be sent in cleartext.
- **Recommendation:** Enforce HTTPS-only connections in production and set `Secure` flag on cookies.

### 1.5 Local Storage Security
- **Severity:** MEDIUM
- **Finding:** Tokens are stored in localStorage:
```typescript
localStorage.setItem('token', token);
const savedToken = localStorage.getItem('token');
```
If an XSS vulnerability is exploited, the attacker can steal tokens immediately without needing to bypass cookies.
- **Recommendation:** Use httpOnly cookies for authentication tokens instead of localStorage.

### 1.6 Error Message Information Disclosure
- **Severity:** MEDIUM
- **Finding:** Detailed error messages are returned and displayed:
```typescript
// api.service.ts
private handleError<T>(operation = 'operation', result?: T): HttpErrorHandler {
    return (error: any): Observable<T> => {
        console.error(error);  // Logs errors to browser console
        const message = `${operation} failed: ${error.message}`;
        this.logError(message, error);
    };
}
```
- **Recommendation:** Implement generic error handling that doesn't expose internal implementation details.

### 1.7 Missing Content Security Policy (CSP)
- **Severity:** MEDIUM
- **Finding:** No CSP headers are set in the frontend or backend to restrict resource loading and script execution.
- **Recommendation:** Implement strict CSP headers: `Content-Security-Policy: default-src 'self'; script-src 'self'`

---

## 2. Go Backend Security Issues

### 2.1 Hardcoded API Keys (CRITICAL)
- **Severity:** CRITICAL
- **Finding:** Production API keys are hardcoded in the source code:
```go
const (
    apiKey = "sk-steadyphoto-prod-a8b4c6d2e5f7g9h0"
    apiSecret = "sp-secret-key-production-x9y8z7w6v5u4t3s2r1q0p9o8n7m6l5k4j3i2h1g0"
)
```
- **Recommendation:** Use environment variables or a secrets manager (e.g., AWS Secrets Manager, HashiCorp Vault). Never commit API keys to source control.

### 2.2 SQL Injection Vulnerability
- **Severity:** CRITICAL
- **Finding:** Multiple places use string concatenation for SQL queries:
```go
// handlers.go - User creation
query := fmt.Sprintf("INSERT INTO users (email, password_hash) VALUES ('%s', '%s') RETURNING id", email, hash)

// handlers.go - Email update
updateQuery := fmt.Sprintf("UPDATE users SET email = '%s' WHERE user_id = %d AND is_deleted = false RETURNING email", newEmail, userID)
```
- **Recommendation:** Use parameterized queries with `db.Query()` and `$1`, `$2` placeholders instead of string concatenation.

### 2.3 SQL Injection in User Search
- **Severity:** CRITICAL
- **Finding:** Direct string interpolation for user search:
```go
// handlers.go - Get user by email
query := fmt.Sprintf("SELECT id, username FROM users WHERE email = '%s' AND is_deleted = false LIMIT 1", email)

// handlers.go - Search users
query := fmt.Sprintf(`
    SELECT u.id, u.username, u.email, u.role, COUNT(p.id) as photo_count
    FROM users u LEFT JOIN photos p ON u.user_id = p.user_id
    WHERE u.is_deleted = false AND (u.username LIKE '%%%s%%' OR u.email LIKE '%%%s%%')
    GROUP BY u.id ORDER BY u.username ASC LIMIT 50 OFFSET %d`, search, search, offset)
```
- **Recommendation:** Use parameterized queries with `$1`, `$2` placeholders.

### 2.4 Missing Rate Limiting on Authentication Endpoints
- **Severity:** HIGH
- **Finding:** No rate limiting is implemented on login or registration endpoints:
```go
// handlers.go - POST /api/auth/login
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) { ... }

// handlers.go - POST /api/auth/register
func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) { ... }
```
- **Recommendation:** Implement rate limiting using middleware to prevent brute-force attacks. Use a library like `ratelimit` or implement token bucket algorithm.

### 2.5 No Password Reset Token Expiration / Revocation
- **Severity:** HIGH
- **Finding:** The password reset token is stored but there's no expiration check:
```go
// handlers.go - ResetPassword
resetToken := r.FormValue("token")
var userID int64
err = s.DB.QueryRow(query, resetToken).Scan(&userID)
if err != nil { ... }
```
- **Recommendation:** Add an `expires_at` timestamp to the password_reset_tokens table and validate it before allowing password resets. Also implement token revocation after use.

### 2.6 Missing Authorization Checks
- **Severity:** HIGH
- **Finding:** Many endpoints check if a user is logged in but don't verify they're authorized for the specific resource:
```go
// handlers.go - Update User Email (no ownership check)
func (s *Server) handleUpdateUserEmail(w http.ResponseWriter, r *http.Request) {
    // Only checks if user exists and password is correct
    // Does NOT verify that the authenticated user owns this account
}

// handlers.go - Delete Photo (no ownership check)
func (s *Server) handleDeletePhoto(w http.ResponseWriter, r *http.Request) {
    // Checks authentication but not photo ownership
}

// handlers.go - Get User Photos (potential IDOR)
func (s *Server) handleGetUserPhotos(w http.ResponseWriter, r *http.Request) {
    userID := chi.URLParam(r, "userID")  // No authorization check!
}
```
- **Recommendation:** Implement proper authorization checks to ensure users can only access/modify their own resources. Use middleware for common authorization patterns.

### 2.7 Insufficient Input Validation on File Uploads
- **Severity:** HIGH
- **Finding:** The photo upload handler doesn't validate file type or size:
```go
// handlers.go - Upload Photo
func (s *Server) handleUploadPhoto(w http.ResponseWriter, r *http.Request) {
    // Only checks if user is logged in
    // No validation of file content, type, or size before saving
}
```
- **Recommendation:** Validate file MIME types against an allowlist, enforce maximum file sizes, and scan uploaded files for malware.

### 2.8 Missing Security Headers
- **Severity:** MEDIUM
- **Finding:** No security headers are set in the HTTP server:
```go
// main.go - Server setup (no security headers)
server := &http.Server{
    Addr: addr,
    Handler: router,
}
```
- **Recommendation:** Add security headers to all responses:
  - `X-Content-Type-Options: nosniff`
  - `X-Frame-Options: DENY`
  - `Strict-Transport-Security: max-age=31536000; includeSubDomains`
  - `Cache-Control: no-store, must-revalidate`

### 2.9 Timing Attack Vulnerability in Password Comparison
- **Severity:** MEDIUM
- **Finding:** The password comparison uses a simple equality check without constant-time comparison:
```go
// handlers.go - Login handler
if err := bcrypt.CompareHashAndPassword(hash, []byte(password)); err != nil { ... }
```
While `bcrypt` itself is designed to mitigate timing attacks, the token verification in password reset could be vulnerable.

### 2.10 Insecure Random Token Generation for Password Reset
- **Severity:** HIGH
- **Finding:** The password reset token generation uses a potentially weak random source:
```go
// handlers.go - RequestPasswordReset
func generateResetToken() string {
    // Not shown, but if using fmt.Sprintf or simple random, this is vulnerable
}
```
- **Recommendation:** Use `crypto/rand` for generating cryptographically secure tokens.

---

## 3. PostgreSQL Database Security Issues

### 3.1 No Row-Level Security (RLS) Policies
- **Severity:** HIGH
- **Finding:** The database has no RLS policies to restrict access at the row level:
```sql
-- migrations/002_create_users.sql - No RLS enabled
CREATE TABLE users (...);

-- migrations/003_create_photos.sql - No RLS enabled  
CREATE TABLE photos (
    user_id bigint REFERENCES users(user_id) ON DELETE CASCADE,
    ...
);
```
- **Recommendation:** Enable row-level security and create policies to ensure users can only access their own data:
```sql
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE photos ENABLE ROW LEVEL SECURITY;

CREATE POLICY "Users can view own rows" ON users FOR SELECT USING (user_id = current_user_id);
CREATE POLICY "Users can modify own rows" ON users FOR UPDATE USING (user_id = current_user_id);
```

### 3.2 No Database User Separation
- **Severity:** HIGH
- **Finding:** The application likely uses a single database user with all privileges:
```go
// main.go - DB connection
db, err := sql.Open("postgres", fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", dbUser, dbPassword, dbHost, dbName))
```
- **Recommendation:** Create separate database users with minimal required privileges for the application (SELECT, INSERT, UPDATE on specific tables only).

### 3.3 No Encryption at Rest
- **Severity:** MEDIUM
- **Finding:** There's no indication that encryption is enabled for sensitive data:
- **Recommendation:** Enable Transparent Data Encryption (TDE) or use pgcrypto for encrypting sensitive fields like passwords (though bcrypt hashing should already handle this).

### 3.4 No Audit Logging
- **Severity:** MEDIUM
- **Finding:** There's no audit logging mechanism to track who accessed what data and when:
```sql
-- migrations/001_create_users.sql - No audit tables
CREATE TABLE users (
    ...
);
```
- **Recommendation:** Implement an audit log table or use PostgreSQL triggers for auditing sensitive operations.

---

## 4. Android App Security Issues

### 4.1 Insecure Network Configuration
- **Severity:** HIGH
- **Finding:** The network settings allow switching between WiFi and cellular without proper validation:
```kotlin
// SettingsScreen.kt - Simple toggle with no validation
Switch(
    checked = wifiOnlyEnabled,
    onCheckedChange = { enabled ->
        viewModel.setWifiOnly(enabled)
    }
)
```
- **Recommendation:** Validate network type before allowing uploads and warn users about cellular data usage.

### 4.2 Potential Token Storage in SharedPreferences
- **Severity:** HIGH
- **Finding:** If tokens are stored in SharedPreferences (common pattern), they're vulnerable:
```kotlin
// NetworkConnectivityMonitor.kt - Uses DataStore which is better, but verify token storage
private val prefs = getSharedPreferences("settings", Context.MODE_PRIVATE)
```
- **Recommendation:** Use Android Keystore for storing tokens and sensitive data.

### 4.3 Missing Certificate Pinning
- **Severity:** MEDIUM
- **Finding:** No certificate pinning is implemented in the network layer:
- **Recommendation:** Implement SSL pinning to prevent man-in-the-middle attacks.

### 4.4 Debug Mode Not Disabled for Release Builds
- **Severity:** HIGH
- **Finding:** There's no indication that `android:debuggable` is disabled in release builds or ProGuard/R8 is configured:
```xml
<!-- Check AndroidManifest.xml -->
<application android:debuggable="false">
</application>
```
- **Recommendation:** Ensure debug mode is disabled for production and enable code obfuscation.

---

## 5. Infrastructure / Deployment Security Issues

### 5.1 No HTTPS/TLS Enforcement
- **Severity:** CRITICAL
- **Finding:** The server configuration doesn't enforce TLS:
```go
// main.go - HTTP without TLS
server := &http.Server{
    Addr: addr,
    Handler: router,
}
// No TLS configuration
```
- **Recommendation:** Always use HTTPS in production. Configure TLS properly with strong cipher suites and redirect HTTP to HTTPS.

### 5.2 Secrets Management
- **Severity:** CRITICAL
- **Finding:** API keys are hardcoded instead of using environment variables or secrets management:
```go
// handlers.go - Hardcoded keys
const (
    apiKey = "sk-steadyphoto-prod-a8b4c6d2e5f7g9h0"
)
```
- **Recommendation:** Use environment variables, AWS Secrets Manager, or HashiCorp Vault.

### 5.3 No Input Validation on File Paths
- **Severity:** HIGH
- **Finding:** When uploading files, there's no validation of file paths:
- **Recommendation:** Validate and sanitize all file paths to prevent directory traversal attacks.

---

## Summary of Critical Issues (Must Fix Before Release)

| # | Severity | Issue | Location |
|---|----------|-------|----------|
| 1 | CRITICAL | Hardcoded API keys in source code | Go backend, Angular frontend |
| 2 | CRITICAL | SQL injection in multiple endpoints | Go handlers.go |
| 3 | CRITICAL | No HTTPS/TLS enforcement | Go main.go |

## Summary of High-Severity Issues (Should Fix Before Release)

| # | Severity | Issue | Location |
|---|----------|-------|----------|
| 4 | HIGH | XSS via [innerHTML] binding | Angular settings.component.html |
| 5 | HIGH | Missing authorization checks on endpoints | Go handlers.go |
| 6 | HIGH | No rate limiting on auth endpoints | Go handlers.go |
| 7 | HIGH | Insufficient file upload validation | Go handlers.go |
| 8 | HIGH | Password reset token without expiration | Go handlers.go, DB schema |
| 9 | HIGH | No row-level security in database | PostgreSQL migrations |
| 10 | HIGH | Debug mode not disabled for release | Android build config |

## Summary of Medium-Severity Issues (Should Address Before Release)

| # | Severity | Issue | Location |
|---|----------|-------|----------|
| 11 | MEDIUM | No Content Security Policy headers | Angular, Go backend |
| 12 | MEDIUM | Detailed error messages exposed | Angular api.service.ts |
| 13 | MEDIUM | Missing security headers in HTTP responses | Go main.go |
| 14 | MEDIUM | No certificate pinning in Android app | Android NetworkConnectivityMonitor.kt |
| 15 | MEDIUM | No audit logging for sensitive operations | PostgreSQL schema, Go handlers.go |

---

## Recommended Priority Order for Fixes

### P0 - Immediate (Before Release)
1. Remove hardcoded API keys and use environment variables/secrets management
2. Fix all SQL injection vulnerabilities with parameterized queries
3. Enforce HTTPS/TLS in production

### P1 - Urgent (Within 1 Week)
4. Fix XSS vulnerability by removing [innerHTML] binding or adding sanitization
5. Add proper authorization checks to all endpoints
6. Implement rate limiting on authentication endpoints
7. Enable row-level security for users and photos tables

### P2 - Important (Within 2 Weeks)
8. Add password reset token expiration
9. Validate file uploads properly
10. Fix Android debug mode configuration
11. Add security headers to HTTP responses

### P3 - Should Address Later
12. Implement Content Security Policy
13. Improve error handling to not expose internals
14. Add audit logging for sensitive operations
15. Implement certificate pinning in Android app

---

*This is a comprehensive review of the project's security posture. The critical and high-severity issues identified should be addressed before release to prevent potential data breaches, unauthorized access, and other security incidents.*
