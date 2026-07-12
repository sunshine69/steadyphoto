import { Component, ElementRef, ViewChild, OnInit, OnDestroy, inject, HostListener, NgZone } from '@angular/core';

import { FormsModule } from '@angular/forms';
import { ShareService, ShareRequest, PublicShareRequest, CreateShareResponseFull, PublicShareLinkResponse } from '../../services/share.service';
import { ShareTriggerService } from '../../services/share-trigger.service';
import { Subscription } from 'rxjs';

@Component({
    selector: 'app-share-modal',
    imports: [FormsModule],
    template: `
    <!-- Backdrop -->
    @if (isVisible) {
      <div class="modal-backdrop" (click)="closeModal()"></div>
    }
    
    <!-- Modal Container -->
    @if (isVisible) {
      <div class="share-modal-container">
        <div class="share-card">
          <!-- Header -->
          <div class="card-header">
            <h2 class="modal-title">Share</h2>
            <button class="close-btn" (click)="closeModal()">✕</button>
          </div>
          <!-- Tabs: User-to-User vs Public Link -->
          <div class="tabs">
            <button
              [class.active]="activeTab === 'user'"
              (click)="activeTab = 'user'">
              Share with people
            </button>
            <button
              [class.active]="activeTab === 'public'"
              (click)="activeTab = 'public'">
              Create public link
            </button>
          </div>
          <!-- Tab 1: User-to-User Sharing -->
          @if (activeTab === 'user' && !isSharing && !shareComplete) {
            <div class="tab-content">
              <!-- Selected items summary -->
              @if (sharedItems.length > 0) {
                <div class="selected-items-summary">
                  <p class="summary-text">{{ sharedItems.length }} item(s) selected to share</p>
                  @if (sharedItems.length > 1) {
                    <button class="clear-btn" (click)="clearSelected()">Clear all</button>
                  }
                </div>
              }
              <!-- User search -->
              <div class="user-search-section">
                <input
                  type="text"
                  placeholder="Search users to share with..."
                  [(ngModel)]="searchQuery"
                  (input)="onSearchInput()"
                  class="search-input"
                  >
                <!-- Search results -->
                @if (showSearchResults) {
                  <div class="user-results">
                    @for (user of searchResults; track user; let i = $index) {
                      <div
                        class="user-item"
                        [class.selected]="isSelected(user.id)"
                        (click)="toggleUserSelection(user)">
                        <span class="avatar">{{ getInitials(user.email) }}</span>
                        <div class="user-info">
                          <p class="user-email">{{ user.email }}</p>
                          @if (user.username) {
                            <p class="user-username">{{ user.username }}</p>
                          }
                        </div>
                      </div>
                    }
                  </div>
                }
                <!-- No results -->
                @if (searchQuery && showSearchResults && searchResults.length === 0) {
                  <p class="no-results">No users found</p>
                }
              </div>
              <!-- Selected users list -->
              @if (selectedUsers.length > 0) {
                <div class="selected-users-section">
                  <h3>Sharing with:</h3>
                  <ul class="selected-users-list">
                    @for (user of selectedUsers; track user; let i = $index) {
                      <li class="user-tag">
                        {{ getInitials(user.email) }} {{ user.email }}
                        <button class="remove-user-btn" (click)="removeUser(i)">✕</button>
                      </li>
                    }
                  </ul>
                </div>
              }
              <!-- Share button -->
              @if (selectedUsers.length > 0 || sharedItems.length === 0) {
                @if (!isSharing && !shareComplete) {
                  <p class="share-hint">
                    Select users to share with, then click "Share" or create a public link
                  </p>
                }
              }
              <button
                class="btn-share-primary"
                (click)="createUserToUserShare()"
                [disabled]="!canCreateShare || isSharing">
                {{ isSharing ? 'Sharing...' : 'Share' }}
              </button>
            </div>
          }
          <!-- Tab 2: Public Link Sharing -->
          @if (activeTab === 'public' && !isCreatingLink && !linkCreated) {
            <div class="tab-content">
              <p class="tab-description">Create a shareable link for the selected item. Anyone with the link can view it.</p>
              <!-- Selected item summary -->
              @if (sharedItems.length > 0) {
                <div class="selected-items-summary">
                  <span class="item-type-icon">{{ getItemTypeIcon() }}</span>
                  <p class="summary-text">{{ sharedItems[0].filename }}</p>
                  <button class="clear-btn" (click)="clearSelected()">Clear</button>
                </div>
              }
              <!-- Password protection -->
              <div class="password-section">
                <label class="checkbox-label">
                  <input type="checkbox" [(ngModel)]="requirePassword">
                  Require password to view link
                </label>
                @if (requirePassword) {
                  <div class="password-input-group">
                    <input
                      type="text"
                      placeholder="Enter a password..."
                      [(ngModel)]="sharePassword"
                      class="password-input"
                      >
                  </div>
                }
              </div>
              <!-- Expiration -->
              <div class="expiration-section">
                <label class="checkbox-label">
                  <input type="checkbox" [(ngModel)]="requireExpiration">
                  Set expiration date
                </label>
                @if (requireExpiration) {
                  <div class="date-input-group">
                    <input
                      type="datetime-local"
                      [(ngModel)]="shareExpiresAt"
                      class="date-input"
                      [min]="getMinDateTime()"
                      >
                  </div>
                }
              </div>
              <!-- Create link button -->
              <button
                class="btn-share-primary"
                (click)="createPublicShareLink()"
                [disabled]="!canCreatePublicLink || isCreatingLink">
                {{ isCreatingLink ? 'Creating...' : 'Create Share Link' }}
              </button>
            </div>
          }
          <!-- Success State -->
          @if (shareComplete || linkCreated) {
            <div class="result-area" [class.success]="!hasError" [class.error]="hasError">
              <span class="icon">{{ hasError ? '⚠️' : '✅' }}</span>
              <p>{{ shareMessage }}</p>
              <!-- Show the public link if created -->
              @if (createdPublicLink && !hasError) {
                <div class="link-box">
                  <input
                    type="text"
                    [value]="getShareableUrl()"
                    readonly
                    class="share-url-input"
                    >
                  <button (click)="copyToClipboard(getShareableUrl())" class="btn-copy-link">Copy</button>
                </div>
              }
              <button class="btn-close-result" (click)="closeModal()">Close</button>
            </div>
          }
        </div>
      </div>
    }
    `,
    styles: [`
    /* Backdrop */
    .modal-backdrop {
      position: fixed; inset: 0; background-color: rgba(0,0,0,0.6); z-index: 998; backdrop-filter: blur(2px);
    }

    /* Modal Container */
    .share-modal-container {
      position: fixed; top: 50%; left: 50%; transform: translate(-50%, -50%); width: 480px; max-width: 95vw; z-index: 1000;
      background-color: #1e293b; border-radius: 16px; box-shadow: 0 20px 40px rgba(0,0,0,0.5); display: flex; flex-direction: column; max-height: 85vh; overflow-y: auto;
    }

    /* Header */
    .card-header { padding: 16px 20px; border-bottom: 1px solid #374151; display: flex; justify-content: space-between; align-items: center; position: sticky; top: 0; background-color: #1e293b; z-index: 1; }
    .modal-title { margin: 0; font-size: 18px; color: #f3f4f6; font-weight: 600; }
    .close-btn { background: none; border: none; color: #9ca3af; cursor: pointer; font-size: 20px; padding: 4px; line-height: 1; transition: color 0.2s; }
    .close-btn:hover { color: white; }

    /* Tabs */
    .tabs { display: flex; border-bottom: 1px solid #374151; margin-top: 8px; position: sticky; top: 56px; background-color: #1e293b; z-index: 1;}
    .tabs button { 
      flex: 1; padding: 12px 16px; border: none; background: none; color: #9ca3af; cursor: pointer; font-size: 14px; transition: all 0.2s; position: relative;
    }
    .tabs button.active { color: #f3f4f6; font-weight: 500; }
    .tabs button.active::after { content: ''; position: absolute; bottom: -1px; left: 0; right: 0; height: 2px; background-color: #6366f1; border-radius: 99px; }

    /* Tab Content */
    .tab-content { padding: 20px; display: flex; flex-direction: column; gap: 16px; }
    .tab-description { color: #9ca3af; font-size: 14px; margin-bottom: 8px; }

    /* Selected Items Summary */
    .selected-items-summary { 
      display: flex; align-items: center; gap: 8px; padding: 12px; background-color: rgba(99, 102, 241, 0.1); border-radius: 8px; font-size: 13px; color: #e5e7eb; 
    }
    .item-type-icon { font-size: 16px; }
    .summary-text { flex: 1; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; margin: 0; }
    .clear-btn { background: none; border: none; color: #9ca3af; cursor: pointer; font-size: 12px; padding: 4px 8px; transition: color 0.2s; }
    .clear-btn:hover { color: white; }

    /* User Search */
    .user-search-section { position: relative; }
    .search-input { 
      width: 100%; padding: 10px 14px; background-color: #374151; border: 1px solid #4b5563; border-radius: 8px; color: #f3f4f6; font-size: 14px; outline: none; transition: border-color 0.2s;
    }
    .search-input:focus { border-color: #6366f1; }
    .search-input::placeholder { color: #6b7280; }

    /* User Results */
    .user-results { 
      position: absolute; top: 100%; left: 0; right: 0; background-color: #374151; border-radius: 8px; margin-top: 4px; max-height: 200px; overflow-y: auto; z-index: 10; box-shadow: 0 4px 12px rgba(0,0,0,0.3);
    }
    .user-item { 
      display: flex; align-items: center; gap: 10px; padding: 10px 14px; cursor: pointer; transition: background-color 0.2s; border-bottom: 1px solid #4b5563;
    }
    .user-item:last-child { border-bottom: none; }
    .user-item:hover, .user-item.selected { background-color: rgba(99, 102, 241, 0.2); }
    .avatar { 
      width: 32px; height: 32px; border-radius: 50%; background-color: #6366f1; display: flex; align-items: center; justify-content: center; color: white; font-size: 12px; font-weight: 600; flex-shrink: 0;
    }
    .user-info { flex: 1; min-width: 0; }
    .user-email { margin: 0; font-size: 13px; color: #e5e7eb; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
    .user-username { margin: 2px 0 0; font-size: 11px; color: #9ca3af; }

    /* No Results */
    .no-results { padding: 8px 14px; color: #6b7280; font-size: 13px; text-align: center; }

    /* Selected Users Section */
    .selected-users-section h3 { margin: 0 0 8px; font-size: 14px; color: #9ca3af; }
    .selected-users-list { list-style: none; padding: 0; margin: 0; display: flex; flex-wrap: wrap; gap: 6px; max-height: 120px; overflow-y: auto; }
    .user-tag { 
      display: inline-flex; align-items: center; gap: 4px; background-color: rgba(99, 102, 241, 0.3); border-radius: 99px; padding: 4px 8px 4px 4px; font-size: 12px; color: #e5e7eb;
    }
    .remove-user-btn { background: none; border: none; color: #f87171; cursor: pointer; font-size: 12px; padding: 0; line-height: 1; transition: color 0.2s; }
    .remove-user-btn:hover { color: white; }

    /* Share Hint */
    .share-hint { color: #6b7280; font-size: 13px; text-align: center; margin: 8px 0; }

    /* Password Section */
    .password-section, .expiration-section { padding: 12px; background-color: rgba(99, 102, 241, 0.05); border-radius: 8px; }
    .checkbox-label { display: flex; align-items: center; gap: 8px; color: #e5e7eb; font-size: 14px; cursor: pointer; margin-bottom: 8px; }
    .checkbox-label input[type="checkbox"] { accent-color: #6366f1; width: auto; height: auto; }
    
    .password-input-group, .date-input-group { padding-left: 20px; }
    .password-input { 
      width: 100%; padding: 8px 12px; background-color: #374151; border: 1px solid #4b5563; border-radius: 6px; color: #f3f4f6; font-size: 14px; outline: none; transition: border-color 0.2s;
    }
    .password-input:focus { border-color: #6366f1; }
    .date-input { 
      width: 100%; padding: 8px 12px; background-color: #374151; border: 1px solid #4b5563; border-radius: 6px; color: #f3f4f6; font-size: 14px; outline: none; transition: border-color 0.2s;
    }
    .date-input:focus { border-color: #6366f1; }

    /* Buttons */
    .btn-share-primary { 
      background-color: #6366f1; color: white; border: none; padding: 12px 24px; border-radius: 8px; cursor: pointer; font-weight: 500; transition: all 0.2s ease; margin-top: auto;
    }
    .btn-share-primary:hover:not(:disabled) { background-color: #4f46e5; transform: translateY(-1px); box-shadow: 0 4px 12px rgba(99, 102, 241, 0.3); }
    .btn-share-primary:disabled { opacity: 0.5; cursor: not-allowed; }

    /* Result Area */
    .result-area { padding: 24px; text-align: center; display: flex; flex-direction: column; align-items: center; gap: 16px; }
    .icon { font-size: 32px; margin-bottom: 8px;}
    
    /* Success / Error Colors */
    .result-area.success p { color: #4ade80; } 
    .result-area.error p { color: #f87171; }

    .btn-close-result { background-color: #374151; border: none; padding: 8px 20px; border-radius: 6px; cursor: pointer; margin-top: 12px;}
    
    /* Link Box */
    .link-box { width: 90%; display: flex; gap: 8px; }
    .share-url-input { 
      flex: 1; padding: 8px 12px; background-color: #374151; border: 1px solid #4b5563; border-radius: 6px; color: #e5e7eb; font-size: 13px; outline: none;
    }
    .btn-copy-link { 
      background-color: #6366f1; color: white; border: none; padding: 8px 14px; border-radius: 6px; cursor: pointer; font-size: 13px; transition: all 0.2s ease;
    }
    .btn-copy-link:hover { background-color: #4f46e5; }

    @media (max-width: 480px) { 
      .share-modal-container { width: 95vw; top: auto; bottom: 0; transform: translateY(0); border-radius: 16px 16px 0 0; max-height: 80vh;}
    }
  `]
})

