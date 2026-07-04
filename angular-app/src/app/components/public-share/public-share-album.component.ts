import { Component, OnInit, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule } from '@angular/router';
import { ActivatedRoute, Router } from '@angular/router';
import { HttpClient, HttpParams } from '@angular/common/http';
import { FormsModule } from '@angular/forms';
import { PhotoService } from '../../services/photo.service';
import { environment } from '../../../environments/environment';
import { PhotoCardComponent } from '../photo-card/photo-card.component';

@Component({
  selector: 'app-public-share-album',
  standalone: true,
  imports: [CommonModule, RouterModule, FormsModule, PhotoCardComponent],
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
      <div *ngIf="!albumData && !showError && !showPasswordModal" class="text-center py-5">
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
      <div *ngIf="loading && !showPasswordModal && !showError" class="loading-spinner d-flex justify-content-center my-5">
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
    console.log('=== PUBLIC SHARE DEBUG: ngOnInit ===');
    console.log('Route token:', token);
    console.log('Full route snapshot:', this.route.snapshot);
    console.log('sessionStorage keys:', Object.keys(sessionStorage));
    if (token) {
      const storedPassword = sessionStorage.getItem(`share_password_${token}`);
      console.log('Stored password in sessionStorage:', storedPassword ? 'YES (exists)' : 'NO (null)');
      console.log('SessionStorage share_password:', sessionStorage.getItem(`share_password_${token}`));
      console.log('All sessionStorage items:');
      for (let i = 0; i < sessionStorage.length; i++) {
        const key = sessionStorage.key(i);
        const value = sessionStorage.getItem(key ?? '');
        console.log(`  ${key}: ${value}`);
      }
      console.log('=== CALLING loadPublicShareAlbum ===');
      this.loadPublicShareAlbum(token);
    } else {
      console.log('=== NO TOKEN FOUND - SHOWING ERROR ===');
      this.showError = true;
      this.errorMessage = 'Invalid share link';
    }
  }

  private loadPublicShareAlbum(token: string): void {
    // Check for password in sessionStorage - include it in the initial request if available
    const password = sessionStorage.getItem(`share_password_${token}`);
    let params: any = undefined;
    if (password) {
      params = new HttpParams().set('password', password);
    }
    
    this.http.get<any>(`${environment.apiBaseUrl}/public/shares/album/${token}`, { 
      params: params 
    }).subscribe({
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
        console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
        console.log('❌ [DEBUG] loadPublicShareAlbum ERROR');
        console.log('   Token:', token);
        console.log('   Status:', err.status);
        console.log('   Error status === 403:', err.status === 403);
        console.log('   showPasswordModal:', this.showPasswordModal);
        console.log('   Password in sessionStorage:', sessionStorage.getItem(`share_password_${token}`));
        console.log('   Error response:', err.error);
        console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
        if (err.status === 403 && !this.showPasswordModal) {
          // Password required - show password modal
          // But first check if we already have a password in sessionStorage
          const storedPassword = sessionStorage.getItem(`share_password_${token}`);
          console.log('   → Error status 403, showPasswordModal:', this.showPasswordModal);
          console.log('   → storedPassword:', storedPassword ? 'YES (exists, value: ***' + storedPassword.substring(Math.max(0, storedPassword.length-4)) + '...)' : 'NO (null)');
          if (storedPassword) {
            // We have a password but it's wrong - show error and clear it
            this.showError = true;
            this.errorMessage = 'Incorrect password. Please try again.';
          } else {
            // No password stored yet, show the password modal
            this.showPasswordModal = true;
          }
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
    console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
    console.log('🔓 [DEBUG] verifyPassword() called');
    const token = this.route.snapshot.paramMap.get('token');
    console.log('   Token:', token);
    console.log('   Password length:', this.passwordInput?.length);
    if (!token || !this.passwordInput) {
      console.log('   Early return: token or password missing');
      console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
      return;
    }
    
    // Try with password parameter
    const params = new HttpParams().set('password', this.passwordInput);
    console.log('   Request params:', params.toString());
    console.log('   Request URL:', `${environment.apiBaseUrl}/public/shares/album/${token}?${params.toString()}`);
    console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
    
    this.http.get<any>(`${environment.apiBaseUrl}/public/shares/album/${token}`, { 
      params: params,
      headers: { 'Accept': 'application/json' }
    }).subscribe({
      next: (response) => {
        console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
        console.log('🟢 [DEBUG] verifyPassword() SUCCESS');
        console.log('   Response:', JSON.stringify(response, null, 2));
        console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
        // Backend returns SharedAlbumWithMedia directly (not wrapped in Album)
        const albumData = response;
        this.albumData = albumData;
        this.albumName = albumData.name || 'Untitled Album';
        this.albumDescription = albumData.description;
        
        // Store password in sessionStorage for subsequent requests
        sessionStorage.setItem(`share_password_${token}`, this.passwordInput);
        console.log('   Password stored in sessionStorage');
        
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
        console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
        console.log('❌ [DEBUG] verifyPassword() ERROR');
        console.log('   Status:', err.status);
        console.log('   Error response:', err.error);
        console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
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
    const password = sessionStorage.getItem(`share_password_${token}`);
    let params = new HttpParams().set('limit', this.albumLimit).set('offset', this.albumOffset);
    if (password) {
      params = params.set('password', password);
    }
    
    this.http.get<any>(`${environment.apiBaseUrl}/public/shares/album/${token}/media`, { 
      params: params 
    }).subscribe({
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

    // For public shares, always use the public share thumbnail endpoint (API returns authenticated endpoint)
    const token = this.route.snapshot.paramMap.get('token');
    let photoThumbUrl: string = '';
    if (token && path) {
      // Use public share thumbnail endpoint for public shares
      photoThumbUrl = this.getThumbnailUrl(id, path);
    } else if (p.thumbnailUrl || p.ThumbnailUrl) {
      const rawThumb = p.thumbnailUrl || p.ThumbnailUrl;
      photoThumbUrl = rawThumb.startsWith('http') ? rawThumb : (environment.mediaBaseUrl + rawThumb);
    } else {
      photoThumbUrl = this.getThumbnailUrl(id, path);
    }

    return {
      id: id,
      path: path,
      thumbnailUrl: photoThumbUrl,
      filename: p.Filename ?? p.filename ?? '',
      captured_at: p.capturedAt ?? p.capturedAt ?? '',
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
    const password = sessionStorage.getItem(`share_password_${token}`);
    // Use the album media thumbnail endpoint with the share token and media path
    let url = `${environment.apiBaseUrl}/public/shares/album/${token}/media/thumb?path=${encodeURIComponent(path)}`;
    if (password) {
      url += `&password=${encodeURIComponent(password)}`;
    }
    return url;
  }

  changeAlbumPage(dir: number): void {
    const newOffset = this.albumOffset + (dir * this.albumLimit);
    
    if (newOffset < 0 || newOffset >= this.totalAlbumPhotos) return;
    
    // For public shares, use the album media endpoint
    const token = this.route.snapshot.paramMap.get('token');
    if (!token) return;
    
    const password = sessionStorage.getItem(`share_password_${token}`);
    let params = new HttpParams().set('limit', this.albumLimit).set('offset', newOffset);
    if (password) {
      params = params.set('password', password);
    }
    
    this.loading = true;
    this.http.get<any>(`${environment.apiBaseUrl}/public/shares/album/${token}/media`, { 
      params: params 
    }).subscribe({
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
      
      const password = sessionStorage.getItem(`share_password_${token}`);
      let params = new HttpParams().set('limit', this.albumLimit).set('offset', targetOffset);
      if (password) {
        params = params.set('password', password);
      }
      
      this.loading = true;
      this.http.get<any>(`${environment.apiBaseUrl}/public/shares/album/${token}/media`, { 
        params: params 
      }).subscribe({
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
    console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
    console.log('📸 [DEBUG] PublicShareAlbum.onPhotoClick()');
    console.log('   Photo ID:', id);
    const token = this.route.snapshot.paramMap.get('token');
    const path = this.photos.find(p => p.id === id)?.path || '';
    const ids = this.photos.map(p => p.id).join(',');
    const paths = this.photos.map(p => p.path || '').join(',');
    console.log('   Token:', token);
    console.log('   Path:', path);
    console.log('   albumIds:', ids);
    console.log('   source:', 'shared');
    console.log('   mediaPaths:', paths);
    console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
    if (token && path) {
      // Use public share endpoint with token and media path
      console.log('→ Using public share endpoint with token and media path');
      this.router.navigate(['/photos', id], { queryParams: { albumIds: ids, source: 'shared', shareToken: token, mediaPath: path, mediaPaths: paths } });
    } else if (token) {
      // Token but no path, fall back to authenticated shared media endpoint
      console.log('→ Using public share endpoint with token, no path');
      this.router.navigate(['/photos', id], { queryParams: { albumIds: ids, source: 'shared', shareToken: token } });
    } else {
      console.log('→ Using authenticated shared media endpoint');
      this.router.navigate(['/photos', id], { queryParams: { albumIds: ids, source: 'shared' } });
    }
  }

  goBack(): void { 
    window.history.back(); 
  }
}
