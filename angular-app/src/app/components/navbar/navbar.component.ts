import { Component, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router, RouterModule } from '@angular/router';
import { SearchService } from '../../services/search.service';
import { AuthService } from '../../services/auth.service';
import { PresentationService, MediaItem } from '../../services/presentation.service';
import { PhotoService } from '../../services/photo.service';

@Component({
  selector: 'app-navbar',
  standalone: true,
  imports: [CommonModule, RouterModule, FormsModule],
  template: `
    <nav class="navbar navbar-expand-lg navbar-dark bg-dark sticky-top shadow-sm">
      <div class="container-fluid px-3">
        <!-- Brand -->
        <a class="navbar-brand d-flex align-items-center" routerLink="/">
          <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="me-2">
            <path d="M12 2v4"/><path d="m17 9-5-5-5 5"/><path d="m17 15-5 5-5 5"/>
          </svg>
          SteadyPhoto
        </a>

        <!-- Hamburger Menu Button -->
        <button 
          class="navbar-toggler border-0 hamburger-btn" 
          type="button" 
          [attr.aria-expanded]="isMenuOpen"
          aria-label="Toggle menu"
          (click)="toggleMenu()">
          <span class="hamburger-icon">
            <span></span>
            <span></span>
            <span></span>
          </span>
        </button>

        <!-- Search Bar - Desktop Only -->
        <div class="search-container flex-grow-1 mx-auto d-none d-lg-block" *ngIf="authService.isAuthenticated()">
          <div class="input-group">
            <span class="input-group-text bg-secondary border-0 text-white">
              <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>
            </span>
            <input 
              type="text" 
              class="form-control bg-secondary text-white border-0" 
              placeholder="Search by filename..."
              [(ngModel)]="searchTerm"
              (ngModelChange)="onSearch($event)"
              (keyup.enter)="onSearch(searchTerm)"
            >
          </div>
        </div>

        <!-- Mobile Search - Only shown when menu is open -->
        <div *ngIf="authService.isAuthenticated() && isMenuOpen" class="mobile-search mt-3">
          <div class="input-group">
            <span class="input-group-text bg-secondary border-0 text-white">
              <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>
            </span>
            <input 
              type="text" 
              class="form-control bg-secondary text-white border-0" 
              placeholder="Search by filename..."
              [(ngModel)]="searchTerm"
              (ngModelChange)="onSearch($event)"
              (keyup.enter)="onSearch(searchTerm)"
            >
          </div>
        </div>

        <!-- Mobile Tag Filter - Only shown when menu is open -->
        <div *ngIf="authService.isAuthenticated() && isMenuOpen" class="mobile-tag-filter mt-2">
          <input 
            type="text" 
            class="form-control form-control-sm bg-secondary text-white border-0"
            placeholder="#Tag filter..."
            [(ngModel)]="tagFilter"
            (keyup.enter)="onTagFilter()"
          >
        </div>

        <!-- Mobile Online Status - Only shown when menu is open -->
        <div *ngIf="authService.isAuthenticated() && isMenuOpen" class="mobile-status mt-2 text-white opacity-50">
          <small>Online</small>
        </div>

        <!-- Menu Dropdown Panel (Presentation Mode Only) -->
        <div class="menu-panel" [class.show]="isMenuOpen">
          <div class="menu-panel-content">
            <ul class="list-unstyled mb-0">
              <li *ngIf="authService.isAuthenticated()">
                <button 
                  (click)="startPresentationMode(); closeMenu();"
                  class="btn btn-outline-light text-white w-100 py-3 fs-5 menu-item-btn">
                  🎬 Presentation Mode
                </button>
              </li>
            </ul>
          </div>
        </div>

        <!-- Desktop Navigation - Only shown on larger screens -->
        <div class="navbar-nav ms-auto d-none d-lg-flex align-items-center" *ngIf="authService.isAuthenticated()">
          <span class="text-white me-3" style="font-size: 0.9rem; opacity: 0.8;">Online</span>
          <a class="nav-link text-white me-2" routerLink="/albums">Albums</a>
          
          <!-- Account Dropdown -->
          <div class="dropdown">
            <button 
              class="btn btn-outline-light dropdown-toggle" 
              (click)="toggleAccountDropdown($event)" 
              type="button">
               Account
            </button>
            <ul class="dropdown-menu dropdown-menu-end shadow" [class.show]="isAccountDropdownOpen">
              <li><a class="dropdown-item" routerLink="/settings" (click)="isAccountDropdownOpen = false">Settings</a></li> 
              <li><hr class="dropdown-divider"></li>
              <li><a class="dropdown-item text-danger" (click)="onLogout()" style="cursor: pointer;">Logout</a></li>
            </ul>
          </div>
        </div>

        <!-- Guest Links - Desktop Only -->
        <div class="navbar-nav ms-auto d-none d-lg-flex align-items-center gap-3" *ngIf="!authService.isAuthenticated()">
          <a class="nav-link text-white" routerLink="/login">Login</a>
          <a class="btn btn-sm btn-primary text-white px-3" routerLink="/register">Sign Up</a>
        </div>
      </div>
    </nav>
  `,
  styles: [`
    /* Search Container Styles */
    .search-container {
      max-width: 600px;
    }

    .search-container .form-control:focus {
      background-color: #495057 !important;
      box-shadow: none;
      border: 1px solid #6c757d;
    }

    .search-container .input-group-text {
      border-radius: 8px 0 0 8px;
    }

    .search-container .form-control {
      border-radius: 0 8px 8px 0;
    }

    /* Hamburger Menu Button Styles */
    .hamburger-btn {
      display: flex;
      align-items: center;
      justify-content: center;
      padding: 0.5rem;
    }

    .hamburger-icon {
      display: flex;
      flex-direction: column;
      justify-content: space-between;
      width: 24px;
      height: 18px;
      cursor: pointer;
    }

    .hamburger-icon span {
      display: block;
      height: 3px;
      background-color: white;
      border-radius: 2px;
      transition: all 0.3s ease;
    }

    /* Menu Panel Styles */
    .menu-panel {
      max-height: 0;
      overflow: hidden;
      transition: max-height 0.3s ease-out;
      background-color: #1a1d20;
      border-radius: 8px;
      margin-top: 1rem;
    }

    .menu-panel.show {
      max-height: 200px;
    }

    .menu-panel-content {
      padding: 1.5rem;
    }

    .menu-item-btn {
      border-radius: 8px;
      transition: all 0.3s ease;
    }

    .menu-item-btn:hover {
      background-color: rgba(255, 255, 255, 0.1);
      transform: scale(1.02);
    }

    /* Mobile Search */
    .mobile-search {
      max-width: 400px;
    }

    .mobile-tag-filter {
      max-width: 300px;
    }

    /* Responsive adjustments */
    @media (max-width: 991.98px) {
      .search-container,
      .navbar-nav.d-none.d-lg-flex {
        display: none !important;
      }
      
      .mobile-search,
      .mobile-tag-filter,
      .mobile-status {
        width: 100%;
      }
    }

    @media (min-width: 992px) {
      .menu-panel {
        display: none !important;
      }
      
      .hamburger-btn {
        display: none !important;
      }
    }
  `]
})
export class NavbarComponent {
  searchTerm = '';
  tagFilter = '';
  isAccountDropdownOpen = false;
  isMenuOpen = false;

