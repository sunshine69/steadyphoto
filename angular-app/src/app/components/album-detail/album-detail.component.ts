import { Component, inject, OnInit, OnDestroy, ChangeDetectorRef, NgZone } from '@angular/core';

import { ActivatedRoute, RouterModule, Router } from '@angular/router';
import { HttpClient } from '@angular/common/http';
import { AlbumService } from '../../services/album.service';
import { PhotoService } from '../../services/photo.service';
import { PresentationService, MediaItem } from '../../services/presentation.service';
import { Photo, ListPhotosResponse } from '../../models/photo.model';
import { Album } from '../../models/album.model';
import { ShareTriggerService } from '../../services/share-trigger.service';
import { PhotoCardComponent } from '../photo-card/photo-card.component';
import { FormsModule } from '@angular/forms';

@Component({
    selector: 'app-album-detail',
    imports: [RouterModule, PhotoCardComponent, FormsModule],
    template: `
    <div class="container mt-4">
      <!-- Remove Media Mode Banner -->
      @if (selectedRemovePhotoIds.size > 0 && !loading) {
        <div class="alert alert-danger d-flex align-items-center justify-content-between mb-3">
          <span>{{ selectedRemovePhotoIds.size }} items selected for removal from "{{ albumName }}"</span>
          <div class="d-flex gap-2">
            <button class="btn btn-sm btn-light" (click)="executeRemoveSelected()">Remove Selected</button>
            <button class="btn btn-sm btn-outline-light" (click)="clearSelection()">Cancel Selection</button>
          </div>
        </div>
      }
    
      <div class="d-flex justify-content-between align-items-center mb-4">
        <h2>{{ albumName }}</h2>
        <div class="d-flex align-items-center gap-2">
          <button class="btn btn-outline-secondary" style="margin-right: 20px !important;" (click)="goBack()">Back to Albums</button>
          @if (totalAlbumPhotos > 0 && !loading) {
            <button
              class="btn btn-success"
              (click)="openAddMediaModal()">
              ➕ Add Media
            </button>
          }
          @if (photos.length > 0 && !loading) {
            <button
              class="btn btn-warning"
              (click)="startPresentationFromAlbum()"
              [disabled]="photos.length === 0">
              🎬 Presentation Mode
            </button>
          }
          @if (photos.length > 0 && !loading) {
            <button
              class="btn btn-info"
              (click)="shareAlbum()">
              📤 Share
            </button>
          }
        </div>
      </div>
    
      <!-- Add Media Modal -->
      @if (showAddMediaModal) {
        <div class="modal-overlay">
          <!-- New wrapper for flex layout -->
          <div class="add-media-modal-wrapper bg-dark border rounded p-4">
            <div class="d-flex justify-content-between align-items-center mb-3 modal-header-section">
              <h5 class="text-white m-0">Add Media to Album</h5>
              <button type="button" class="btn-close btn-close-white" (click)="closeAddMediaModal()"></button>
            </div>
            <!-- Add Media Toolbar -->
            <div class="d-flex justify-content-between align-items-center mb-3 p-2 rounded modal-toolbar">
              <span class="text-white">Select items to add ({{ selectedAddMediaIds.size }} selected)</span>
              <button
                class="btn btn-success btn-sm"
                (click)="executeAddMedia()">
                Done ({{ selectedAddMediaIds.size }})
              </button>
            </div>
            <!-- Media Grid Container -->
            <div class="media-grid-container">
              @if (allPhotos.length === 0 && !loadingAllPhotos) {
                <div class="text-center text-muted py-4">
                  No media items available to add.
                </div>
              }
              @if (allPhotos.length > 0) {
                <div class="grid-container">
                  @for (photo of allPhotos; track photo) {
                    <div class="grid-item position-relative">
                      <!-- Selection Checkbox -->
                      <button type="button"
                        class="selection-checkbox-btn-add"
                        [class.selected]="isAddMediaSelected(photo.id)"
                        (click)="toggleAddMediaSelection(photo.id); $event.stopPropagation()">
                        @if (isAddMediaSelected(photo.id)) {
                          <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="white" viewBox="0 0 16 16">
                            <path d="M12.736 3.97a.733.733 0 0 1 1.047 0c.286.289.29.756.01 1.05L7.88 12.01a.733.733 0 0 1-1.065.02L3.217 8.384a.757.757 0 0 1 0-1.06.733.733 0 0 1 1.047 0l3.052 3.093 5.777-6.817z"/>
                          </svg>
                        }
                      </button>
                      <app-photo-card [photo]="photo" (cardClick)="onPhotoClick(photo.id)"></app-photo-card>
                    </div>
                  }
                </div>
              }
            </div>
            <!-- Pagination for Add Media Modal -->
            @if (!loadingAllPhotos && totalLibraryItems > addMediaLimit) {
              <div class="pagination-controls mt-3">
                <button class="btn btn-outline-primary me-2" [disabled]="addMediaOffset === 0" (click)="changeAddMediaPage(-1)">Previous</button>
                <div class="pagination-jump d-flex align-items-center mx-3">
                  <span class="me-2 text-nowrap">Page</span>
                  <input type="number" class="form-control form-control-sm jump-input me-2" [(ngModel)]="addMediaJumpInput" (keyup.enter)="onAddMediaJumpToPage()" min="1" [max]="totalLibraryPages">
                  <button class="btn btn-primary btn-sm jump-btn" type="button" (click)="onAddMediaJumpToPage()">Go</button>
                  <span class="ms-2 text-nowrap">of {{ totalLibraryPages }}</span>
                </div>
                <button class="btn btn-outline-primary ms-2" [disabled]="addMediaOffset + addMediaLimit >= totalLibraryItems" (click)="changeAddMediaPage(1)">Next</button>
              </div>
            }
            <!-- Loading Spinner for Add Media -->
            @if (loadingAllPhotos) {
              <div class="loading-spinner d-flex justify-content-center my-3">
                <div class="spinner-border text-primary" role="status">
                  <span class="visually-hidden">Loading...</span>
                </div>
              </div>
            }
          </div> <!-- End add-media-modal-wrapper -->
        </div>
        } <!-- End modal-overlay -->
    
        <!-- Media Grid -->
        @if (!loading && totalAlbumPhotos > 0) {
          <div>
            @if (photos.length > 0) {
              <div class="grid-container">
                @for (photo of photos; track photo; let i = $index) {
                  <div class="grid-item position-relative">
                    <!-- Selection Checkbox - ALWAYS VISIBLE, placed BEFORE photo-card -->
                    <button type="button"
                      class="selection-checkbox-btn-remove-mode"
                      [class.selected]="isPhotoSelected(photo.id)"
                      (click)="toggleSelection(photo.id); $event.stopPropagation()">
                      <!-- Empty circle when not selected, checkmark when selected -->
                      @if (!isPhotoSelected(photo.id)) {
                        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="none" stroke="#fff" stroke-width="2" viewBox="0 0 16 16">
                          <circle cx="8" cy="8" r="7"/>
                        </svg>
                      }
                      @if (isPhotoSelected(photo.id)) {
                        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="#dc3545" stroke="#fff" stroke-width="2" viewBox="0 0 16 16">
                          <circle cx="8" cy="8" r="7"/>
                          <path d="M12.736 3.97a.733.733 0 0 1 1.047 0c.286.289.29.756.01 1.05L7.88 12.01a.733.733 0 0 1-1.065.02L3.217 8.384a.757.757 0 0 1 0-1.06.733.733 0 0 1 1.047 0l3.052 3.093 5.777-6.817z"/>
                        </svg>
                      }
                    </button>
                    <app-photo-card [photo]="photo" (cardClick)="onPhotoClick(photo.id)"></app-photo-card>
                  </div>
                }
              </div>
            }
            <!-- Pagination for Album Grid -->
            @if (totalAlbumPhotos > albumLimit) {
              <div class="pagination-controls mt-4">
                <button class="btn btn-outline-primary me-2" [disabled]="albumOffset === 0" (click)="changeAlbumPage(-1)">Previous</button>
                <div class="pagination-jump d-flex align-items-center mx-3">
                  <span class="me-2 text-nowrap">Page</span>
                  <input type="number" class="form-control form-control-sm jump-input me-2" [(ngModel)]="albumJumpInput" (keyup.enter)="onAlbumJumpToPage()" min="1" [max]="totalAlbumPages">
                  <button class="btn btn-primary btn-sm jump-btn" type="button" (click)="onAlbumJumpToPage()">Go</button>
                  <span class="ms-2 text-nowrap">of {{ totalAlbumPages }}</span>
                </div>
                <button class="btn btn-outline-primary ms-2" [disabled]="albumOffset + albumLimit >= totalAlbumPhotos" (click)="changeAlbumPage(1)">Next</button>
              </div>
            }
          </div>
        }
    
        <!-- Empty State -->
        @if (!loading && photos.length === 0) {
          <div class="empty-state py-5 text-center border rounded bg-light">
            <p class="lead text-muted">This album is empty.</p>
          </div>
        }
    
        <!-- Loading Spinner -->
        @if (loading) {
          <div class="loading-spinner d-flex justify-content-center my-5">
            <div class="spinner-border text-primary" role="status">
              <span class="visually-hidden">Loading...</span>
            </div>
          </div>
        }
      </div>
    `,
    styles: [`
    .grid-container {
      display: grid;
      grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
      gap: 1.5rem;
      width: 100%;
    }

    /* Modal-specific grid container - overrides the main one */
    .add-media-modal-wrapper .grid-container {
      display: grid !important;
      grid-template-columns: repeat(auto-fill, minmax(200px, 1fr)) !important;
      gap: 1.5rem !important;
      width: 100% !important;
    }
    .grid-item {
      position: relative;
      height: 100%;
    }

    /* Remove mode selection button - matches photo-list pattern */
    .selection-checkbox-btn-remove-mode {
      position: absolute; 
      top: 8px; 
      left: 8px; 
      z-index: 20; 
      width: 26px; 
      height: 26px; 
      border-radius: 50%; 
      background: rgba(255, 255, 255, 0.8); 
      backdrop-filter: blur(4px); 
      border: 1px solid rgba(0,0,0,0.1); 
      display: flex; 
      align-items: center; 
      justify-content: center; 
      padding: 0; 
      cursor: pointer; 
      transition: all 0.2s ease; 
      box-shadow: 0 2px 4px rgba(0,0,0,0.15);
    }

    .selection-checkbox-btn-remove-mode:hover { 
      transform: scale(1.1); 
      background: #fff; 
    }

    .selection-checkbox-btn-remove-mode.selected { 
      background: #dc3545; 
      border-color: #a71d2a;
    }

    .selection-checkbox-btn-add {
      position: absolute; top: 8px; left: 8px; z-index: 20; width: 26px; height: 26px; border-radius: 50%; background: rgba(255, 255, 255, 0.8); backdrop-filter: blur(4px); border: 1px solid rgba(0,0,0,0.1); display: flex; align-items: center; justify-content: center; padding: 0; cursor: pointer; transition: all 0.2s ease; box-shadow: 0 2px 4px rgba(0,0,0,0.15);
    }

    .selection-checkbox-btn-add:hover { transform: scale(1.1); background: #fff; }
    .selection-checkbox-btn-add.selected { background: #198754; border-color: #146c43; } /* Green for add mode */

    .empty-state {
        margin-top: 2rem;
    }

    /* Modal Overlay Styles */
    .modal-overlay {
      position: fixed;
      top: 0; left: 0; width: 100%; height: 100%;
      background-color: rgba(0, 0, 0, 0.85); /* Darker for better focus */
      display: flex; align-items: center; justify-content: center;
      z-index: 2000; /* High z-index to prevent background issues */
    }

    .modal-content {
      min-width: 400px; max-width: 90%;
      pointer-events: auto; /* Ensure interaction works within modal */
    }

    /* Modal Wrapper for Flex Layout */
    .add-media-modal-wrapper {
      width: 90vw; 
      max-width: 1200px;
      height: 90vh; /* Fixed height to allow internal scrolling */
      display: flex;
      flex-direction: column;
      overflow: hidden; /* Prevent outer scroll */
    }

    .modal-header-section {
      flex-shrink: 0; /* Don't shrink the header */
    }

    .modal-toolbar {
      flex-shrink: 0; /* Don't shrink the toolbar */
      background-color: #495057 !important;
    }

    /* The Grid Container - takes all remaining space and scrolls internally */
    .media-grid-container {
      flex-grow: 1;
      overflow-y: auto; /* Scroll vertically within this area only */
      padding-right: 8px; /* Space for scrollbar */
      width: 100%; /* Ensure it fills the wrapper width */
    }

    /* Pagination Controls - matching photo-list pattern */
    .pagination-controls { 
      display: flex; 
      justify-content: center; 
      align-items: center; 
      margin-top: 2rem; 
      padding-bottom: 1rem; 
    }
    .jump-input { width: 60px !important; text-align: center; }

  `]
})
export class AlbumDetailComponent implements OnInit, OnDestroy {
  private route = inject(ActivatedRoute);
  private router = inject(Router);
  private albumService = inject(AlbumService);
  private photoService = inject(PhotoService);
  private presentationService = inject(PresentationService);
  private cdr = inject(ChangeDetectorRef);
  private ngZone = inject(NgZone);
  private http = inject(HttpClient);
  private shareTrigger = inject(ShareTriggerService);

