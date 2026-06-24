import { Component, inject, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router, RouterModule } from '@angular/router';
import { ShareService, SharedMediaItem, SharedAlbumItem } from '../../services/share.service';
import { Photo } from '../../models/photo.model';

@Component({
  selector: 'app-shared-with-me',
  standalone: true,
  imports: [CommonModule, RouterModule],
  template: `
    <div class="shared-with-me-container">
      <!-- Header -->
      <div class="header-section">
        <h1>Shared with me</h1>
        
        <!-- Tabs for media/albums -->
        <div class="tabs">
          <button [class.active]="activeTab === 'media'" (click)="activeTab = 'media'">Photos & Videos</button>
          <button [class.active]="activeTab === 'albums'" (click)="activeTab = 'albums'">Albums</button>
        </div>
      </div>

      <!-- Loading state -->
      <div class="loading-state" *ngIf="isLoading && contentItems.length === 0">
        <span class="spinner"></span>
        <p>Loading...</p>
      </div>

      <!-- Empty state -->
      <div class="empty-state" *ngIf="!isLoading && contentItems.length === 0">
        <div class="empty-icon">👥</div>
        <h3>No shared items yet</h3>
        <p>When someone shares photos, videos, or albums with you, they'll appear here.</p>
      </div>

      <!-- Content grid -->
      <ng-container *ngIf="!isLoading && contentItems.length > 0">
        <!-- Media/Photos view -->
        <div class="content-grid" *ngIf="activeTab === 'media'">
          <div *ngFor="let item of sharedMedia; let i = index" class="shared-media-card">
            <!-- Thumbnail with sharer badge -->
            <div class="thumbnail-wrapper">
              <img 
                [src]="item.thumbnailUrl || getFallbackThumbnail()" 
                [alt]="item.filename"
                (error)="onImageError($event)"
                loading="lazy"
              >
              <span class="sharer-badge">{{ item.sharerName }}</span>
            </div>
            
            <!-- Info -->
            <div class="media-info">
              <p class="filename">{{ item.filename }}</p>
              <p class="shared-date">{{ item.sharedAt | date:'shortDate' }}</p>
            </div>

            <!-- Actions -->
            <div class="media-actions">
              <button class="action-btn" (click)="viewItem(item)">👁️ View</button>
              <button class="action-btn" *ngIf="!item.isFavorite" (click)="toggleFavorite(item)">❤️</button>
              <button class="action-btn" *ngIf="item.isFavorite" (click)="toggleFavorite(item)" style="color: #ef4444;">♥️</button>
            </div>
          </div>

          <!-- Load more button -->
          <div class="load-more-container" *ngIf="hasMoreMedia">
            <button class="btn-load-more" (click)="loadMoreSharedMedia()">Load More Photos & Videos</button>
          </div>
        </div>

        <!-- Albums view -->
        <div class="content-grid" *ngIf="activeTab === 'albums'">
          <div *ngFor="let item of sharedAlbums; let i = index" class="shared-album-card">
            <!-- Album thumbnail with sharer badge -->
            <div class="thumbnail-wrapper">
              <img 
                [src]="item.thumbnailUrl || getFallbackThumbnail()" 
                [alt]="item.name"
                (error)="onImageError($event)"
                loading="lazy"
              >
              <span class="sharer-badge">{{ item.sharerName }}</span>
            </div>

            <!-- Info -->
            <div class="album-info">
              <p class="name">{{ item.name }}</p>
              <p class="description" *ngIf="item.description">{{ item.description }}</p>
              <p class="shared-date">{{ item.sharedAt | date:'shortDate' }}</p>
            </div>

            <!-- Actions -->
            <div class="album-actions">
              <button class="action-btn" (click)="viewAlbum(item)">👁️ View Album</button>
              <button class="action-btn" *ngIf="!item.isFavorite" (click)="toggleFavorite(item)">❤️</button>
              <button class="action-btn" *ngIf="item.isFavorite" (click)="toggleFavorite(item)" style="color: #ef4444;">♥️</button>
            </div>

            <!-- Media count -->
            <p class="media-count">{{ item.mediaCount }} items in album</p>
          </div>

          <!-- Load more button for albums -->
          <div class="load-more-container" *ngIf="hasMoreAlbums">
            <button class="btn-load-more" (click)="loadMoreSharedAlbums()">Load More Albums</button>
          </div>
        </div>
      </ng-container>
    </div>
  `,
  styles: [`
    .shared-with-me-container {
      padding: 24px;
      max-width: 1200px;
      margin: 0 auto;
    }

    /* Header */
    .header-section h1 {
      font-size: 28px;
      color: #f3f4f6;
      margin-bottom: 16px;
    }

    /* Tabs */
    .tabs {
      display: flex;
      gap: 8px;
      border-bottom: 1px solid #374151;
      padding-bottom: 0;
    }
    .tabs button {
      padding: 8px 20px;
      background: none;
      border: none;
      color: #9ca3af;
      cursor: pointer;
      font-size: 14px;
      transition: all 0.2s;
    }
    .tabs button.active {
      color: #f3f4f6;
      border-bottom: 2px solid #6366f1;
      margin-bottom: -1px;
    }

    /* Loading */
    .loading-state {
      display: flex;
      align-items: center;
      justify-content: center;
      gap: 12px;
      padding: 40px;
    }
    .spinner {
      width: 24px;
      height: 24px;
      border: 3px solid #4b5563;
      border-top-color: #6366f1;
      border-radius: 50%;
      animation: spin 0.8s linear infinite;
    }
    @keyframes spin { to { transform: rotate(360deg); } }

    /* Empty state */
    .empty-state {
      text-align: center;
      padding: 60px 20px;
    }
    .empty-icon { font-size: 48px; margin-bottom: 16px; opacity: 0.5; }
    .empty-state h3 { color: #e5e7eb; margin-bottom: 8px; }
    .empty-state p { color: #9ca3af; font-size: 14px; max-width: 400px; margin: 0 auto; }

    /* Grid */
    .content-grid {
      display: grid;
      grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
      gap: 20px;
      margin-top: 24px;
    }

    /* Shared media card */
    .shared-media-card {
      background-color: #1e293b;
      border-radius: 12px;
      overflow: hidden;
      transition: transform 0.2s, box-shadow 0.2s;
    }
    .shared-media-card:hover {
      transform: translateY(-4px);
      box-shadow: 0 8px 20px rgba(0, 0, 0, 0.3);
    }

    /* Thumbnail wrapper */
    .thumbnail-wrapper {
      position: relative;
      width: 100%;
      aspect-ratio: 4/3;
      overflow: hidden;
      background-color: #0f172a;
    }
    .thumbnail-wrapper img {
      width: 100%;
      height: 100%;
      object-fit: cover;
    }

    /* Sharer badge */
    .sharer-badge {
      position: absolute;
      top: 8px;
      left: 8px;
      background-color: rgba(99, 102, 241, 0.9);
      color: white;
      padding: 4px 10px;
      border-radius: 999px;
      font-size: 11px;
      font-weight: 500;
    }

    /* Media info */
    .media-info {
      padding: 12px 16px;
      background-color: #1e293b;
    }
    .filename {
      margin: 0;
      font-size: 14px;
      color: #e5e7eb;
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
    }
    .shared-date {
      margin: 4px 0 0;
      font-size: 12px;
      color: #9ca3af;
    }

    /* Media actions */
    .media-actions {
      display: flex;
      gap: 8px;
      padding: 8px 16px;
      border-top: 1px solid #374151;
    }
    .action-btn {
      background: none;
      border: none;
      cursor: pointer;
      font-size: 12px;
      color: #9ca3af;
      padding: 6px 8px;
      border-radius: 4px;
      transition: all 0.2s;
    }
    .action-btn:hover { background-color: rgba(99, 102, 241, 0.2); color: #f3f4f6; }

    /* Album card */
    .shared-album-card {
      background-color: #1e293b;
      border-radius: 12px;
      overflow: hidden;
      transition: transform 0.2s, box-shadow 0.2s;
    }
    .shared-album-card:hover {
      transform: translateY(-4px);
      box-shadow: 0 8px 20px rgba(0, 0, 0, 0.3);
    }

    /* Album info */
    .album-info {
      padding: 12px 16px;
      background-color: #1e293b;
    }
    .album-info .name {
      margin: 0;
      font-size: 15px;
      color: #f3f4f6;
      font-weight: 500;
    }
    .album-info .description {
      margin: 2px 0 0;
      font-size: 12px;
      color: #9ca3af;
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
    }

    /* Album actions */
    .album-actions {
      display: flex;
      gap: 8px;
      padding: 12px 16px;
      border-top: 1px solid #374151;
    }

    /* Media count */
    .media-count {
      margin: 0;
      font-size: 12px;
      color: #9ca3af;
      padding: 0 16px 12px;
    }

    /* Load more button */
    .load-more-container {
      text-align: center;
      margin-top: 32px;
      grid-column: 1 / -1;
    }
    .btn-load-more {
      background-color: #6366f1;
      color: white;
      border: none;
      padding: 10px 24px;
      border-radius: 8px;
      cursor: pointer;
      font-size: 14px;
      transition: all 0.2s;
    }
    .btn-load-more:hover { background-color: #4f46e5; transform: translateY(-1px); }

    /* Image error fallback */
    @keyframes fadeIn { from { opacity: 0; } to { opacity: 1; } }
    
    @media (max-width: 768px) {
      .shared-with-me-container { padding: 16px; }
      .content-grid { grid-template-columns: repeat(auto-fill, minmax(240px, 1fr)); gap: 12px; }
    }
  `]
})