  constructor(
    private searchService: SearchService,
    private router: Router,
    public authService: AuthService, // Made public for template access
    private presentationService: PresentationService,
    private photoService: PhotoService
  ) {}

  toggleMenu(): void {
    this.isMenuOpen = !this.isMenuOpen;
  }

  closeMenu(): void {
    this.isMenuOpen = false;
  }

  toggleAccountDropdown(event: Event): void {
    event.preventDefault();
    this.isAccountDropdownOpen = !this.isAccountDropdownOpen;
  }

  onSearch(term: string): void {
    this.searchTerm = term;
    this.searchService.setSearchTerm(term);
    
    if (this.router.url !== '/') {
      this.router.navigate(['/']);
    }
  }

  onTagFilter(): void {
    if (this.tagFilter) {
      // Normalize tag: remove leading # if present
      const normalizedTag = this.tagFilter.replace(/^#/, '').toLowerCase().trim();
      this.router.navigate(['/'], { queryParams: { tag: normalizedTag }, replaceUrl: true });
    } else {
      this.clearTagFilter();
    }
  }

  clearTagFilter(): void {
    this.tagFilter = '';
    if (this.router.url.includes('?tag=')) {
      // Remove query params and navigate to home
      const urlWithoutQuery = this.router.url.split('?')[0];
      this.router.navigate([urlWithoutQuery]);
    }
  }

  onLogout(): void {
    this.isAccountDropdownOpen = false;
    this.authService.logout().subscribe({
      next: () => this.router.navigate(['/login']),
      error: (err) => console.error('Logout failed', err)
    });
  }

  startPresentationMode(): void {
    // Fetch gallery items for presentation mode
    this.photoService.listMedia(50, 0).subscribe({
      next: (response) => {
        const mediaItems: MediaItem[] = response.photos.map(p => ({
          id: p.id,
          path: p.path,
          filename: p.filename,
          mediaType: p.mediaType
        }));
        
        if (mediaItems.length > 0) {
          this.presentationService.open(mediaItems, 0);
          this.router.navigate(['/presentation']);
        } else {
          alert('No items available for presentation.');
        }
      },
      error: (err) => {
        console.error('Failed to load media for presentation', err);
        alert('Failed to start presentation mode.');
      }
    });
  }
}
