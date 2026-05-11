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

@Component({
  selector: 'app-photo-list',
  standalone: true,
  imports: [CommonModule, RouterModule, PhotoCardComponent, FormsModule],
  template: `
    <div class="photo-list-container">
      <!-- Active Tag Filter Display -->
      <div *ngIf="activeTagFilter" class="alert alert-info d-flex align-items-center justify-content-between mb-3">
        <span>Showing items with tag: <strong>#{{ activeTagFilter }}</strong></span>
        <button (click)="clearTagFilter()" class="btn btn-sm btn-outline-danger">Clear Filter</button>
      </div>

      <div class="empty-state" *ngIf="!loading && (!photos || photos.length === 0)">
        <p *ngIf="!currentSearchTerm && !activeTagFilter">No photos found. Start by importing your photo library.</p>
        <p *ngIf="currentSearchTerm && activeTagFilter">No items match search "{{currentSearchTerm}}" and tag "#{{activeTagFilter}}"</p>
        <p *ngIf="!currentSearchTerm && activeTagFilter">No items match tag "#{{activeTagFilter}}"</p>
        <p *ngIf="currentSearchTerm && !activeTagFilter">No photos match "{{currentSearchTerm}}"</p>
      </div >
      
      <div class="grid-container" *ngIf="!loading && photos && photos.length > 0">
        <div class="grid-item" *ngFor="let photo of photos">
          <app-photo-card [photo]="photo" (cardClick)="onPhotoClick(photo.id)"></app-photo-card>
        </div >
      </div >

      <div class="pagination-controls" *ngIf="!loading && totalPhotos > limit">
        <button class="btn btn-outline-primary me-2" 
                [disabled]="offset === 0" 
                (click)="changePage(-1)">
          Previous
        </button>

        <div class="pagination-jump d-flex align-items-center mx-3">
          <span class="me-2 text-nowrap">Page</span>
          <div class="input-group input-group-sm" style="width: 130px;">
            <input 
              type="number" 
              class="form-control text-center jump-input" 
              [(ngModel)]="jumpPageInput"
              (keyup.enter)="onJumpToPage()"
              min="1"
              [max]="totalPages"
            >
            <button class="btn btn-primary jump-btn" type="button" (click)="onJumpToPage()">Go</button>
          </div >
          <span class="ms-2 text-nowrap">of {{ totalPages }}</span>
        </div >

        <button class="btn btn-outline-primary ms-2" 
                [disabled]="offset + limit >= totalPhotos" 
                (click)="changePage(1)">
          Next
        </button>
      </div >

      <div class="loading-spinner" *ngIf="loading">
        <div class="spinner-border text-primary" role="status">
          <span class="visually-hidden">Loading...</span>
        </div >
      </div >
    </div >
  `,
  styles: [`
    .photo-list-container {
      padding: 1rem;
      width: 100%;
    }
    .grid-container {
      display: grid;
      grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
      gap: 1.5rem;
      width: 100%;
    }
    .grid-item {
      display: flex;
      height: 100%;
    }
    .empty-state {
      text-align: center;
      padding: 4rem 2rem;
      color: #6c757d;
      font-size: 1.2rem;
    }
    .loading-spinner {
      display: flex;
      justify-content: center;
      align-items: center;
      min-height: 300px;
    }
    .pagination-controls {
      display: flex;
      justify-content: center;
      align-items: center;
      margin-top: 2rem;
      padding-bottom: 2rem;
    }
    
    /* Fix for the visibility issue */
    .jump-input {
      background-color: #ffffff !important;
      color: #000000 !important; /* Force black text */
      border: 1px solid #dee2e6 !important;
      font-weight: bold;
    }

    .jump-btn {
      font-weight: 500;
    }

    /* Remove spin buttons for a cleaner look if desired, 
       but keeping them for UX if they don't block the text */
    .jump-input::-webkit-inner-spin-button,
    .jump-input::-webkit-outer-spin-button {
      opacity: 1;
    }
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
  activeTagFilter: string | null = null;
  private subscription?: Subscription;
  private searchSubscription?: Subscription;
  private routeSub?: Subscription;
  
  private readonly SCROLL_KEY = 'photo_list_scroll_pos';

  constructor(
    @Optional() @Inject(PhotoService) private photoService: PhotoService,
    private router: Router,
    private galleryState: GalleryStateService,
    private searchService: SearchService,
    private route: ActivatedRoute
  ) {}

  get totalPages(): number {
    return Math.ceil(this.totalPhotos / this.limit) || 1;
  }

  ngOnInit(): void {
    const savedPage = this.galleryState.getCurrentPage();
    this.currentPage = savedPage;
    this.offset = (savedPage - 1) * this.limit;
    
    // Listen for search term changes
    this.searchSubscription = this.searchService.searchTerm$.subscribe(term => {
      this.currentSearchTerm = term;
      this.loadPhotos();
    });

    // Listen for query params (tag filter)
    this.routeSub = this.route.queryParams.subscribe(params => {
      if (params['tag']) {
        this.activeTagFilter = params['tag'];
      } else {
        this.activeTagFilter = null;
      }
      this.loadPhotos();
    });

    this.loadPhotos();
  }

  loadPhotos(): void {
    if (!this.photoService) {
      this.loading = false;
      return;
    }

    this.loading = true;
    
    // If there is no search term, we want to list all media (including videos)
    // If there IS a search term, we can continue using listPhotos (which is filtered by type in backend)
    // or we can use listMedia and filter client-side. 
    // To show videos in the main list, we should use listMedia.
    const request$ = this.currentSearchTerm.trim() !== '' 
      ? this.photoService.listPhotos(this.limit, this.offset)
      : this.photoService.listMedia(this.limit, this.offset);

    this.subscription = request$.subscribe({
      next: (response: ListPhotosResponse) => {
        let allMedia = response.photos;
        
        // Apply tag filter if active
        if (this.activeTagFilter) {
          const tagLower = this.activeTagFilter.toLowerCase();
          allMedia = allMedia.filter(p => {
            const tags = this.getTagsForPhoto(p);
            return tags.some(tag => tag.toLowerCase().includes(tagLower));
          });
        }

        if (this.currentSearchTerm.trim() !== '') {
          const term = this.currentSearchTerm.toLowerCase();
          allMedia = allMedia.filter(p => 
            p.filename.toLowerCase().includes(term)
          );
        }

        this.photos = allMedia;
        this.totalPhotos = response.total;

        this.loading = false;

        setTimeout(() => {
          const savedScrollPos = sessionStorage.getItem(this.SCROLL_KEY);
          if (savedScrollPos) {
            window.scrollTo({
              top: parseInt(savedScrollPos, 10),
              behavior: 'instant'
            });
          }
        }, 0);
      },
      error: (err) => {
        console.error('Error fetching photos', err);
        this.photos = [];
        this.totalPhotos = 0;
        this.loading = false;
      }
    });
  }

  getTagsForPhoto(photo: Photo): string[] {
    if (!photo.tags) return [];
    // Handle different possible formats for tags
    if (Array.isArray(photo.tags)) return photo.tags;
    if (typeof photo.tags === 'string') {
      try {
        const parsed = JSON.parse(photo.tags);
        return Array.isArray(parsed) ? parsed : [parsed];
      } catch {
        return [photo.tags];
      }
    }
    // Handle object with tag array inside
    if (typeof photo.tags === 'object' && !Array.isArray(photo.tags)) {
      const possibleArrays = Object.values(photo.tags);
      for (const val of possibleArrays) {
        if (Array.isArray(val)) return val;
      }
    }
    return [];
  }

  clearTagFilter(): void {
    this.activeTagFilter = null;
    this.router.navigate(['/'], { replaceUrl: true });
  }

  changePage(direction: number): void {
    this.offset += (direction * this.limit);
    this.currentPage += direction;
    this.galleryState.saveCurrentPage(this.currentPage);
    this.loadPhotos();
    window.scrollTo(0, 0);
  }

  onJumpToPage(): void {
    const targetPage = this.jumpPageInput;
    if (targetPage && targetPage >= 1 && targetPage <= this.totalPages) {
      this.currentPage = targetPage;
      this.offset = (this.currentPage - 1) * this.limit;
      this.galleryState.saveCurrentPage(this.currentPage);
      this.loadPhotos();
      this.jumpPageInput = null;
      window.scrollTo(0, 0);
    } else {
      alert(`Please enter a valid page between 1 and ${this.totalPages}`);
    }
  }

  ngOnDestroy(): void {
    sessionStorage.setItem(this.SCROLL_KEY, window.scrollY.toString());
    this.subscription?.unsubscribe();
    this.searchSubscription?.unsubscribe();
    this.routeSub?.unsubscribe();
  }

  onPhotoClick(id: string): void {
    this.router.navigate(['/photos', id]);
  }
}