export class ShareModalComponent implements OnInit, OnDestroy {
  private shareService = inject(ShareService);
  private shareTrigger = inject(ShareTriggerService);

  isVisible = false;
  
  // Tab selection
  activeTab: 'user' | 'public' = 'user';
  
  // User-to-user sharing state
  searchQuery = '';
  selectedUsers: Array<{ id: string; email: string; username?: string }> = [];
  searchResults: Array<{ id: string; email: string; username?: string }> = [];
  isSearching = false;

  // NEW: Track whether to show the search results dropdown (decoupled from searchResults array)
  showSearchResults = false;

  // Public link sharing state
  requirePassword = false;
  sharePassword = '';
  requireExpiration = false;
  shareExpiresAt = '';

  // Item type being shared (set when opening modal)
  itemType: 'media' | 'album' = 'media';

  // Items being shared (set when opening modal)
  sharedItems: Array<{ id: string; filename?: string }> = [];

  // Sharing action state
  isSharing = false;
  isCreatingLink = false;
  shareComplete = false;
  linkCreated = false;
  hasError = false;
  shareMessage = '';
  createdPublicLink?: PublicShareLinkResponse;

  // User to user sharing state
  isSharingUserToUser = false;

  private searchTimeout: any;
  private shareSubscription?: Subscription;
  
