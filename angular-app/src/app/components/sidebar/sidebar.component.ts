import { Component, OnInit, OnDestroy, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule, RouterLinkActive } from '@angular/router';
import { AuthService } from '../../services/auth.service';
import { Router } from '@angular/router';
import { Subscription } from 'rxjs';

@Component({
  selector: 'app-sidebar',
  standalone: true,
  imports: [CommonModule, RouterModule],
  template: `
    <div class="sidebar">
      <!-- Logo Section -->
      <div class="sidebar-header">
        <div class="logo-container">
          <svg width="32" height="32" viewBox="0 0 100 100" fill="none" xmlns="http://www.w3.org/2000/svg">
            <circle cx="50" cy="50" r="48" fill="#6366F1"/>
            <path d="M50 20C33.431 20 20 33.431 20 50s13.431 30 30 30 30-13.431 30-30S66.569 20 50 20zm0 8c13.255 0 24 10.745 24 24S63.255 74 50 74s-24-10.745-24-24S36.745 28 50 28z" fill="white"/>
            <circle cx="50" cy="50" r="12" fill="#A5B4FC"/>
          </svg>
          <span class="logo-text">SteadyPhoto</span>
        </div>
      </div>

      <!-- Main Navigation -->
      <nav class="sidebar-nav">
        <a routerLink="/" routerLinkActive="active" [routerLinkActiveOptions]="{exact: true}" class="nav-item">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>
            <circle cx="8.5" cy="8.5" r="1.5"/>
            <polyline points="21 15 16 10 5 21"/>
          </svg>
          <span>Photos</span>
        </a>

        <a routerLink="/explore" routerLinkActive="active" class="nav-item">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="11" cy="11" r="8"/>
            <line x1="21" y1="21" x2="16.65" y2="16.65"/>
          </svg>
          <span>Explore</span>
        </a>

        <a routerLink="/map" routerLinkActive="active" class="nav-item">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polygon points="1 6 1 22 8 18 16 22 23 18 23 2 16 6 8 2 1 6"/>
            <line x1="8" y1="2" x2="8" y2="18"/>
            <line x1="16" y1="6" x2="16" y2="22"/>
          </svg>
          <span>Map</span>
        </a>

        <a routerLink="/sharing" routerLinkActive="active" class="nav-item">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/>
            <circle cx="9" cy="7" r="4"/>
            <path d="M23 21v-2a4 4 0 0 0-3-3.87"/>
            <path d="M16 3.13a4 4 0 0 1 0 7.75"/>
          </svg>
          <span>Sharing</span>
        </a>

        <!-- Library Section -->
        <div class="nav-section">
          <h3 class="section-title">LIBRARY</h3>
          
          <a routerLink="/favorites" routerLinkActive="active" class="nav-item">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z"/>
            </svg>
            <span>Favorites</span>
          </a>

          <a routerLink="/albums" routerLinkActive="active" class="nav-item">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M19 3H5a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2V5a2 2 0 0 0-2-2z"/>
              <rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>
            </svg>
            <span>Albums</span>
          </a>

          <div class="nav-subsection">
            <a routerLink="/camera" routerLinkActive="active" class="nav-item sub-item">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M23 19a2 2 0 0 1-2 2H3a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h4l2-3h6l2 3h4a2 2 0 0 1 2 2z"/>
                <circle cx="12" cy="13" r="4"/>
              </svg>
              <span>Camera</span>
            </a>

            <a routerLink="/screenshots" routerLinkActive="active" class="nav-item sub-item">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <rect x="2" y="3" width="20" height="14" rx="2" ry="2"/>
                <line x1="8" y1="21" x2="16" y2="21"/>
                <line x1="12" y1="17" x2="12" y2="21"/>
              </svg>
              <span>Screenshots</span>
            </a>

            <a routerLink="/messenger" routerLinkActive="active" class="nav-item sub-item">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/>
              </svg>
              <span>Messenger</span>
            </a>
          </div>

          <a routerLink="/utilities" routerLinkActive="active" class="nav-item">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="3"/>
              <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/>
            </svg>
            <span>Utilities</span>
          </a>

          <a routerLink="/archive" routerLinkActive="active" class="nav-item">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="21 8 21 21 3 21 3 8"/>
              <rect x="1" y="3" width="22" height="5"/>
              <line x1="10" y1="12" x2="14" y2="12"/>
            </svg>
            <span>Archive</span>
          </a>

          <a routerLink="/locked" routerLinkActive="active" class="nav-item">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect x="3" y="11" width="18" height="11" rx="2" ry="2"/>
              <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
            </svg>
            <span>Locked Folder</span>
          </a>

          <a routerLink="/trash" routerLinkActive="active" class="nav-item">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="3 6 5 6 21 6"/>
              <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
            </svg>
            <span>Trash</span>
          </a>
        </div>
      </nav>

      <!-- User Profile Section -->
      <div class="sidebar-footer">
        <div class="user-profile" (click)="toggleMenu()" [class.open]="isMenuOpen">
          <div class="avatar">{{ emailInitial }}</div>
          <div class="user-info">
            <span class="username">{{ emailUsername }}</span>
            <span class="storage-info">2.3 GB used</span>
          </div>
        </div>

        <!-- Dropdown Menu -->
        <div class="dropdown-menu" *ngIf="isMenuOpen" (click)="$event.stopPropagation()">
          <ul class="menu-list">
            <li class="menu-item" (click)="onSettings()">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="12" cy="12" r="3"/>
                <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/>
              </svg>
              <span>Settings</span>
            </li>
            <li class="menu-item" (click)="onLogout()">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/>
                <polyline points="16 17 21 12 16 7"/>
                <line x1="21" y1="12" x2="9" y2="12"/>
              </svg>
              <span>Logout</span>
            </li>
          </ul>
        </div>
      </div>
    </div>
  `,
  styles: [`
    .sidebar {
      position: fixed;
      left: 0;
      top: 0;
      bottom: 0;
      width: 260px;
      background-color: #1a1b2e;
      color: #9ca3af;
      display: flex;
      flex-direction: column;
      z-index: 100;
      overflow-y: auto;
    }

    .sidebar::-webkit-scrollbar {
      width: 4px;
    }

    .sidebar::-webkit-scrollbar-track {
      background: transparent;
    }

    .sidebar::-webkit-scrollbar-thumb {
      background-color: #374151;
      border-radius: 20px;
    }

    /* Logo Section */
    .sidebar-header {
      padding: 20px;
      border-bottom: 1px solid #2d3748;
    }

    .logo-container {
      display: flex;
      align-items: center;
      gap: 12px;
    }

    .logo-text {
      font-size: 20px;
      font-weight: bold;
      color: #e5e7eb;
    }

    /* Navigation */
    .sidebar-nav {
      flex: 1;
      padding: 16px 8px;
    }

    .nav-section {
      margin-top: 24px;
    }

    .section-title {
      font-size: 11px;
      font-weight: 700;
      text-transform: uppercase;
      letter-spacing: 0.5px;
      color: #6b7280;
      padding: 0 12px;
      margin-bottom: 8px;
    }

    .nav-item {
      display: flex;
      align-items: center;
      gap: 12px;
      padding: 10px 12px;
      border-radius: 8px;
      color: #9ca3af;
      text-decoration: none;
      font-size: 14px;
      transition: all 0.2s ease;
      margin-bottom: 2px;
    }

    .nav-item:hover {
      background-color: rgba(99, 102, 241, 0.1);
      color: #e5e7eb;
    }

    .nav-item.active {
      background-color: rgba(99, 102, 241, 0.2);
      color: #818cf8;
    }

    .nav-item svg {
      flex-shrink: 0;
    }

    /* Subsection */
    .nav-subsection {
      margin-left: 16px;
      padding-left: 12px;
      border-left: 1px solid #374151;
    }

    .sub-item {
      font-size: 13px;
    }

    /* Footer */
    .sidebar-footer {
      padding: 16px;
      border-top: 1px solid #2d3748;
      position: relative;
    }

    .user-profile {
      display: flex;
      align-items: center;
      gap: 12px;
      cursor: pointer;
      padding: 8px;
      border-radius: 8px;
      transition: background-color 0.2s ease;
    }

    .user-profile:hover {
      background-color: rgba(99, 102, 241, 0.1);
    }

    .avatar {
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
    }

    .user-info {
      display: flex;
      flex-direction: column;
      gap: 2px;
    }

    .username {
      font-size: 14px;
      color: #e5e7eb;
      font-weight: 500;
    }

    .storage-info {
      font-size: 12px;
      color: #6b7280;
    }

    /* Dropdown Menu */
    .dropdown-menu {
      position: absolute;
      bottom: 100%;
      left: 16px;
      right: 16px;
      background-color: #1e293b;
      border: 1px solid #374151;
      border-radius: 8px;
      box-shadow: 0 -4px 12px rgba(0, 0, 0, 0.3);
      margin-bottom: 8px;
      overflow: hidden;
    }

    .menu-list {
      list-style: none;
      padding: 8px 0;
      margin: 0;
    }

    .menu-item {
      display: flex;
      align-items: center;
      gap: 12px;
      padding: 10px 16px;
      color: #9ca3af;
      cursor: pointer;
      transition: all 0.2s ease;
      font-size: 14px;
    }

    .menu-item:hover {
      background-color: rgba(99, 102, 241, 0.1);
      color: #e5e7eb;
    }

    .menu-item svg {
      flex-shrink: 0;
    }
  `]
})
export class SidebarComponent implements OnInit, OnDestroy {
  private authService = inject(AuthService);
  private router = inject(Router);
  private subscription?: Subscription;