  albumName: string = 'Loading...';
  photos: Photo[] = [];
  loading = true;
  albumId!: string;

  // Track whether we're viewing a shared album (for path construction)
  isSharedAlbumView: boolean = false;

  // Album Grid Pagination State
  totalAlbumPhotos = 0;
  albumLimit = 20;
  albumOffset = 0;
  albumJumpInput: number | null = null;

  get totalAlbumPages(): number { return Math.ceil(this.totalAlbumPhotos / this.albumLimit) || 1; }

  // Add Media Modal State
  showAddMediaModal = false;
  allPhotos: Photo[] = [];
  selectedAddMediaIds: Set<string> = new Set();
  loadingAllPhotos = false;

  // Add Media Pagination State
  totalLibraryItems = 0;
  addMediaLimit = 20;
  addMediaOffset = 0;
  addMediaJumpInput: number | null = null;
  currentAddMediaPage = 1;
  existingAlbumIds: Set<string> = new Set(); // IDs already in this album

  get totalLibraryPages(): number { return Math.ceil(this.totalLibraryItems / this.addMediaLimit) || 1; }

  // Remove Media Mode State
  selectedRemovePhotoIds: Set<string> = new Set();

  /** 
   * Fuzzy normalization for IDs: removes non-alphanumeric chars, lowercase.
   * This ensures "abc-123" matches "ABC_123".
   */
  private normalizeId(id: string): string {
    if (!id) return '';
    return id.toString().toLowerCase().replace(/[^a-z0-9]/g, '');
  }

