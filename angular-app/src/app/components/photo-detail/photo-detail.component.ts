import { Component, OnInit, OnDestroy, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router, RouterModule } from '@angular/router';
import { Subscription } from 'rxjs';
import { PhotoService } from '../../services/photo.service';
import { AlbumService } from '../../services/album.service';
import { PresentationService, MediaItem } from '../../services/presentation.service';
import { ShareTriggerService } from '../../services/share-trigger.service';
import { Photo } from '../../models/photo.model';

@Component({
  selector: 'app-photo-detail',
  standalone: true,
  imports: [CommonModule, RouterModule, FormsModule],
  template: `
    <div class="container mt-4">
      <div class="row">
        <div class="col-md-8">
          <div class="photo-detail-container" *ngIf="photo; else loading">
            <!-- Video Player for videos -->
            <div class="video-viewer-wrapper" *ngIf="isVideo()">
              <video 
                [attr.src]="mediaSrcUrl()" 
                controls
                preload="metadata"
                class="main-video rounded shadow w-100"
                crossorigin="use-credentials"
                (error)="onVideoError($event)"
                (loadstart)="onMediaLoadStart()"
              >
                Your browser does not support the video tag.
              </video>
            </div>
            
            <!-- Image display for photos -->
            <div class="image-viewer-wrapper" *ngIf="!isVideo()">
              <img 
                [src]="thumbnailUrl()" 
                [alt]="photo.filename" 
                class="main-image rounded shadow"
                crossorigin="use-credentials"
                (error)="onImageError($event)"
                (loadstart)="onMediaLoadStart()"
              >
            </div>
            
            <div class="mt-3 d-flex justify-content-between align-items-start">
              <div>
                <h3 class="mb-1">{{ photo.filename }}</h3>
                <p class="text-muted mb-0">Captured: {{ photo.captured_at | date:'medium' }}</p>
              </div>
              <div class="btn-group">
                <a [href]="originalUrl()" download="{{ photo.filename }}" class="btn btn-outline-secondary">
                  <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="me-1"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2 2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
                  Download
                </a>
                <button (click)="sharePhoto()" class="btn btn-info ms-2">
                  <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="me-1"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>
                  Share
                </button>
                <button (click)="startPresentation()" class="btn btn-warning ms-2">
                  🎬 Presentation Mode
                </button>
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
              <li class="list-group-item">
                <span class="text-muted">Filename:</span> {{ photo?.filename }}
              </li>
              <li class="list-group-item">
                <span class="text-muted">Type:</span> {{ isVideo() ? '🎥 Video' : (photo?.type || 'Photo') }}
              </li>
              <li class="list-group-item" *ngIf="isVideo()">
                <span class="text-muted">Duration:</span> {{ formatDuration(photo?.videoMetadata?.duration) }}
              </li>
              <li class="list-group-item">
                <span class="text-muted">Captured:</span> {{ photo?.captured_at | date:'fullDate' }}
              </li>
              <li class="list-group-item" *ngIf="photo?.width || photo?.height">
                <span class="text-muted">Dimensions:</span> {{ photo?.width }} x {{ photo?.height }}
              </li>
              <li class="list-group-item" *ngIf="isVideo()">
                <span class="text-muted">Video Codec:</span> {{ photo?.videoMetadata?.video_codec || 'Unknown' }}
              </li>
              <li class="list-group-item" *ngIf="isVideo()">
                <span class="text-muted">Audio Codec:</span> {{ photo?.videoMetadata?.audio_codec || 'Unknown' }}
              </li>
              <li class="list-group-item" *ngIf="isVideo() && (photo?.videoMetadata?.frame_rate ?? 0) > 0">
                <span class="text-muted">Frame Rate:</span> {{ photo?.videoMetadata?.frame_rate }} fps
              </li>
              <li class="list-group-item" *ngIf="photo?.size">
                <span class="text-muted">Size:</span> {{ formatFileSize(photo.size) }}
              </li>
              
              <!-- Tags Section -->
              <li class="list-group-item">
                <div class="d-flex align-items-center mb-2 gap-3">
                  <span class="text-muted" style="margin-right: 16px !important;">Tags:</span>
                  <button *ngIf="!isEditingTags" (click)="startEditingTags()" class="btn btn-sm btn-outline-primary py-1 px-2" style="font-size: 0.75rem;">
                    ✏️ Edit
                  </button>
                </div>
                
                <!-- Tag Display Mode -->
                <div *ngIf="!isEditingTags">
                  <span *ngIf="photo?.tags && getTagList(photo.tags).length > 0" class="d-flex flex-wrap gap-1 mb-2">
                    <span 
                      *ngFor="let tag of getTagList(photo.tags)" 
                      (click)="searchByTag(tag)"
                      class="badge bg-primary text-white cursor-pointer" 
                      style="cursor: pointer;"
                    >
                      {{ tag }} ×
                    </span>
                  </span>
                  <div *ngIf="!photo?.tags || getTagList(photo.tags).length === 0" class="text-muted small">
                    No tags added yet. Click "Edit" to add tags.
                  </div>
                </div>
                
                <!-- Tag Edit Mode -->
                <div *ngIf="isEditingTags">
                  <input 
                    type="text" 
                    [(ngModel)]="tagInput" 
                    (keyup.enter)="saveTags()"
                    placeholder="Enter tags separated by commas..."
                    class="form-control form-control-sm mb-2"
                    #tagInputRef
                  >
                  <div class="btn-group btn-group-sm">
                    <button (click)="saveTags()" class="btn btn-success">Save</button>
                    <button (click)="cancelEditingTags()" class="btn btn-secondary">Cancel</button>
                  </div>
                </div>
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
  private albumService = inject(AlbumService);
  private presentationService = inject(PresentationService);
  private shareTrigger = inject(ShareTriggerService);
  private subscription?: Subscription;

  // Tag editing state
  isEditingTags = false;
  tagInput = '';

  ngOnInit(): void {
    const id = this.route.snapshot.paramMap.get('id');
    
    // Check if navigating from shared media (source=shared query param)
    const isSharedMedia = this.route.snapshot.queryParams['source'] === 'shared';
    
    console.log('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
    console.log(`🔍 [DEBUG] PhotoDetailComponent.ngOnInit`);
    console.log(`   Route params: id=${id}, source=${isSharedMedia ? 'shared' : 'owner'}`);
    console.log(`   Query params:`, this.route.snapshot.queryParams);
    
    if (id) {
      // Use getSharedMedia() for shared access, or getMedia() for ownership check
      const fetch$ = isSharedMedia 
        ? this.photoService.getSharedMedia(id)
        : this.photoService.getMedia(id);
      
      console.log(`   API call: ${isSharedMedia ? 'GET /media/shared/' + id : 'GET /media/' + id}`);
      
      this.subscription = fetch$.subscribe({
        next: (photo) => {
          console.log(`━━━━━━━━━━━━━━━━━━━━━━━━━━━━`);
          console.log(`🟢 [DEBUG] PhotoDetailComponent photo loaded successfully!`);
          console.log(`   ID: ${photo.id}`);
          console.log(`   Filename: ${photo.filename}`);
          console.log(`   mediaType: ${photo.mediaType || 'N/A'}`);
          console.log(`   path (for <img>/<video> src): ${photo.path}`);
          console.log(`   thumbnailUrl: ${photo.thumbnailUrl || 'N/A'}`);
          console.log(`━━━━━━━━━━━━━━━━━━━━━━━━━━━━`);
          this.photo = photo;
        },
        error: (err) => {
          console.error('❌ [DEBUG] PhotoDetailComponent Error fetching media', err);
          console.error('   Status:', err.status);
          console.error('   URL:', err.url || 'N/A');
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

  // Computed URL properties for shared vs owned media views
  
  /** Returns the correct thumbnail URL based on shared/owned context */
  thumbnailUrl(): string {
    if (!this.photo) return '';
    
    const isShared = this.route.snapshot.queryParams['source'] === 'shared';
    if (isShared && this.photo.id) {
      return `${this.photoService['API_BASE_URL']}/media/shared/${this.photo.id}/thumb`;
    }
    
    // For owned media, use thumbnailUrl from the photo object if available
    return this.photo.thumbnailUrl || this.photo.path;
  }

  /** Returns the correct original file URL for video playback */
  mediaSrcUrl(): string {
    if (!this.photo) return '';
    
    const isShared = this.route.snapshot.queryParams['source'] === 'shared';
    if (isShared && this.photo.id) {
      return `${this.photoService['API_BASE_URL']}/media/shared/${this.photo.id}/original`;
    }
    
    return this.photo.path;
  }

  /** Returns the correct original file URL for download */
  originalUrl(): string {
    if (!this.photo) return '';
    
    const isShared = this.route.snapshot.queryParams['source'] === 'shared';
    if (isShared && this.photo.id) {
      return `${this.photoService['API_BASE_URL']}/media/shared/${this.photo.id}/original`;
    }
    
    return this.photo.path;
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
    const err = (event as any).target?.error;
    console.error('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
    console.error('❌ [DEBUG] Video media load FAILED!');
    console.error(`   Media type: video`);
    console.error(`   URL requested: ${this.photo?.path}`);
    console.error(`   Error details:`, err ? `code=${err.code}, message=${err.message || 'N/A'}` : 'No error object available');
    console.error('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
  }

  onImageError(event: Event): void {
    const img = (event as any).target;
    console.error('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
    console.error('❌ [DEBUG] Image load FAILED!');
    console.error(`   URL requested: ${this.photo?.path}`);
    console.error(`   Media type: photo`);
    console.error(`   Alt text (filename): ${img?.alt || 'N/A'}`);
    if (typeof img?.complete !== 'undefined') {
      console.error(`   Image complete flag: ${img.complete}, naturalWidth: ${img.naturalWidth || 0}, naturalHeight: ${img.naturalHeight || 0}`);
    }
    const err = (event as any).target?.error;
    if (err) {
      console.error(`   Error details:`, `code=${err.code}, message=${err.message || 'N/A'}`);
    } else {
      console.error('   No error event - image may have been blocked by CORS/same-origin policies');
    }
    console.error('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
  }

  onMediaLoadStart(): void {
    console.log(`🟡 [DEBUG] Media load STARTED: ${this.photo?.path}`);
  }

  goBack(): void {
    window.history.back();
  }

  startPresentation(): void {
    if (!this.photo) return;

    // Check if viewing from a shared album (source=shared query param + albumIds present)
    const isSharedMedia = this.route.snapshot.queryParams['source'] === 'shared';
    const albumIdsParam = this.route.snapshot.queryParams['albumIds'];
    
    let mediaItems: MediaItem[];
    
    if (albumIdsParam) {
      // We came from an album - fetch actual photo details to get correct media types
      const albumPhotoIds: string[] = albumIdsParam.split(',').map((id: string) => id.trim()).filter(Boolean);
      
      import('rxjs').then(({ forkJoin, of, catchError }) => {
        const requests$ = albumPhotoIds.map(id => 
          isSharedMedia 
            ? this.photoService.getSharedMedia(id).pipe(
                catchError(() => of(null)) // Skip failed fetches gracefully
              )
            : this.photoService.getMedia(id).pipe(
                catchError(() => of(null)) // Skip failed fetches gracefully
              )
        );
        
        forkJoin(requests$).subscribe({
          next: (photos) => {
            mediaItems = photos.filter((p): p is Photo => p !== null).map(p => ({
              id: p.id,
              path: isSharedMedia 
                ? `${this.photoService['API_BASE_URL']}/media/shared/${p.id}/original`
                : `${this.photoService['API_BASE_URL']}/media/${p.id}/original`,
              filename: p.filename || '',
              mediaType: p.mediaType || 'photo'
            }));

            const startIndex = mediaItems.findIndex(item => item.id === this.photo?.id);
            
            if (startIndex !== -1 && mediaItems.length > 0) {
              this.presentationService.open(mediaItems, startIndex);
              this.router.navigate(['/presentation']);
            } else {
              alert('No items available for presentation.');
            }
          },
          error: (err) => {
            console.error('Failed to load album media for presentation', err);
            alert('Failed to start presentation mode.');
          }
        });
      });
    } else {
      // No album context - fetch all gallery items using the appropriate endpoint based on shared context
      if (isSharedMedia) {
        this.photoService.listSharedMedia(100, 0).subscribe({
          next: (response: any) => {
            mediaItems = response.items.map((p: any) => ({
              id: p.id || p.ID,
              path: `${this.photoService['API_BASE_URL']}/media/shared/${(p.id || p.ID)}/original`,
              filename: p.filename || '',
              mediaType: (p.mediaType || 'photo')
            }));

            const startIndex = mediaItems.findIndex(item => item.id === this.photo?.id);
            
            if (startIndex !== -1 && mediaItems.length > 0) {
              this.presentationService.open(mediaItems, startIndex);
              this.router.navigate(['/presentation']);
            } else {
              alert('No items available for presentation.');
            }
          },
          error: (err) => {
            console.error('Failed to load shared media for presentation', err);
            alert('Failed to start presentation mode.');
          }
        });
      } else {
        this.photoService.listMedia(100, 0).subscribe({
          next: (response: any) => {
            mediaItems = response.photos.map((p: any) => ({
              id: p.id,
              path: `${this.photoService['API_BASE_URL']}/media/${p.id}/original`,
              filename: p.filename,
              mediaType: p.mediaType || 'photo'
            }));

            const startIndex = mediaItems.findIndex(item => item.id === this.photo?.id);
            
            if (startIndex !== -1 && mediaItems.length > 0) {
              this.presentationService.open(mediaItems, startIndex);
              this.router.navigate(['/presentation']);
            } else {
              alert('No items available for presentation.');
            }
          },
          error: (err) => {
            console.error('Failed to load media for presentation', err);
            alert('Failed to start presentation mode.');
          }
        });
      }
    }
  }

  // Tag editing methods
  startEditingTags(): void {
    this.isEditingTags = true;
    if (this.photo?.tags) {
      const tagList = this.getTagList(this.photo.tags);
      // Join with colon as per spec for the input field
      this.tagInput = tagList.join(': '); 
    } else {
      this.tagInput = '';
    }
  }

  saveTags(): void {
    if (!this.photo) return; // Allow empty string to clear tags
    
    // Use the parser which now supports colon, then join with colon for storage
    const tagsString = this.parseTagString(this.tagInput).join(':'); 
    this.photoService.updateTags(this.photo.id, tagsString).subscribe({
      next: (updated) => {
        if (this.photo && updated) {
          // Update the local photo object with new tags to trigger change detection
          // The backend only returns { status: "success" }, so we manually set the tags field
          this.photo = { ...this.photo, tags: tagsString };
        }
        this.isEditingTags = false;
        this.tagInput = '';
      },
      error: (err) => {
        console.error('Error updating tags', err);
        alert('Failed to update tags');
      }
    });
  }

  cancelEditingTags(): void {
    this.isEditingTags = false;
    if (this.photo?.tags) {
      const tagList = this.getTagList(this.photo.tags);
      this.tagInput = tagList.join(': ');
    } else {
      this.tagInput = '';
    }
  }

  getTagList(tags: any): string[] {
    if (!tags) return [];
    // If it's already an array, just use it
    if (Array.isArray(tags)) return tags;
    
    if (typeof tags === 'string') {
      // Split by colon as per spec requirements
      return tags.split(':')
        .map(t => t.trim())
        .filter(t => t.length > 0);
    }
    return [];
  }

  parseTagString(input: string): string[] {
    // Split by colon, trim whitespace, remove empty strings
    return input.split(':')
      .map(tag => tag.trim())
      .filter(tag => tag.length > 0);
  }

  sharePhoto(): void {
    if (!this.photo) return;
    this.shareTrigger.open(this.photo.id, 'media');
  }

  searchByTag(tag: string): void {
    this.router.navigate(['/'], { queryParams: { tag: tag } });
  }
}
