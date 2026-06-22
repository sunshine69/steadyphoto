import { Component, OnInit, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule } from '@angular/router';
import { ActivatedRoute, Router } from '@angular/router';
import { HttpClient, HttpParams } from '@angular/common/http';
import { PhotoService } from '../../services/photo.service';
import { environment } from '../../../environments/environment';

@Component({
  selector: 'app-public-share-album',
  standalone: true,
  imports: [CommonModule, RouterModule],
  template: `
    <div class="container mt-4">
      <!-- Password Protection Modal -->
      <div class="modal-overlay" *ngIf="showPasswordModal && !albumData">
        <div class="password-modal-content bg-dark border rounded p-4 text-center">
          <h3>🔒 This album is password protected</h3>
          <p class="text-muted mt-2 mb-3">Enter the password to view this content</p>
          <div class="row justify-content-center">
            <div class="col-md-6">
              <input 
                type="password" 
                [(ngModel)]="passwordInput" 
                (keyup.enter)="verifyPassword()"
                placeholder="Enter password..."
                class="form-control mb-3 text-dark"
              >
              <button (click)="verifyPassword()" class="btn btn-primary w-100">
                Unlock Album
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- Loading State -->
      <div *ngIf="!albumData && !showError" class="text-center py-5">
        <div class="spinner-border text-primary" role="status">
          <span class="visually-hidden">Loading...</span>
        </div>
        <p class="mt-2">Loading album content...</p>
      </div>

      <!-- Error State -->
      <div *ngIf="showError" class="text-center py-5">
        <h3>😕 Album Not Available</h3>
        <p class="text-muted">{{ errorMessage }}</p>
        <a routerLink="/" class="btn btn-primary mt-3">Back to Gallery</a>
      </div>

      <!-- Album Content -->
      <div *ngIf="albumData" class="row">
        <div class="col-md-8">
          <!-- Album Header -->
          <div class="d-flex justify-content-between align-items-center mb-4">
            <h2>{{ albumName }}</h2>
            <button (click)="goBack()" class="btn btn-primary">
              Back to Gallery
            </button>
          </div>

          <!-- Album Description -->
          <p class="text-muted" *ngIf="albumDescription">{{ albumDescription }}</p>

          <!-- Media Grid -->
          <div *ngIf="!loading && photos.length > 0">
            <div class="grid-container">
              <div class="grid-item position-relative" *ngFor="let photo of photos; let i = index">
                <app-photo-card [photo]="photo" (cardClick)="onPhotoClick(photo.id)"></app-photo-card>
              </div>
            </div>

            <!-- Pagination -->
            <div class="pagination-controls mt-4" *ngIf="totalAlbumPhotos > albumLimit">
              <button class="btn btn-outline-primary me-2" [disabled]="albumOffset === 0" (click)="changeAlbumPage(-1)">Previous</button>
              <div class="pagination-jump d-flex align-items-center mx-3">
                <span class="me-2 text-nowrap">Page</span>
                <input type="number" class="form-control form-control-sm jump-input me-2" [(ngModel)]="albumJumpInput" (keyup.enter)="onAlbumJumpToPage()" min="1" [max]="totalAlbumPages">
                <button class="btn btn-primary btn-sm jump-btn" type="button" (click)="onAlbumJumpToPage()">Go</button>
                <span class="ms-2 text-nowrap">of {{ totalAlbumPages }}</span>
              </div>
              <button class="btn btn-outline-primary ms-2" [disabled]="albumOffset + albumLimit >= totalAlbumPhotos" (click)="changeAlbumPage(1)">Next</button>
            </div>
          </div>

          <!-- Empty State -->
          <div *ngIf="!loading && photos.length === 0" class="empty-state py-5 text-center border rounded bg-light">
            <p class="lead text-muted">This album is empty.</p>
          </div>
        </div>
        
        <!-- Album Details Sidebar -->
        <div class="col-md-4">
          <div class="card shadow-sm sticky-top" style="top: 100px;">
            <div class="card-header bg-light">
              <h5 class="mb-0">Album Info</h5>
            </div>
            <ul class="list-group list-group-flush">
              <li class="list-group-item">
                <span class="text-muted">Photos:</span> {{ totalAlbumPhotos }}
              </li>
              <li class="list-group-item" *ngIf="albumData?.createdAt">
                <span class="text-muted">Created:</span> {{ albumData.createdAt | date:'fullDate' }}
              </li>
            </ul>
          </div>
        </div>
      </div>

      <!-- Loading Spinner for Media -->
      <div *ngIf="loading" class="loading-spinner d-flex justify-content-center my-5">
        <div class="spinner-border text-primary" role="status">
          <span class="visually-hidden">Loading...</span>
        </div>
      </div>
    </div>
  `,
  styles: [`
    .grid-container {
      display: grid;
      grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
      gap: 1.5rem;
      width: 100%;
    }
    .grid-item {
      position: relative;
      height: 100%;
    }
    
    /* Password Modal Styles */
    .modal-overlay {
      position: fixed;
      top: 0; left: 0; width: 100%; height: 100%;
      background-color: rgba(0, 0, 0, 0.85);
      display: flex; align-items: center; justify-content: center;
      z-index: 2000;
    }
    .password-modal-content {
      min-width: 400px; max-width: 90%;
    }
    
    /* Pagination Styles */
    .pagination-controls { 
      display: flex; 
      justify-content: center; 
      align-items: center; 
      margin-top: 2rem; 
      padding-bottom: 1rem; 
    }
    .jump-input { width: 60px !important; text-align: center; }
    
    .empty-state {
        margin-top: 2rem;
    }
    
    /* Loading Spinner */
    .loading-spinner {
      display: flex;
      justify-content: center;
      align-items: center;
    }
  `]
})
export class PublicShareAlbumComponent implements OnInit {
  private route = inject(ActivatedRoute);
  private router = inject(Router);
  private http = inject(HttpClient);
  private photoService = inject(PhotoService);

