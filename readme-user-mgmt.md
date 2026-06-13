# User Management Feature

**Status:** ✅ Backend API Completed — Frontend UI Status Unknown  
**Last Updated:** June 13, 2026

---

## Overview

Admin users can manage other user accounts through a modal interface accessible from the top-right avatar icon. This feature provides full lifecycle management of user accounts including status changes, role assignments, and account deletion.

---

## Data Model

### `users` Table
| Column | Type | Description |
| :--- | :--- | :--- |
| `id` | UUID | Primary Key |
| `email` | VARCHAR | User email address (unique) — enforced at DB level via UNIQUE constraint |
| `password_hash` | TEXT | Securely hashed password (Argon2id) |
| `status` | VARCHAR | Account status: pending, active, disabled, rejected |
| `role` | VARCHAR | User role: admin, user |
| `created_at` | TIMESTAMP | Account creation time |
| `updated_at` | TIMESTAMP | Last update time |

---

## API Specifications (Admin Endpoints) *(Updated paths from `/api/` to `/api/v1/`)*

| Method | Path | Description | Auth Required | Admin Only? |
|--------|------|-------------|---------------|-------------|
| `GET` | `/api/v1/admin/users` | List all users with optional status filter (`?status=active`) | ✅ Admin JWT | Yes |
| `GET` | `/api/v1/admin/users/{id}` | Get specific user details | ✅ Admin JWT | Yes |
| `PATCH` | `/api/v1/admin/users/{id}` | Update user status or role (validated against allowed values) | ✅ Admin JWT | Yes |
| `DELETE` | `/api/v1/admin/users/{id}` | Soft-delete: set status to disabled, revoke all sessions | ✅ Admin JWT | Yes |
| `POST` | `/api/v1/admin/users/bulk-approve` | Bulk approve pending users (set status = active) | ✅ Admin JWT | Yes |
| `POST` | `/api/v1/admin/users/bulk-disable` | Bulk disable users (set status to disabled, revoke sessions) | ✅ Admin JWT | Yes |
| `DELETE` | `/api/v1/admin/users/bulk-delete` | Bulk delete: revoke all sessions for selected users before deletion | ✅ Admin JWT | Yes |

---

## User Profile Endpoints (Authenticated Users)

| Method | Path | Description | Auth Required? |
|--------|------|-------------|----------------|
| `GET` | `/api/v1/auth/profile` | Get current user profile | Yes — JWT/Session cookie or Bearer token |
| `PATCH` | `/api/v1/auth/profile/email` | Update email address | Yes |
| `PATCH` | `/api/v1/auth/profile/password` | Change password (requires current + new password) | Yes |
| `DELETE` | `/api/v1/auth/profile` | Delete user account — revokes all sessions, sets status to disabled | Yes |

---

## Frontend Implementation (Angular) ✅ Updated — Security Fix Applied

The Angular auth service (`auth.service.ts`) confirms the following features exist:

- ✅ **Login** with email/password → returns `access_token`, `refresh_token`, `user_id`, `role`, `status`
- ✅ **Registration** creates pending user account (requires admin approval)
- ✅ **Logout** clears local storage and calls backend logout endpoint
- ✅ **Token refresh** via `/auth/refresh` using HttpOnly cookie

### Security Fixes Applied
- ⚠️ **Security Fix**: access_token is now stored in `sessionStorage` instead of `localStorage`. This means the token is cleared when the browser tab is closed, reducing XSS risk. The HttpOnly cookie remains the primary authentication mechanism for normal browser requests (sent automatically via `withCredentials`).
- ⚠️ **Security Fix**: refresh_token is NOT stored on the client at all — it's managed by server-side session rotation and HttpOnly cookies only.

---

## Related Documents

- [Architecture Overview](readme-arch.md) — High-level system architecture, security summary
