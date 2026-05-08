import { Component, Inject, Optional } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router, RouterModule } from '@angular/router';
import { PhotoService } from './services/photo.service';
import { PhotoListComponent } from './components/photo-list/photo-list.component';
import { NavbarComponent } from './components/navbar/navbar.component';

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [CommonModule, RouterModule, PhotoListComponent, NavbarComponent],
  template: `
    <!-- Navigation Bar -->
    <app-navbar></app-navbar>

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
