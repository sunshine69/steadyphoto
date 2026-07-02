# Select All Feature Implementation

## Overview
Implemented a "Select All" button that selects all photos on the current page, as specified in the requirements.

## Files Modified

### 1. `angular-app/src/app/components/photo-list/photo-list.component.ts`
**Changes:**
- Added subscription to `selectionService.selectAllTrigger$` in `ngOnInit()`
- When triggered, calls `selectionService.selectAll()` with all photo IDs from the current page

**Key Code:**
```typescript
// Subscribe to "Select All" trigger from the top header
this.selectAllTriggerSub = this.selectionService.selectAllTrigger$.subscribe(() => {
  if (this.photos && this.photos.length > 0) {
    this.selectionService.selectAll(this.photos.map(p => p.id));
  }
});
```

### 2. `angular-app/src/app/app.component.ts`
**Changes:**
- Added `SelectionService` import
- Added "Select All" button in the header actions (first button, before upload)
- Injected `SelectionService` in constructor
- Added `onSelectAll()` method that triggers the select all functionality

**HTML Changes:**
```html
<!-- Select All Button -->
<button class="icon-btn select-all-btn" title="Select All on Page" (click)="onSelectAll()">
  <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
    <rect x="3" y="3" width="7" height="7"/>
    <rect x="14" y="3" width="7" height="7"/>
    <rect x="3" y="14" width="7" height="7"/>
    <rect x="14" y="14" width="7" height="7"/>
  </svg>
</button>
```

**TypeScript Changes:**
```typescript
constructor(
  public presentationService: PresentationService,
  private searchService: SearchService,
  private selectionService: SelectionService
) {}

onSelectAll(): void {
  this.selectionService.triggerSelectAll();
}
```

### 3. `angular-app/src/app/services/selection.service.ts`
**Status:** No changes needed - already had the required infrastructure:
- `selectAllTrigger$` Subject for broadcasting select all events
- `triggerSelectAll()` method to emit the trigger
- `selectAll(ids)` method to set the selected IDs

## How It Works

1. **User clicks the "Select All" button** in the top header
2. **`AppComponent.onSelectAll()`** is called
3. **`SelectionService.triggerSelectAll()`** emits a value to `selectAllTrigger$`
4. **`PhotoListComponent`** receives the trigger and selects all photos on the current page
5. **Bulk Action Toolbar** appears showing the count of selected items

## Button Position
The button is positioned **first in the header actions**, before the Upload button:
- Icon: 2x2 grid pattern (representing multiple items)
- Tooltip: "Select All on Page"
- Style: Consistent with other header action buttons (40x40px, hover effects)

## Behavior
- **Selects only current page items**: The button selects all photos currently visible (up to 20 items per page)
- **Replaces current selection**: If items are already selected, it replaces them with all current page items
- **Works with search results**: If users are viewing search results, it selects all items in the search result set on that page
- **Responsive**: Button scales with other header actions

## Testing
To verify the implementation:
1. Navigate to the gallery page
2. Click the grid icon button (first button in top-right header)
3. All photos on the current page should be selected
4. The bulk action toolbar should appear showing the count
5. Navigate to a different page and click again - only that page's items should be selected
