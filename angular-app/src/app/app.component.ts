import { Component, OnInit, OnDestroy, inject, ViewChild } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { RouterModule } from '@angular/router';
import { PresentationModeComponent } from './components/presentation-mode/presentation-mode.component';
import { PresentationService } from './services/presentation.service';
import { UploadModalComponent } from './components/upload-modal/upload-modal.component';
import { UploadTriggerService } from './services/upload-trigger.service';
import { SidebarComponent } from './components/sidebar/sidebar.component';
import { UserManagementComponent } from './components/user-management/user-management.component';
import { AuthService } from './services/auth.service';
import { ShareModalComponent } from './components/share-modal/share-modal.component';
import { ShareTriggerService } from './services/share-trigger.service';

import { SearchService, SearchScope } from './services/search.service';
import { Subject, Subscription, debounceTime, distinctUntilChanged } from 'rxjs';
import { SelectionService } from './services/selection.service';
import { PhotoService } from './services/photo.service';
import { AlbumService } from './services/album.service';
import { Album } from './models/album.model';
import { ExifTriggerService } from './services/exif-trigger.service';

@Component({
    selector: 'app-root',
    imports: [CommonModule, RouterModule, PresentationModeComponent, SidebarComponent, FormsModule, UploadModalComponent, UserManagementComponent, ShareModalComponent],
    template: `
    <!-- Main Layout Container -->
    <div class="app-layout">
      <!-- Fixed Sidebar Navigation -->
      <app-sidebar></app-sidebar>
    
      <!-- Main Content Area -->
      <main class="main-content">
        <!-- Top Header Bar (Search, Settings, User) -->
        <header class="top-header">
          <div class="search-container">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="11" cy="11" r="8"/>
              <line x1="21" y1="21" x2="16.65" y2="16.65"/>
            </svg>
    
            <!-- Search Scope Dropdown -->
            <select
              class="search-scope-select"
              [(ngModel)]="selectedScope"
              (ngModelChange)="onScopeChange($event)">
              <option value="all">All</option>
              <option value="name">Name</option>
              <option value="tags">Tags</option>
              <option value="date">Date</option>
              <option value="location">GPS lat, lon</option>
              <option value="place">Place</option>
            </select>
    
            <input
              type="text"
              placeholder="Search your photos"
              class="search-input"
              [(ngModel)]="searchTerm"
              (keyup)="onKeyUp($event)"
              />
            @if (searchTerm) {
              <button class="clear-search-btn" (click)="clearSearch()">×</button>
            }
    
            <!-- Date Range Input -->
            @if (selectedScope === 'date') {
              <div class="date-range-container">
                <div class="date-part-group">
                  <input
                    type="text"
                    class="date-segment"
                    placeholder="DD"
                    [(ngModel)]="startDay"
                    (keyup.enter)="onDateSearch()"
                    maxlength="2"
                    inputmode="numeric"
                    />
                  <span class="date-separator">/</span>
                  <input
                    type="text"
                    class="date-segment"
                    placeholder="MM"
                    [(ngModel)]="startMonth"
                    (keyup.enter)="onDateSearch()"
                    maxlength="2"
                    inputmode="numeric"
                    />
                  <span class="date-separator">/</span>
                  <input
                    type="text"
                    class="date-segment date-year"
                    placeholder="YYYY"
                    [(ngModel)]="startYear"
                    (keyup.enter)="onDateSearch()"
                    maxlength="4"
                    inputmode="numeric"
                    />
                </div>
                <span class="date-range-separator">to</span>
                <div class="date-part-group">
                  <input
                    type="text"
                    class="date-segment"
                    placeholder="DD"
                    [(ngModel)]="endDay"
                    (keyup.enter)="onDateSearch()"
                    maxlength="2"
                    inputmode="numeric"
                    />
                  <span class="date-separator">/</span>
                  <input
                    type="text"
                    class="date-segment"
                    placeholder="MM"
                    [(ngModel)]="endMonth"
                    (keyup.enter)="onDateSearch()"
                    maxlength="2"
                    inputmode="numeric"
                    />
                  <span class="date-separator">/</span>
                  <input
                    type="text"
                    class="date-segment date-year"
                    placeholder="YYYY"
                    [(ngModel)]="endYear"
                    (keyup.enter)="onDateSearch()"
                    maxlength="4"
                    inputmode="numeric"
                    />
                </div>
              </div>
            }
          </div>
    
          <div class="header-actions">
            <!-- Selection Actions Panel (only when items are selected) -->
            @if (selectedCount > 0) {
              <div class="selection-actions-panel">
                <div class="selected-count-badge" title="{{ selectedCount }} item(s) selected">{{ selectedCount }}</div>
                <select class="selection-action-select" [(ngModel)]="selectedAction" (change)="onActionSelected()">
                  <option [ngValue]="null">Select action...</option>
                  <option value="addAlbum">Add to album</option>
                  <option value="removeAlbum">Remove from album</option>
                  <option value="delete">Delete media</option>
                  <option value="addTags">Add tags</option>
                </select>
                <button class="btn-cancel-selection" (click)="clearSelection()" title="Cancel selection">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <line x1="18" y1="6" x2="6" y2="18"/>
                    <line x1="6" y1="6" x2="18" y2="18"/>
                  </svg>
                </button>
              </div>
            }
    
            <!-- Select All Button -->
            <button class="icon-btn select-all-btn" title="Select All on Page" (click)="onSelectAll()">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <rect x="3" y="3" width="7" height="7"/>
                <rect x="14" y="3" width="7" height="7"/>
                <rect x="3" y="14" width="7" height="7"/>
                <rect x="14" y="14" width="7" height="7"/>
              </svg>
            </button>
    
            <!-- Upload Button -->
            <button class="icon-btn upload-trigger" title="Upload Media" (click)="uploadTrigger.open()">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
                <polyline points="17 8 12 3 7 8"/>
                <line x1="12" y1="3" x2="12" y2="15"/>
              </svg>
            </button>
    
            <!-- Share Button - Opens share modal for the currently selected item -->
            <button class="icon-btn share-trigger" title="Share" (click)="openShareModal()">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/>
                <circle cx="9" cy="7" r="4"/>
                <path d="M23 21v-2a4 4 0 0 0-3-3.87"/>
                <path d="M16 3.13a4 4 0 0 1 0 7.75"/>
              </svg>
            </button>
    
            <!-- User Avatar (click to open user management) -->
            <div class="user-avatar" (click)="openUserManagement()" title="{{ isAdmin ? 'User Management' : 'Profile' }}">{{ avatarInitial }}</div>
          </div>
        </header>
    
        <!-- Router Outlet for Page Content -->
        <router-outlet></router-outlet>
    
        <!-- Upload Modal (shown when upload trigger service is open) -->
        @if (uploadTrigger.isUploadModalOpen$ | async) {
          <app-upload-modal
            >
          </app-upload-modal>
        }
    
        <!-- Presentation Mode Overlay (shown when presentation service is open) -->
        @if (presentationService.isOpen$ | async) {
          <app-presentation-mode
            [items]="presentationService.getItems()"
            [startIndex]="presentationService.getCurrentIndex()">
          </app-presentation-mode>
        }
    
        <!-- Share Modal (shown when share trigger service is open) -->
        @if (shareTrigger.isShareModalOpen$ | async) {
          <app-share-modal
            >
          </app-share-modal>
        }
    
        <!-- User Management Modal -->
        <app-user-management #userManagement></app-user-management>
    
        <!-- Dialog: Add to Album -->
        @if (showAddAlbumDialog) {
          <div class="modal-overlay">
            <div class="modal-content bg-dark border rounded p-4" style="border-color: #6c757d;">
              <h5 class="mb-3 text-white">Add to Album</h5>
              <p class="text-muted mb-3">{{ selectedCount }} items will be added to the selected album.</p>
              <div class="mb-3">
                <label class="form-label text-white">Select Album:</label>
                <select class="form-select" [(ngModel)]="targetAlbumId" style="background-color: #495057; border-color: #6c757d; color: white;">
                  <option [ngValue]="undefined">Choose an album...</option>
                  @for (album of albums; track album) {
                    <option [ngValue]="album.id">{{ album.name }}</option>
                  }
                </select>
              </div>
              <div class="d-flex gap-2 justify-content-end">
                <button class="btn btn-secondary" (click)="closeAddAlbumDialog()">Cancel</button>
                <button class="btn btn-primary" (click)="executeAddToAlbum()" [disabled]="!targetAlbumId">Add to Album</button>
              </div>
            </div>
          </div>
        }
    
        <!-- Dialog: Remove from Album -->
        @if (showRemoveAlbumDialog) {
          <div class="modal-overlay">
            <div class="modal-content bg-dark border rounded p-4" style="border-color: #6c757d;">
              <h5 class="mb-3 text-white">Remove from Album</h5>
              <p class="text-muted mb-3">{{ selectedCount }} items will be removed from the selected album.</p>
              <div class="mb-3">
                <label class="form-label text-white">Select Album:</label>
                <select class="form-select" [(ngModel)]="targetAlbumIdForRemoval" style="background-color: #495057; border-color: #6c757d; color: white;">
                  <option [ngValue]="undefined">Choose an album...</option>
                  @for (album of albums; track album) {
                    <option [ngValue]="album.id">{{ album.name }}</option>
                  }
                </select>
              </div>
              <div class="d-flex gap-2 justify-content-end">
                <button class="btn btn-secondary" (click)="closeRemoveAlbumDialog()">Cancel</button>
                <button class="btn btn-outline-danger" (click)="executeRemoveFromAlbum()" [disabled]="!targetAlbumIdForRemoval">Remove from Album</button>
              </div>
            </div>
          </div>
        }
    
        <!-- Dialog: Delete Media -->
        @if (showDeleteDialog) {
          <div class="modal-overlay">
            <div class="modal-content bg-dark border rounded p-4" style="border-color: #6c757d;">
              <h5 class="mb-3 text-danger">Delete Media</h5>
              <p class="text-muted mb-3">{{ selectedCount }} item(s) will be permanently deleted. This action cannot be undone.</p>
              <div class="alert alert-warning" role="alert">
                Are you sure you want to delete the selected media?
              </div>
              <div class="d-flex gap-2 justify-content-end">
                <button class="btn btn-secondary" (click)="closeDeleteDialog()">Cancel</button>
                <button class="btn btn-danger" (click)="executeDelete()" [disabled]="isDeleting">
                  {{ isDeleting ? 'Deleting...' : 'Delete' }}
                </button>
              </div>
            </div>
          </div>
        }
    
        <!-- Dialog: Add Tags -->
        @if (showAddTagsDialog) {
          <div class="modal-overlay">
            <div class="modal-content bg-dark border rounded p-4" style="border-color: #6c757d;">
              <h5 class="mb-3 text-white">Add Tags</h5>
              <p class="text-muted mb-3">{{ selectedCount }} items will be tagged.</p>
              <div class="mb-3">
                <label class="form-label text-white">Enter Tags:</label>
                <input
                  type="text"
                  [(ngModel)]="tagInput"
                  placeholder="tag1:tag2:tag3..."
                  class="form-control"
                  style="background-color: #495057; border-color: #6c757d; color: white;"
                  autofocus
                  >
                <small class="text-muted">Separate tags with colons (e.g., vacation:sunset:beach)</small>
              </div>
              <div class="d-flex gap-2 justify-content-end">
                <button class="btn btn-secondary" (click)="closeAddTagsDialog()">Cancel</button>
                <button class="btn btn-success" (click)="executeAddTags()" [disabled]="!tagInput.trim()">Add Tags</button>
              </div>
            </div>
          </div>
        }
      </main>
    </div>
    
    <!-- Footer (optional, can be removed if not needed) -->
    @if (false) {
      <footer class="bg-dark text-white py-3 mt-5">
        <div class="container text-center">
          <p class="mb-0">© 2024 SteadyPhoto. All rights reserved.</p>
        </div>
      </footer>
    }
    `,
    styles: [`
    .app-layout {
      display: flex;
      min-height: 100vh;
      background-color: #1a1b2e;
    }

    /* Main Content Area */
    .main-content {
      flex: 1;
      margin-left: 260px; /* Same as sidebar width */
      display: flex;
      flex-direction: column;
      min-height: 100vh;
    }

    /* Top Header Bar */
    .top-header {
      position: sticky;
      top: 0;
      z-index: 50;
      background-color: #1a1b2e;
      border-bottom: 1px solid #2d3748;
      padding: 12px 24px;
      display: flex;
      align-items: center;
      justify-content: space-between;
    }

    .search-container {
      position: relative;
      width: 100%;
      max-width: 600px;
      display: flex;
      align-items: center;
    }

    .search-container svg {
      position: absolute;
      left: 12px;
      top: 50%;
      transform: translateY(-50%);
      color: #9ca3af;
      z-index: 1;
    }

    /* Search Scope Dropdown */
    .search-scope-select {
      width: auto;
      padding: 10px 32px 10px 48px;
      background-color: #2d3748;
      border: 1px solid #374151;
      border-right: none;
      border-radius: 8px 0 0 8px;
      color: #e5e7eb;
      font-size: 14px;
      outline: none;
      cursor: pointer;
      appearance: none;
      -webkit-appearance: none;
      background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 24 24' fill='none' stroke='%239ca3af' stroke-width='2'%3E%3Cpath d='M6 9l6 6 6-6'/%3E%3C/svg%3E");
      background-repeat: no-repeat;
      background-position: right 10px center;
    }

    .search-scope-select:focus {
      border-color: #6366f1;
      background-color: #1e293b;
    }

    .search-scope-select option {
      background-color: #2d3748;
      color: #e5e7eb;
    }

    /* Search Input */
    .search-input {
      flex: 1;
      padding: 10px 40px 10px 16px;
      background-color: #2d3748;
      border: 1px solid #374151;
      border-left: none;
      border-radius: 0 8px 8px 0;
      color: #e5e7eb;
      font-size: 14px;
      outline: none;
      transition: all 0.2s ease;
    }

    .search-input::placeholder {
      color: #6b7280;
    }

    .search-input:focus {
      border-color: #6366f1;
      background-color: #1e293b;
    }

    .clear-search-btn {
      position: absolute;
      right: 10px;
      top: 50%;
      transform: translateY(-50%);
      background: transparent;
      border: none;
      color: #9ca3af;
      font-size: 20px;
      cursor: pointer;
      padding: 4px 8px;
      line-height: 1;
    }

    .clear-search-btn:hover {
      color: #e5e7eb;
    }

    /* Date Range Container */
    .date-range-container {
      display: flex;
      align-items: center;
      gap: 8px;
      margin-left: 12px;
      padding: 4px 12px;
      background-color: #2d3748;
      border: 1px solid #374151;
      border-radius: 8px;
    }

    .date-part {
      display: flex;
      align-items: center;
      gap: 2px;
    }

    .date-part-group {
      display: flex;
      align-items: center;
      gap: 2px;
    }

    .date-segment {
      width: 32px;
      height: 32px;
      padding: 4px 2px;
      background-color: #1e293b;
      border: 1px solid #374151;
      border-radius: 4px;
      color: #e5e7eb;
      font-size: 13px;
      text-align: center;
      outline: none;
      transition: all 0.2s ease;
    }

    .date-segment:focus {
      border-color: #6366f1;
      background-color: #0f172a;
    }

    .date-segment::placeholder {
      color: #6b7280;
      font-size: 12px;
    }

    .date-year {
      width: 40px;
    }

    .date-separator {
      color: #9ca3af;
      font-size: 13px;
      font-weight: 500;
      margin: 0 2px;
    }

    .date-range-separator {
      color: #9ca3af;
      font-size: 13px;
      font-weight: 500;
      margin: 0 6px;
    }

    .header-actions {
      display: flex;
      align-items: center;
      gap: 12px;
      margin-left: auto;
      flex-wrap: nowrap;
    }

    /* ===== Selection Actions Panel (inline in header) ===== */
    .selection-actions-panel {
      display: flex;
      align-items: center;
      gap: 8px;
      margin-right: 8px;
      padding: 4px 10px;
      background: linear-gradient(135deg, #374151, #4b5563);
      border-radius: 8px;
      border: 1px solid #4b5563;
      box-shadow: 0 2px 8px rgba(0, 0, 0, 0.3);
      white-space: nowrap;
    }

    .selected-count-badge {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      min-width: 28px;
      height: 28px;
      padding: 0 6px;
      background: #0d6efd;
      color: white;
      border-radius: 14px;
      font-size: 13px;
      font-weight: 600;
      flex-shrink: 0;
    }

    .selection-action-select {
      width: auto;
      min-width: 140px;
      padding: 5px 28px 5px 8px;
      background-color: #374151;
      border: 1px solid #4b5563;
      border-radius: 6px;
      color: #e5e7eb;
      font-size: 13px;
      outline: none;
      cursor: pointer;
      appearance: none;
      -webkit-appearance: none;
      background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='10' height='10' viewBox='0 0 24 24' fill='none' stroke='%239ca3af' stroke-width='2'%3E%3Cpath d='M6 9l6 6 6-6'/%3E%3C/svg%3E");
      background-repeat: no-repeat;
      background-position: right 6px center;
      transition: all 0.2s ease;
    }

    .selection-action-select:hover {
      background-color: #4b5563;
      border-color: #6366f1;
    }

    .selection-action-select:focus {
      border-color: #6366f1;
      box-shadow: 0 0 0 2px rgba(99, 102, 241, 0.2);
    }

    .selection-action-select option {
      background-color: #374151;
      color: #e5e7eb;
    }

    .btn-cancel-selection {
      width: 28px;
      height: 28px;
      display: flex;
      align-items: center;
      justify-content: center;
      background: transparent;
      border: 1px solid #4b5563;
      border-radius: 6px;
      color: #9ca3af;
      cursor: pointer;
      transition: all 0.2s ease;
      flex-shrink: 0;
    }

    .btn-cancel-selection:hover {
      background-color: rgba(239, 68, 68, 0.15);
      border-color: #ef4444;
      color: #ef4444;
    }

    .icon-btn {
      width: 40px;
      height: 40px;
      border-radius: 8px;
      background: transparent;
      border: none;
      color: #9ca3af;
      cursor: pointer;
      display: flex;
      align-items: center;
      justify-content: center;
      transition: all 0.2s ease;
      flex-shrink: 0;
    }

    .icon-btn:hover {
      background-color: rgba(99, 102, 241, 0.1);
      color: #e5e7eb;
    }

    .user-avatar {
      width: 36px;
      height: 36px;
      border-radius: 50%;
      background: linear-gradient(135deg, #6366f1, #8b5cf6);
      display: flex;
      align-items: center;
      justify-content: center;
      color: white;
      font-weight: bold;
      font-size: 14px;
      cursor: pointer;
      flex-shrink: 0;
    }

    /* Router Outlet Content */
    router-outlet + * {
      flex: 1;
      padding: 24px;
      background-color: #0f172a;
    }
  `]
})
export class AppComponent implements OnInit, OnDestroy {
  searchTerm = '';
  selectedScope: SearchScope = 'all';
  
