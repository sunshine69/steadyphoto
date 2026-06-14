import { Component, OnInit, OnDestroy, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule } from '@angular/router';
import { ShareService, SharedMediaItem, SharedAlbumItem } from '../../services/share.service';
import { Subscription } from 'rxjs';

@Component({
  selector: 'app-sharing-dashboard',
  standalone: true,
  imports: [CommonModule, RouterModule],
  template: `
    <div class="sharing-page">
      <!-- Header -->
      <div class="page-header">
        <h1>Sharing</h1>
        <p class="subtitle">Manage your shared photos and albums</p>
      </div>

      <!-- Tabs -->
      <div class="tabs-container">
        <button 
          [class.active]="activeTab === 'shared-with-me'" 
          (click)="activeTab = 'shared-with-me'">
          Shared with Me
          <span class="badge" *ngIf="sharedMediaCount > 0">{{ sharedMediaCount }}</span>
        </button>
        <button 
          [class.active]="activeTab === 'my-shares'" 
          (click)="activeTab = 'my-shares'">
          My Shares
        </button>
      </div>

      <!-- Tab Content: Shared with Me -->
      <ng-container *ngIf="activeTab === 'shared-with-me'">
        
        <!-- Media Section -->
        <section class="sharing-section" *ngIf="sharedMediaItems.length > 0 || loadingSharedMedia">
          <div class="section-header">
            <h2>Photos Shared with You</h2>
            <span class="count-badge">{{ sharedMediaCount }} items</span>
          </div>

          <!-- Loading State -->
          <div *ngIf="loadingSharedMedia" class="loading-state">
            <div class="spinner"></div>
            <p>Loading photos...</p>
          </div>

          <!-- Empty State -->
          <div *ngIf="!loadingSharedMedia && sharedMediaItems.length === 0" class="empty-state">
            <span class="icon">📷</span>
            <h3>No photos shared with you yet</h3>
            <p>When someone shares a photo with you, it will appear here.</p>
          </div>

          <!-- Photo Grid -->
          <div *ngIf="!loadingSharedMedia && sharedMediaItems.length > 0" class="photo-grid">
            <div 
              *ngFor="let item of sharedMediaItems; let i = index" 
              class="shared-photo-card"
              (click)="viewPhoto(item.media_id)">
              <div class="thumb-container">
                <img [src]="item.thumbnail_url" [alt]="item.filename" class="photo-thumb">
                <div class="share-badge">Shared with you</div>
              </div>
              <div class="card-info">
                <p class="filename">{{ item.filename }}</p>
                <p class="shared-date">Shared {{ formatDate(item.created_at) }}</p>
              </div>
            </div>

            <!-- Load More -->
            <button 
              *ngIf="hasMoreSharedMedia" 
              (click)="loadMoreSharedMedia()" 
              class="load-more-btn">
              Load more photos
            </button>
          </div>
        </section>

        <!-- Albums Section -->
        <section class="sharing-section" *ngIf="sharedAlbums.length > 0 || loadingSharedAlbums">
          <div class="section-header">
            <h2>Albums Shared with You</h2>
            <span class="count-badge">{{ sharedAlbumCount }} albums</span>
          </div>

          <!-- Loading State -->
          <div *ngIf="loadingSharedAlbums" class="loading-state">
            <div class="spinner"></div>
            <p>Loading albums...</p>
          </div>

          <!-- Empty State -->
          <div *ngIf="!loadingSharedAlbums && sharedAlbums.length === 0" class="empty-state">
            <span class="icon">📁</span>
            <h3>No albums shared with you yet</h3>
            <p>When someone shares an album with you, it will appear here.</p>
          </div>

          <!-- Album Grid -->
          <div *ngIf="!loadingSharedAlbums && sharedAlbums.length > 0" class="album-grid">
            <div 
              *ngFor="let item of sharedAlbums; let i = index" 
              class="shared-album-card"
              (click)="viewAlbum(item.album_id)">
              <div class="thumb-container">
                <img [src]="item.thumbnail_url || 'assets/placeholder-album.jpg'" [alt]="item.name" class="album-thumb">
                <div class="share-badge">Shared with you</div>
              </div>
              <div class="card-info">
                <p class="filename">{{ item.name }}</p>
                <p class="shared-date">{{ item.media_count }} photos · Shared {{ formatDate(item.created_at) }}</p>
              </div>
            </div>

            <!-- Load More -->
            <button 
              *ngIf="hasMoreSharedAlbums" 
              (click)="loadMoreSharedAlbums()" 
              class="load-more-btn">
              Load more albums
            </button>
          </div>
        </section>

        <!-- Nothing to show message -->
        <div *ngIf="!loadingSharedMedia && !loadingSharedAlbums && sharedMediaItems.length === 0 && sharedAlbums.length === 0" class="nothing-state">
          <span class="icon">🤝</span>
          <h3>No sharing activity yet</h3>
          <p>You haven't received any shares or created any public links.</p>
        </div>
      </ng-container>

      <!-- Tab Content: My Shares -->
      <ng-container *ngIf="activeTab === 'my-shares'">
        
        <!-- Public Links Section -->
        <section class="sharing-section" *ngIf="publicShares.length > 0 || loadingPublicShares">
          <div class="section-header">
            <h2>My Share Links</h2>
            <span class="count-badge">{{ publicSharesCount }} links</span>
          </div>

          <!-- Loading State -->
          <div *ngIf="loadingPublicShares" class="loading-state">
            <div class="spinner"></div>
            <p>Loading share links...</p>
          </div>

          <!-- Empty State -->
          <div *ngIf="!loadingPublicShares && publicShares.length === 0" class="empty-state">
            <span class="icon">🔗</span>
            <h3>No share links yet</h3>
            <p>Create a public link from the photo detail view to get started.</p>
          </div>

          <!-- Share Links List -->
          <div *ngIf="!loadingPublicShares && publicShares.length > 0" class="shares-list">
            <div 
              *ngFor="let share of publicShares; let i = index" 
              class="share-link-card">
              
              <div class="share-info">
                <p class="filename">{{ share.resourceType === 'media' ? '📷 Photo' : '📁 Album' }}: {{ getResourceTitle(share) }}</p>
                <p class="link-url" [title]="getShareableUrl(share)">
                  {{ getShortUrl(getShareableUrl(share)) }}
                </p>
              </div>

              <!-- Share Actions -->
              <div class="share-actions">
                <button 
                  (click)="copyLink(share)" 
                  class="btn-copy"
                  [title]="'Copy share link'">
                  📋 Copy
                </button>
                <button 
                  *ngIf="!isExpired(share.expires_at)"
                  (click)="revokeShare(share.id)" 
                  class="btn-revoke"
                  [disabled]="revokingId === share.id"
                  [title]="'Revoke share link'">
                  {{ revokingId === share.id ? 'Revoking...' : '🗑️ Revoke' }}
                </button>
              </div>

              <!-- Expiration Info -->
              <div class="share-meta">
                <span *ngIf="isExpired(share.expires_at)" class="expired-badge">⏰ Expired</span>
                <span *ngIf="!isExpired(share.expires_at) && share.expires_at" class="expires-info">Expires {{ formatDate(share.expires_at) }}</span>
              </div>
            </div>

            <!-- Load More -->
            <button 
              *ngIf="hasMorePublicShares" 
              (click)="loadMorePublicShares()" 
              class="load-more-btn">
              Load more links
            </button>
          </div>
        </section>

        <!-- User-to-User Shares Section -->
        <section class="sharing-section" *ngIf="userToUserShares.length > 0 || loadingUserToUserShares">
          <div class="section-header">
            <h2>User-to-User Shares</h2>
            <span class="count-badge">{{ userToUserShareCount }} shares</span>
          </div>

          <!-- Loading State -->
          <div *ngIf="loadingUserToUserShares" class="loading-state">
            <div class="spinner"></div>
            <p>Loading shared photos...</p>
          </div>

          <!-- Empty State -->
          <div *ngIf="!loadingUserToUserShares && userToUserShares.length === 0" class="empty-state">
            <span class="icon">👥</span>
            <h3>No shared photos yet</h3>
            <p>Select photos and share them with people using the Share button.</p>
          </div>

          <!-- Shared Photos Grid -->
          <div *ngIf="!loadingUserToUserShares && userToUserShares.length > 0" class="photo-grid">
            <div 
              *ngFor="let item of userToUserShares; let i = index" 
              class="shared-photo-card"
              (click)="viewPhoto(item.media_id)">
              <div class="thumb-container">
                <img [src]="item.thumbnail_url" [alt]="item.filename" class="photo-thumb">
                <div class="share-badge">Shared with {{ getRecipientName(item) }}</div>
              </div>
              <div class="card-info">
                <p class="filename">{{ item.filename }}</p>
                <p class="shared-date">Shared {{ formatDate(item.created_at) }}</p>
              </div>
            </div>

            <!-- Load More -->
            <button 
              *ngIf="hasMoreUserToUserShares" 
              (click)="loadMoreUserToUserShares()" 
              class="load-more-btn">
              Load more photos
            </button>
          </div>
        </section>

        <!-- Nothing to show message -->
        <div *ngIf="!loadingPublicShares && !loadingUserToUserShares && publicShares.length === 0 && userToUserShares.length === 0" class="nothing-state">
          <span class="icon">🤝</span>
          <h3>No sharing activity yet</h3>
          <p>You haven't created any share links or shared photos with people.</p>
        </div>
      </ng-container>

    </div>
  `,
  styles: [`
    .sharing-page {
      max-width: 1200px;
      margin: 0 auto;
      padding: 32px 24px;
    }

    /* Header */
    .page-header {
      margin-bottom: 32px;
    }

    .page-header h1 {
      font-size: 28px;
      color: #f3f4f6;
      margin: 0 0 8px 0;
      font-weight: 700;
    }

    .subtitle {
      color: #9ca3af;
      font-size: 15px;
      margin: 0;
    }

    /* Tabs */
    .tabs-container {
      display: flex;
      gap: 8px;
      margin-bottom: 32px;
      border-bottom: 1px solid #374151;
      padding-bottom: 0;
    }

    .tabs-container button {
      background: none;
      border: none;
      color: #9ca3af;
      font-size: 15px;
      font-weight: 500;
      padding: 12px 24px;
      cursor: pointer;
      position: relative;
      transition: all 0.2s ease;
    }

    .tabs-container button:hover {
      color: #e5e7eb;
    }

    .tabs-container button.active {
      color: #818cf8;
    }

    .tabs-container button.active::after {
      content: '';
      position: absolute;
      bottom: -1px;
      left: 0;
      right: 0;
      height: 2px;
      background-color: #6366f1;
      border-radius: 99px;
    }

    .badge {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      min-width: 20px;
      height: 20px;
      padding: 0 6px;
      background-color: #6366f1;
      color: white;
      font-size: 11px;
      border-radius: 99px;
      margin-left: 8px;
    }

    /* Sections */
    .sharing-section {
      margin-bottom: 40px;
    }

    .section-header {
      display: flex;
      align-items: center;
      justify-content: space-between;
      margin-bottom: 20px;
    }

    .section-header h2 {
      font-size: 18px;
      color: #f3f4f6;
      margin: 0;
      font-weight: 600;
    }

    .count-badge {
      background-color: rgba(99, 102, 241, 0.15);
      color: #818cf8;
      padding: 4px 12px;
      border-radius: 99px;
      font-size: 13px;
    }

    /* Loading State */
    .loading-state {
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      padding: 48px 0;
      color: #9ca3af;
    }

    .spinner {
      width: 32px;
      height: 32px;
      border: 3px solid #374151;
      border-top-color: #6366f1;
      border-radius: 50%;
      animation: spin 0.8s linear infinite;
      margin-bottom: 16px;
    }

    @keyframes spin {
      to { transform: rotate(360deg); }
    }

    /* Empty States */
    .empty-state, .nothing-state {
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      text-align: center;
      padding: 48px 0;
    }

    .icon {
      font-size: 48px;
      margin-bottom: 16px;
    }

    .empty-state h3, .nothing-state h3 {
      color: #e5e7eb;
      font-size: 16px;
      margin: 0 0 8px 0;
    }

    .empty-state p, .nothing-state p {
      color: #9ca3af;
      font-size: 14px;
      margin: 0;
    }

    /* Photo Grid */
    .photo-grid {
      display: grid;
      grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
      gap: 16px;
    }

    .shared-photo-card {
      background-color: #1e293b;
      border-radius: 12px;
      overflow: hidden;
      cursor: pointer;
      transition: transform 0.2s ease, box-shadow 0.2s ease;
    }

    .shared-photo-card:hover {
      transform: translateY(-4px);
      box-shadow: 0 8px 24px rgba(0, 0, 0, 0.3);
    }

    .thumb-container {
      position: relative;
      width: 100%;
      aspect-ratio: 16/9;
      overflow: hidden;
    }

    @supports not (aspect-ratio: 16/9) {
      .thumb-container {
        padding-top: 56.25%;
      }
      .photo-thumb {
        position: absolute;
        top: 0;
        left: 0;
      }
    }

    .photo-thumb, .album-thumb {
      width: 100%;
      height: 100%;
      object-fit: cover;
      display: block;
    }

    .share-badge {
      position: absolute;
      top: 8px;
      right: 8px;
      background-color: rgba(99, 102, 241, 0.9);
      color: white;
      padding: 4px 8px;
      border-radius: 6px;
      font-size: 11px;
      font-weight: 500;
    }

    .card-info {
      padding: 12px;
    }

    .filename {
      margin: 0;
      color: #e5e7eb;
      font-size: 14px;
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
    }

    .shared-date, .album-info p {
      margin: 4px 0 0;
      color: #9ca3af;
      font-size: 12px;
    }

    /* Album Grid */
    .album-grid {
      display: grid;
      grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
      gap: 16px;
    }

    .shared-album-card {
      background-color: #1e293b;
      border-radius: 12px;
      overflow: hidden;
      cursor: pointer;
      transition: transform 0.2s ease, box-shadow 0.2s ease;
    }

    .shared-album-card:hover {
      transform: translateY(-4px);
      box-shadow: 0 8px 24px rgba(0, 0, 0, 0.3);
    }

    /* Share Links List */
    .shares-list {
      display: flex;
      flex-direction: column;
      gap: 12px;
    }

    .share-link-card {
      background-color: #1e293b;
      border-radius: 12px;
      padding: 16px;
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 16px;
    }

    .share-info {
      flex: 1;
      min-width: 0;
    }

    .link-url {
      color: #818cf8;
      font-size: 13px;
      word-break: break-all;
      margin-top: 4px;
    }

    .share-actions {
      display: flex;
      gap: 8px;
      flex-shrink: 0;
    }

    .btn-copy, .btn-revoke {
      padding: 6px 12px;
      border-radius: 6px;
      border: none;
      cursor: pointer;
      font-size: 13px;
      transition: all 0.2s ease;
    }

    .btn-copy {
      background-color: rgba(99, 102, 241, 0.15);
      color: #818cf8;
    }

    .btn-copy:hover {
      background-color: rgba(99, 102, 241, 0.3);
    }

    .btn-revoke {
      background-color: rgba(239, 68, 68, 0.15);
      color: #f87171;
    }

    .btn-revoke:hover:not(:disabled) {
      background-color: rgba(239, 68, 68, 0.3);
    }

    .btn-revoke:disabled {
      opacity: 0.5;
      cursor: not-allowed;
    }

    .share-meta {
      display: flex;
      gap: 12px;
    }

    .expired-badge {
      color: #f87171;
      font-size: 12px;
    }

    .expires-info {
      color: #9ca3af;
      font-size: 12px;
    }

    /* Load More Button */
    .load-more-btn {
      display: block;
      width: 100%;
      padding: 16px;
      margin-top: 24px;
      background-color: rgba(99, 102, 241, 0.1);
      border: none;
      color: #818cf8;
      font-size: 15px;
      cursor: pointer;
      border-radius: 8px;
      transition: all 0.2s ease;
    }

    .load-more-btn:hover {
      background-color: rgba(99, 102, 241, 0.2);
    }

    /* Responsive */
    @media (max-width: 768px) {
      .sharing-page {
        padding: 24px 16px;
      }

      .photo-grid, .album-grid {
        grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
        gap: 12px;
      }

      .share-link-card {
        flex-direction: column;
        align-items: stretch;
      }

      .share-actions {
        justify-content: center;
      }

      .tabs-container button {
        padding: 10px 16px;
        font-size: 14px;
      }
    }
  `]
})
export class SharingDashboardComponent implements OnInit, OnDestroy {
  private shareService = inject(ShareService);
  