  // Guard flag to prevent infinite loop on modal close.
  // Without this, the subscription in ngOnInit fires when isShareModalOpen$ turns false 
  // and isVisible is still true (the 100ms setTimeout hasn't fired yet), causing 
  // closeModal() → shareTrigger.close() → subscriber calls closeModal() again — forever.
  private _isClosing = false;

  ngOnInit(): void {
    // Subscribe to share trigger service for open/close events (like UploadModalComponent)
    this.shareSubscription = this.shareTrigger.isShareModalOpen$.subscribe((isOpen: boolean) => {
      if (isOpen && !this.isVisible) {
        const data = this.shareTrigger['shareDataSubject'].value;
        if (data?.itemId) {
          this.open(data.itemId, data.itemType || 'media');
        }
      } else if (!isOpen && this.isVisible) {
        // Modal was already closed by an explicit action (click close / ESC / backdrop).
        // Don't call closeModal() again — that would re-trigger the subject to false
        // and cause a race loop. Just hide the modal here since the trigger said "not open".
        this.isVisible = false;
      }
    });

    window.addEventListener('keydown', this.handleEscapeKey);
  }

  ngOnDestroy(): void {
    if(this.shareSubscription) this.shareSubscription.unsubscribe();
    window.removeEventListener('keydown', this.handleEscapeKey);
  }