  // Custom date parts — separate DD/MM/YYYY to avoid native date input cursor issues
  startDay: string = '';
  startMonth: string = '';
  startYear: string = '';
  endDay: string = '';
  endMonth: string = '';
  endYear: string = '';
  isAdmin = false;
  avatarInitial = 'U';

  // Selection state (tracked here so the header can show inline actions)
  selectedIds: Set<string> = new Set();
  get selectedCount(): number { return this.selectedIds.size; }

  // Selection action state
  selectedAction: string | null = null;

  // Dialog visibility flags
  showAddAlbumDialog = false;
  showRemoveAlbumDialog = false;
  showDeleteDialog = false;
  showAddTagsDialog = false;
  
  // Album & tag data
  targetAlbumId: string | undefined = undefined;
  targetAlbumIdForRemoval: string | undefined = undefined;
  albums: Album[] = [];
  
  // Tag input for dialog
  tagInput = '';
  
  isDeleting = false;

  // RxJS Subject for debounced search input
  private searchInput$ = new Subject<string>();
  private searchSubscription?: Subscription;

  public uploadTrigger = inject(UploadTriggerService);
  public shareTrigger = inject(ShareTriggerService);
  
  @ViewChild('userManagement', { static: false }) userManagementComponent!: UserManagementComponent;