export class SharedWithMeComponent implements OnInit {
  activeTab: 'media' | 'albums' = 'media';
  
  // Shared media items (photos/videos)
  sharedMedia: Array<SharedMediaItem & { sharerName?: string; sharedAt?: string; isFavorite?: boolean; mediaCount?: number; thumbnailUrl?: string | null }> = [];
  hasMoreMedia = false;
  private mediaOffset = 0;

  // Shared albums
  sharedAlbums: Array<SharedAlbumItem & { sharerName?: string; mediaCount?: number; sharedAt?: string; isFavorite?: boolean; description?: string }> = [];
  hasMoreAlbums = false;
  private albumOffset = 0;

  isLoading = true;

  // Computed array of items based on active tab for empty/loading state display
  get contentItems(): Array<any> {
    if (this.activeTab === 'media') {
      return this.sharedMedia as any[];
    } else {
      return this.sharedAlbums as any[];
    }
  }

  private router = inject(Router);

  constructor(private shareService: ShareService) {}

  ngOnInit(): void {
    this.loadSharedMedia();
    this.loadSharedAlbums();
  }

  // Load shared media (photos/videos)
  loadSharedMedia(): void {
    this.isLoading = true;
    
    this.shareService.listSharedMedia(20, this.mediaOffset).subscribe({
      next: (response) => {
        if (!this.sharedMedia.length) {
          this.sharedMedia = []; // Clear for first page
        }
        
        const items = response.items || [];
        this.sharedMedia.push(...items);
        this.hasMoreMedia = this.mediaOffset + 20 < response.total;
        this.isLoading = false;

        if (this.activeTab === 'media') {
          // Fetch thumbnail URLs for each item in the background
          items.forEach((item: SharedMediaItem) => this.loadItemThumbnail(item.id));
        }
      },
      error: (err) => {
        console.error('Failed to load shared media:', err);
        this.isLoading = false;
      }
    });
  }