  // Tab state
  activeTab: 'shared-with-me' | 'my-shares' = 'shared-with-me';

  // Shared with Me - Media
  sharedMediaItems: Array<SharedMediaItem> = [];
  sharedMediaCount = 0;
  loadingSharedMedia = false;
  hasMoreSharedMedia = true;
  private sharedMediaOffset = 0;
  private readonly SHARED_MEDIA_LIMIT = 20;

  // Shared with Me - Albums
  sharedAlbums: Array<SharedAlbumItem> = [];
  sharedAlbumCount = 0;
  loadingSharedAlbums = false;
  hasMoreSharedAlbums = true;
  private sharedAlbumOffset = 0;
  private readonly SHARED_ALBUM_LIMIT = 20;

  // My Shares - Public Links
  publicShares: Array<any> = [];
  publicSharesCount = 0;
  loadingPublicShares = false;
  hasMorePublicShares = true;
  private publicShareOffset = 0;
  private readonly PUBLIC_SHARE_LIMIT = 20;

  // My Shares - User-to-User Shares
  userToUserShares: Array<SharedMediaItem> = [];
  userToUserShareCount = 0;
  loadingUserToUserShares = false;
  hasMoreUserToUserShares = true;
  private userToUserOffset = 0;
  private readonly USER_TO_USER_LIMIT = 20;