  emailInitial = 'U';
  emailUsername = 'User';
  isMenuOpen = false;

  ngOnInit(): void {
    // Subscribe to auth state changes to update display values reactively
    this.subscription = this.authService.isAuthenticated$.subscribe(isAuth => {
      if (isAuth) {
        // User is logged in - get fresh values from localStorage
        this.emailInitial = this.authService.getUsername();
        this.emailUsername = this.authService.getEmailUsername();
      } else {
        // User is logged out - reset to defaults
        this.emailInitial = 'U';
        this.emailUsername = 'User';
      }
    });

    // Close menu when clicking outside
    document.addEventListener('click', this.closeMenuOnOutsideClick);
  }

  ngOnDestroy(): void {
    document.removeEventListener('click', this.closeMenuOnOutsideClick);
    this.subscription?.unsubscribe();
  }

  private closeMenuOnOutsideClick = (event: MouseEvent): void => {
    const sidebar = document.querySelector('.sidebar-footer');
    if (sidebar && !sidebar.contains(event.target as Node)) {
      this.isMenuOpen = false;
    }
  };

  toggleMenu(): void {
    this.isMenuOpen = !this.isMenuOpen;
  }

  onSettings(): void {
    console.log('Settings clicked');
    // TODO: Implement settings page navigation later
    this.isMenuOpen = false;
  }

  onLogout(): void {
    console.log('Logout clicked');
    this.authService.logout().subscribe({
      next: () => {
        this.authService.clearUser();
        localStorage.removeItem('access_token');
        localStorage.removeItem('refresh_token');
        this.router.navigate(['/login']);
      },
      error: (err) => {
        console.error('Logout failed', err);
        // Still clear local state even if server fails
        this.authService.clearUser();
        localStorage.removeItem('access_token');
        localStorage.removeItem('refresh_token');
        this.router.navigate(['/login']);
      }
    });
  }
}
