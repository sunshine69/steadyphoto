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
            <!-- Video Player for videos -->
            <div class="video-viewer-wrapper" *ngIf="isVideo()">
              <video 
                [attr.src]="photo.path"
                controls
                preload="metadata"
                class="main-video rounded shadow w-100"
                (error)="onVideoError($event)"
              >
                Your browser does not support the video tag.
              </video>
            </div>
            
            <!-- Image display for photos -->
            <div class="image-viewer-wrapper" *ngIf="!isVideo()">
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
                <a [href]="photo.path" download="{{ photo.filename }}" class="btn btn-outline-secondary">
                  <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="me-1"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
                  Download
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
              <p class="mt-2">Loading details...</p>
            </div>
          </ng-template>
        </div>
        <div class="col-md-4">
          <div class="card shadow-sm sticky-top" style="top: 100px;">
            <div class="card-header bg-light">
              <h5 class="mb-0">Details</h5>
            </div>
            <ul class="list-group list-group-flush">
              <li class="list-group-item d-flex justify-content-between align-items-center">
                <span class="text-muted">Filename</span>
                <span class="text-end small text-break ms-2">{{ photo?.filename }}</span>
              </li>
              <li class="list-group-item d-flex justify-content-between align-items-center">
                <span class="text-muted">Type</span>
                <span>{{ isVideo() ? '🎥 Video' : (photo?.type || 'Photo') }}</span>
              </li>
              <li class="list-group-item d-flex justify-content-between align-items-center" *ngIf="isVideo()">
                <span class="text-muted">Duration</span>
                <span>{{ formatDuration(photo?.videoMetadata?.duration) }}</span>
              </li>
              <li class="list-group-item d-flex justify-content-between align-items-center">
                <span class="text-muted">Captured</span>
                <span>{{ photo?.captured_at | date:'fullDate' }}</span>
              </li>
              <li class="list-group-item d-flex justify-content-between align-items-center" *ngIf="photo?.width || photo?.height">
                <span class="text-muted">Dimensions</span>
                <span>{{ photo?.width }} x {{ photo?.height }}</span>
              </li>
              <li class="list-group-item d-flex justify-content-between align-items-center" *ngIf="isVideo()">
                <span class="text-muted">Video Codec</span>
                <span>{{ photo?.videoMetadata?.video_codec || 'Unknown' }}</span>
              </li>
              <li class="list-group-item d-flex justify-content-between align-items-center" *ngIf="isVideo()">
                <span class="text-muted">Audio Codec</span>
                <span>{{ photo?.videoMetadata?.audio_codec || 'Unknown' }}</span>
              </li>
              <li class="list-group-item d-flex justify-content-between align-items-center" *ngIf="isVideo() && (photo?.videoMetadata?.frame_rate ?? 0) > 0">
                <span class="text-muted">Frame Rate</span>
                <span>{{ photo?.videoMetadata?.frame_rate }} fps</span>
              </li>
              <li class="list-group-item d-flex justify-content-between align-items-center" *ngIf="photo?.size">
                <span class="text-muted">Size</span>
                <span>{{ formatFileSize(photo.size) }}</span>
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
    .video-viewer-wrapper {
      width: 100%;
      background-color: #000;
      border-radius: 8px;
      overflow: hidden;
      min-height: 300px;
    }
    .main-video {
      max-width: 100%;
      max-height: 75vh; /* Keeps the video within the viewport height */
      display: block;
      background-color: #000;
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
      // Use getMedia instead of getPhoto since it works for both photos and videos
      this.subscription = this.photoService.getMedia(id).subscribe({
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

  isVideo(): boolean {
    return this.photo?.mediaType === 'video';
  }

  formatDuration(seconds?: number): string {
    if (!seconds || seconds <= 0) return 'Unknown';
    const hrs = Math.floor(seconds / 3600);
    const mins = Math.floor((seconds % 3600) / 60);
    const secs = Math.floor(seconds % 60);
    
    if (hrs > 0) {
      return `${hrs}:${mins.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`;
    }
    return `${mins}:${secs.toString().padStart(2, '0')}`;
  }

  formatFileSize(bytes?: number): string {
    if (!bytes) return 'Unknown';
    const kb = bytes / 1024;
    const mb = kb / 1024;
    
    if (mb >= 1) {
      return `${mb.toFixed(2)} MB`;
    }
    return `${kb.toFixed(2)} KB`;
  }

  onVideoError(event: Event): void {
    console.error('Video playback error:', event);
  }

  goBack(): void {
    window.history.back();
  }
}
