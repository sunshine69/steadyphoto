import { Component, OnInit, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ShareService, PublicShareListItem } from '../../services/share.service';

@Component({
    selector: 'app-my-shares',
    imports: [CommonModule],
    template: `
    <div class="my-shares-container">
      <!-- Header -->
      <div class="header-section">
        <h1>My Shared Links</h1>
        <p class="subtitle">Public share links you've created for photos and albums</p>
      </div>
    
      <!-- Loading state -->
      @if (isLoading && shares.length === 0) {
        <div class="loading-state">
          <span class="spinner"></span>
          <p>Loading your shared links...</p>
        </div>
      }
    
      <!-- Empty state -->
      @if (!isLoading && shares.length === 0) {
        <div class="empty-state">
          <div class="empty-icon">🔗</div>
          <h3>No shared links yet</h3>
          <p>Create a shareable link from any photo or album to share it with others.</p>
        </div>
      }
    
      <!-- Shared links list -->
      @if (!isLoading && shares.length > 0) {
        <div class="shares-list">
          @for (share of shares; track share; let i = $index) {
            <div class="share-item" [class.expired]="isExpired(share)">
              <!-- Resource type badge -->
              <span class="resource-badge" [class.media]="share.resourceType === 'media'" [class.album]="share.resourceType === 'album'">
                {{ share.resourceType === 'media' ? '📷 Photo/Video' : '📁 Album' }}
              </span>
              <!-- Share details -->
              <div class="share-details">
                <p class="resource-name">{{ getResourceName(share) }}</p>
                <div class="share-meta">
                  <!-- Password protection indicator -->
                  @if (share.password_protected) {
                    <span class="password-protected">🔒 Password protected</span>
                  }
                  <!-- Expiration date -->
                  @if (share.expires_at && !isExpired(share)) {
                    <span class="expiration-date">Expires: {{ share.expires_at | date:'short' }}</span>
                  }
                  <!-- Expired indicator -->
                  @if (isExpired(share)) {
                    <span class="expired-indicator">⚠️ Expired</span>
                  }
                  <!-- Access count -->
                  <span class="access-count">{{ share.access_count }} view{{ share.access_count === 1 ? '' : 's' }}</span>
                </div>
                <!-- Share link with copy button -->
                <div class="share-link-row">
                  <input
                    type="text"
                    [value]="getShareUrl(share)"
                    readonly
                    (click)="copyToClipboard(getShareUrl(share))"
                    class="share-url-input"
                    >
                  <button class="btn-copy" (click)="copyToClipboard(getShareUrl(share))">📋 Copy</button>
                </div>
                <!-- Revocation -->
                @if (!isExpired(share)) {
                  <button
                    class="btn-revoke"
                    (click)="revokeShare(share)">
                    Revoke Link
                  </button>
                }
              </div>
            </div>
          }
        </div>
        <!-- No more shares message -->
        @if (shares.length > 0) {
          <p class="no-more-message">That's all your shared links</p>
        }
      }
    </div>
    `,
    changeDetection: ChangeDetectionStrategy.Eager,
    styles: [`
    .my-shares-container {
      padding: 24px;
      max-width: 800px;
      margin: 0 auto;
    }

    /* Header */
    .header-section h1 {
      font-size: 28px;
      color: #f3f4f6;
      margin-bottom: 8px;
    }
    .subtitle {
      color: #9ca3af;
      font-size: 14px;
      margin-top: 0;
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

    /* Shared links list */
    .shares-list {
      display: flex;
      flex-direction: column;
      gap: 16px;
      margin-top: 24px;
    }

    /* Share item card */
    .share-item {
      background-color: #1e293b;
      border-radius: 12px;
      padding: 16px;
      display: flex;
      gap: 16px;
      transition: transform 0.2s, box-shadow 0.2s;
    }
    .share-item:hover {
      transform: translateY(-2px);
      box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
    }

    /* Expired style */
    .share-item.expired {
      opacity: 0.6;
      border-left: 3px solid #f87171;
    }

    /* Resource type badge */
    .resource-badge {
      padding: 4px 10px;
      border-radius: 999px;
      font-size: 12px;
      flex-shrink: 0;
    }
    .resource-badge.media {
      background-color: rgba(99, 102, 241, 0.3);
      color: #a5b4fc;
    }
    .resource-badge.album {
      background-color: rgba(236, 72, 153, 0.3);
      color: #f9a8d4;
    }

    /* Share details */
    .share-details {
      flex: 1;
      min-width: 0;
    }
    .resource-name {
      margin: 0 0 8px;
      font-size: 15px;
      color: #f3f4f6;
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
    }

    /* Meta info */
    .share-meta {
      display: flex;
      flex-wrap: wrap;
      gap: 8px;
      margin-bottom: 12px;
    }
    .password-protected, .expiration-date, .access-count {
      font-size: 12px;
      padding: 2px 8px;
      border-radius: 999px;
    }
    .password-protected { background-color: rgba(234, 179, 8, 0.3); color: #fde68a; }
    .expiration-date { background-color: rgba(52, 211, 153, 0.3); color: #6ee7b7; }
    .expired-indicator { background-color: rgba(248, 113, 113, 0.3); color: #fca5a5; }
    .access-count { background-color: rgba(99, 102, 241, 0.3); color: #a5b4fc; }

    /* Share link row */
    .share-link-row {
      display: flex;
      gap: 8px;
      align-items: center;
    }
    .share-url-input {
      flex: 1;
      padding: 6px 10px;
      background-color: #374151;
      border: 1px solid #4b5563;
      border-radius: 6px;
      color: #e5e7eb;
      font-size: 12px;
      outline: none;
    }
    .btn-copy {
      background-color: #6366f1;
      color: white;
      border: none;
      padding: 6px 12px;
      border-radius: 6px;
      cursor: pointer;
      font-size: 12px;
      transition: all 0.2s;
    }
    .btn-copy:hover { background-color: #4f46e5; transform: translateY(-1px); }

    /* Revoke button */
    .btn-revoke {
      margin-top: 8px;
      background-color: rgba(239, 68, 68, 0.3);
      color: #fca5a5;
      border: none;
      padding: 6px 14px;
      border-radius: 6px;
      cursor: pointer;
      font-size: 12px;
      transition: all 0.2s;
    }
    .btn-revoke:hover { background-color: rgba(239, 68, 68, 0.5); transform: translateY(-1px); }

    /* No more message */
    .no-more-message {
      text-align: center;
      color: #6b7280;
      font-size: 14px;
      margin-top: 32px;
    }

    @media (max-width: 640px) {
      .my-shares-container { padding: 16px; }
      .share-item { flex-direction: column; gap: 8px; }
    }
  `]
})