  private authService = inject(AuthService);
  private authSubscription?: Subscription;

  // Track whether the last Enter press triggered a search
  private lastEnteredTerm = '';

  constructor(
    public presentationService: PresentationService,
    private searchService: SearchService,
    private selectionService: SelectionService,
    private photoService: PhotoService,
    private albumService: AlbumService,
    private exifTriggerService: ExifTriggerService
  ) {}

  onKeyUp(event: KeyboardEvent): void {
    if (event.key === 'Enter') {
      // Always trigger search on Enter, even if term hasn't changed
      this.lastEnteredTerm = this.searchTerm;
      this.searchInput$.next(this.searchTerm);
    }
  }

  onDateSearch(): void {
    // Only fire on Enter when both dates are set
    if (this.startDay && this.startMonth && this.startYear && 
        this.endDay && this.endMonth && this.endYear) {
      const dateRange = this.buildDateRange();
      this.searchService.triggerSearch('', 'date', dateRange);
    }
  }

  ngOnInit(): void {
    // Set up the RxJS search pipeline — debounce 500ms so typing doesn't fire search
    // Note: removed distinctUntilChanged to allow Enter to always trigger search, even with the same term
    this.searchSubscription = this.searchInput$
      .pipe(
        debounceTime(500)
      )
      .subscribe({
        next: (term: string) => {
          if (term.trim()) {
            this.searchService.setSearchTerm(term);
            this.searchService.setSearchScope(this.selectedScope);
          }
        }
      });

    // Populate date inputs on load: today - 3 months → today
    this.populateDateDefaults();

    // Listen for presentation mode close events from child components
    window.addEventListener('presentationModeClosed', this.handlePresentationClose.bind(this));
    
    // Listen for upload complete event to refresh gallery
    window.addEventListener('media-upload-complete', () => {
      // Refresh gallery logic can be added here if needed
    });

    // Subscribe to auth state changes to keep isAdmin and avatarInitial reactive
    this.authSubscription = this.authService.isAuthenticated$.subscribe(isAuth => {
      if (isAuth) {
        this.checkAdminStatus();
        this.avatarInitial = this.authService.getUsername();
      } else {
        this.isAdmin = false;
        this.avatarInitial = 'U';
      }
    });

    // Subscribe to selection state changes from the service so the header can
    // show the inline action buttons when items are selected
    this.selectionService.selectedIds$.subscribe(ids => {
      this.selectedIds = new Set(ids);
    });

    // Subscribe to "Select All" trigger to refresh the badge count (already
    // subscribed above, but this keeps the count reactive to the service)
    this.selectionService.selectAllTrigger$.subscribe(() => {
      // Badge count already updates via selectedIds$ subscription above
    });

    // Try to restore auth state from session cookie
    this.authService.initializeAuth();

    // Load albums for the dropdown
    this.albumService.getAlbums().subscribe({
      next: (albums: Album[]) => this.albums = albums,
      error: (err: any) => console.error('Error loading albums for header actions', err)
    });

    // Initial check for admin status and avatar
    this.checkAdminStatus();
  }

