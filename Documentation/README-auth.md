## Authentication & Multi-User Architecture

**Status:** ✅ Complete — All features working including admin status tracking  
**Last Updated:** June 22, 2026

---

### Overview
To transition from a single-user system to a multi-tenant application, we implemented an **Opaque Token + Refresh Pattern**. This provides high security against XSS/CSRF while allowing stateless API interaction and granular session control.

**Core Security Model:** 
- Tokens are stored in `HttpOnly`, `Secure` cookies (inaccessible to JavaScript).
- Short-lived Access Token (~15m) for requests via `/api`.
- Long-lived Refresh Token (~7d) used only at `/api/auth/refresh` to renew access.

**System Administration & Lifecycle:**
- **Bootstrap Admin**: A default super admin account (`admin@steadyphoto.com` / `password`) is generated during database initialization for immediate system management and testing purposes.
- **Account Management**: Users can update their email addresses and change passwords via authenticated API endpoints.

---

### 1. Design Principles
- **Data Isolation**: Every media asset must be linked to a `user_id`. All database queries include this filter.
- **Physical Storage Partitioning**: Each user is allocated their own top-level directory under the storage root for complete file isolation at the OS level.
  - Structure: `storage/{user_id}/YYYY/MM/DD/...`
- **Zero-Knowledge Frontend**: The Angular app never touches the actual tokens; they are stored in browser-managed, encrypted cookies.
- **Stateless API with Revocability**: Using opaque tokens allows us to instantly invalidate sessions via the database/cache.

### 2. Token Lifecycle Management
We use two distinct tokens to balance security and user experience:

| Feature | Access Token | Refresh Token |
| :--- | :--- | :--- |
| **Format** | Random Opaque String | Long Random String |
| **Lifespan** | ~15 Minutes | ~7 Days |
| **Storage Method** | `HttpOnly` Cookie (`Path=/api`) | `HttpOnly` Cookie (`Path=/api/auth/refresh`) |
| **Security Goal** | Short window of risk if compromised. | Used only to get new access tokens; highly restricted path. |

---

### 3. API Endpoints
- `POST /api/v1/auth/register`: Create user account.
- `POST /api/v1/auth/login`: Validate credentials and issue the Access + Refresh token cookie pair.
- `POST /api/v1/auth/refresh`: Receives the refresh cookie; issues a new access token if valid.
- `POST /api/v1/auth/logout`: Clears cookies and revokes the session in the DB.
- `GET /api/v1/auth/profile`: Get current user profile.
- `PATCH /api/v1/auth/profile/email`: Update email address.
- `PATCH /api/v1/auth/profile/password`: Change password.
- `GET /api/v1/auth/users/search?query=`: Search users by email/username for sharing purposes.

---

### 4. Frontend Implementation (Angular)

#### Auth Interceptor
Automatically detects `401 Unauthorized` responses, triggers the `/refresh` call silently, then retries the original failed request to ensure zero user interruption.

#### Route Guards (`CanActivate`)
Prevents unauthenticated access to media dashboard and settings routes.

#### Credential Handling
All HTTP calls are configured with `{ withCredentials: true }` to allow cookie transmission.

---

### 5. Auth Service (`auth.service.ts`) — Full State Management

The `AuthService` provides complete authentication state management:

#### Observable State Streams
- **`isAuthenticated$`** — Observable for auth state changes (true/false)
- **`isAdmin$`** — Observable for admin status (true/false) — dynamically updates on login/logout
- **`currentUser$`** — Observable for the current user object (`CurrentUser | null`)
- **`isLoggingOut$`** — Observable for logout-in-progress state

#### Key Methods
| Method | Description |
|--------|-------------|
| `login(email, password)` | Authenticates user and sets all state: `currentUser`, `isAdmin`, `isAuthenticated` |
| `logout()` | Clears cookies, clears local storage, resets all state via `finalize` |
| `initializeAuth()` | Checks session on page load via `/auth/profile`; sets auth state accordingly |
| `refreshToken()` | Refreshes the access token via `/auth/refresh` |
| `handle401()` | Handles 401 responses — deduplicates concurrent refresh requests |
| `setCurrentUser(user)` | Sets the current user and admin status in the BehaviorSubject |
| `setAuthenticated(status)` | Updates the auth state BehaviorSubject |
| `setAdminStatus(isAdmin)` | Updates the admin status BehaviorSubject based on user role |
| `clearUser()` | Clears ALL local auth state: `currentUser`, admin status, all localStorage/sessionStorage items |
| `getProfile()` | Fetches user profile from backend |
| `updateEmail(newEmail)` | Updates user's email |
| `changePassword(current, new)` | Changes user's password |

#### Auth State Persistence
- **`access_token`** — Stored in `sessionStorage` (cleared on tab close) for Bearer token usage
- **`email`** — Stored in `localStorage` for display purposes
- **`username`** — Stored in `localStorage` for display purposes
- **`currentUser`** — Stored in `localStorage` as JSON (parsed into `CurrentUser` object)
- **`refresh_token`** — NOT stored on the client at all — managed by server-side session rotation and HttpOnly cookies only

#### Security Fixes Applied
- ⚠️ **Security Fix**: access_token moved to `sessionStorage` instead of `localStorage` — token is cleared when the browser tab is closed, reducing XSS risk
- ⚠️ **Security Fix**: refresh_token is NOT stored on the client — managed by server-side HttpOnly cookies only

---

### 6. Security Hardening
- **XSS Mitigation**: Using `HttpOnly` cookies ensures JavaScript cannot read or steal tokens.
- **CSRF Mitigation**: Use `SameSite=Strict` flags on all authentication cookies and implement Double Submit Cookie pattern for sensitive mutations.
- **Rate Limiting**: Applied to `/login`, `/register`, and `/refresh` endpoints.

---

### Current Status Summary

| Feature | Status | Notes |
|---------|--------|-------|
| **User Registration** | ✅ Complete | Creates pending user account (requires admin approval) |
| **Password Hashing** | ✅ Secure | Argon2id via `golang.org/x/crypto/argon2` |
| **Session Management**| ✅ Functional | Token rotation and revocation verified |
| **Profile Updates** | ✅ Functional | Email & Password updates via `/auth/profile` |
| **Soft-Delete (IAM)** | ✅ Complete | SQL query confirms `status = 'disabled'` |
| **Admin Status Tracking** | ✅ Functional | `isAdmin$` observable updates dynamically on login/logout |
| **Logout State Clearing**| ✅ Functional | Clears all localStorage, sessionStorage, and BehaviorSubject state |
| **401 Retry Interceptor**| ✅ Functional | Deduplicates concurrent refresh requests |
| **User Search** | ✅ Functional | Search users by email/username for sharing |
| **Auth Interceptor** | ✅ Functional | Automatic 401 handling with token refresh |
