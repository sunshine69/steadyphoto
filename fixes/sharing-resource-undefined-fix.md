# Sharing Fix: Resource Undefined Issue

## Problem
The sharing page displays "Resource undefined" for both albums and photos in the "My Shares" section.

## Root Cause
The `getResourceTitle` method in `sharing-dashboard.component.ts` accesses `share.resourceId`, but this field is either:
1. Not being returned by the backend API
2. Having a different case (snake_case vs camelCase)
3. Not being properly mapped from the API response

## Solution
Add a dedicated method to the ShareService that fetches resource titles and update the component to use it.

## Files Changed
- `angular-app/src/app/services/share.service.ts` - Added `getResourceTitle` method
- `angular-app/src/app/components/sharing-dashboard/sharing-dashboard.component.ts` - Updated to use new method

## Implementation
1. Added `getResourceTitle(share)` to ShareService that returns the resource type with ID
2. Updated component template to use the service method
3. Added fallback handling for cases where resource ID is missing

## Testing
- Check that "Album: Resource undefined" now shows actual album names
- Check that "Photo: Resource undefined" now shows actual photo filenames
- Verify the fix works for both public share links and user-to-user shares
