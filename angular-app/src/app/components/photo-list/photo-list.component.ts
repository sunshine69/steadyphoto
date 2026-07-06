import { Component, OnInit, OnDestroy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Subscription } from 'rxjs';
import { PhotoService } from '../../services/photo.service';
import { GalleryStateService } from '../../services/gallery-state.service';
import { SearchService, SearchScope, SearchResponse } from '../../services/search.service';
import { Photo, ListPhotosResponse } from '../../models/photo.model';
import { Router, RouterModule, ActivatedRoute } from '@angular/router';
import { PhotoCardComponent } from '../photo-card/photo-card.component';
import { SelectionService } from '../../services/selection.service';

@Component({
  selector: 'app-photo-list',
  standalone: true,
  imports: [CommonModule, RouterModule, PhotoCardComponent, FormsModule],
  template: `
    <div class="photo-list-container">
      <!-- Active Tag Filter Display -->
      <div *ngIf="activeTagFilter && selectedPhotoIds.size === 0" class="alert alert-info d-flex align-items-center justify-content-between mb-3">
        <span>Showing items with tag: <strong class="text-uppercase">#{{ activeTagFilter }}</strong></span>
        <button (click)="clearTagFilter()" class="btn btn-sm btn-outline-danger">Clear Filter</button>
      </div>

      <!-- Empty State -->
      <div class="empty-state" *ngIf="!loading && (!photos || photos.length === 0)">
        <p *ngIf="!currentSearchTerm && !activeTagFilter">No photos found. Start by importing your photo library.</p>
        <p *ngIf="currentSearchTerm && activeTagFilter">No items match search "{{currentSearchTerm}}" and tag "#{{activeTagFilter}}"</p>
        <p *ngIf="currentSearchTerm && !activeTagFilter">No items match search "{{currentSearchTerm}}"</p>
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
    .photo-list-container { padding-top: 5rem; padding-bottom: 2rem; width: 100%; position: relative; }
    .grid-container { display: grid; grid-template-columns: repeat(auto-fill, minmax(180px, 1fr)); gap: 1.5rem; width: 100%; }
    .grid-item { position: relative; height: 100%; }

    .selection-checkbox-btn {
      position: absolute; top: 8px; left: 8px; z-index: 20; width: 26px; height: 26px; border-radius: 50%; background: rgba(255, 255, 255, 0.8); backdrop-filter: blur(4px); border: 1px solid rgba(0,0,0,0.1); display: flex; align-items: center; justify-content: center; padding: 0; cursor: pointer; transition: all 0.2s ease; box-shadow: 0 2px 4px rgba(0,0,0,0.15);
    }

    .selection-checkbox-btn:hover { transform: scale(1.1); background: #fff; }
    .selection-checkbox-btn.selected { background: #0d6efd; border-color: #0a58ca; }
    .empty-state { text-align: center; padding: 4rem 2rem; color: #6c757d; font-size: 1.2rem; }
    .loading-spinner { display: flex; justify-content: center; align-items: center; min-height: 300px; }
    .pagination-controls { display: flex; justify-content: center; align-items: center; margin-top: 2rem; padding-bottom: 2rem; }
    .jump-input { width: 60px !important; text-align: center; }
  `]
})
export class PhotoListComponent implements OnInit, OnDestroy {
  photos: Photo[] = [];
  totalPhotos = 0;
  limit = 20;
  offset = 0;
  currentPage = 1;
  jumpPageInput: number | null = null;

  loading = true;
  currentSearchTerm = '';
  currentDateRange = '';
  activeTagFilter: string | null = null;
  searchScope: SearchScope = 'all';
  
  // Local selection state — kept in sync with the service via selectedIds$ subscription
  selectedPhotoIds: Set<string> = new Set();

  private subscription: Subscription | null = null;
  private searchSubscription: Subscription | null = null;
  private routeSub: Subscription | null = null;
  private selectAllTriggerSub: Subscription | null = null;
  private selectedIdsSub: Subscription | null = null;
  private readonly SCROLL_KEY = 'photo_list_scroll_pos';