  ngOnInit(): void {
    const id = this.route.snapshot.paramMap.get('id');
    
    // Check if navigating from shared media (source=shared query param)
    const isSharedAlbum = this.route.snapshot.queryParams['source'] === 'shared';
    
    if (id) {
      this.albumId = id;
      this.loadAlbumContent(id, isSharedAlbum);
    } else {
      this.router.navigate(['/albums']);
    }
  }

  ngOnDestroy(): void {
    // Clean up presentation state when leaving album detail
  }

  private loadAlbumContent(id: string, isShared = false): void {
    this.loading = true;
    this.isSharedAlbumView = isShared;
    
    if (isShared) {
      // Use shared album endpoint for sharee access
      this.http.get<any>(`${this.photoService['API_BASE_URL']}/albums/shared/${id}`).subscribe({
        next: (albumResponse) => {
          const albumData = albumResponse.Album || albumResponse;
          this.albumName = albumData.name || 'Untitled Album';
          this.fetchMedia(id, isShared);
        },
        error: (err: any) => {
          console.error('Error loading shared album metadata', err);
          this.loading = false;
          this.albumName = 'Error loading album';
        }
      });
    } else {
      // Use ownership-checking album endpoint
      this.albumService.getAlbum(id).subscribe({
        next: (album: Album) => {
          this.albumName = album.name || 'Untitled Album';
          this.fetchMedia(id);
        },
        error: (err: any) => {
          console.error('Error loading album metadata', err);
          this.loading = false;
          this.albumName = 'Error loading album';
        }
      });
    }
  }