  // Load shared albums
  loadSharedAlbums(): void {
    this.shareService.listSharedAlbums(20, this.albumOffset).subscribe({
      next: (response) => {
        if (!this.sharedAlbums.length) {
          this.sharedAlbums = []; // Clear for first page
        }
        
        const items = response.items || [];
        items.forEach((item: SharedAlbumItem) => {
          // Map the thumbnail from the API response
          const albumItem: SharedAlbumItem & { sharerName?: string; mediaCount?: number; sharedAt?: string; isFavorite?: boolean; description?: string } = {
            ...item,
            thumbnailUrl: item.thumbnailUrl || undefined
          };
          this.sharedAlbums.push(albumItem);
        });
        this.hasMoreAlbums = this.albumOffset + 20 < response.total;
      },
      error: (err) => {
        console.error('Failed to load shared albums:', err);
      }
    });
  }

  // Load more items on scroll/pagination
  loadMoreSharedMedia(): void {
    this.mediaOffset += 20;
    this.loadSharedMedia();
  }

  loadMoreSharedAlbums(): void {
    this.albumOffset += 20;
    this.loadSharedAlbums();
  }

  // Load thumbnail for a shared media item - delegated to the implementation below with real API call

  // Load thumbnail for a shared album - delegated to the implementation below with real API call