  // Revoking state
  revokingId: string | null = null;

  private subscription?: Subscription;

  ngOnInit(): void {
    this.loadSharedMedia();
    this.loadSharedAlbums();
    this.loadPublicShares();
    this.loadUserToUserShares();
    
    // Listen for share modal events to refresh data
    window.addEventListener('share-modal-open', () => {});
  }

  ngOnDestroy(): void {
    this.subscription?.unsubscribe();
    window.removeEventListener('share-modal-open', () => {});
  }

  // --- Shared with Me - Media ---

  loadSharedMedia(): void {
    if (this.loadingSharedMedia || !this.hasMoreSharedMedia) return;
    
    this.loadingSharedMedia = true;
    this.shareService.listSharedMedia(this.SHARED_MEDIA_LIMIT, this.sharedMediaOffset).subscribe({
      next: (response) => {
        this.sharedMediaItems.push(...(response.items as any));
        this.sharedMediaCount = response.total || 0;
        this.hasMoreSharedMedia = this.sharedMediaOffset + this.SHARED_MEDIA_LIMIT < this.sharedMediaCount;
        this.sharedMediaOffset += this.SHARED_MEDIA_LIMIT;
      },
      error: (err) => {
        console.error('Failed to load shared media:', err);
      },
      complete: () => {
        this.loadingSharedMedia = false;
      }
    });
  }