  private fetchMedia(id: string, isShared = false): void {
    // Load first page with pagination params
    this.loadAlbumPageFromAPI(id, 0, isShared);
  }

  private loadAlbumPageFromAPI(id: string, offset: number, isShared = false): void {
    this.loading = true;
    
    const limit = this.albumLimit;
    const url = isShared 
      ? `${this.photoService['API_BASE_URL']}/albums/shared/${id}/media?limit=${limit}&offset=${offset}`
      : `${this.photoService['API_BASE_URL']}/albums/${id}/media?limit=${limit}&offset=${offset}`;
    
    this.http.get<any>(url).subscribe({
      next: (response) => {
        const rawPhotos: any[] = response.items || response.media || [];
        const totalItems = response.total ?? response.totalItems ?? 0;
        
        this.photos = rawPhotos.map((p: any) => this.normalizePhoto(p, isShared));
        this.totalAlbumPhotos = totalItems;
        this.albumOffset = offset;
        this.loading = false;
      },
      error: (err: any) => {
        console.error('Error loading album media', err);
        this.loading = false;
      }
    });
  }

  changeAlbumPage(dir: number): void {
    const newOffset = this.albumOffset + (dir * this.albumLimit);
    
    // Validate offset bounds
    if (newOffset < 0 || newOffset >= this.totalAlbumPhotos) return;
    
    if (!this.albumId) return;
    
    this.loadAlbumPageFromAPI(this.albumId, newOffset, this.isSharedAlbumView);
    window.scrollTo(0, 0);
  }