  albumData: any = null;
  albumName: string = 'Loading...';
  albumDescription: string | undefined;
  
  photos: any[] = [];
  loading = true;
  showError = false;
  errorMessage = '';

  // Password protection state
  showPasswordModal = false;
  passwordInput = '';

  // Pagination State
  totalAlbumPhotos = 0;
  albumLimit = 20;
  albumOffset = 0;
  albumJumpInput: number | null = null;

  get totalAlbumPages(): number { return Math.ceil(this.totalAlbumPhotos / this.albumLimit) || 1; }

  ngOnInit(): void {
    const token = this.route.snapshot.paramMap.get('token');
    if (token) {
      this.loadPublicShareAlbum(token);
    } else {
      this.showError = true;
      this.errorMessage = 'Invalid share link';
    }
  }

  private loadPublicShareAlbum(token: string): void {
    // Try to fetch without password first, if it fails with 403 try with password modal
    this.http.get<any>(`${environment.apiBaseUrl}/public/shares/album/${token}`).subscribe({
      next: (response) => {
        console.log('=== PUBLIC SHARE DEBUG ===');
        console.log('Full response:', JSON.stringify(response, null, 2));
        // Backend returns SharedAlbumWithMedia directly (not wrapped in Album)
        const albumData = response;
        console.log('Album data:', JSON.stringify(albumData, null, 2));
        this.albumData = albumData;
        this.albumName = albumData.name || 'Untitled Album';
        this.albumDescription = albumData.description;
        
        // Load album media if present in the response
        const mediaItems = albumData.media_items || [];
        console.log('Media items from album data:', mediaItems);
        console.log('Media items type:', typeof mediaItems, Array.isArray(mediaItems));
        console.log('Media items length:', mediaItems.length);
        if (Array.isArray(mediaItems) && mediaItems.length > 0) {
          this.photos = mediaItems.map((p: any) => this.normalizePhoto(p));
          this.totalAlbumPhotos = albumData.totalItems || mediaItems.length;
          this.loading = false;
          console.log('Loaded photos from album data:', this.photos);
          console.log('Photos length:', this.photos.length);
        } else {
          // Fetch media via pagination endpoint if not in response
          console.log('No media items found, calling fetchMedia');
          this.fetchMedia(token);
        }
      },
      error: (err: any) => {
        if (err.status === 403 && !this.showPasswordModal) {
          // Password required - show password modal
          this.showPasswordModal = true;
          return;
        } else if (err.status === 410 || err.status === 404) {
          // Link expired or not found
          this.showError = true;
          const errorData = err.error as any;
          this.errorMessage = errorData?.error || 'Share link not found or has expired';
        } else {
          console.error('Error loading public share album:', err);
          this.showError = true;
          this.errorMessage = 'Failed to load album content';
        }
      }
    });
  }

  private verifyPassword(): void {
    const token = this.route.snapshot.paramMap.get('token');
    if (!token || !this.passwordInput) return;
    
    // Try with password parameter
    const params = new HttpParams().set('password', this.passwordInput);
    
    this.http.get<any>(`${environment.apiBaseUrl}/public/shares/album/${token}`, { 
      params: params,
      headers: { 'Accept': 'application/json' }
    }).subscribe({
      next: (response) => {
        // Backend returns SharedAlbumWithMedia directly (not wrapped in Album)
        const albumData = response;
        this.albumData = albumData;
        this.albumName = albumData.name || 'Untitled Album';
        this.albumDescription = albumData.description;
        
        // Load album media if present in the response
        if (albumData.media_items && Array.isArray(albumData.media_items)) {
          this.photos = albumData.media_items.map((p: any) => this.normalizePhoto(p));
          this.totalAlbumPhotos = albumData.totalItems || albumData.media_items.length;
          this.loading = false;
        } else {
          // Fetch media via pagination endpoint if not in response
          this.fetchMedia(token);
        }
        
        this.showPasswordModal = false;
        this.passwordInput = '';
      },
      error: (err: any) => {
        if (err.status === 403) {
          // Wrong password - show error and clear input
          alert('Incorrect password. Please try again.');
          this.passwordInput = '';
        } else {
          console.error('Error with password:', err);
          this.showError = true;
          this.errorMessage = 'Failed to access shared content';
        }
      }
    });
  }