  loadMoreSharedMedia(): void {
    this.loadSharedMedia();
  }

  // --- Shared with Me - Albums ---

  loadSharedAlbums(): void {
    if (this.loadingSharedAlbums || !this.hasMoreSharedAlbums) return;
    
    this.loadingSharedAlbums = true;
    this.shareService.listSharedAlbums(this.SHARED_ALBUM_LIMIT, this.sharedAlbumOffset).subscribe({
      next: (response) => {
        this.sharedAlbums.push(...(response.items as any));
        this.sharedAlbumCount = response.total || 0;
        this.hasMoreSharedAlbums = this.sharedAlbumOffset + this.SHARED_ALBUM_LIMIT < this.sharedAlbumCount;
        this.sharedAlbumOffset += this.SHARED_ALBUM_LIMIT;
      },
      error: (err) => {
        console.error('Failed to load shared albums:', err);
      },
      complete: () => {
        this.loadingSharedAlbums = false;
      }
    });
  }

  loadMoreSharedAlbums(): void {
    this.loadSharedAlbums();
  }

  // --- My Shares - Public Links ---

  loadPublicShares(): void {
    if (this.loadingPublicShares || !this.hasMorePublicShares) return;
    
    this.loadingPublicShares = true;
    this.shareService.listMyPublicShares().subscribe({
      next: (shares) => {
        // Filter out expired shares from display but keep count
        const activeShares = shares.filter((s: any) => !this.isExpired(s.expires_at));
        this.publicShares.push(...activeShares);
        this.publicSharesCount = shares.length;
        this.hasMorePublicShares = false; // API returns all at once
      },
      error: (err) => {
        console.error('Failed to load public shares:', err);
      },
      complete: () => {
        this.loadingPublicShares = false;
      }
    });
  }

