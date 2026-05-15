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