  checkAdminStatus(): void {
    const currentUser = localStorage.getItem('currentUser');
    if (currentUser) {
      try {
        const user = JSON.parse(currentUser);
        this.isAdmin = user.role === 'admin';
      } catch (e) {
        // Silently fail if user data is corrupted
      }
    }
  }

  clearSearch(): void {
    this.searchTerm = '';
    this.selectedScope = 'all';
    this.startDay = '';
    this.startMonth = '';
    this.startYear = '';
    this.endDay = '';
    this.endMonth = '';
    this.endYear = '';
    this.searchService.setSearchTerm('');
    this.searchService.setSearchScope('all');
    this.searchService.setSearchDate('');
    this.searchService.triggerSearch('', 'all');
  }

  populateDateDefaults(): void {
    const today = new Date();
    const threeMonthsAgo = new Date();
    threeMonthsAgo.setMonth(today.getMonth() - 3);
    
    const fmt = (d: Date) => ({
      day: String(d.getDate()).padStart(2, '0'),
      month: String(d.getMonth() + 1).padStart(2, '0'),
      year: String(d.getFullYear())
    });
    
    const s = fmt(threeMonthsAgo);
    this.startDay = s.day;
    this.startMonth = s.month;
    this.startYear = s.year;
    
    const e = fmt(today);
    this.endDay = e.day;
    this.endMonth = e.month;
    this.endYear = e.year;
  }