  private handleEscapeKey = (e: KeyboardEvent): void => {
     if(e.key === 'Escape' && this.isVisible && !this._isClosing) this.closeModal();
  };

  @HostListener('document:click', ['$event'])
  onDocumentClick(event: MouseEvent): void {
    // Close the user search dropdown when clicking outside of it
    const target = event.target as HTMLElement;
    if (this.showSearchResults && 
        !target.closest('.user-search-section') && 
        !target.closest('.user-item')) {
      this.closeSearchDropdown();
    }
  }

  /** Open the modal for a specific item */
  open(itemId: string, itemType: 'media' | 'album'): void {
    
    // Reset state
    this.activeTab = 'user';
    this.searchQuery = '';
    this.selectedUsers = [];
    this.searchResults = [];
    this.showSearchResults = false;
    this.requirePassword = false;
    this.sharePassword = '';
    this.requireExpiration = false;
    this.shareExpiresAt = '';
    this.isSharing = false;
    this.isCreatingLink = false;
    this.shareComplete = false;
    this.linkCreated = false;
    this.hasError = false;
    this.createdPublicLink = undefined;

    // Store item type and set the item being shared
    this.itemType = itemType;
    this.sharedItems = [{ id: itemId }];


    this.isVisible = true;
  }

  closeModal(): void {
    if (this._isClosing) return; // Prevent re-entrance — the guard flag stops infinite loops.
    this._isClosing = true;

    // Notify the trigger service that the modal is being closed
    this.shareTrigger.close();
     setTimeout(() => { 
       this.isVisible = false;
       this._isClosing = false;
     }, 100); 
  }

