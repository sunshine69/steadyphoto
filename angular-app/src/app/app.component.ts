import { Component, OnInit, OnDestroy, Inject, Optional } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router, RouterModule } from '@angular/router';
import { PhotoService } from './services/photo.service';
import { PhotoListComponent } from './components/photo-list/photo-list.component';

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [CommonModule, RouterModule, PhotoListComponent],
  template: `
    <!-- Navigation Bar -->
    <nav class="navbar navbar-expand-lg navbar-dark bg-dark sticky-top">
      <div class="container">
        <a class="navbar-brand" routerLink="/">
          <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 2v4"/><path d="m17 9-5-5-5 5"/><path d="m17 15-5 5-5-5"/></svg>
          SteadyPhoto
        </a>
        <div class="collapse navbar-collapse" id="navbarNav">
          <ul class="navbar-nav ms-auto">
            <li class="nav-item">
              <a class="nav-link" routerLink="/">Home</a>
            </li>
          </ul>
        </div>
      </div>
    </nav>

    <!-- Main Content -->
    <main class="container py-4">
      <router-outlet></router-outlet>
    </main>

    <footer class="bg-dark text-white py-3 mt-5">
      <div class="container text-center">
        <p class="mb-0">© 2024 SteadyPhoto. All rights reserved.</p>
      </div>
    </footer>
  `
})
export class AppComponent {
  constructor(
    @Optional() @Inject(PhotoService) private photoService?: PhotoService
  ) {
    if (this.photoService) {
      console.log('PhotoService injected successfully');
    } else {
      console.warn('PhotoService NOT injected. Check your providers in main.ts');
    }
  }
}