  onScopeChange(scope: string): void {
    if (scope === 'date') {
      this.populateDateDefaults();
    }
    // Clear search term when changing scope
    this.searchTerm = '';
    this.searchService.setSearchTerm('');
  }

  /**
   * Builds a date range string in the format expected by the backend parser.
   * Format: "dd/mm/yyyy - dd/mm/yyyy" or "dd/mm/yyyy" if only one date is selected.
   */
  private buildDateRange(): string {
    const fmtDate = (day: string, month: string, year: string): string => {
      if (!day || !month || !year) return '';
      return `${day}/${month}/${year}`;
    };

    const startRange = fmtDate(this.startDay, this.startMonth, this.startYear);
    const endRange = fmtDate(this.endDay, this.endMonth, this.endYear);

    if (startRange && endRange) {
      return `${startRange} - ${endRange}`;
    } else if (startRange) {
      return startRange;
    } else if (endRange) {
      return endRange;
    }
    return '';
  }

  openUserManagement(): void {
    if (!this.isAdmin) {
      alert('You do not have permission to access User Management.');
      return;
    }
    
    if (this.userManagementComponent) {
      this.userManagementComponent.open();
    }
  }

  ngOnDestroy(): void {
    window.removeEventListener('presentationModeClosed', this.handlePresentationClose.bind(this));
    this.searchSubscription?.unsubscribe();
    this.authSubscription?.unsubscribe();
  }

