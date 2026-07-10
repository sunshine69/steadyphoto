import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { TrashService } from '../../services/trash.service';
import { Photo, ListPhotosResponse } from '../../models/photo.model';

@Component({
  selector: 'app-trash',
  standalone: true,
  imports: [CommonModule, FormsModule],
  template: `
    <div class="trash-container">
      <!-- Header -->
      <div class="header-section" *ngIf="!isLoading && photos.length > 0; else trashEmptyState">
        <h1>Trash</h1>
        <p class="subtitle">{{ totalItems }} item{{ totalItems !== 1 ? 's' : '' }} in trash</p>
      </div>

      <!-- Loading State -->
      <div *ngIf="isLoading" class="loading-container">
        <div class="spinner"></div>
        <p>Loading trashed items...</p>
      </div>

      <!-- Empty Trash State -->
      <ng-template #trashEmptyState>
        <div class="empty-state">
          <svg xmlns="http://www.w3.org/2000/svg" width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="#9ca3af" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
            <polyline points="3 6 5 6 21 6"></polyline>
            <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>
            <line x1="10" y1="11" x2="10" y2="17"></line>
            <line x1="14" y1="11" x2="14" y2="17"></line>
          </svg>
          <h2>{{ isLoading ? 'Loading...' : (totalItems === 0 && !isLoading ? 'Trash is empty' : '') }}</h2>
          <p class="empty-text">{{ totalItems === 0 && !isLoading ? 'Deleted items will appear here.' : '' }}</p>
        </div>
      </ng-template>

      <!-- Trash Items List -->
      <div *ngIf="!isLoading" class="trash-list">
        <div 
          *ngFor="let photo of photos; let i = index" 
          class="trash-item"
          [class.expanded]="expandedItem === photo.id"
        >
          <!-- Item Row -->
          <div class="item-row" (click)="toggleExpand(photo)">
            <!-- Thumbnail -->
            <div class="thumbnail-wrapper">
              <img *ngIf="photo.thumbnailUrl; else noThumb" [src]="photo.thumbnailUrl" alt="{{ photo.filename }}" />
              <ng-template #noThumb>
                <div class="placeholder-thumb">
                  {{ getInitials(photo) }}
                </div>
              </ng-template>
            </div>

            <!-- Info -->
            <div class="item-info">
              <h3 class="filename">{{ photo.filename }}</h3>
              <p class="meta-text" *ngIf="photo.captured_at || photo.size">
                {{ getFormattedDate(photo) }} · {{ formatSize(photo.size) }}
              </p>
            </div>

            <!-- Actions (visible on hover/expanded or always visible for mobile) -->
            <div class="item-actions" [class.show]="isExpandedOrMobile()">
              
              <!-- Restore Button -->
              <button 
                *ngIf="!showDeleteConfirm[photo.id]"
                class="btn btn-restore"
                (click)="handleRestore(photo, $event)"
                title="Restore to library">
                <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <polyline points="1 4 1 10 7 10"></polyline>
                  <path d="M3.51 15a9 9 0 1 0 2.13-9.36L1 10"></path>
                </svg>
                Restore
              </button>

              <!-- Delete Forever Button -->
              <button 
                *ngIf="!showDeleteConfirm[photo.id]"
                class="btn btn-delete"
                (click)="handlePermanentDelete(photo, $event)"
                title="Permanently delete">
                <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <polyline points="3 6 5 6 21 6"></polyline>
                  <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>
                </svg>
                Delete Forever
              </button>

            </div>
          </div>

          <!-- Confirmation Dialog (Delete Forever) -->
          <div *ngIf="showDeleteConfirm[photo.id]" class="confirm-delete">
            <p><strong>Delete "{{ photo.filename }}" forever?</strong></p>
            <p>This action cannot be undone.</p>
            <div class="confirm-actions">
              <button 
                class="btn btn-cancel"
                (click)="cancelDelete(photo)">Cancel</button>
              <button 
                class="btn btn-confirm-delete"
                [disabled]="deletingId === photo.id"
                (click)="confirmPermanentDelete(photo, $event)">Yes, Delete Forever</button>
            </div>
          </div>

        </div>
      </div>

      <!-- Pagination -->
      <div *ngIf="!isLoading && totalItems > photos.length" class="pagination">
        <button 
          [disabled]="offset <= 0 || loadingMore" 
          (click)="loadPrevious()"
          class="btn btn-page">← Previous</button>
        
        <span class="page-info">{{ offset + 1 }} - {{ Math.min(offset + photos.length, totalItems) }} of {{ totalItems }}</span>

        <button 
          [disabled]="offset + photos.length >= totalItems || loadingMore" 
          (click)="loadNext()"
          class="btn btn-page">Next →</button>
      </div>
    </div>
  `,
  styles: [`
    .trash-container {
      padding: 24px;
      max-width: 100%;
    }

    /* Header */
    .header-section h1 {
      font-size: 32px;
      color: #e5e7eb;
      margin-bottom: 8px;
    }

    .subtitle {
      color: #9ca3af;
      font-size: 14px;
    }

    /* Loading */
    .loading-container, .empty-state {
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      padding: 80px 20px;
      text-align: center;
    }

    .spinner {
      width: 40px;
      height: 40px;
      border: 3px solid #e5e7eb;
      border-top-color: #6b7280;
      border-radius: 50%;
      animation: spin 1s linear infinite;
      margin-bottom: 16px;
    }

    @keyframes spin {
      to { transform: rotate(360deg); }
    }

    .empty-state svg {
      margin-bottom: 24px;
    }

    .empty-state h2 {
      font-size: 18px;
      color: #e5e7eb;
      margin-bottom: 8px;
    }

    .empty-text {
      color: #9ca3af;
      font-size: 14px;
    }

    /* Trash List */
    .trash-list {
      display: flex;
      flex-direction: column;
      gap: 8px;
      margin-top: 20px;
    }

    .trash-item {
      border-radius: 12px;
      overflow: hidden;
      transition: all 0.2s ease;
    }

    /* Item Row */
    .item-row {
      display: flex;
      align-items: center;
      padding: 16px;
      gap: 16px;
      cursor: pointer;
      background-color: #374151;
      border-radius: 12px;
      transition: background-color 0.2s ease;
    }

    .item-row:hover {
      background-color: #4b5563;
    }

    /* Thumbnail */
    .thumbnail-wrapper {
      width: 80px;
      height: 80px;
      border-radius: 8px;
      overflow: hidden;
      flex-shrink: 0;
      background-color: #1f2937;
    }

    .thumbnail-wrapper img {
      width: 100%;
      height: 100%;
      object-fit: cover;
    }

    .placeholder-thumb {
      display: flex;
      align-items: center;
      justify-content: center;
      font-size: 24px;
      color: #6b7280;
      background-color: #1f2937;
    }

    /* Info */
    .item-info {
      flex: 1;
      min-width: 0;
    }

    .filename {
      font-size: 15px;
      color: #e5e7eb;
      margin-bottom: 4px;
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
    }

    .meta-text {
      font-size: 13px;
      color: #9ca3af;
    }

    /* Actions */
    .item-actions {
      display: flex;
      gap: 8px;
      opacity: 0.4;
      transition: opacity 0.2s ease;
    }

    .trash-item:hover .item-actions,
    .item-row.show .item-actions {
      opacity: 1;
    }

    /* Buttons */
    .btn {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      padding: 8px 14px;
      border-radius: 8px;
      font-size: 13px;
      font-weight: 500;
      cursor: pointer;
      transition: all 0.2s ease;
    }

    .btn-restore {
      background-color: #6b7280;
      color: white;
      border: none;
    }

    .btn-restore:hover:not(:disabled) {
      background-color: #9ca3af;
    }

    .btn-delete {
      background-color: transparent;
      color: #ef4444;
      border: 1px solid #7f1d1d;
    }

    .btn-delete:hover:not(:disabled) {
      background-color: rgba(239, 68, 68, 0.1);
    }

    /* Confirm Delete */
    .confirm-delete {
      padding: 16px;
      background-color: #451a1a;
      border-top: 1px solid rgba(239, 68, 68, 0.2);
    }

    .confirm-delete p:first-child {
      color: #fca5a5;
      margin-bottom: 4px;
    }

    .confirm-delete p:last-of-type {
      font-size: 13px;
      color: #9ca3af;
      margin-bottom: 12px;
    }

    .confirm-actions {
      display: flex;
      gap: 8px;
    }

    .btn-cancel, .btn-confirm-delete {
      padding: 6px 14px;
      border-radius: 6px;
      font-size: 13px;
      cursor: pointer;
      transition: all 0.2s ease;
    }

    .btn-cancel {
      background-color: #374151;
      color: #e5e7eb;
      border: none;
    }

    .btn-cancel:hover:not(:disabled) {
      background-color: #4b5563;
    }

    .btn-confirm-delete {
      background-color: #dc2626;
      color: white;
      border: none;
    }

    .btn-confirm-delete:hover:not(:disabled) {
      background-color: #ef4444;
    }

    /* Pagination */
    .pagination {
      display: flex;
      align-items: center;
      justify-content: space-between;
      padding: 20px 8px;
      margin-top: 16px;
    }

    .btn-page {
      background-color: #374151;
      color: #e5e7eb;
      border: none;
      padding: 8px 16px;
      border-radius: 8px;
      font-size: 14px;
    }

    .btn-page:hover:not(:disabled) {
      background-color: #4b5563;
    }

    .page-info {
      color: #9ca3af;
      font-size: 14px;
    }

    /* Disabled state */
    :host ::ng-deep button:disabled,
    [disabled] {
      opacity: 0.5;
      cursor: not-allowed !important;
    }

    @media (max-width: 768px) {
      .item-actions.show {
        flex-direction: column;
        width: 100%;
        margin-top: 12px;
      }

      .btn {
        justify-content: center;
      }
    }
  `]
})
export class TrashComponent implements OnInit {
  photos: Photo[] = [];
  totalItems = 0;
  offset = 0;
  limit = 20;
  isLoading = false;
  loadingMore = false;