  // Get fallback thumbnail URL when image fails to load
  getFallbackThumbnail(): string {
    return 'data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMTAwIiBoZWlnaHQ9IjEwMCIgdmlld0JveD0iMCAwIDEwMCAxMDAiIGZpbGw9Im5vbmUiIHhtbG5zPSJodHRwOi8vd3d3LnczLm9yZy8yMDAwL3N2ZyI+PHBhdGggZD0iTTEwIDEwSDkwVjkwSDEwVjEwWiIgZmlsbD0iIzM3NDE1MSIvPjxjaXJjbGUgY3g9IjUwIiBjeT0iMzUiIHI9IjEyIiBmaWxsPSIjNDk2MzY4Ii8+PHBhdGggZD0iTTE1IDkwVjcwQzE1IDY2Ljc0NjcgMTYuNzM3NSA2My42ODUyIDE5LjEwNTUgNjEuNDk0OEMxOS45MzAxIDYwLjcyMDIgMjAuOTIyNyA1OS41ODc2IDIxLjk3MTggNTguNTEyNEMyMy4wMjEyIDU3LjQzMzcgMjQuMjEzNiA1Ni40NDE5IDI1LjUyNjYgNTUuNjk5OUMyNy4xMDk2IDU1LjAwODIgMjguNDYwMiA1NC43OTQ5IDMwIDU0Ljc5NDlDMzEuNTM5OCA1NC43OTQ5IDMyLjg5MDQgNTUuMDA4MiAzNC40NzM0IDU1LjY5OTlDNTEuODExNiA1Ni40NDE5IDUzLjA2NDAgNTcuNDMzNyA1NC4xMTM0IDU4LjUxMjRDMTUuMTYyNSA1OS41ODc2IDU2LjE1NTEgNjAuNzIwMiA1Ni45OTQ1IDYxLjQ5NDhDNjkuMzYyNSA2My42ODUyIDcwIDEwIDIwIDkwVjkwWiIgZmlsbD0iIzQ5NjM2OCIvPjwvc3ZnPg==';
  }

  // Handle image load error - replace with fallback
  onImageError(event: Event): void {
    const img = event.target as HTMLImageElement;
    if (img) {
      img.src = this.getFallbackThumbnail();
    }
  }

  // View the shared media item (navigate to photo detail page with 'source=shared')
  viewItem(item: SharedMediaItem & { sharerName?: string }): void {
    console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
    console.log(`🔍 [DEBUG] SharedWithMeComponent.viewItem`);
    console.log(`   Item ID: ${item.id}`);
    console.log(`   Filename: ${item.filename}`);
    console.log(`   Thumbnail URL (shown on this page):`, (item as any).thumbnailUrl || 'N/A');
    console.log(`   Navigating to: /photos/${item.id}?source=shared`);
    console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━' + '━'.repeat(10));
    
    // Navigate using Angular Router with a query param so PhotoDetailComponent knows
    // to use the shared-media endpoint instead of ownership-checking endpoint
    this.router.navigate(['/photos', item.id], { queryParams: { source: 'shared' } });
  }

  // View a shared album (navigate to album detail page with 'source=shared')
  viewAlbum(item: SharedAlbumItem & { sharerName?: string }): void {
    console.log('Viewing shared album:', item.id);
    
    // Navigate using Angular Router with a query param so AlbumDetailComponent knows
    // to use the shared-album endpoint instead of ownership-checking endpoint
    this.router.navigate(['/albums', item.id], { queryParams: { source: 'shared' } });
  }

  // Load thumbnail for a shared media item using the dedicated shared-media endpoint
  private loadItemThumbnail(mediaId: string): void {
    this.shareService.getSharedMediaThumbnailUrl(mediaId).subscribe({
      next: (thumbUrl) => {
        if (thumbUrl) {
          const item = this.sharedMedia.find(i => i.id === mediaId);
          if (item) {
            item.thumbnailUrl = thumbUrl;
          }
        }
      },
      error: (err) => {
        console.error('Failed to load thumbnail for shared media:', err, mediaId);
      }
    });
  }

  // Toggle favorite status for a shared item (optional feature)
  toggleFavorite(item: SharedMediaItem | SharedAlbumItem): void {
    // TODO: Add API endpoint to toggle favorite status on shared items
    console.log('Toggling favorite for:', item.id);
  }
}