  onAlbumJumpToPage(): void { 
    if (this.albumJumpInput && this.albumJumpInput >= 1 && this.albumJumpInput <= this.totalAlbumPages) {
      const targetOffset = (this.albumJumpInput - 1) * this.albumLimit;
      
      if (!this.albumId) return;
      
      this.loadAlbumPageFromAPI(this.albumId, targetOffset, this.isSharedAlbumView);
      window.scrollTo(0, 0);
    } 
  }

  /**
   * Normalizes raw photo data from the API to ensure thumbnailUrl is properly constructed.
   */
  private normalizePhoto(p: any, isShared = false): Photo {
    const id = p.ID ?? p.id ?? '';
    
    let mediaType: 'photo' | 'video' | undefined;
    const rawMediaType = p.MediaType || p.mediaType || p.media_type;
    if (rawMediaType) {
      const mt = String(rawMediaType).toLowerCase();
      if (mt === 'video') {
        mediaType = 'video';
      } else {
        mediaType = 'photo';
      }
    }

    // Normalize path to API URL endpoint instead of raw file system path
    const apiBaseUrl = this.photoService['API_BASE_URL'];
    let normalizedPath = '';
    if (id) {
      if (isShared) {
        // For shared albums, construct the /original endpoint from the ID
        normalizedPath = `${apiBaseUrl}/media/shared/${id}/original`;
      } else {
        // For regular albums, use whatever path the backend provides
        normalizedPath = p.Path ?? p.path ?? '';
      }
    }

    return {
      id: id,
      path: normalizedPath,
      thumbnailUrl: isShared 
        ? (id ? `${apiBaseUrl}/media/shared/${id}/thumb` : '')
        : (id ? `${apiBaseUrl}/media/${id}/thumb` : ''),
      filename: p.Filename ?? p.filename ?? '',
      captured_at: p.capturedAt ?? p.capturedAt ?? '',
      width: p.Width ?? p.width,
      height: p.Height ?? p.height,
      size: p.Size ?? p.size,
      type: p.Type ?? p.type,
      mediaType: mediaType || 'photo',
      metadata: p.Metadata ? this.photoService.normalizeMetadata(p.Metadata) : undefined,
      videoMetadata: p.VideoMetadata ? {
        duration: p.VideoMetadata.Duration ?? p.VideoMetadata.duration,
        bitrate: p.VideoMetadata.Bitrate ?? p.VideoMetadata.bitrate,
        video_codec: p.VideoMetadata.VideoCodec ?? p.VideoMetadata.video_codec,
        audio_codec: p.VideoMetadata.AudioCodec ?? p.VideoMetadata.audio_codec,
        frame_rate: p.VideoMetadata.FrameRate ?? p.VideoMetadata.frame_rate,
      } : undefined,
      tags: p.Tags ?? p.tags ?? ''
    };
  }