  openShareModal(itemId?: string, itemType: 'media' | 'album' = 'media'): void {
    if (itemId) {
      this.shareTrigger.open(itemId, itemType);
    }
  }

  handlePresentationClose(): void {
    this.presentationService.close();
  }

  onSelectAll(): void {
    // Trigger select all - the photo-list component will handle it
    // by listening to selectAllTrigger$ and selecting all photos on the current page
    this.selectionService.triggerSelectAll();
    // Also trigger the EXIF popup
    this.exifTriggerService.triggerExifPopup();
  }

  // ===== Selection Action Handlers =====

  onActionSelected(): void {
    if (!this.selectedAction) return;
    
    switch (this.selectedAction) {
      case 'addAlbum':
        this.showAddAlbumDialog = true;
        break;
      case 'removeAlbum':
        this.showRemoveAlbumDialog = true;
        break;
      case 'delete':
        if (!confirm(`Are you sure you want to delete ${this.selectedCount} item(s)? This action cannot be undone.`)) {
          this.clearSelection();
          return;
        }
        this.executeDelete();
        break;
      case 'addTags':
        this.showAddTagsDialog = true;
        break;
    }
  }

  clearSelection(): void { 
    this.selectionService.clear();
    this.selectedAction = null;
    this.targetAlbumId = undefined;
    this.targetAlbumIdForRemoval = undefined;
    this.showAddAlbumDialog = false;
    this.showRemoveAlbumDialog = false;
    this.showDeleteDialog = false;
    this.showAddTagsDialog = false;
    this.tagInput = '';
    this.isDeleting = false;
  }