  expandedItem: string | null = null;
  showDeleteConfirm: Record<string, boolean> = {};
  deletingId: string | null = null;

  Math = Math; // Expose for template use

  constructor(private trashService: TrashService) {}

  ngOnInit(): void {
    this.loadTrash();
  }

  loadTrash(pageOffset?: number): void {
    const offsetToUse = pageOffset ?? this.offset;
    
    if (pageOffset !== undefined && pageOffset === this.offset + this.limit) {
      // Loading more items, don't clear existing ones
      this.loadingMore = true;
    } else {
      this.isLoading = true;
    }

    const offsetParam = pageOffset ?? 0;
    
    this.trashService.listTrash(this.limit, offsetParam).subscribe({
      next: (response) => {
        if (pageOffset !== undefined && pageOffset === this.offset + this.limit) {
          // Append to existing list when loading more pages
          const newPhotos = response.photos.filter(
            p1 => !this.photos.some(p2 => p1.id === p2.id)
          );
          this.photos.push(...newPhotos);
        } else {
          this.photos = response.photos;
        }

        this.totalItems = response.total;
        
        if (pageOffset !== undefined && pageOffset === this.offset + this.limit) {
          this.loadingMore = false;
        } else {
          this.isLoading = false;
        }
      },
      error: (error) => {
        console.error('Failed to load trash:', error);
        if (pageOffset !== undefined && pageOffset === this.offset + this.limit) {
          this.loadingMore = false;
        } else {
          this.isLoading = false;
        }
      }
    });

    // Update offset for next navigation
    if (!this.photos.length || !pageOffset) {
      this.offset = 0;
    }
  }

