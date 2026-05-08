import { Component, OnInit, OnDestroy, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router, RouterModule } from '@angular/router';
import { Subscription } from 'rxjs';
import { PhotoService } from '../../services/photo.service';
import { Photo } from '../../models/photo.model';

@Component({
  selector: 'app-photo-detail',
  standalone: true,
  imports: [CommonModule, RouterModule],
  template: `
    <div class="container mt-4">
      <div class="row">
        <div class="col-md-8">
          <div class="photo-detail-container" *ngIf="photo; else loading">
            <div class="image-viewer-wrapper">
              <img 
                [src]="photo.path" 
                [alt]="photo.filename" 
                class="main-image rounded shadow"
              >
            </div>
            
            <div class="mt-3 d-flex justify-content-between align-items-start">
              <div>
                <h3 class="mb-1">{{ photo.filename }}</h3>
                <p class="text-muted mb-0">Captured: {{ photo.captured_at | date:'medium' }}</p>
              </div>
              <div class="btn-group">
                <a [href]="photo.path" target="_blank" class="btn btn-outline-secondary">
                  <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="me-1"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" y1="14" x2="21" y2="3"/></svg>
                  Full Size
                </a>
                <button (click)="goBack()" class="btn btn-primary ms-2">
                  Back to Gallery
                </button>
              </div>
            </div>
          </div>
          
          <ng-template #loading>
            <div class="text-center py-5">
              <div class="spinner-border text-primary" role="status">
                <span class="visually-hidden">Loading...</span>
              </div>
              <p class="mt-2">Loading photo details...</p>
            </div>
          </ng-template>
        </div>
        <div class="col-md-4">
          <div class="card shadow-sm sticky-top" style="top: 100px;">
            <div class="card-header bg-light">
              <h5 class="mb-0">Photo Details</h5>
            </div>
            <ul class="list-group list-group-flush">
              <li class="list-group-item d-flex justify-content-between align-items-center">
                <span class="text-muted">Filename</span>
                <span class="text-end small text-break ms-2">{{ photo?.filename }}</span>
              </li>
              <li class="list-group-item d-flex justify-content-between align-items-center">
                <span class="text-muted">Path</span>
                <span class="text-end small text-break ms-2">{{ photo?.path }}</span>
              </li>
              <li class="list-group-item d-flex justify-content-between align-items-center">
                <span class="text-muted">Captured</span>
                <span>{{ photo?.captured_at | date:'fullDate' }}</span>
              </li>
              <li class="list-group-item d-flex justify-content-between align-items-center">
                <span class="text-muted">Dimensions</span>
                <span>{{ photo?.width }} x {{ photo?.height }}</span>
              </li>
              <li class="list-group-item d-flex justify-content-between align-items-center">
                <span class="text-muted">Type</span>
                <span>{{ photo?.type || 'Unknown' }}</span>
              </li>
              <li class="list-group-item d-flex justify-content-between align-items-center">
                <span class="text-muted">Size</span>
                <span>{{ (photo?.size || 0) / 1024 | number:'1.0-2' }} KB</span>
              </li>
            </ul>
          </div>
        </div>
      </div>
    </div>
  `,
  styles: [`
    .image-viewer-wrapper {
      width: 100%;
      display: flex;
      justify-content: center;
      align-items: center;
      background-color: #f8f9fa;
      border-radius: 8px;
      overflow: hidden;
      min-height: 300px;
    }
    .main-image {
      max-width: 100%;
      max-height: 75vh; /* Keeps the image within the viewport height */
      object-fit: contain; /* Ensures the whole image is visible without cropping */
      display: block;
    }
  `]
})
export class PhotoDetailComponent implements OnInit, OnDestroy {
  photo: Photo | null = null;
  private route = inject(ActivatedRoute);
  private router = inject(Router);
  private photoService = inject(PhotoService);
  private subscription?: Subscription;

  ngOnInit(): void {
    const id = this.route.snapshot.paramMap.get('id');
    if (id) {
      this.subscription = this.photoService.getPhoto(id).subscribe({
        next: (photo) => {
          this.photo = photo;
        },
        error: (err) => {
          console.error('Error fetching photo', err);
          this.router.navigate(['/']);
        }
      });
    } else {
      this.router.navigate(['/']);
    }
  }

  ngOnDestroy(): void {
    this.subscription?.unsubscribe();
  }

  goBack(): void {
    window.history.back();
  }
}