  closeAddAlbumDialog(): void {
    this.showAddAlbumDialog = false;
    this.selectedAction = null;
  }

  executeAddToAlbum(): void {
    if (!this.targetAlbumId || typeof this.targetAlbumId !== 'string' || this.selectedIds.size === 0) {
      alert('Please select a valid album first');
      return;
    }
    const mediaIds = Array.from(this.selectedIds);
    this.albumService.addMediaToAlbum(this.targetAlbumId, mediaIds).subscribe({
      next: () => { 
        alert('Added to album successfully!'); 
        this.closeAddAlbumDialog();
        this.clearSelection();
      },
      error: (err: any) => alert(`Error adding to album - ${err.message || 'Unknown error'}`)
    });
  }

  closeRemoveAlbumDialog(): void {
    this.showRemoveAlbumDialog = false;
    this.selectedAction = null;
  }

  executeRemoveFromAlbum(): void {
    if (!this.targetAlbumIdForRemoval || typeof this.targetAlbumIdForRemoval !== 'string' || this.selectedIds.size === 0) {
      alert('Please select an album to remove items from');
      return;
    }

    const mediaIds = Array.from(this.selectedIds);
    this.albumService.bulkRemoveMediaFromAlbum(this.targetAlbumIdForRemoval, mediaIds).subscribe({
      next: () => { 
        alert('Removed from album successfully!'); 
        this.closeRemoveAlbumDialog();
        this.clearSelection();
      },
      error: (err: any) => alert(`Error removing from album - ${err.message || 'Unknown error'}`)
    });
  }