  /** Close the search dropdown */
  private closeSearchDropdown(): void {
    this.searchResults = [];
    this.showSearchResults = false;
    // Don't clear searchQuery here - let user keep typing if they click back in
  }

  // --- User-to-User Sharing ---

  onSearchInput(): void {
    if (this.searchTimeout) clearTimeout(this.searchTimeout);
    
    const query = this.searchQuery.trim();
    
    if (query.length < 2) {
      this.searchResults = [];
      this.showSearchResults = false;
      return;
    }

    // Show the dropdown while searching
    this.showSearchResults = true;

    // Debounce search request and call the real API
    this.searchTimeout = setTimeout(() => {
      this.isSearching = true;
      
      this.shareService.searchUsers(query).subscribe({
        next: (users) => {
          // Map backend SearchUser results to component's user interface (add username as undefined since we don't have it from API)
          this.searchResults = users.map(u => ({ id: u.id, email: u.email, username: '' }));
          this.isSearching = false;
        },
        error: (err) => {
          console.error('[DEBUG] Failed to search users:', err);
          this.searchResults = [];
          this.isSearching = false;
        }
      });
    }, 300);
  }

  isSelected(userId: string): boolean {
    return this.selectedUsers.some(u => u.id === userId);
  }

  toggleUserSelection(user: { id: string; email: string; username?: string }): void {
    
    const index = this.selectedUsers.findIndex(u => u.id === user.id);
    if (index >= 0) {
      // Remove from selection - don't close dropdown, let user keep selecting others
      this.selectedUsers.splice(index, 1);
    } else {
      // Add to selection and ALWAYS close the search dropdown
      this.selectedUsers.push(user);
      this.closeSearchDropdown();
    }
    
  }

  removeUser(index: number): void {
    this.selectedUsers.splice(index, 1);
  }

  clearSelected(): void {
    this.sharedItems = [];
    this.selectedUsers = [];
    this.itemType = 'media';
  }

  get canCreateShare(): boolean {
    return this.selectedUsers.length > 0;
  }

