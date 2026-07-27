import { Component, OnInit, inject, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule } from '@angular/router';
import { ActivatedRoute, Router } from '@angular/router';
import { HttpClient, HttpParams } from '@angular/common/http';
import { FormsModule } from '@angular/forms';
import { PhotoService } from '../../services/photo.service';
import { ShareService } from '../../services/share.service';
import { environment } from '../../../environments/environment';

@Component({
    selector: 'app-public-share-media',
    imports: [CommonModule, RouterModule, FormsModule],
    template: `
    <div class="container mt-4">
      <!-- Password Protection Modal -->
      @if (showPasswordModal) {
        <div class="modal-overlay">
          <div class="password-modal-content bg-dark border rounded p-4 text-center">
            <h3>🔒 This share is password protected</h3>
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
                  Unlock
                </button>
              </div>
            </div>
          </div>
        </div>
      }
    
      <!-- Loading State -->
      @if (!mediaItem && !showError) {
        <div class="text-center py-5">
          <div class="spinner-border text-primary" role="status">
            <span class="visually-hidden">Loading...</span>
          </div>
          <p class="mt-2">Loading media...</p>
        </div>
      }
    
      <!-- Error State -->
      @if (showError) {
        <div class="text-center py-5">
          <h3>😕 Content Not Available</h3>
          <p class="text-muted">{{ errorMessage }}</p>
          <a routerLink="/" class="btn btn-primary mt-3">Back to Gallery</a>
        </div>
      }
    
      <!-- Media Display -->
      @if (mediaItem) {
        <div class="row">
          <div class="col-md-8">
            <div class="photo-detail-container">
              <!-- Video Player for videos -->
              @if (isVideo()) {
                <div class="video-viewer-wrapper">
                  <video
                    [src]="getMediaPath('original')"
                    controls
                    preload="metadata"
                    class="main-video rounded shadow w-100"
                    crossorigin="use-credentials"
                    >
                    Your browser does not support the video tag.
                  </video>
                </div>
              }
              <!-- Image display for photos -->
              @if (!isVideo()) {
                <div class="image-viewer-wrapper">
                  <img
                    [src]="getMediaPath('original')"
                    [alt]="mediaItem.filename"
                    class="main-image rounded shadow"
                    crossorigin="use-credentials"
                    >
                </div>
              }
              <div class="mt-3 d-flex justify-content-between align-items-start">
                <div>
                  <h3 class="mb-1">{{ mediaItem?.filename }}</h3>
                  @if (mediaItem?.captured_at) {
                    <p class="text-muted mb-0">Captured: {{ mediaItem.captured_at | date:'medium' }}</p>
                  }
                </div>
                <div class="btn-group">
                  <button (click)="downloadMedia()" class="btn btn-outline-secondary" type="button">
                    <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="me-1"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2 2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
                    Download
                  </button>
                  <button (click)="goBack()" class="btn btn-outline-primary" type="button">
                    <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="me-1"><line x1="19" y1="12" x2="5" y2="12"/><polyline points="12 19 5 12 12 5"/></svg>
                    Gallery
                  </button>
                </div>
              </div>
            </div>
          </div>
          <div class="col-md-4">
            <div class="card shadow-sm sticky-top" style="top: 100px;">
              <div class="card-header bg-light">
                <h5 class="mb-0">Details</h5>
              </div>
              <ul class="list-group list-group-flush">
                <li class="list-group-item">
                  <span class="text-muted">Filename:</span> {{ mediaItem?.filename }}
                </li>
                <li class="list-group-item">
                  <span class="text-muted">Type:</span> {{ isVideo() ? '🎥 Video' : (mediaItem?.type || 'Photo') }}
                </li>
                @if (isVideo()) {
                  <li class="list-group-item">
                    <span class="text-muted">Duration:</span> {{ formatDuration($safeNavigationMigration(mediaItem?.videoMetadata?.duration)) }}
                  </li>
                }
                <li class="list-group-item">
                  <span class="text-muted">Captured:</span> {{ $safeNavigationMigration(mediaItem?.captured_at) | date:'fullDate' }}
                </li>
                @if (mediaItem?.width || mediaItem?.height) {
                  <li class="list-group-item">
                    <span class="text-muted">Dimensions:</span> {{ mediaItem?.width }} x {{ mediaItem?.height }}
                  </li>
                }
                @if (isVideo()) {
                  <li class="list-group-item">
                    <span class="text-muted">Video Codec:</span> {{ mediaItem?.videoMetadata?.video_codec || 'Unknown' }}
                  </li>
                }
                @if (isVideo()) {
                  <li class="list-group-item">
                    <span class="text-muted">Audio Codec:</span> {{ mediaItem?.videoMetadata?.audio_codec || 'Unknown' }}
                  </li>
                }
                @if (mediaItem?.size) {
                  <li class="list-group-item">
                    <span class="text-muted">Size:</span> {{ formatFileSize(mediaItem.size) }}
                  </li>
                }
              </ul>
            </div>
          </div>
        </div>
      }
    </div>
    `,
    changeDetection: ChangeDetectionStrategy.Eager,
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
      max-height: 75vh;
      object-fit: contain;
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
      max-height: 75vh;
      display: block;
      background-color: #000;
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
  `]
})
export class PublicShareMediaComponent implements OnInit {
  private route = inject(ActivatedRoute);
  private router = inject(Router);
  private http = inject(HttpClient);
  
  mediaItem: any = null;
  showError = false;
  errorMessage = '';
  
  // Password protection state
  showPasswordModal = false;
  passwordInput = '';

  ngOnInit(): void {
    const token = this.route.snapshot.paramMap.get('token');
    
    if (token) {
      this.loadPublicShareMedia(token);
    } else {
      this.showError = true;
      this.errorMessage = 'Invalid share link';
    }
  }

  private loadPublicShareMedia(token: string): void {
    const url = `${environment.apiBaseUrl}/public/shares/media/${token}`;
    
    this.http.get<any>(url).subscribe({
      next: (response) => {
        // This is the SUCCESS path - 200 OK
        const mediaData = response?.Media || response;
        
        if (!mediaData) {
          this.showError = true;
          this.errorMessage = 'No media content found';
          return;
        }
        
        try {
          this.mediaItem = this.normalizePhoto(mediaData);
        } catch (e) {
          this.showError = true;
          this.errorMessage = 'Failed to process media content';
        }
      },
      error: (err: any) => {
        let statusCode: number | null = null;
        
        // Try multiple ways to get the status code
        if (typeof err.status === 'number') {
          statusCode = err.status;
        } else if (err.error && typeof err.error === 'object' && typeof err.error.status === 'number') {
          statusCode = err.error.status;
        } else if (err.status) {
          // Try to coerce it (could be string like "403")
          statusCode = Number(err.status);
        } else if (err.error && err.error.status) {
          statusCode = Number(err.error.status);
        }

        // 403 explicitly means password is required/incorrect -> show modal instead of error page
        if (statusCode === 403) {
          this.showPasswordModal = true;
          this.mediaItem = null; // Ensure !mediaItem condition is met
          return; // Exit early so showError stays false
        } 

        // For other errors (404, 410, network issues), show error state
        this.showError = true;
        
        if (statusCode === 410) {
          this.errorMessage = 'Share link has expired';
        } else if (statusCode === 404) {
          this.errorMessage = 'Share link not found or invalid';
        } else {
          const backendMsg = err.error?.error || '';
          this.errorMessage = backendMsg ? `${backendMsg}` : 'Failed to load media content';
        }
      }
    });
  }

  public verifyPassword(): void {
 
    const token = this.route.snapshot.paramMap.get('token');
    if (!token || !this.passwordInput) return;
    
 
  
    
    // Try with password parameter
    const params = new HttpParams().set('password', this.passwordInput);
    const url = `${environment.apiBaseUrl}/public/shares/media/${token}`;
    
 
    
    this.http.get<any>(url, { 
      params: params,
      headers: { 'Accept': 'application/json' }
    }).subscribe({
      next: (response) => {
 
        const mediaData = response?.Media || response;
        this.mediaItem = this.normalizePhoto(mediaData);
        this.showPasswordModal = false;
        this.passwordInput = '';
      },
      error: (err: any) => {
  
       console.error('Error with password:', err);
        const statusCode = err.status ?? err.error?.status;
        
        if (statusCode === 403) {
          // Wrong password - show error and clear input
          alert('Incorrect password. Please try again.');
          this.passwordInput = '';
        } else {
          this.showError = true;
          const errorData = err.error as any;
          this.errorMessage = errorData?.error || 'Failed to access shared content';
        }
      }
    });
  }

  private normalizePhoto(p: any): any {
  
    
    const id = p.ID ?? p.id ?? '';
    
    let mediaType: string | undefined;
    const rawMediaType = p.MediaType || p.mediaType || p.media_type;
    if (rawMediaType) {
      const mt = String(rawMediaType).toLowerCase();
      mediaType = mt === 'video' ? 'video' : 'photo';
    }

    return {
      id: id,
      path: '',
      thumbnailUrl: p.thumbnailUrl || '',
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

  isVideo(): boolean {
    return this.mediaItem?.mediaType === 'video';
  }

  getMediaPath(type: 'original' | 'thumb'): string {
    const token = this.route.snapshot.paramMap.get('token');
    if (!token) return '';
    return `${environment.apiBaseUrl}/public/shares/media/${token}/${type}`;
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


  /**
   * Downloads the original file as a Blob to ensure browser initiates a download
   * rather than opening the file inline (which browsers do for images/videos).
   */
  downloadMedia(): void {
    if (!this.mediaItem) return;

    const token = this.route.snapshot.paramMap.get('token');
    if (!token) return;

    // Fetch the file as a Blob
    const url = `${environment.apiBaseUrl}/public/shares/media/${token}/original`;
    const password = sessionStorage.getItem(`share_password_${token}`);
    let downloadUrl = url;
    if (password) {
      downloadUrl += `?password=${encodeURIComponent(password)}`;
    }

    this.http.get(downloadUrl, { responseType: 'blob' }).subscribe({
      next: (blob: Blob) => {
        // Create a Blob URL
        const blobUrl = URL.createObjectURL(blob);

        // Create a temporary anchor element with the download attribute
        const link = document.createElement('a');
        link.href = blobUrl;
        link.download = this.mediaItem?.filename || 'download';

        // Click it programmatically to trigger the download
        document.body.appendChild(link);
        link.click();

        // Clean up the Blob URL after a short delay
        setTimeout(() => {
          URL.revokeObjectURL(blobUrl);
          document.body.removeChild(link);
        }, 100);
      },
      error: (err: any) => {
        console.error('Error downloading file', err);
        alert('Failed to download file.');
      }
    });
  }
  goBack(): void {
    window.history.back();
  }
}