  closeDeleteDialog(): void {
    this.showDeleteDialog = false;
    this.selectedAction = null;
    this.isDeleting = false;
  }

  executeDelete(): void {
    if (this.selectedIds.size === 0 || !this.photoService) return;

    const mediaIds = Array.from(this.selectedIds);
    let completed = 0;
    const total = mediaIds.length;
    const errors: string[] = [];

    this.isDeleting = true;

    mediaIds.forEach(id => {
      this.photoService.deleteMedia(id).subscribe({
        next: () => {
          completed++;
          if (completed === total) {
            alert(`Successfully deleted ${total} item(s)`);
            this.clearSelection();
          }
        },
        error: (err) => {
          console.error(`Error deleting media ${id}`, err);
          errors.push(id);
          completed++;
          if (completed === total) {
            const successCount = total - errors.length;
            let message = `Deleted ${successCount} item(s)`;
            if (errors.length > 0) {
              message += `, but failed to delete ${errors.length} item(s).`;
            } else {
              message += '.';
            }
            alert(message);
            this.clearSelection();
          }
        }
      });
    });

    if (this.showDeleteDialog) {
      this.closeDeleteDialog();
    } else {
      this.isDeleting = false;
    }
  }

  closeAddTagsDialog(): void {
    this.showAddTagsDialog = false;
    this.selectedAction = null;
    this.tagInput = '';
  }

  executeAddTags(): void {
    if (!this.tagInput.trim() || this.selectedIds.size === 0) {
      alert('Please enter tags and select items');
      return;
    }

    const mediaIds = Array.from(this.selectedIds);
    
    // Parse the tag input - split by colon as per spec
    const newTags = this.tagInput.split(':')
      .map(tag => tag.trim())
      .filter(tag => tag.length > 0)
      .join(':');

    if (newTags.length === 0) {
      alert('No valid tags entered');
      return;
    }

    // Add tags to each selected photo
    let completed = 0;
    const total = mediaIds.length;
    
    mediaIds.forEach(id => {
      this.photoService.getMedia(id).subscribe({
        next: (photo) => {
          if (!photo) return;
          
          // Get existing tags and append new ones
          let existingTags: string[] = [];
          if (Array.isArray(photo.tags)) {
            existingTags = photo.tags as string[];
          } else if (typeof photo.tags === 'string') {
            existingTags = photo.tags.split(':').filter(t => t.trim().length > 0);
          }

          // Add new tags that don't already exist
          const allNewTags = newTags.split(':');
          allNewTags.forEach(tag => {
            if (!existingTags.includes(tag)) {
              existingTags.push(tag);
            }
          });

          // Update the photo with combined tags
          this.photoService.updateTags(id, existingTags.join(':')).subscribe({
            next: () => {
              completed++;
              if (completed === total) {
                alert(`Added ${allNewTags.length} tag(s) to ${total} item(s)`);
                this.closeAddTagsDialog();
                this.clearSelection();
              }
            },
            error: (err) => {
              console.error(`Error updating tags for photo ${id}`, err);
              completed++;
              if (completed === total) {
                alert(`Added tags to ${total} item(s), but some updates may have failed`);
                this.closeAddTagsDialog();
                this.clearSelection();
              }
            }
          });
        },
        error: (err) => {
          console.error(`Error fetching photo ${id}`, err);
          completed++;
          if (completed === total) {
            alert(`Added tags to ${total} item(s), but some updates may have failed`);
            this.closeAddTagsDialog();
            this.clearSelection();
          }
        }
      });
    });
  }
}