  async createUserToUserShare(): Promise<void> {

    if (this.sharedItems.length === 0) {
      this.hasError = true;
      this.shareMessage = 'No items selected to share';
      this.shareComplete = true;
      return;
    }

    if (this.selectedUsers.length === 0) {
      this.hasError = true;
      this.shareMessage = 'Select at least one user to share with';
      this.shareComplete = true;
      return;
    }

    this.isSharing = true;

    try {
      const itemIds = this.sharedItems.filter(i => i.id).map(i => i.id);
      

      // Build request body - use FormData or JSON based on backend expectations
      const requestBody: ShareRequest = {
        sharee_user_ids: this.selectedUsers.map(u => u.id),
      };

      // Use the correct ID field based on item type
      if (this.itemType === 'media') {
        requestBody.media_ids = itemIds;
      } else {
        requestBody.album_ids = itemIds;
      }


      await this.shareService.createShare(requestBody).toPromise();
      
      // User-to-user share completed — no public link needed

      this.hasError = false;
      this.shareMessage = 'Items shared successfully!';
      this.shareComplete = true;
    } catch (err) {
      console.error('[DEBUG] Error details:', err);
      // Log the error response if available
      if ((err as any)?.error) {
        console.error('[DEBUG] Error response body:', JSON.stringify((err as any).error));
      }
      this.hasError = true;
      this.shareMessage = 'Failed to share. Please try again.';
    } finally {
      this.isSharing = false;
    }
  }

  // --- Public Link Sharing ---

  get canCreatePublicLink(): boolean {
    return this.sharedItems.length > 0;
  }

  getItemTypeIcon(): string {
    if (this.sharedItems[0]?.id) {
      // Return correct icon based on item type
      if (this.itemType === 'album') {
        return '📁';
      }
      return '📷';
    }
    return '';
  }

  getMinDateTime(): string {
    const now = new Date();
    now.setMinutes(now.getMinutes() + 60); // Minimum: 1 hour from now
    return now.toISOString().slice(0, 16); // Format for datetime-local input
  }

  async createPublicShareLink(): Promise<void> {
    if (this.sharedItems.length === 0) {
      this.hasError = true;
      this.shareMessage = 'No item selected to share';
      this.linkCreated = true;
      return;
    }

    const item = this.sharedItems[0];
    
    // Use the correct resource type based on item type
    const resourceType: 'media' | 'album' = this.itemType;

    this.isCreatingLink = true;

    try {
      const requestBody: PublicShareRequest = {
        resource_type: resourceType,
        resource_id: item.id,
      };


      if (this.requirePassword && this.sharePassword) {
        requestBody.password = this.sharePassword;
      }

      if (this.requireExpiration && this.shareExpiresAt) {
        requestBody.expires_at = this.shareExpiresAt;
      }

      const response = await this.shareService.createPublicShareLink(requestBody).toPromise();
      
      
      this.createdPublicLink = response;
      this.hasError = false;
      this.shareMessage = 'Share link created!';
      this.linkCreated = true;
    } catch (err) {
      console.error('[DEBUG] Failed to create public share link:', err);
      if ((err as any)?.error) {
        console.error('[DEBUG] Error response body:', JSON.stringify((err as any).error));
      }
      this.hasError = true;
      this.shareMessage = 'Failed to create share link. Please try again.';
    } finally {
      this.isCreatingLink = false;
    }
  }

  getShareableUrl(): string {
    if (!this.createdPublicLink) return '';
    
    // Construct the public share URL based on resource type
    const baseUrl = window.location.origin + '/public/shares';
    
    
    if (this.createdPublicLink.resourceType === 'media') {
      return `${baseUrl}/media/${this.createdPublicLink.token}`;
    } else {
      return `${baseUrl}/album/${this.createdPublicLink.token}`;
    }
  }

  copyToClipboard(text: string): void {
    
    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(text).then(() => {
        this._showCopyFeedback();
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
        this._showCopyFeedback();
      } else {
        console.error('[DEBUG] execCommand copy failed');
      }
    } catch (err) {
      console.error('[DEBUG] Fallback copy failed:', err);
    }
    document.body.removeChild(textArea);
  }

  private _showCopyFeedback(): void {
    const btn = event?.target as HTMLButtonElement;
    if (btn) {
      const originalText = btn.textContent;
      btn.textContent = 'Copied!';
      setTimeout(() => { btn.textContent = originalText; }, 1500);
    }
  }

  // --- Helpers ---

  getInitials(email: string): string {
    const parts = email.split('@');
    return (parts[0] || 'U').charAt(0).toUpperCase();
  }
}