  goBack(): void { this.router.navigate(['/albums']); }

  shareAlbum(): void {
    if (!this.albumId) return;
    this.shareTrigger.open(this.albumId, 'album');
  }

  onPhotoClick(id: string): void { 
    const ids = this.photos.map(p => p.id).join(',');
    
    // Pass source=shared if viewing from a shared album, so PhotoDetailComponent uses the correct endpoint
    const isSharedAlbum = this.route.snapshot.queryParams['source'] === 'shared';
    
    if (isSharedAlbum) {
      this.router.navigate(['/photos', id], { queryParams: { albumIds: ids, source: 'shared' } }); 
    } else {
      this.router.navigate(['/photos', id], { queryParams: { albumIds: ids } }); 
    }
  }

  startPresentationFromAlbum(): void {
    // === DEBUGGING: Track presentation from album detail ===
    
    
    
    
    
    
    
    // Show first 3 photo IDs to verify data integrity
    if (this.photos.length > 0) {
      
      
      
    }
    ;

    if (this.photos.length === 0 || this.loading) {
      
      return;
    }

    const isSharedAlbum = this.route.snapshot.queryParams['source'] === 'shared';
    
    // Convert photos to MediaItem format for presentation service - use correct URL based on album type
    const apiBaseUrl = this.photoService['API_BASE_URL'];
    const mediaItems: MediaItem[] = this.photos.map(p => ({
      id: p.id,
      path: isSharedAlbum 
        ? `${apiBaseUrl}/media/shared/${p.id}/original`  // Shared album → use shared endpoint
        : `${apiBaseUrl}/media/${p.id}/original`,          // Regular album → use regular endpoint
      filename: p.filename,
      mediaType: p.mediaType || 'photo'
    }));

    console.group('🎬 FINAL ITEM CHECK');
    
    
    ;

    if (mediaItems.length > 0) {
      
      this.presentationService.open(mediaItems, 0);
      this.router.navigate(['/presentation']);
    } else {
      console.error('❌ Presentation blocked - mediaItems is empty!');
      console.error('   This should not happen if photos.length > 0');
      alert('No items available for presentation.');
    }
  }

  // Add Media Modal Methods - NOW PAGINATED
  openAddMediaModal(): void {
    if (!this.albumId) return;
    
    // Build set of normalized IDs from existing photos for fuzzy matching
    this.existingAlbumIds = new Set(this.photos?.map(p => this.normalizeId(p.id)) || []); 
    
    
    // Reset pagination and fetch first page
    this.addMediaOffset = 0;
    this.currentAddMediaPage = 1;
    this.selectedAddMediaIds.clear();
    this.loadingAllPhotos = true;
    this.showAddMediaModal = true;
    
    this._fetchAddMediaPage();
  }