  loadNext(): void {
    const newOffset = Math.min(this.offset + this.limit, this.totalItems - this.limit);
    if (newOffset > this.offset && newOffset >= 0) {
      this.loadTrash(newOffset);
      window.scrollTo({ top: document.querySelector('.trash-list')?.getBoundingClientRect().top || 0 });
    }
  }

  loadPrevious(): void {
    const newOffset = Math.max(this.offset - this.limit, 0);
    if (newOffset < this.offset) {
      this.loadTrash(newOffset);
      window.scrollTo({ top: document.querySelector('.trash-list')?.getBoundingClientRect().top || 0 });
    }
  }

  toggleExpand(photo: Photo): void {
    // Don't expand when clicking buttons directly - let button handlers manage state
    if (this.expandedItem === photo.id) {
      this.expandedItem = null;
    } else {
      this.expandedItem = photo.id;
    }
  }

  isExpandedOrMobile(): boolean {
    return window.innerWidth <= 768 || !!this.expandedItem;
  }

  handleRestore(photo: Photo, event?: Event): void {
    if (event) {
      event.stopPropagation(); // Prevent row click toggle
    }

    this.trashService.restoreMedia(photo.id).subscribe({
      next: () => {
        
        this.photos = this.photos.filter(p => p.id !== photo.id);
        this.totalItems--;
        
        if (this.showDeleteConfirm[photo.id]) {
          delete this.showDeleteConfirm[photo.id];
        }

        // Show success feedback - could use a toast in future
      },
      error: (error) => {
        console.error('Failed to restore:', photo.filename, error);
        alert(`Failed to restore "${photo.filename}". Please try again.`);
      }
    });
  }

