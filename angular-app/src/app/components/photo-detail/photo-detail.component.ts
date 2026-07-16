import { Component, OnInit, OnDestroy, inject, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router, RouterModule } from '@angular/router';
import { Subscription } from 'rxjs';
import { PhotoService } from '../../services/photo.service';
import { AlbumService } from '../../services/album.service';
import { PresentationService, MediaItem } from '../../services/presentation.service';
import { SearchService, SearchScope } from '../../services/search.service';
import { ShareTriggerService } from '../../services/share-trigger.service';
import { ExifTriggerService } from '../../services/exif-trigger.service';
import { GalleryStateService } from '../../services/gallery-state.service';
import { ExifDataPopupComponent } from '../exif-data-popup/exif-data-popup.component';
import { Photo } from '../../models/photo.model';

@Component({
    selector: 'app-photo-detail',
    imports: [CommonModule, RouterModule, FormsModule, ExifDataPopupComponent],
    template: `
    <div class="container mt-4">
      <div class="row">
        <div class="col-md-8">
          @if (lastPresentationItem) {
            <div class="alert alert-info d-flex justify-content-between align-items-center" style="font-size: 13px;">
              <span>
                <strong>Presentation returned:</strong> Last viewed item ID — {{ lastPresentationItem }}
              </span>
              <button class="btn btn-sm btn-outline-primary" (click)="dismissLastPresentation()">Dismiss</button>
            </div>
          }
          @if (photo) {
            <div class="photo-detail-container">
              @if (isFromAlbum) {
                <div class="alert alert-info d-flex justify-content-between align-items-center" style="font-size: 13px;">
                  <span>
                    <strong>Album View:</strong> Viewing photo from album
                  </span>
                  <button class="btn btn-sm btn-outline-primary" (click)="goBack()">Back to Album</button>
                </div>
              }
              <!-- Video Player for videos -->
              @if (isVideo()) {
                <div class="video-viewer-wrapper">
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
              }
              <!-- Image display for photos -->
              @if (!isVideo()) {
                <div class="image-viewer-wrapper">
                  <img
                    [src]="originalUrl()"
                    [alt]="photo.filename"
                    class="main-image rounded shadow"
                    crossorigin="use-credentials"
                    (error)="onImageError($event)"
                    (loadstart)="onMediaLoadStart()"
                    >
                </div>
              }
              <div class="mt-3 d-flex justify-content-between align-items-start">
                <div>
                  <h3 class="mb-1">{{ photo.filename }}</h3>
                  <p class="text-muted mb-0">Captured: {{ capturedDate | date:'medium' }}</p>
                </div>
                <div class="btn-group position-relative">
                  <button (click)="startEditingTags()" class="btn btn-outline-success ms-2" [class.active]="isEditingTags">
                    <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="me-1"><path d="M20.59 13.41l-7.17 7.17a2 2 0 0 1-2.83 0L2 12V2h10l8.59 8.59a2 2 0 0 1 0 2.82z"/><line x1="7" y1="7" x2="7.01" y2="7"/></svg>
                    Tag
                  </button>
                  @if (isEditingTags) {
                    <div class="tag-editing-popup position-absolute bg-white border rounded shadow-sm p-3 mt-2" style="z-index: 1000; min-width: 300px;">
                      <input
                        type="text"
                        [(ngModel)]="tagInput"
                        (keyup.enter)="saveTags()"
                        placeholder="Enter tags separated by colons..."
                        class="form-control form-control-sm mb-2"
                        >
                      <div class="btn-group btn-group-sm">
                        <button (click)="saveTags()" class="btn btn-success">Save</button>
                        <button (click)="cancelEditingTags()" class="btn btn-secondary">Cancel</button>
                      </div>
                    </div>
                  }
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
                    {{ isFromAlbum ? 'Back to Album' : 'Back to Gallery' }}
                  </button>
                </div>
              </div>
            </div>
          } @else {
            <div class="text-center py-5">
              <div class="spinner-border text-primary" role="status">
                <span class="visually-hidden">Loading...</span>
              </div>
              <p class="mt-2">Loading details...</p>
            </div>
          }
    
        </div>
        <div class="col-md-4">
          <div class="card shadow-sm sticky-top" style="top: 100px;">
            <div class="card-header bg-light">
              <h5 class="mb-0">Details</h5>
            </div>
            <ul class="list-group list-group-flush">
              <!-- Basic Info -->
              <li class="list-group-item">
                <span class="text-muted">Filename:</span> {{ photo?.filename }}
              </li>
              <li class="list-group-item">
                <span class="text-muted">Type:</span> {{ isVideo() ? '🎥 Video' : (photo?.type || 'Photo') }}
              </li>
              @if (isVideo()) {
                <li class="list-group-item">
                  <span class="text-muted">Duration:</span> {{ formatDuration($safeNavigationMigration(photo?.videoMetadata?.duration)) }}
                </li>
              }
              <li class="list-group-item">
                <span class="text-muted">Captured:</span> {{ capturedDate | date:'fullDate' }}
              </li>
              @if (photo?.width || photo?.height) {
                <li class="list-group-item">
                  <span class="text-muted">Dimensions:</span> {{ photo?.width }} x {{ photo?.height }}
                </li>
              }
              @if (isVideo()) {
                <li class="list-group-item">
                  <span class="text-muted">Video Codec:</span> {{ photo?.videoMetadata?.video_codec || 'Unknown' }}
                </li>
              }
              @if (isVideo()) {
                <li class="list-group-item">
                  <span class="text-muted">Audio Codec:</span> {{ photo?.videoMetadata?.audio_codec || 'Unknown' }}
                </li>
              }
              @if (isVideo() && (photo?.videoMetadata?.frame_rate ?? 0) > 0) {
                <li class="list-group-item">
                  <span class="text-muted">Frame Rate:</span> {{ photo?.videoMetadata?.frame_rate }} fps
                </li>
              }
              @if (photo?.size) {
                <li class="list-group-item">
                  <span class="text-muted">Size:</span> {{ formatFileSize(photo.size) }}
                </li>
              }
    
              <!-- Tags Display -->
              @if (getTagList(photo?.tags || '').length > 0) {
                <li class="list-group-item">
                  <div class="d-flex align-items-center mb-2 gap-3">
                    <span class="text-muted" style="margin-right: 16px !important;">Tags:</span>
                  </div>
                  <span class="d-flex flex-wrap gap-1">
                    @for (tag of getTagList(photo?.tags || ''); track tag) {
                      <span
                        class="badge bg-primary text-white"
                        >
                        {{ tag }}
                      </span>
                    }
                  </span>
                </li>
              }
            </ul>
          </div>
        </div>
      </div>
    </div>
    
    @if (showExifPopup) {
      <app-exif-data-popup
        [exifData]="exifData"
        (close)="showExifPopup = false"
      ></app-exif-data-popup>
    }
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
    .cursor-pointer {
      cursor: pointer;
    }
    h3 {
      font-size: 14px;
      font-weight: 500;
      color: #e5e7eb;
      margin-bottom: 4px;
    }
    .photo-detail-container > div:first-child {
      font-size: 14px;
      color: #9ca3af;
    }
    .list-group-item {
      font-size: 14px;
      padding: 10px 16px;
    }
    .list-group-item .text-muted {
      font-size: 14px;
      color: #6b7280;
      margin-right: 8px;
    }
    .card-header h5 {
      font-size: 14px;
      font-weight: 500;
      color: #e5e7eb;
    }
    .btn {
      font-size: 14px;
    }
    p {
      font-size: 14px;
    }
    .tag-editing-popup {
      animation: fadeIn 0.2s ease-in-out;
    }
    @keyframes fadeIn {
      from { opacity: 0; transform: translateY(-5px); }
      to { opacity: 1; transform: translateY(0); }
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
  private searchService = inject(SearchService);
  private shareTrigger = inject(ShareTriggerService);
  private exifTrigger = inject(ExifTriggerService);
  private subscription?: Subscription;

  // Tag editing state
  isEditingTags = false;
  tagInput = '';

  // EXIF popup state
  showExifPopup = false;

  /** Saved presentation item ID displayed as a banner */
  lastPresentationItem: string | null = null;

  private galleryState = inject(GalleryStateService);

  get exifData(): any {
    if (!this.photo?.metadata) return null;
    return this.photo.metadata;
  }

  /** Returns the EXIF datetime_original if available, otherwise falls back to ModifyDate (Samsung), then captured_at */
  get capturedDate(): Date | string {
    const meta = this.photo?.metadata;
    
    // Try DateTimeOriginal first (standard EXIF capture time)
    if (meta?.datetime_original) {
      return this.parseExifDate(meta.datetime_original);
    }
    
    // Fallback: ModifyDate (used by Samsung Galaxy phones)
    if (meta?.modifydate || meta?.ModifyDate) {
      const dateStr: string | undefined = meta.modifydate || meta.ModifyDate;
      if (dateStr) {
        return this.parseExifDate(dateStr);
      }
    }
    
    // Final fallback: database captured_at
    return this.photo?.captured_at || new Date();
  }

  /** Parses EXIF date format like "2026:04:25 11:13:43" to Date object */
  private parseExifDate(dateStr: string): Date {
    // Convert "2026:04:25 11:13:43" to "2026-04-25T11:13:43" for proper date parsing
    const cleaned = dateStr.replace(/(\d{4}):(\d{2}):(\d{2})\s+(\d{2}):(\d{2}):(\d{2})/, '$1-$2-$3T$4:$5:$6');
    const date = new Date(cleaned);
    return isNaN(date.getTime()) ? new Date() : date;
  }

  ngOnInit(): void {
    // Read the saved presentation item ID from localStorage
    const savedId = this.galleryState.getPresentationItem();
    if (savedId) {
      this.lastPresentationItem = savedId;
    }

    // Subscribe to EXIF trigger service - opens EXIF popup when triggered
    this.exifTrigger.exifTrigger$.subscribe(() => {
      this.showExifPopup = true;
    });
    
    // Detect if we're coming from an album view (albumIds or currentAlbumId query param present)
    const albumIdsParam = this.route.snapshot.queryParams['albumIds'];
    const currentAlbumIdParam = this.route.snapshot.queryParams['currentAlbumId'];
    if (albumIdsParam || currentAlbumIdParam) {
      this.isFromAlbum = true;
    }
    
    // Subscribe to route parameter changes to handle navigation between photos
    this.route.paramMap.subscribe(params => {
      const id = params.get('id');
      
      if (id) {
        // Re-check if coming from album on route change (check both albumIds and currentAlbumId)
        const currentAlbumIds = this.route.snapshot.queryParams['albumIds'];
        const currentAlbumId = this.route.snapshot.queryParams['currentAlbumId'];
        this.isFromAlbum = !!currentAlbumIds || !!currentAlbumId;
      }
      
      if (!id) {
        this.router.navigate(['/']);
        return;
      }
      
      let fetch$: any;
      const isSharedMedia = this.route.snapshot.queryParams['source'] === 'shared';
      const shareToken = this.route.snapshot.queryParams['shareToken'];
      
      if (isSharedMedia && shareToken) {
        // Use public share endpoint (no auth required)
        const mediaPath = this.route.snapshot.queryParams['mediaPath'] || '';
        fetch$ = this.photoService.getPublicShareMedia(shareToken, mediaPath);
      } else if (isSharedMedia) {
        // Use authenticated shared media endpoint
        fetch$ = this.photoService.getSharedMedia(id);
      } else {
        // Use ownership endpoint
        fetch$ = this.photoService.getMedia(id);
      }
      
      this.subscription?.unsubscribe();
      this.subscription = fetch$.subscribe({
        next: (photo: any) => {
          this.photo = photo;
        },
        error: (err: any) => {
          console.error('Error fetching media', err);
          this.router.navigate(['/']);
        }
      });
    });
  }

  ngOnDestroy(): void {
    this.subscription?.unsubscribe();
  }

  // Computed URL properties for shared vs owned media views
  
  /** Returns the correct thumbnail URL based on shared/owned context */
  thumbnailUrl(): string {
    if (!this.photo) return '';
    
    const isShared = this.route.snapshot.queryParams['source'] === 'shared';
    const shareToken = this.route.snapshot.queryParams['shareToken'];
    
    if (isShared && shareToken) {
      // Use public share thumbnail endpoint
      const mediaPath = this.route.snapshot.queryParams['mediaPath'] || '';
      return this.photoService.getPublicShareThumbnailUrl(shareToken, mediaPath);
    }
    
    if (isShared && this.photo.id) {
      // Use authenticated shared media thumbnail endpoint
      return `${this.photoService['API_BASE_URL']}/media/shared/${this.photo.id}/thumb`;
    }
    
    // For owned media, use thumbnailUrl from the photo object if available
    return this.photo.thumbnailUrl || this.photo.path;
  }

  /** Returns the correct original file URL for video playback */
  mediaSrcUrl(): string {
    if (!this.photo) return '';
    
    const isShared = this.route.snapshot.queryParams['source'] === 'shared';
    const shareToken = this.route.snapshot.queryParams['shareToken'];
    
    if (isShared && shareToken) {
      // Use public share original endpoint
      const mediaPath = this.route.snapshot.queryParams['mediaPath'] || '';
      return this.photoService.getPublicShareOriginalUrl(shareToken, mediaPath);
    }
    
    if (isShared && this.photo.id) {
      // Use authenticated shared media original endpoint
      return `${this.photoService['API_BASE_URL']}/media/shared/${this.photo.id}/original`;
    }
    
    return this.photo.path;
  }

  /** Returns the correct original file URL for download */
  originalUrl(): string {
    if (!this.photo) return '';
    
    const isShared = this.route.snapshot.queryParams['source'] === 'shared';
    const shareToken = this.route.snapshot.queryParams['shareToken'];
    
    if (isShared && shareToken) {
      // Use public share original endpoint for download
      const mediaPath = this.route.snapshot.queryParams['mediaPath'] || '';
      return this.photoService.getPublicShareOriginalUrl(shareToken, mediaPath);
    }
    
    if (isShared && this.photo.id) {
      // Use authenticated shared media original endpoint for download
      return `${this.photoService['API_BASE_URL']}/media/shared/${this.photo.id}/original`;
    }
    
    return this.photo.path;
  }

  isVideo(): boolean {
    return this.photo?.mediaType === 'video';
  }

  /** Whether the photo was opened from an album */
  isFromAlbum = false;

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

  formatExposureTime(exposureTime?: string | number): string {
    if (!exposureTime) return 'Unknown';
    
    // Handle string format like "1/250"
    if (typeof exposureTime === 'string' && exposureTime.includes('/')) {
      return exposureTime;
    }
    
    // Handle numeric value (fraction of a second)
    const value = typeof exposureTime === 'string' ? parseFloat(exposureTime) : exposureTime;
    if (isNaN(value) || value <= 0) return 'Unknown';
    
    // Convert to fraction if it's a decimal
    if (value < 1) {
      const denominator = Math.round(1 / value);
      return `1/${denominator}`;
    }
    
    return value.toString();
  }

  onVideoError(event: Event): void {
    const err = (event as any).target?.error;
    console.error('Video media load failed', err);
  }

  onImageError(event: Event): void {
    const err = (event as any).target?.error;
    console.error('Image load failed', err);
  }

  onMediaLoadStart(): void {
    // Media load started
  }

  /**
   * Clears the saved presentation item from localStorage and hides the banner.
   */
  dismissLastPresentation(): void {
    this.galleryState.clearPresentationItem();
    this.lastPresentationItem = null;
  }

  goBack(): void {
    // If coming from an album, navigate back to the album detail page
    if (this.isFromAlbum) {
      const currentAlbumId = this.route.snapshot.queryParams['currentAlbumId'];
      const source = this.route.snapshot.queryParams['source'];
      const shareToken = this.route.snapshot.queryParams['shareToken'];
      
      const queryParams: any = {};
      if (source === 'shared') {
        queryParams.source = 'shared';
        if (shareToken) {
          queryParams.shareToken = shareToken;
        }
      }
      
      // Navigate to the actual album using currentAlbumId (the real album ID)
      if (currentAlbumId) {
        this.router.navigate(['/albums', currentAlbumId], { queryParams });
      } else {
        this.router.navigate(['/albums']);
      }
      return;
    }

    // Read the saved page from GalleryStateService
    const savedPage = this.galleryState.getCurrentPage() || 1;

    // Rebuild query params from current route
    const queryParams: any = {};
    if (this.route.snapshot.queryParams['searchTerm']) {
      queryParams.searchTerm = this.route.snapshot.queryParams['searchTerm'];
      queryParams.searchScope = this.route.snapshot.queryParams['searchScope'];
    }
    if (this.route.snapshot.queryParams['tag']) {
      queryParams.tag = this.route.snapshot.queryParams['tag'];
    }
    if (this.route.snapshot.queryParams['source'] === 'shared') {
      queryParams.source = 'shared';
      if (this.route.snapshot.queryParams['shareToken']) {
        queryParams.shareToken = this.route.snapshot.queryParams['shareToken'];
      }
    }

    this.router.navigate(['/' ], { queryParams, queryParamsHandling: 'merge' });
  }

  startPresentation(): void {
    if (!this.photo) return;

    // Check if viewing from a shared album (source=shared query param + albumIds present)
    const isSharedMedia = this.route.snapshot.queryParams['source'] === 'shared';
    const shareToken = this.route.snapshot.queryParams['shareToken'];
    const albumIdsParam = this.route.snapshot.queryParams['albumIds'];
    const currentAlbumIdParam = this.route.snapshot.queryParams['currentAlbumId'];
    const albumMediaPaths = this.route.snapshot.queryParams['mediaPaths'];
    const searchParam = this.route.snapshot.queryParams['searchTerm'];
    const searchScopeParam = this.route.snapshot.queryParams['searchScope'] || 'all';
    
    let mediaItems: MediaItem[];
    
    // Helper: build media items array from photos (shared or not)
    const buildMediaItems = (photos: (Photo | null)[], sourceContext: string): MediaItem[] => {
      return photos
        .filter((p): p is Photo => p !== null)
        .map(p => ({
          id: p.id,
          path: (isSharedMedia && shareToken)
            ? this.photoService.getPublicShareOriginalUrl(shareToken, p.path || '')
            : isSharedMedia && !shareToken
            ? `${this.photoService['API_BASE_URL']}/media/shared/${p.id}/original`
            : `${this.photoService['API_BASE_URL']}/media/${p.id}/original`,
          filename: p.filename || '',
          mediaType: p.mediaType || 'photo'
        }));
    };

    const startPresentationWithItems = (items: MediaItem[], navigateOpts: any) => {
      const currentPhotoId = this.photo?.id || 'NO_PHOTO_ID';
      const foundIndex = items.findIndex(item => item.id === currentPhotoId);

      if (foundIndex !== -1 && items.length > 0) {
        this.presentationService.open(items, foundIndex);
        // Ensure navigateOpts wraps params in { queryParams: ... } for router.navigate
        const navOpts = (navigateOpts && navigateOpts.queryParams !== undefined)
          ? navigateOpts
          : { queryParams: navigateOpts || {} };
        this.router.navigate(['/presentation'], navOpts);
      } else {
        alert('No items available for presentation.');
      }
    };

    // Build navigation queryParams that preserve album context for return navigation
    const buildNavQueryParams = (): any => {
      const qp: any = {};
      // Always preserve source/album context so goBack() works after presentation
      if (albumIdsParam) {
        qp.albumIds = albumIdsParam;
      }
      if (currentAlbumIdParam) {
        qp.currentAlbumId = currentAlbumIdParam;
      }
      if (isSharedMedia && shareToken) {
        qp.source = 'shared';
        qp.shareToken = shareToken;
      }
      return qp;
    };

    if (albumIdsParam) {
      // We came from an album - build media items from album IDs
      const albumPhotoIds: string[] = albumIdsParam.split(',').map((id: string) => id.trim()).filter(Boolean);
      const albumMediaPathsArray: string[] = albumMediaPaths ? albumMediaPaths.split(',').map((p: string) => p.trim()) : [];
      
      import('rxjs').then(({ forkJoin, of, catchError }) => {
        const requests$ = albumPhotoIds.map(id => 
          isSharedMedia && !shareToken
            ? this.photoService.getSharedMedia(id).pipe(catchError(() => of(null)))
            : isSharedMedia && shareToken
            ? this.photoService.getPublicShareMedia(shareToken, albumMediaPathsArray[albumPhotoIds.indexOf(id)] || '').pipe(catchError(() => of(null)))
            : this.photoService.getMedia(id).pipe(catchError(() => of(null)))
        );
        
        forkJoin(requests$).subscribe({
          next: (photos: (Photo | null)[]) => {
            mediaItems = buildMediaItems(photos, 'album');
            startPresentationWithItems(mediaItems, buildNavQueryParams());
          },
          error: (err: unknown) => {
            console.error('Failed to load album media for presentation', err);
            alert('Failed to start presentation mode.');
          }
        });
      });
    } else if ((searchParam || this.route.snapshot.queryParams['tag'] || this.route.snapshot.queryParams['dateRange']) && !isSharedMedia) {
      // Came from search context (owner gallery, tag filter, or date range) - use search results
      const term = searchParam || '';
      let scope = searchScopeParam as SearchScope;

      if (this.route.snapshot.queryParams['tag']) {
        scope = 'tags';
      } else if (this.route.snapshot.queryParams['dateRange'] && (searchScopeParam === 'all' || !term)) {
         // If there is a date range, ensure scope is set to 'date' even if it was passed as 'all' or empty term
         scope = 'date';
      }

      const dateRange = this.route.snapshot.queryParams['dateRange'];

      this.searchService.searchMedia(term, scope, 200, 0, dateRange).subscribe({
        next: (response: any) => {
          mediaItems = response.results.map((p: any) => ({
            id: p.id,
            path: `${this.photoService['API_BASE_URL']}/media/${p.id}/original`,
            filename: p.filename || '',
            mediaType: p.mediaType || 'photo'
          }));
          startPresentationWithItems(mediaItems, buildNavQueryParams());
        },
        error: (err: unknown) => {
          console.error('Failed to load search results for presentation', err);
          alert('Failed to start presentation mode.');
        }
      });
    } else if (isSharedMedia && shareToken) {
      // Public shared media - use public share endpoint
      this.photoService.listPublicShareMedia(shareToken, 100, 0).subscribe({
        next: (response: any) => {
          mediaItems = response.media.map((p: any) => ({
            id: p.ID || p.id,
            path: `${this.photoService.getPublicShareOriginalUrl(shareToken, p.Path || p.path || '')}`,
            filename: p.Filename || p.filename || '',
            mediaType: (p.MediaType || p.mediaType || 'photo')
          }));
          startPresentationWithItems(mediaItems, buildNavQueryParams());
        },
        error: (err) => {
          console.error('Failed to load public share media for presentation', err);
          alert('Failed to start presentation mode.');
        }
      });
    } else if (isSharedMedia) {
      // Authenticated shared media
      this.photoService.listSharedMedia(100, 0).subscribe({
        next: (response: any) => {
          mediaItems = response.items.map((p: any) => ({
            id: p.id || p.ID,
            path: `${this.photoService['API_BASE_URL']}/media/shared/${(p.id || p.ID)}/original`,
            filename: p.filename || '',
            mediaType: (p.mediaType || 'photo')
          }));
          startPresentationWithItems(mediaItems, buildNavQueryParams());
        },
        error: (err) => {
          console.error('Failed to load shared media for presentation', err);
          alert('Failed to start presentation mode.');
        }
      });
    } else {
      // No search or album context - fetch gallery items around the current page
      const galleryPage = this.galleryState.getCurrentPage() || 1;
      const limit = 20;  // Same as gallery component
      const offset = (galleryPage - 1) * limit;
      
      this.photoService.listMedia(200, offset).subscribe({
        next: (response: any) => {
          mediaItems = response.photos.map((p: any) => ({
            id: p.id,
            path: `${this.photoService['API_BASE_URL']}/media/${p.id}/original`,
            filename: p.filename,
            mediaType: p.mediaType || 'photo'
          }));
          startPresentationWithItems(mediaItems, {});
        },
        error: (err) => {
          console.error('Failed to load media for presentation', err);
          alert('Failed to start presentation mode.');
        }
      });
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
