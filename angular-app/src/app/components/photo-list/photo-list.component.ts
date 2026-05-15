import { Component, Inject, OnInit, OnDestroy, Optional } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Subscription } from 'rxjs';
import { PhotoService } from '../../services/photo.service';
import { GalleryStateService } from '../../services/gallery-state.service';
import { SearchService } from '../../services/search.service';
import { Photo, ListPhotosResponse } from '../../models/photo.model';
import { Router, RouterModule, ActivatedRoute } from '@angular/router';
import { PhotoCardComponent } from '../photo-card/photo-card.component';
import { AlbumService } from '../../services/album.service';
import { Album } from '../../models/album.model';

@Component({
  selector: 'app-photo-list',
  standalone: true,
  imports: [CommonModule, RouterModule, PhotoCardComponent, FormsModule],
  template: `
    <div class="photo-list-container">
      <!-- Bulk Action Toolbar -->
      <div class="bulk-action-toolbar mb-3" *ngIf="selectedPhotoIds.size > 0">
        <div class="d-flex align-items-center justify-content-between bg-white p-2 rounded shadow-sm border">
          <div>
            <span class="badge bg-primary me-2">{{ selectedPhotoIds.size }}</span> items selected
          </div>
          <div class="actions d-flex gap-2">
             <!-- Simplified Album Selection for now -->
             <div class="d-flex align-items-center gap-2">
                <select class="form-select form-select-sm w-auto" [(ngModel)]="targetAlbumId">
                  <option [ngValue]="undefined">Add to album...</option>
                  <option *ngFor="let album of albums" [value]="album.id">{{ album.name }}</option>
                </select>
                <button class="btn btn-sm btn-primary" (click)="addToSelectedAlbum()" [disabled]="!targetAlbumId">Apply</button>
             </div>
            <button class="btn btn-sm btn-outline-danger" (click)="clearSelection()">Cancel</button>
          </div>
        </div>
      </div>

      <!-- Active Tag Filter Display -->
      <div *ngIf="activeTagFilter && selectedPhotoIds.size === 0" class="alert alert-info d-flex align-items-center justify-content-between mb-3">
        <span>Showing items with tag: <strong class="text-uppercase">#{{ activeTagFilter }}</strong></span>
        <button (click)="clearTagFilter()" class="btn btn-sm btn-outline-danger">Clear Filter</button>
      </div>

      <!-- Empty State -->
      <div class="empty-state" *ngIf="!loading && (!photos || photos.length === 0)">
        <p *ngIf="!currentSearchTerm && !activeTagFilter">No photos found. Start by importing your photo library.</p>
        <p *ngIf="currentSearchTerm && activeTagFilter">No items match search "{{currentSearchTerm}}" and tag "#{{activeTagFilter}}"</p>
      </div>
      
      <!-- Photo Grid -->
      <div class="grid-container" *ngIf="!loading && photos && photos.length > 0">
        <div class="grid-item position-relative" *ngFor="let photo of photos">
          <!-- Discrete Selection Button at Top Left Corner -->
          <button type="button" 
                  class="selection-checkbox-btn" 
                  [class.selected]="isPhotoSelected(photo.id)"
                  (click)="toggleSelection(photo.id); $event.stopPropagation()">
            <svg *ngIf="isPhotoSelected(photo.id)" xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="white" viewBox="0 0 16 16">
              <path d="M12.736 3.97a.733.733 0 0 1 1.047 0c.286.289.29.756.01 1.05L7.88 12.01a.733.733 0 0 1-1.065.02L3.217 8.384a.757.757 0 0 1 0-1.06.733.733 0 0 1 1.047 0l3.052 3.093 5.777-6.817z"/>
            </svg>
          </button>

          <app-photo-card [photo]="photo" (cardClick)="onPhotoClick(photo.id)"></app-photo-card>
        </div>
      </div>

      <!-- Pagination -->
       <!-- ... same pagination code as before ... -->
      <div class="pagination-controls" *ngIf="!loading && totalPhotos > limit">
        <button class="btn btn-outline-primary me-2" [disabled]="offset === 0" (click)="changePage(-1)">Previous</button>
        <div class="pagination-jump d-flex align-items-center mx-3">
          <span class="me-2 text-nowrap">Page</span>
          <input type="number" class="form-control form-control-sm jump-input me-2" [(ngModel)]="jumpPageInput" (keyup.enter)="onJumpToPage()" min="1" [max]="totalPages">
          <button class="btn btn-primary btn-sm jump-btn" type="button" (click)="onJumpToPage()">Go</button>
          <span class="ms-2 text-nowrap">of {{ totalPages }}</span>
        </div>
        <button class="btn btn-outline-primary ms-2" [disabled]="offset + limit >= totalPhotos" (click)="changePage(1)">Next</button>
      </div>

      <!-- Loading Spinner -->
      <div class="loading-spinner" *ngIf="loading">
        <div class="spinner-border text-primary" role="status"><span class="visually-hidden">Loading...</span></div>
      </div>
    </div>
  `,
  styles: [`
    .photo-list-container { padding: 1rem; width: 100%; }
    .grid-container { display: grid; grid-template-columns: repeat(auto-fill, minmax(180px, 1fr)); gap: 1.5rem; width: 100%; }
    .grid-item { position: relative; height: 100%; }

    /* New discrete selection button style */
    .selection-checkbox-btn {
      position: absolute;
      top: 8px;
      left: 8px;
      z-index: 20; /* Must be higher than photo card and any overlays */
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

    .selection-checkbox-btn:hover {
      transform: scale(1.1);
      background: #fff;
    }

    .selection-checkbox-btn.selected {
      background: #0d6efd;
      border-color: #0a58ca;
    }

    /* Existing styles... */
    .empty-state { text-align: center; padding: 4rem 2rem; color: #6c757d; font-size: 1.2rem; }
    .loading-spinner { display: flex; justify-content: center; align-items: center; min-height: 300px; }
    .pagination-controls { display: flex; justify-content: center; align-items: center; margin-top: 2rem; padding-bottom: 2rem; }
    .jump-input { width: 60px !important; text-align: center; }
  `]
})
export class PhotoListComponent implements OnInit, OnDestroy {
  // ... Logic remains identical to previous stable version ...
  photos: Photo[] = [];
  totalPhotos = 0;
  limit = 20;
  offset = 0;
  currentPage = 1;
  jumpPageInput: number | null = null;