  loadMorePublicShares(): void {
    this.loadPublicShares();
  }

  // --- My Shares - User-to-User Shares ---

  loadUserToUserShares(): void {
    if (this.loadingUserToUserShares || !this.hasMoreUserToUserShares) return;
    
    this.loadingUserToUserShares = true;
    this.shareService.listSharedMedia(this.USER_TO_USER_LIMIT, this.userToUserOffset).subscribe({
      next: (response) => {
        // These are items shared with us via user-to-user sharing
        // Filter to only show those where we're the recipient and it was a direct share
        const ourShares = response.items.filter((item: any) => item.shared_by !== null);
        this.userToUserShares.push(...ourShares as any);
        this.userToUserShareCount = response.total || 0;
        this.hasMoreUserToUserShares = this.userToUserOffset + this.USER_TO_USER_LIMIT < this.userToUserShareCount;
        this.userToUserOffset += this.USER_TO_USER_LIMIT;
      },
      error: (err) => {
        console.error('Failed to load user-to-user shares:', err);
      },
      complete: () => {
        this.loadingUserToUserShares = false;
      }
    });
  }

  loadMoreUserToUserShares(): void {
    this.loadUserToUserShares();
  }

  // --- Actions ---

  viewPhoto(mediaId: string): void {
    window.location.href = `/photos/${mediaId}`;
  }

