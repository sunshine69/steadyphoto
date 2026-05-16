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
      <div class="container">
        <!-- Brand -->
        <a class="navbar-brand d-flex align-items-center" routerLink="/">
          <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="me-2">
            <path d="M12 2v4"/><path d="m17 9-5-5-5 5"/><path d="m17 15-5 5-5 5"/>
          </svg>
          SteadyPhoto
        </a>

        <!-- Hamburger Menu Button -->
        <button 
          class="navbar-toggler border-0" 
          type="button" 
          (click)="toggleMenu()"
          aria-label="Toggle menu">
          <span class="navbar-toggler-icon"></span>
        </button>

        <!-- Search Bar (Middle) - Only visible when menu is open on mobile -->
        <div class="flex-grow-1 d-flex justify-content-center mx-4" *ngIf="authService.isAuthenticated() && !isMenuOpen">
          <div class="search-container w-100" style="max-width: 500px;">
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
        </div>

        <!-- Tag Filter (Middle) - Only for Authenticated Users -->
        <div class="tag-filter d-flex align-items-center me-3" *ngIf="authService.isAuthenticated() && !isMenuOpen">
          <input 
            type="text" 
            class="form-control form-control-sm bg-secondary text-white border-0 tag-input"
            placeholder="#Tag filter..."
            [(ngModel)]="tagFilter"
            (keyup.enter)="onTagFilter()"
          >
          <button *ngIf="tagFilter" (click)="clearTagFilter()" class="btn btn-sm btn-outline-light ms-1">
            ×
          </button>
        </div>

        <!-- Right Side Actions -->
        <div class="navbar-nav ms-auto">
          <ul class="navbar-nav align-items-center">
            <!-- If Authenticated: Show Profile/Logout -->
            <ng-container *ngIf="authService.isAuthenticated(); else guestLinks">
              <li class="nav-item me-3 text-white d-flex align-items-center" style="font-size: 0.9rem; opacity: 0.8;">
                Online
              </li>
              <li class="nav-item">
                <a class="nav-link px-2" routerLink="/albums">Albums</a>
              </li>
              <li class="nav-item dropdown">
                <a class="nav-link btn btn-outline-light text-white border-0 p-0 ms-2" (click)="toggleAccountDropdown($event)" role="button">
                   Account
                </a>
                <ul class="dropdown-menu dropdown-menu-end shadow" [class.show]="isAccountDropdownOpen">
                  <li><a class="dropdown-item" routerLink="/settings" (click)="isAccountDropdownOpen = false">Settings</a></li> 
                  <li><hr class="dropdown-divider"></li>
                  <li><a class="dropdown-item text-danger" (click)="onLogout()" style="cursor: pointer;">Logout</a></li>
                </ul>
              </li>
            </ng-container>

            <!-- If Guest: Show Login/Register -->
            <ng-template #guestLinks>
              <li class="nav-item">
                <a class="nav-link" routerLink="/login">Login</a>
              </li>
              <li class="nav-item ms-2">
                <a class="btn btn-sm btn-primary text-white px-3" routerLink="/register">Sign Up</a>
              </li>
            </ng-template>
          </ul>
        </div>

        <!-- Expanded Menu (shown when hamburger is clicked) -->
        <div class="expanded-menu" [class.show]="isMenuOpen">
          <div class="menu-content p-3">
            <!-- Search in expanded menu -->
            <div *ngIf="authService.isAuthenticated()" class="mb-3">
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

            <!-- Menu Items -->
            <ul class="list-unstyled mb-0">
              <li class="mb-2">
                <a routerLink="/" (click)="closeMenu()" class="text-white text-decoration-none d-block py-2" style="opacity: 0.8;">
                  📷 Gallery
                </a>
              </li>
              <li class="mb-2">
                <a routerLink="/albums" (click)="closeMenu()" class="text-white text-decoration-none d-block py-2" style="opacity: 0.8;">
                  💾 Albums
                </a>
              </li>
              <li class="mb-2">
                <button 
                  *ngIf="authService.isAuthenticated()"
                  (click)="startPresentationMode(); closeMenu();"
                  class="btn btn-outline-light text-white w-100 text-start py-2"
                  style="opacity: 0.8;">
                  🎬 Presentation Mode
                </button>
              </li>
            </ul>

            <!-- Tag Filter in expanded menu -->
            <div *ngIf="authService.isAuthenticated()" class="mt-3">
              <input 
                type="text" 
                class="form-control form-control-sm bg-secondary text-white border-0 tag-input"
                placeholder="#Tag filter..."
                [(ngModel)]="tagFilter"
                (keyup.enter)="onTagFilter()"
              >
            </div>
          </div>
        </div>
      </div>
    </nav>
  `,
  styles: [`
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

    /* Hamburger Menu Styles */
    .navbar-toggler {
      display: none;
    }

    @media (max-width: 991.98px) {
      .navbar-toggler {
        display: flex;
        align-items: center;
        justify-content: center;
      }

      .expanded-menu {
        max-height: 0;
        overflow: hidden;
        transition: max-height 0.3s ease-out;
        background-color: #212529;
      }

      .expanded-menu.show {
        max-height: 500px;
      }

      .menu-content {
        border-top: 1px solid rgba(255, 255, 255, 0.1);
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