  loading = true;
  currentSearchTerm = '';
  activeTagFilter: string | null = null;
  
  // Selection State
  selectedPhotoIds: Set<string> = new Set();
  targetAlbumId: string | undefined = undefined;
  albums: Album[] = [];

  private subscription?: Subscription;
  private searchSubscription?: Subscription;
  private routeSub?: Subscription;
  private readonly SCROLL_KEY = 'photo_list_scroll_pos';

  constructor(
    @Optional() @Inject(PhotoService) private photoService: PhotoService,
    private router: Router,
    private galleryState: GalleryStateService,
    private searchService: SearchService,
    private route: ActivatedRoute,
    private albumService: AlbumService
  ) {}

  get totalPages(): number { return Math.ceil(this.totalPhotos / this.limit) || 1; }

  ngOnInit(): void {
    const savedPage = this.galleryState.getCurrentPage();
    this.currentPage = savedPage;
    this.offset = (savedPage - 1) * this.limit;
    
    this.searchSubscription = this.searchService.searchTerm$.subscribe(term => {
      this.currentSearchTerm = term;
      this.loadPhotos();
    });

    this.routeSub = this.route.queryParams.subscribe(params => {
      this.activeTagFilter = params['tag'] || null;
      this.loadPhotos();
    });

    this.albumService.getAlbums().subscribe(albums => this.albums = albums);
    this.loadPhotos();
  }

  toggleSelection(id: string): void {
    if (this.selectedPhotoIds.has(id)) {
      this.selectedPhotoIds.delete(id);
    } else {
      this.selectedPhotoIds.add(id);
    }
  }

  isPhotoSelected(id: string): boolean { return this.selectedPhotoIds.has(id); }

  clearSelection(): void { 
    this.selectedPhotoIds.clear(); 
    this.targetAlbumId = undefined; 
  }

  addToSelectedAlbum(): void {
    if (!this.targetAlbumId || this.selectedPhotoIds.size === 0) return;
    const mediaIds = Array.from(this.selectedPhotoIds);
    this.albumService.addMediaToAlbum(this.targetAlbumId, mediaIds).subscribe({
      next: () => { alert('Added to album successfully!'); this.clearSelection(); },
      error: (err) => alert('Error adding to album')
    });
  }

  loadPhotos(): void {
    if (!this.photoService) { this.loading = false; return; }
    this.loading = true;
    const request$ = this.currentSearchTerm.trim() !== '' 
      ? this.photoService.listPhotos(this.limit, this.offset)
      : this.photoService.listMedia(this.limit, this.offset);

    this.subscription = request$.subscribe({
      next: (response: ListPhotosResponse) => {
        let allMedia = response.photos;
        if (this.activeTagFilter) {
          const tagLower = this.activeTagFilter.toLowerCase();
          allMedia = allMedia.filter(p => this.getTagsForPhoto(p).some(t => t.toLowerCase().includes(tagLower)));
        }
        if (this.currentSearchTerm.trim() !== '') {
          const term = this.currentSearchTerm.toLowerCase();
          allMedia = allMedia.filter(p => p.filename.toLowerCase().includes(term));
        }
        this.photos = allMedia;
        this.totalPhotos = response.total;
        this.loading = false;
      },
      error: () => { this.photos = []; this.totalPhotos = 0; this.loading = false; }
    });
  }

  getTagsForPhoto(photo: Photo): string[] {
    if (!photo.tags) return [];
    return Array.isArray(photo.tags) ? photo.tags : (typeof photo.tags === 'string' ? [photo.tags] : []);
  }

  clearTagFilter(): void { this.activeTagFilter = null; this.router.navigate(['/'], { replaceUrl: true }); }
  changePage(dir: number): void { this.offset += (dir * this.limit); this.currentPage += dir; this.galleryState.saveCurrentPage(this.currentPage); this.loadPhotos(); window.scrollTo(0, 0); }
  onJumpToPage(): void { if (this.jumpPageInput && this.jumpPageInput <= this.totalPages) { this.currentPage = this.jumpPageInput; this.offset = (this.currentPage - 1) * this.limit; this.galleryState.saveCurrentPage(this.currentPage); this.loadPhotos(); window.scrollTo(0, 0); } }
  onPhotoClick(id: string): void { this.router.navigate(['/photos', id]); }

  ngOnDestroy(): void {
    sessionStorage.setItem(this.SCROLL_KEY, window.scrollY.toString());
    this.subscription?.unsubscribe();
    this.searchSubscription?.unsubscribe();
    this.routeSub?.unsubscribe();
  }
}
