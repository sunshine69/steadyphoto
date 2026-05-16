import { Component, OnInit, OnDestroy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router, RouterModule } from '@angular/router';
import { PhotoService } from './services/photo.service';
import { PhotoListComponent } from './components/photo-list/photo-list.component';
import { NavbarComponent } from './components/navbar/navbar.component';
import { PresentationModeComponent } from './components/presentation-mode/presentation-mode.component';
import { PresentationService, MediaItem } from './services/presentation.service';

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [CommonModule, RouterModule, PhotoListComponent, NavbarComponent, PresentationModeComponent],
  template: `
    <!-- Navigation Bar -->
    <app-navbar></app-navbar>

    <!-- Main Content -->
    <main class="container py-4">
      <router-outlet></router-outlet>
    </main>

    <!-- Presentation Mode Overlay (shown when presentation service is open) -->
    <app-presentation-mode 
      *ngIf="presentationService.isOpen$ | async"
      [items]="presentationService.getItems()"
      [startIndex]="presentationService.getCurrentIndex()">
    </app-presentation-mode>

    <footer class="bg-dark text-white py-3 mt-5">
      <div class="container text-center">
        <p class="mb-0">© 2024 SteadyPhoto. All rights reserved.</p>
      </div>
    </footer>
  `
})
export class AppComponent implements OnInit, OnDestroy {
  constructor(
    public presentationService: PresentationService,
    private photoService?: PhotoService
  ) {
    if (this.photoService) {
      console.log('PhotoService injected successfully');
    } else {
      console.warn('PhotoService NOT injected. Check your providers in main.ts');
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