  private _fetchAddMediaPage(): void {
    if (!this.albumId) return;
    
    this.loadingAllPhotos = true;
    
    // Fetch all library items to filter out existing ones (client-side filtering for now)
    const limit = 100;
    let offset = 0;
    let allLibraryPhotos: Photo[] = [];
    let totalItemsFromApi = 0;

    const fetchPage = (): Promise<void> => {
      return new Promise((resolve) => {
        this.photoService.listMedia(limit, offset).subscribe({
          next: (response) => {
            if (totalItemsFromApi === 0 && response.total !== undefined) {
              totalItemsFromApi = response.total;
            }

            // Filter out photos already in this album using Fuzzy ID matching
            const filteredPhotos: Photo[] = [];
            
            for (const photo of response.photos) {
              const normId = this.normalizeId(photo.id);
              
              if (!this.existingAlbumIds.has(normId)) {
                filteredPhotos.push(photo); // Keep it! It's NOT in the album.
              }
            }
            
            allLibraryPhotos.push(...filteredPhotos);
            offset += limit;

            resolve();
          },
          error: (err) => {
            console.error('Error loading media for add modal', err);
            alert('Failed to load media items');
            resolve();
          }
        });
      });
    };

    // Fetch all pages first, then paginate in UI
    const fetchAll = async () => {
      try {
        do {
          await fetchPage();
        } while (offset < totalItemsFromApi && allLibraryPhotos.length > 0); 
        
        this.totalLibraryItems = allLibraryPhotos.length;
        
        // Now load first page for display
        this._loadAddMediaPage(allLibraryPhotos);
      } catch (error) {
        console.error('Error in fetchAll:', error);
      } finally {
        this.loadingAllPhotos = false;
      }
    };

    fetchAll();
  }

  private _loadAddMediaPage(allPhotos: Photo[]): void {
    const start = this.addMediaOffset;
    const end = Math.min(start + this.addMediaLimit, allPhotos.length);
    this.allPhotos = allPhotos.slice(start, end);
    this.cdr.detectChanges();
  }

  changeAddMediaPage(dir: number): void {
    this.addMediaOffset += (dir * this.addMediaLimit);
    // Reload and slice to new page
    if (!this.albumId) return;
    
    this.loadingAllPhotos = true;
    
    const limit = 100;
    let offset = 0;
    let allLibraryPhotos: Photo[] = [];
    let totalItemsFromApi = 0;

    const fetchPage = (): Promise<void> => {
      return new Promise((resolve) => {
        this.photoService.listMedia(limit, offset).subscribe({
          next: (response) => {
            if (totalItemsFromApi === 0 && response.total !== undefined) {
              totalItemsFromApi = response.total;
            }

            const filteredPhotos: Photo[] = [];
            
            for (const photo of response.photos) {
              const normId = this.normalizeId(photo.id);
              
              if (!this.existingAlbumIds.has(normId)) {
                filteredPhotos.push(photo);
              }
            }
            
            allLibraryPhotos.push(...filteredPhotos);
            offset += limit;

            resolve();
          },
          error: (err) => {
            console.error('Error loading media for add modal', err);
            resolve();
          }
        });
      });
    };

    const fetchAll = async () => {
      try {
        do {
          await fetchPage();
        } while (offset < totalItemsFromApi && allLibraryPhotos.length > 0); 
        
        this.totalLibraryItems = allLibraryPhotos.length;
        this._loadAddMediaPage(allLibraryPhotos);
      } catch (error) {
        console.error('Error:', error);
      } finally {
        this.loadingAllPhotos = false;
      }
    };

    fetchAll();
  }

