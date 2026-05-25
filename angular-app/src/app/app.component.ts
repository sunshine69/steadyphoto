import { Component, OnInit, OnDestroy, inject, ViewChild } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router, RouterModule } from '@angular/router';
import { PhotoService } from './services/photo.service';
import { PresentationModeComponent } from './components/presentation-mode/presentation-mode.component';
import { PresentationService, MediaItem } from './services/presentation.service';
import { UploadModalComponent } from './components/upload-modal/upload-modal.component';
import { UploadTriggerService } from './services/upload-trigger.service';
import { SidebarComponent } from './components/sidebar/sidebar.component';
import { UserManagementComponent } from './components/user-management/user-management.component';
import { AuthService, CurrentUser } from './services/auth.service';

import { SearchService, SearchScope } from './services/search.service';
import { Subscription } from 'rxjs';

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [CommonModule, RouterModule, PresentationModeComponent, SidebarComponent, FormsModule, UploadModalComponent, UserManagementComponent],
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
              (change)="onSearch()">
              <option value="all">All</option>
              <option value="name">Name</option>
              <option value="tags">Tags</option>
            </select>
            
            <input 
              type="text" 
              placeholder="Search your photos" 
              class="search-input"
              [(ngModel)]="searchTerm"
              (keyup)="onKeyUp($event)"
            />
            <button *ngIf="searchTerm" class="clear-search-btn" (click)="clearSearch()">×</button>
          </div>

          <div class="header-actions">
<!-- Upload Button -->
            <button class="icon-btn upload-trigger" title="Upload Media" (click)="uploadTrigger.open()">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
                <polyline points="17 8 12 3 7 8"/>
                <line x1="12" y1="3" x2="12" y2="15"/>
              </svg>
            </button>

            <!-- User Avatar (click to open user management) -->
            <div class="user-avatar" (click)="openUserManagement()" title="{{ isAdmin ? 'User Management' : 'Profile' }}">{{ avatarInitial }}</div>
          </div>
        </header>

        <!-- Router Outlet for Page Content -->
        <router-outlet></router-outlet>

        <!-- Upload Modal (shown when upload trigger service is open) -->
        <app-upload-modal 
          *ngIf="uploadTrigger.isUploadModalOpen$ | async">
        </app-upload-modal>

        <!-- Presentation Mode Overlay (shown when presentation service is open) -->
        <app-presentation-mode 
          *ngIf="presentationService.isOpen$ | async"
          [items]="presentationService.getItems()"
          [startIndex]="presentationService.getCurrentIndex()">
        </app-presentation-mode>

        <!-- User Management Modal -->
        <app-user-management #userManagement></app-user-management>
      </main>
    </div>

    <!-- Footer (optional, can be removed if not needed) -->
    <footer class="bg-dark text-white py-3 mt-5" *ngIf="false">
      <div class="container text-center">
        <p class="mb-0">© 2024 SteadyPhoto. All rights reserved.</p>
      </div>
    </footer>
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

    .header-actions {
      display: flex;
      align-items: center;
      gap: 12px;
      margin-left: auto;
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
  private searchTimeout?: any;
  isAdmin = false;
  avatarInitial = 'U';

  public uploadTrigger = inject(UploadTriggerService);
  
  @ViewChild('userManagement', { static: false }) userManagementComponent!: UserManagementComponent;

  private authService = inject(AuthService);
  private authSubscription?: Subscription;

  constructor(
    public presentationService: PresentationService,
    private searchService: SearchService
  ) {}

  onKeyUp(event: KeyboardEvent): void {
    // Only trigger search on Enter key or when typing stops (debounce)
    if (event.key === 'Enter') {
      this.performSearch();
    } else {
      // Debounce for regular typing
      this.onSearch();
    }
  }

  onSearch(): void {
    // Clear any existing timeout to debounce rapid typing
    if (this.searchTimeout) {
      clearTimeout(this.searchTimeout);
    }
    
    // Wait 300ms after user stops typing before searching
    this.searchTimeout = setTimeout(() => {
      this.performSearch();
    }, 300);
  }

  private performSearch(): void {
    this.searchService.setSearchTerm(this.searchTerm);
    this.searchService.setSearchScope(this.selectedScope);
  }

  clearSearch(): void {
    this.searchTerm = '';
    this.selectedScope = 'all';
    this.searchService.setSearchTerm('');
    this.searchService.setSearchScope('all');
    if (this.searchTimeout) {
      clearTimeout(this.searchTimeout);
    }
  }

  ngOnInit(): void {
    // Listen for presentation mode close events from child components
    window.addEventListener('presentationModeClosed', this.handlePresentationClose.bind(this));
    
    // Listen for upload complete event to refresh gallery
    window.addEventListener('media-upload-complete', () => {
      console.log('Upload completed, refreshing media list...');
      // Trigger a reload of the photo service data if needed
      // This could be adapted based on how your app handles state updates
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
        console.error('Failed to parse current user', e);
      }
    }
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
  }

  handlePresentationClose(): void {
    this.presentationService.close();
  }
}