  private fetchMedia(token: string): void {
    if (!token) return;
    
    this.loading = true;
    const url = `${environment.apiBaseUrl}/public/shares/album/${token}/media?limit=${this.albumLimit}&offset=${this.albumOffset}`;
    
    this.http.get<any>(url).subscribe({
      next: (response) => {
        const rawPhotos: any[] = response.media || [];
        const totalItems = response.totalItems || 0;
        
        this.photos = rawPhotos.map((p: any) => this.normalizePhoto(p));
        this.totalAlbumPhotos = totalItems;
        this.albumOffset = 0;
        this.loading = false;
      },
      error: (err: any) => {
        console.error('Error loading album media via public endpoint', err);
        this.loading = false;
      }
    });
  }

  private normalizePhoto(p: any): any {
    const id = p.ID ?? p.id ?? '';
    const path = p.Path ?? p.path ?? '';

    let mediaType: string | undefined;
    const rawMediaType = p.MediaType || p.mediaType || p.media_type;
    if (rawMediaType) {
      const mt = String(rawMediaType).toLowerCase();
      mediaType = mt === 'video' ? 'video' : 'photo';
    }

    return {
      id: id,
      path: path,
      thumbnailUrl: this.getThumbnailUrl(id, path),
      filename: p.Filename ?? p.filename ?? '',
      captured_at: p.CapturedAt ?? p.captured_at ?? '',
      width: p.Width ?? p.width,
      height: p.Height ?? p.height,
      size: p.Size ?? p.size,
      type: p.Type ?? p.type,
      mediaType: mediaType || 'photo',
      metadata: p.Metadata ? {
        camera: p.Metadata.Camera,
        iso: p.Metadata.Iso,
        aperture: p.Metadata.Aperture,
        focal_length: p.Metadata.FocalLength,
        gps_lat: p.Metadata.GpsLat,
        gps_lon: p.Metadata.GpsLon,
      } : undefined,
      videoMetadata: p.VideoMetadata ? {
        duration: p.VideoMetadata.Duration ?? p.VideoMetadata.duration,
        bitrate: p.VideoMetadata.Bitrate ?? p.VideoMetadata.bitrate,
        video_codec: p.VideoMetadata.VideoCodec ?? p.VideoMetadata.video_codec,
        audio_codec: p.VideoMetadata.AudioCodec ?? p.VideoMetadata.audio_codec,
        frame_rate: p.VideoMetadata.FrameRate ?? p.VideoMetadata.frame_rate,
      } : undefined,
    };
  }

  getThumbnailUrl(id: string, path: string): string {
    if (!id || !path) return '';
    const token = this.route.snapshot.paramMap.get('token');
    if (!token) return '';
    // Use the album media thumbnail endpoint with the share token and media path
    return `${environment.apiBaseUrl}/public/shares/album/${token}/media/thumb?path=${encodeURIComponent(path)}`;
  }

  changeAlbumPage(dir: number): void {
    const newOffset = this.albumOffset + (dir * this.albumLimit);
    
    if (newOffset < 0 || newOffset >= this.totalAlbumPhotos) return;
    
    // For public shares, use the album media endpoint
    const token = this.route.snapshot.paramMap.get('token');
    if (!token) return;
    
    const url = `${environment.apiBaseUrl}/public/shares/album/${token}/media?limit=${this.albumLimit}&offset=${newOffset}`;
    
    this.loading = true;
    this.http.get<any>(url).subscribe({
      next: (response) => {
        const rawPhotos: any[] = response.media || [];
        
        this.photos = rawPhotos.map((p: any) => this.normalizePhoto(p));
        this.albumOffset = newOffset;
        this.loading = false;
      },
      error: (err: any) => {
        console.error('Error loading album media', err);
        this.loading = false;
      }
    });
  }

  onAlbumJumpToPage(): void { 
    if (this.albumJumpInput && this.albumJumpInput >= 1 && this.albumJumpInput <= this.totalAlbumPages) {
      const targetOffset = (this.albumJumpInput - 1) * this.albumLimit;
      
      const token = this.route.snapshot.paramMap.get('token');
      if (!token) return;
      
      const url = `${environment.apiBaseUrl}/public/shares/album/${token}/media?limit=${this.albumLimit}&offset=${targetOffset}`;
      
      this.loading = true;
      this.http.get<any>(url).subscribe({
        next: (response) => {
          const rawPhotos: any[] = response.media || [];
          
          this.photos = rawPhotos.map((p: any) => this.normalizePhoto(p));
          this.albumOffset = targetOffset;
          this.loading = false;
        },
        error: (err: any) => {
          console.error('Error loading album media', err);
          this.loading = false;
        }
      });
    } 
  }

  onPhotoClick(id: string): void { 
    const ids = this.photos.map(p => p.id).join(',');
    this.router.navigate(['/photos', id], { queryParams: { albumIds: ids, source: 'shared' } }); 
  }

  goBack(): void { 
    window.history.back(); 
  }
}