  onAddMediaJumpToPage(): void { 
    if (this.addMediaJumpInput && this.addMediaJumpInput >= 1 && this.addMediaJumpInput <= this.totalLibraryPages) {
      this.addMediaOffset = (this.addMediaJumpInput - 1) * this.addMediaLimit;
      // Reload and jump to page
      if (!this.albumId) return;
      
      this.loadingAllPhotos = true;
      
      const limit = 100;
      let offset = 0;
      let allLibraryPhotos: Photo[] = [];
      let totalItemsFromApi = 0;

      const fetchPage = (): Promise<void> => {
        return new Promise((resolve) => {
          this.photoService.listMedia(limit, offset).subscribe({
            next: (response) => {
              if (totalItemsFromApi === 0 && response.total !== undefined) {
                totalItemsFromApi = response.total;
              }

              const filteredPhotos: Photo[] = [];
              
              for (const photo of response.photos) {
                const normId = this.normalizeId(photo.id);
                
                if (!this.existingAlbumIds.has(normId)) {
                  filteredPhotos.push(photo);
                }
              }
              
              allLibraryPhotos.push(...filteredPhotos);
              offset += limit;

              resolve();
            },
            error: (err) => {
              console.error('Error loading media for add modal', err);
              resolve();
            }
          });
        });
      };

      const fetchAll = async () => {
        try {
          do {
            await fetchPage();
          } while (offset < totalItemsFromApi && allLibraryPhotos.length > 0); 
          
          this.totalLibraryItems = allLibraryPhotos.length;
          this._loadAddMediaPage(allLibraryPhotos);
        } catch (error) {
          console.error('Error:', error);
        } finally {
          this.loadingAllPhotos = false;
        }
      };

      fetchAll();
    } 
  }

  closeAddMediaModal(): void {
    this.showAddMediaModal = false;
    this.allPhotos = [];
    this.selectedAddMediaIds.clear();
    this.existingAlbumIds.clear();
  }

  toggleAddMediaSelection(id: string): void {
    if (this.selectedAddMediaIds.has(id)) { 
      this.selectedAddMediaIds.delete(id); 
    } else { 
      this.selectedAddMediaIds.add(id); 
    }
  }

  isAddMediaSelected(id: string): boolean { 
    return this.selectedAddMediaIds.has(id); 
  }

  executeAddMedia(): void {
    if (this.selectedAddMediaIds.size === 0) return;

    const mediaIds = Array.from(this.selectedAddMediaIds);
    
    this.albumService.addMediaToAlbum(this.albumId, mediaIds).subscribe({
      next: () => {
        alert(`Successfully added ${mediaIds.length} item(s) to "${this.albumName}"`);
        this.closeAddMediaModal();
        // Reload album content with fresh pagination state
        this.albumOffset = 0;
        this.fetchMedia(this.albumId); 
      },
      error: (err: any) => {
        console.error('Error adding media to album', err);
        alert(`Failed to add media - ${err.message || 'Unknown error'}`);
      }
    });
  }

  // Selection Methods (always active)
  toggleSelection(id: string): void {
    if (this.selectedRemovePhotoIds.has(id)) { 
      this.selectedRemovePhotoIds.delete(id); 
    } else { 
      this.selectedRemovePhotoIds.add(id); 
    }
  }

  isPhotoSelected(id: string): boolean { 
    return this.selectedRemovePhotoIds.has(id); 
  }

  executeRemoveSelected(): void {
    if (this.selectedRemovePhotoIds.size === 0) {
      alert('Please select items to remove first.');
      return;
    }

    const mediaIds = Array.from(this.selectedRemovePhotoIds);
    
    if (!confirm(`Are you sure you want to remove ${mediaIds.length} item(s) from "${this.albumName}"?`)) {
      return;
    }

    this.albumService.bulkRemoveMediaFromAlbum(this.albumId, mediaIds).subscribe({
      next: () => {
        alert(`Successfully removed ${mediaIds.length} item(s) from "${this.albumName}"`);
        this.selectedRemovePhotoIds.clear(); // Clear selection after removal
        // Reload album content with fresh pagination state
        this.albumOffset = 0;
        this.fetchMedia(this.albumId); 
      },
      error: (err: any) => {
        console.error('Error removing media from album', err);
        alert(`Failed to remove media - ${err.message || 'Unknown error'}`);
      }
    });
  }

  clearSelection(): void { 
    this.selectedRemovePhotoIds.clear(); 
  }
}
