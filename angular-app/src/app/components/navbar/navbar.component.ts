import { Component, Inject, Optional } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router, RouterModule } from '@angular/router';
import { SearchService } from '../../services/search.service';

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
            <path d="M12 2v4"/><path d="m17 9-5-5-5 5"/><path d="m17 15-5 5-5-5"/>
          </svg>
          SteadyPhoto
        </a>

        <!-- Search Bar (Middle) -->
        <div class="flex-grow-1 d-flex justify-content-center mx-4">
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

        <!-- Right Side Actions -->
        <div class="navbar-nav ms-auto">
          <ul class="navbar-nav">
            <li class="nav-item">
              <a class="nav-link" routerLink="/">Home</a>
            </li>
          </ul>
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
  `]
})
export class NavbarComponent {
  searchTerm = '';

  constructor(
    private searchService: SearchService,
    private router: Router
  ) {}

  onSearch(term: string): void {
    this.searchTerm = term;
    this.searchService.setSearchTerm(term);
    
    if (this.router.url !== '/') {
      this.router.navigate(['/']);
    }
  }
}