  constructor(
    private photoService: PhotoService,
    private router: Router,
    private galleryState: GalleryStateService,
    private searchService: SearchService,
    private route: ActivatedRoute,
    private selectionService: SelectionService
  ) {}

  get totalPages(): number { return Math.ceil(this.totalPhotos / this.limit) || 1; }

  ngOnInit(): void {
    const savedPage = this.galleryState.getCurrentPage();
    this.currentPage = savedPage;
    this.offset = (savedPage - 1) * this.limit;
    
    // Subscribe to search service changes
    this.searchSubscription = this.searchService.searchTerm$.subscribe(term => {
      // Reset to page 1 when search is cleared (transitioning from search to full list)
      if (term === '' && this.currentSearchTerm !== '') {
        this.currentPage = 1;
        this.offset = 0;
      }
      this.currentSearchTerm = term;
      
      // For date scope with empty term, don't reload yet — wait for date range to arrive
      // so we don't call listMedia() with stale data
      if (this.searchScope === 'date' && !term.trim()) {
        console.log('[PHOTO-LIST] date scope with empty term, waiting for date range...');
        return;
      }
      
      this.loadPhotos();
    });

    // Subscribe to "Select All" trigger from the top header
    this.selectAllTriggerSub = this.selectionService.selectAllTrigger$.subscribe(() => {
      if (this.photos && this.photos.length > 0) {
        const ids = this.photos.map(p => p.id);
        this.selectionService.selectAll(ids);
      }
    });

    // Subscribe to the service's selectedIds$ BehaviorSubject so the local
    // selectedPhotoIds Set stays in sync whenever any selection action occurs
    // (selectAll, add, remove, toggle, clear). The service is the single source
    // of truth — we never write to selectedPhotoIds directly except via the
    // subscription callback.
    this.selectedIdsSub = this.selectionService.selectedIds$.subscribe(ids => {
      this.selectedPhotoIds = new Set(ids);
    });

    this.searchService.searchScope$.subscribe(scope => {
      this.searchScope = scope;
      console.log('[PHOTO-LIST] scope changed to:', scope);
      this.loadPhotos();
    });

    // Subscribe to date range changes for date-only search
    this.searchService.searchDate$.subscribe(dateRange => {
      this.currentDateRange = dateRange;
      console.log('[PHOTO-LIST] dateRange changed to:', dateRange);
      // Only reload if we're in date scope or there's an active search
      if (this.searchScope === 'date' || this.currentSearchTerm.trim()) {
        this.loadPhotos();
      }
    });

    this.routeSub = this.route.queryParams.subscribe(params => {
      this.activeTagFilter = params['tag'] || null;
      this.loadPhotos();
    });

    this.loadPhotos();
  }

  /**
   * Toggle the selection state of a photo.
   * Delegates to the service so the service's BehaviorSubject is updated,
   * which triggers the selectedIds$ subscription that updates our local Set.
   */
  toggleSelection(id: string): void {
    this.selectionService.toggle(id);
  }

  /**
   * Check whether a photo is currently selected.
   * Reads from the local Set which is kept in sync via the subscription.
   */
  isPhotoSelected(id: string): boolean { return this.selectedPhotoIds.has(id); }

  /**
   * Clear all selections.
   * Delegates to the service; the subscription will reset our local Set.
   */
  clearSelection(): void { 
    this.selectionService.clear();
  }

