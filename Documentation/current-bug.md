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

### ✅ Selection Toolbar Layout — Overlapping Search Input (FIXED)
**Problem**: The "Selection Actions" dropdown appeared as a floating element that overlapped the search input field, making the search input unusable when items were selected.

**Root Cause**: The toolbar was rendered as a floating/sticky overlay positioned above the search input.

**Fix**: Moved the Selection Actions dropdown to the left side of the "Select All" button and changed it from a floating element to an inline element within the toolbar, so the search input remains usable.

---

### ✅ SelectionService `selectAll()` Replaces Instead of Accumulates (FIXED)
**Problem**: When items were already selected and the user searched for new photos and clicked "Select All", the previous selection was cleared and only the current view's items were selected.

**Root Cause**: `SelectionService.selectAll()` created a new empty `Set` and only added the IDs passed to it, replacing any previous selection.

**Fix**: Changed `selectAll()` to initialize the new set from the existing selection (`new Set(this._selectedIds$.value)`) before adding the new IDs on top. Now `selectAll` **accumulates** selections — previously selected items are retained. Clearing is only done via the explicit "Clear" (×) button which calls `clear()`.

---

### ✅ PhotoListComponent Not Syncing with SelectionService (FIXED)
**Problem**: `PhotoListComponent` maintained its own local selection state and only subscribed to `selectionService.selectAllTrigger$`, but not to `selectionService.selectedIds$`. This caused the component's UI to fall out of sync when selections were made through other components (e.g., album media selection, shared media selection).

**Root Cause**: The component directly accessed the private `_selectedIds$` in tests but relied on a local `Set<string>` that was never updated from the service's `BehaviorSubject`.

**Fix**: Refactored `PhotoListComponent` to subscribe exclusively to `selectionService.selectedIds$` for all state synchronization, removing the local set and any direct access to the service's private state. The component now reflects the service's selection state reactively.

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