export class MySharesComponent implements OnInit {
  shares: PublicShareListItem[] = [];
  isLoading = true;

  constructor(private shareService: ShareService) {}

  ngOnInit(): void {
    this.loadMyShares();
  }

  // Load all public share links created by the current user
  loadMyShares(): void {
    this.isLoading = true;
    
    this.shareService.listMyPublicShares().subscribe({
      next: (response) => {
        this.shares = response || [];
        this.isLoading = false;
        
        // Fetch resource names in background for better UX
        if (this.shares.length > 0) {
          this.loadResourceNames();
        }
      },
      error: (err) => {
        console.error('Failed to load my shares:', err);
        this.isLoading = false;
      }
    });
  }

  // Load resource names for each share link (photo/album name)
  private loadResourceNames(): void {
    this.shares.forEach((share: PublicShareListItem) => {
      if (share.resourceType === 'media') {
        // TODO: Add API endpoint to get media name by ID
        console.log('Loading photo name for:', share.id);
      } else if (share.resourceType === 'album') {
        // TODO: Add API endpoint to get album name by ID
        console.log('Loading album name for:', share.id);
      }
    });
  }

  // Get the display name of a resource (photo/album) - will be populated from backend
  getResourceName(share: PublicShareListItem): string {
    if (!share.resourceType) return 'Unknown';
    
    // For now, show the ID as a placeholder until we implement resource name lookup
    return `Resource (${share.resourceType})`;
  }

  // Get the full share URL for a public link
  getShareUrl(share: PublicShareListItem): string {
    const baseUrl = window.location.origin + '/public/shares';
    
    if (share.resourceType === 'media') {
      return `${baseUrl}/media/${share.token}`;
    } else {
      return `${baseUrl}/album/${share.token}`;
    }
  }

  // Check if a share link has expired
  isExpired(share: PublicShareListItem): boolean {
    if (!share.expires_at) return false;
    
    const expiresAt = new Date(share.expires_at);
    return expiresAt < new Date();
  }

  // Copy the share URL to clipboard
  copyToClipboard(text: string): void {
    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(text).then(() => {
        console.log('Copied to clipboard');
      }).catch((err) => {
        console.error('Clipboard API failed, using fallback:', err);
        this._copyFallback(text);
      });
    } else {
      // Fallback for browsers without Clipboard API (non-HTTPS, older browsers)
      this._copyFallback(text);
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

  // Revoke (delete) a share link
  revokeShare(share: PublicShareListItem): void {
    if (!confirm('Are you sure you want to revoke this share link? It will no longer be accessible.')) {
      return;
    }

    this.shareService.revokePublicShareLink(share.id).subscribe({
      next: (response) => {
        console.log('Share link revoked:', response);
        // Remove from local list
        this.shares = this.shares.filter(s => s.id !== share.id);
      },
      error: (err) => {
        console.error('Failed to revoke share link:', err);
      }
    });
  }
}
