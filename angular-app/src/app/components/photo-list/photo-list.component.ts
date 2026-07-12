import { Component, OnInit, OnDestroy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Subscription, combineLatest, switchMap, debounceTime, distinctUntilChanged, of } from 'rxjs';
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
  
  // Local selection state
  selectedPhotoIds: Set<string> = new Set();

  private mainSub: Subscription | null = null;
  private directSub: Subscription | null = null;
  private selectAllTriggerSub: Subscription | null = null;
  private selectedIdsSub: Subscription | null = null;
  private restorePage = false;
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
    // Prevent switchMap from resetting page on initial load
    this.restorePage = true;
    this.offset = (savedPage - 1) * this.limit;
    
    // Use combineLatest to react to any search parameter change, but only
    // trigger ONE request after 300ms of inactivity (debounce) and skip
    // identical consecutive states (distinctUntilChanged).
    const searchParams$ = combineLatest([
      this.searchService.searchTerm$,
      this.searchService.searchDate$,
      this.searchService.searchScope$
    ]).pipe(
      debounceTime(300),
      distinctUntilChanged((prev: any[], curr: any[]) => 
        prev[0] === curr[0] && prev[1] === curr[1] && prev[2] === curr[2]
      ),
      switchMap(([term, dateRange, scope]) => {
        this.currentSearchTerm = term;
        this.currentDateRange = dateRange;
        this.searchScope = scope;
        
        // Reset page when search starts or scope changes (skip during initial restore)
        if (!this.restorePage && this.currentPage !== 1) {
          this.currentPage = 1;
          this.offset = 0;
        }

        const hasSearchTerm = term.trim() !== '';
        const isDateSearch = scope === 'date' && dateRange !== '';

        if (hasSearchTerm || isDateSearch) {
          return this.searchService.searchMedia(term, scope, this.limit, this.offset, isDateSearch ? dateRange : undefined);
        } else {
          return this.photoService.listMedia(this.limit, this.offset).pipe(
            switchMap((response: ListPhotosResponse) => of({
              results: response.photos,
              total: response.total,
              limit: this.limit,
              offset: this.offset
            }))
          );
        }
      })
    );

    // Subscribe to the combined search stream
    this.mainSub = searchParams$.subscribe({
      next: (response: any) => {
        // The listMedia path wraps its response, so results is always available
        let allMedia = response.results || response.photos || [];

        // Apply tag filter from URL if present
        if (this.activeTagFilter) {
          const tagLower = this.activeTagFilter.toLowerCase();
          allMedia = allMedia.filter((p: Photo) => this.getTagsForPhoto(p).some((t: string) => t.toLowerCase().includes(tagLower)));
        }

        // Clear restore flag after first successful load
        if (this.restorePage) {
          this.restorePage = false;
        }

        this.photos = allMedia;
        this.totalPhotos = response.total ?? 0;
        
        // If we got empty results but photos exist, our saved page is out of range
        // (e.g., user deleted photos between sessions). Reset to last page and reload.
        if (!this.restorePage && allMedia.length === 0 && this.totalPhotos > 0 && this.currentPage > 1) {
          const lastPage = this.totalPages;
          this.currentPage = lastPage;
          this.offset = (lastPage - 1) * this.limit;
          this.galleryState.saveCurrentPage(this.currentPage);
          this.loadPhotosDirect();
          return;
        }

        this.loading = false;
      },
      error: () => {
        this.photos = [];
        this.totalPhotos = 0;
        this.loading = false;
      }
    });

    // Subscribe to "Select All" trigger
    this.selectAllTriggerSub = this.selectionService.selectAllTrigger$.subscribe(() => {
      if (this.photos && this.photos.length > 0) {
        const ids = this.photos.map((p: Photo) => p.id);
        this.selectionService.selectAll(ids);
      }
    });

    // Keep selectedPhotoIds in sync with selection service
    this.selectedIdsSub = this.selectionService.selectedIds$.subscribe((ids: Set<string>) => {
      this.selectedPhotoIds = new Set(ids);
    });

    // Subscribe to route query params for tag filter — just reload directly,
    // no need to go through the search stream debounce.
    this.route.queryParams.subscribe((params: any) => {
      const newTag = params['tag'] || null;
      if (newTag !== this.activeTagFilter) {
        this.activeTagFilter = newTag;
        this.currentPage = 1;
        this.offset = 0;
        this.loadPhotosDirect();
      }
    });

    // Initial load — trigger the stream with existing service state
    this.loading = true;
  }

  /**
   * Direct synchronous load for tag filter changes — bypasses the debounce.
   */
  private loadPhotosDirect(): void {
    if (!this.photoService) { 
      this.loading = false; 
      return; 
    }
    this.loading = true;

    const hasSearchTerm = this.currentSearchTerm.trim() !== '';
    const isDateSearch = this.searchScope === 'date' && this.currentDateRange !== '';

    if (hasSearchTerm || isDateSearch) {
      const dateRange = isDateSearch ? this.currentDateRange : undefined;
      this.directSub = this.searchService.searchMedia(
        this.currentSearchTerm, this.searchScope,
        this.limit, this.offset, dateRange
      ).subscribe({
        next: (response: SearchResponse) => this.applyResults(response.results, response.total),
        error: () => {
          this.photos = [];
          this.totalPhotos = 0;
          this.loading = false;
        }
      });
    } else {
      this.directSub = this.photoService.listMedia(this.limit, this.offset).subscribe({
        next: (response: ListPhotosResponse) => this.applyResults(response.photos, response.total),
        error: () => {
          this.photos = [];
          this.totalPhotos = 0;
          this.loading = false;
        }
      });
    }
  }

  private applyResults(allMedia: Photo[], total: number): void {
    if (this.activeTagFilter) {
      const tagLower = this.activeTagFilter.toLowerCase();
      allMedia = allMedia.filter((p: Photo) => this.getTagsForPhoto(p).some((t: string) => t.toLowerCase().includes(tagLower)));
    }
    this.photos = allMedia;
    this.totalPhotos = total;
    this.loading = false;
  }

  toggleSelection(id: string): void {
    this.selectionService.toggle(id);
  }

  isPhotoSelected(id: string): boolean { return this.selectedPhotoIds.has(id); }

  clearSelection(): void { 
    this.selectionService.clear();
  }

  getTagsForPhoto(photo: Photo): string[] {
    if (!photo.tags) return [];
    return Array.isArray(photo.tags) ? photo.tags : (typeof photo.tags === 'string' ? [photo.tags] : []);
  }

  clearTagFilter(): void { 
    this.activeTagFilter = null; 
    this.router.navigate(['/'], { replaceUrl: true }); 
  }
  
  changePage(dir: number): void { 
    this.offset += (dir * this.limit); 
    this.currentPage += dir; 
    this.galleryState.saveCurrentPage(this.currentPage); 
    this.loadPhotosDirect();
    window.scrollTo(0, 0); 
  }
  
  onJumpToPage(): void { 
    if (this.jumpPageInput && this.jumpPageInput <= this.totalPages) { 
      this.currentPage = this.jumpPageInput; 
      this.offset = (this.currentPage - 1) * this.limit; 
      this.galleryState.saveCurrentPage(this.currentPage); 
      this.loadPhotosDirect();
      window.scrollTo(0, 0); 
    } 
  }
  
  onPhotoClick(id: string): void {
    // Save current page BEFORE navigating so goBack() can restore it
    this.galleryState.saveCurrentPage(this.currentPage);

    const queryParams: any = {};
    if (this.currentSearchTerm) {
      queryParams.searchTerm = this.currentSearchTerm;
      queryParams.searchScope = this.searchScope;
    }
    this.router.navigate(['/photos', id], { queryParams });
  }

  ngOnDestroy(): void {
    sessionStorage.setItem(this.SCROLL_KEY, window.scrollY.toString());
    this.mainSub?.unsubscribe();
    this.directSub?.unsubscribe();
    this.selectAllTriggerSub?.unsubscribe();
    this.selectedIdsSub?.unsubscribe();
  }
}