  /**
   * Load photos from the backend.
   * - If searching by date, use server-side search with date range filter
   * - If there's a search term, use server-side search via SearchService
   * - Otherwise, use the standard list endpoint
   */
  loadPhotos(): void {
    if (!this.photoService) { this.loading = false; return; }
    this.loading = true;

    let request$: Subscription | null = null;

    // Use search endpoint for: text search OR date-only search (scope='date' with dateRange)
    const hasSearchTerm = this.currentSearchTerm.trim() !== '';
    const isDateSearch = this.searchScope === 'date' && this.currentDateRange !== '';

    if (hasSearchTerm || isDateSearch) {
      // Use server-side search
      const dateRange = isDateSearch ? this.currentDateRange : undefined;
      console.log('[PHOTO-LIST] loadPhotos() -> searchMedia:', {
        query: this.currentSearchTerm,
        scope: this.searchScope,
        dateRange: dateRange,
        limit: this.limit,
        offset: this.offset
      });

      request$ = this.searchService.searchMedia(
        this.currentSearchTerm,
        this.searchScope,
        this.limit,
        this.offset,
        dateRange
      ).subscribe({
        next: (response: SearchResponse) => {
          console.log('[PHOTO-LIST] searchMedia response:', { total: response.total, results: response.results.length });
          let allMedia = response.results;

          // Apply tag filter from URL if present (client-side filter on top of search results)
          if (this.activeTagFilter) {
            const tagLower = this.activeTagFilter.toLowerCase();
            allMedia = allMedia.filter(p => this.getTagsForPhoto(p).some(t => t.toLowerCase().includes(tagLower)));
          }

          this.photos = allMedia;
          this.totalPhotos = response.total;
          this.loading = false;
        },
        error: (err) => {
          console.error('[PHOTO-LIST] searchMedia error:', err);
          this.photos = [];
          this.totalPhotos = 0;
          this.loading = false;
        }
      });
    } else {
      // No search term and no date filter - use standard list endpoint
      console.log('[PHOTO-LIST] loadPhotos() -> listMedia (no search, no date filter)');
      request$ = this.photoService.listMedia(this.limit, this.offset).subscribe({
        next: (response: ListPhotosResponse) => {
          let allMedia = response.photos;

          // Apply tag filter from URL if present
          if (this.activeTagFilter) {
            const tagLower = this.activeTagFilter.toLowerCase();
            allMedia = allMedia.filter(p => this.getTagsForPhoto(p).some(t => t.toLowerCase().includes(tagLower)));
          }

          this.photos = allMedia;
          this.totalPhotos = response.total;
          this.loading = false;
        },
        error: () => {
          this.photos = [];
          this.totalPhotos = 0;
          this.loading = false;
        }
      });
    }

    // Store the subscription so we can clean it up
    this.subscription = request$;
  }

  getTagsForPhoto(photo: Photo): string[] {
    if (!photo.tags) return [];
    return Array.isArray(photo.tags) ? photo.tags : (typeof photo.tags === 'string' ? [photo.tags] : []);
  }

  clearTagFilter(): void { this.activeTagFilter = null; this.router.navigate(['/'], { replaceUrl: true }); }
  changePage(dir: number): void { this.offset += (dir * this.limit); this.currentPage += dir; this.galleryState.saveCurrentPage(this.currentPage); this.loadPhotos(); window.scrollTo(0, 0); }
  onJumpToPage(): void { if (this.jumpPageInput && this.jumpPageInput <= this.totalPages) { this.currentPage = this.jumpPageInput; this.offset = (this.currentPage - 1) * this.limit; this.galleryState.saveCurrentPage(this.currentPage); this.loadPhotos(); window.scrollTo(0, 0); } }
  onPhotoClick(id: string): void {
    const queryParams: any = {};
    if (this.currentSearchTerm) {
      queryParams.searchTerm = this.currentSearchTerm;
      queryParams.searchScope = this.searchScope;
    }
    this.router.navigate(['/photos', id], { queryParams });
  }

  ngOnDestroy(): void {
    sessionStorage.setItem(this.SCROLL_KEY, window.scrollY.toString());
    this.subscription?.unsubscribe();
    this.searchSubscription?.unsubscribe();
    this.selectAllTriggerSub?.unsubscribe();
    this.selectedIdsSub?.unsubscribe();
    this.routeSub?.unsubscribe();
  }
}
