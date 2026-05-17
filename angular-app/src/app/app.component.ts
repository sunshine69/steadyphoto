import { Component, OnInit, OnDestroy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router, RouterModule } from '@angular/router';
import { PhotoService } from './services/photo.service';
import { PresentationModeComponent } from './components/presentation-mode/presentation-mode.component';
import { PresentationService, MediaItem } from './services/presentation.service';
import { SidebarComponent } from './components/sidebar/sidebar.component';
import { SearchService } from './services/search.service';

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [CommonModule, RouterModule, PresentationModeComponent, SidebarComponent, FormsModule],
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
            <input 
              type="text" 
              placeholder="Search your photos" 
              class="search-input"
              [(ngModel)]="searchTerm"
              (keyup)="onSearch()"
            />
            <button *ngIf="searchTerm" class="clear-search-btn" (click)="clearSearch()">×</button>
          </div>

          <div class="header-actions">
            <!-- Theme Toggle -->
            <button class="icon-btn" title="Toggle theme">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/>
              </svg>
            </button>

            <!-- Notifications -->
            <button class="icon-btn" title="Notifications">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9"/>
                <path d="M13.73 21a2 2 0 0 1-3.46 0"/>
              </svg>
            </button>

            <!-- User Avatar -->
            <div class="user-avatar">S</div>
          </div>
        </header>

        <!-- Router Outlet for Page Content -->
        <router-outlet></router-outlet>

        <!-- Presentation Mode Overlay (shown when presentation service is open) -->
        <app-presentation-mode 
          *ngIf="presentationService.isOpen$ | async"
          [items]="presentationService.getItems()"
          [startIndex]="presentationService.getCurrentIndex()">
        </app-presentation-mode>
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
    }

    .search-container svg {
      position: absolute;
      left: 12px;
      top: 50%;
      transform: translateY(-50%);
      color: #9ca3af;
    }

    .search-input {
      width: 100%;
      padding: 10px 16px 10px 40px;
      background-color: #2d3748;
      border: 1px solid #374151;
      border-radius: 8px;
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
  private searchTimeout?: any;

  constructor(
    public presentationService: PresentationService,
    private searchService: SearchService
  ) {}

  onSearch(): void {
    // Clear any existing timeout to debounce rapid typing
    if (this.searchTimeout) {
      clearTimeout(this.searchTimeout);
    }
    
    // Wait 300ms after user stops typing before searching
    this.searchTimeout = setTimeout(() => {
      this.searchService.setSearchTerm(this.searchTerm);
    }, 300);
  }

  clearSearch(): void {
    this.searchTerm = '';
    this.searchService.setSearchTerm('');
    if (this.searchTimeout) {
      clearTimeout(this.searchTimeout);
    }
  }

  ngOnInit(): void {
    // Listen for presentation mode close events from child components
    window.addEventListener('presentationModeClosed', this.handlePresentationClose.bind(this));
  }

  ngOnDestroy(): void {
    window.removeEventListener('presentationModeClosed', this.handlePresentationClose.bind(this));
  }

  handlePresentationClose(): void {
    this.presentationService.close();
  }
}