  viewAlbum(albumId: string): void {
    window.location.href = `/albums/${albumId}`;
  }

  copyLink(share: any): void {
    const url = this.getShareableUrl(share);
    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(url).then(() => {
        // Could show a toast notification here
      }).catch((err) => {
        console.error('Clipboard API failed, using fallback:', err);
        this._copyFallback(url);
      });
    } else {
      // Fallback for browsers without Clipboard API (non-HTTPS, older browsers)
      this._copyFallback(url);
    }
  }

  private _copyFallback(text: string): void {
    const textArea = document.createElement('textarea');
    textArea.value = text;
    textArea.style.position = 'fixed';
    textArea.style.left = '-9999px';
    textArea.style.top = '-9999px';
    document.body.appendChild(textArea);
    textArea.focus();
    textArea.select();
    try {
      const successful = document.execCommand('copy');
      if (successful) {
        console.log('Copied to clipboard (fallback)');
      } else {
        console.error('execCommand copy failed');
      }
    } catch (err) {
      console.error('Fallback copy failed:', err);
    }
    document.body.removeChild(textArea);
  }

  async revokeShare(id: string): Promise<void> {
    if (!confirm('Are you sure you want to revoke this share link?')) return;
    
    this.revokingId = id;
    
    try {
      await this.shareService.revokePublicShareLink(id).toPromise();
      // Remove from local list
      this.publicShares = this.publicShares.filter(s => s.id !== id);
      this.publicSharesCount--;
    } catch (err) {
      console.error('Failed to revoke share:', err);
    } finally {
      this.revokingId = null;
    }
  }

  // --- Helpers ---

  getShareableUrl(share: any): string {
    const baseUrl = window.location.origin + '/public/shares';
    
    if (share.resourceType === 'media') {
      return `${baseUrl}/media/${share.token}`;
    } else {
      return `${baseUrl}/album/${share.token}`;
    }
  }

  getShortUrl(url: string): string {
    // Truncate long URLs for display
    if (url.length > 50) {
      return url.substring(0, 47) + '...';
    }
    return url;
  }

  getResourceTitle(share: any): string {
    // This would typically come from the share response with resource metadata
    return `Resource ${share.resourceId}`;
  }

  getRecipientName(item: SharedMediaItem): string {
    // This would come from a user lookup API - for now return placeholder
    return 'Someone';
  }

  isExpired(expiresAt?: string): boolean {
    if (!expiresAt) return false;
    
    try {
      const expirationDate = new Date(expiresAt);
      return expirationDate < new Date();
    } catch (err) {
      console.error('Failed to parse expiration date:', err);
      return true; // Treat invalid dates as expired
    }
  }

  formatDate(dateString?: string): string {
    if (!dateString) return 'Unknown';
    
    try {
      const date = new Date(dateString);
      const now = new Date();
      const diffMs = now.getTime() - date.getTime();
      const daysAgo = Math.floor(diffMs / (1000 * 60 * 60 * 24));

      if (daysAgo === 0) return 'Today';
      if (daysAgo === 1) return 'Yesterday';
      if (daysAgo < 7) return `${daysAgo} days ago`;
      
      return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
    } catch (err) {
      console.error('Failed to format date:', err);
      return dateString;
    }
  }
}