  handlePermanentDelete(photo: Photo, event?: Event): void {
    if (event) {
      event.stopPropagation(); // Prevent row click toggle
    }
    
    this.showDeleteConfirm[photo.id] = true;
    this.expandedItem = photo.id;
  }

  cancelDelete(photo: Photo): void {
    delete this.showDeleteConfirm[photo.id];
    if (this.expandedItem === photo.id) {
      // Keep expanded for the confirmation dialog to remain visible
    } else {
      this.expandedItem = null;
    }
  }

  confirmPermanentDelete(photo: Photo, event?: Event): void {
    if (event) {
      event.stopPropagation();
    }

    this.deletingId = photo.id;

    this.trashService.permanentlyDelete(photo.id).subscribe({
      next: () => {
        
        
        // Remove from list after successful deletion
        setTimeout(() => {
          if (this.showDeleteConfirm[photo.id]) {
            delete this.showDeleteConfirm[photo.id];
          }

          const index = this.photos.findIndex(p => p.id === photo.id);
          if (index > -1) {
            // Fade out effect could be added here
            this.photos.splice(index, 1);
          } else {
            // If not in current page list but still need to decrement total
          }

          this.totalItems = Math.max(0, this.totalItems - 1);
          
          if (this.expandedItem === photo.id) {
            this.expandedItem = null;
          }
        }, 200); // Small delay for visual feedback
        
      },
      error: (error) => {
        console.error('Failed to permanently delete:', photo.filename, error);
        
        if (!this.photos.some(p => p.id === photo.id)) {
          this.totalItems = Math.max(0, this.totalItems - 1);
        }

        alert(`Failed to permanently delete "${photo.filename}". Please try again.`);
      },
      complete: () => {
        this.deletingId = null;
      }
    });
  }

  getInitials(photo: Photo): string {
    const name = photo.filename || '';
    return name.substring(0, 2).toUpperCase();
  }

  getFormattedDate(photo: Photo): string {
    if (!photo.captured_at) return '';
    
    try {
      const date = new Date(photo.captured_at);
      
      // If the captured_at is a valid ISO timestamp (e.g., "2024-12-15T10:30:00Z")
      if (!isNaN(date.getTime())) {
        return date.toLocaleDateString('en-US', { 
          year: 'numeric', 
          month: 'short', 
          day: 'numeric' 
        });
      }

      // If it's a numeric timestamp (e.g., "1702645800") or other format, try parsing as number first
      const ts = parseInt(photo.captured_at, 10);
      if (!isNaN(ts)) {
        return new Date(ts).toLocaleDateString('en-US', { 
          year: 'numeric', 
          month: 'short', 
          day: 'numeric' 
        });
      }

    } catch (e) {}

    // Fallback to raw string or empty
    const dateStr = String(photo.captured_at);
    
    if (/^\d{4}-\d{2}-\d{2}/.test(dateStr)) {
      return new Date(dateStr).toLocaleDateString('en-US', { 
        year: 'numeric', 
        month: 'short', 
        day: 'numeric' 
      });
    }

    if (/^\d+$/.test(dateStr) && dateStr.length === 10) {
      // Unix timestamp (seconds since epoch, e.g., "1734259800")
      return new Date(parseInt(dateStr)).toLocaleDateString('en-US', { 
        year: 'numeric', 
        month: 'short', 
        day: 'numeric' 
      });
    }

    if (/^\d+$/.test(dateStr) && dateStr.length === 13) {
      // Millisecond timestamp (e.g., "1702645800000")
      return new Date(parseInt(dateStr)).toLocaleDateString('en-US', { 
        year: 'numeric', 
        month: 'short', 
        day: 'numeric' 
      });
    }

    // Last resort - try to parse as date string directly and format it nicely if possible, otherwise show raw
    const d = new Date(dateStr);
    return isNaN(d.getTime()) ? '' : d.toLocaleDateString('en-US', { year: 'numeric', month: 'short', day: 'numeric' });
  }

  formatSize(bytes?: number): string {
    if (!bytes) return '';
    
    const units = ['B', 'KB', 'MB', 'GB'];
    let size = bytes;
    let unitIndex = 0;
    
    while (size >= 1024 && unitIndex < units.length - 1) {
      size /= 1024;
      unitIndex++;
    }

    return `${size.toFixed(unitIndex > 0 ? 1 : 0)} ${units[unitIndex]}`;
  }
}
