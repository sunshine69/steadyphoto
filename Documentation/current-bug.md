# Bug Fixes — Status Summary

**Last Updated:** June 22, 2026

---

## Fixed Bugs

### ✅ Admin Status Dropdown Not Showing (FIXED)
**Problem**: The admin role dropdown in the user management modal was not displaying unless the user logged out and back in. Suspected race condition and auth — the component didn't see the user was admin.

**Root Cause**: The `isAdmin` property in `UserManagementComponent` was only set once in `ngOnInit()` and never updated. The `AuthService.logout()` method did not properly clear auth state (missing `clearUser()` and `setAuthenticated(false)` calls).

**Fix**: 
1. Added `isAdmin$` observable to `AuthService`
2. Updated `UserManagementComponent` to subscribe to `isAdmin$` for dynamic updates
3. Fixed `AuthService.logout()` to call `clearUser()` and `setAuthenticated(false)` in the `finalize` block
4. Fixed `AuthService.clearUser()` to also call `setAdminStatus(false)`
5. Fixed `AuthService.login()` to call `setCurrentUser(currentUser)` which sets admin status based on role
6. Fixed TypeScript errors:
   - Added `username?: string` to `CurrentUser` interface
   - Removed manual `setCurrentUser({ email: this.email })` call from `login.component.ts`

**Result**: The admin dropdown now appears immediately after login without needing to refresh or relogin.

---

## Open / Known Issues

### ❌ `.upload-temp/` Directory Growth (KNOWN — PENDING FIX)
The `.upload-temp/` directory grows indefinitely with orphaned chunk files from expired/aborted uploads. See [UPLOAD_FIX_SUMMARY.md](UPLOAD_FIX_SUMMARY.md) for details.

---

## Resolved Bugs

### TypeScript Errors — `CurrentUser` Interface Missing `username` (FIXED)
**Error**: `Argument of type '{ email: string; }' is not assignable to parameter of type 'CurrentUser'. Type '{ email: string; }' is missing the following properties from type 'CurrentUser': id, role, status`

**Fix**: Removed the manual `setCurrentUser({ email: this.email })` call from `login.component.ts` since `login()` already sets the current user correctly with the full response from the API.

### TypeScript Errors — `CurrentUser` Missing `username` Property (FIXED)
**Error**: `Property 'username' does not exist on type 'CurrentUser'` in `auth.service.ts`

**Fix**: Added `username?: string` to the `CurrentUser` interface.

---

*For ongoing bug tracking, see individual feature README files for the latest status.*
