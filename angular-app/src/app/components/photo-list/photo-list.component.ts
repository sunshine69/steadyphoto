import { Component, Inject, OnInit, OnDestroy, Optional } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Subscription } from 'rxjs';
import { PhotoService } from '../../services/photo.service';
import { GalleryStateService } from '../../services/gallery-state.service';
import { SearchService } from '../../services/search.service';
import { Photo, ListPhotosResponse } from '../../models/photo.model';
import { Router, RouterModule } from '@angular/router';
import { PhotoCardComponent } from '../photo-card/photo-card.component';

@Component({
  selector: 'app-photo-list',
  standalone: true,
  imports: [CommonModule, RouterModule, PhotoCardComponent, FormsModule],
  template: `
    <div class="photo-list-container">
      <div class="empty-state" *ngIf="!loading && (!photos || photos.length === 0)">
        <p *ngIf="!currentSearchTerm">No photos found. Start by importing your photo library.</p>
        <p *ngIf="currentSearchTerm">No photos match "{{currentSearchTerm}}"</p>
      </div >
      
      <div class="grid-container" *ngIf="!loading && photos && photos.length > 0">
        <div class="grid-item" *ngFor="let photo of photos">
          <app-photo-card [photo]="photo" (cardClick)="onPhotoClick(photo.id)"></app-photo-card>
        </div >
      </div >

      <div class="pagination-controls" *ngIf="!loading && totalPhotos > photos.length && !currentSearchTerm">
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
  private subscription?: Subscription;
  private searchSubscription?: Subscription;
  
  private readonly SCROLL_KEY = 'photo_list_scroll_pos';

  constructor(
    @Optional() @Inject(PhotoService) private photoService: PhotoService,
    private router: Router,
    private galleryState: GalleryStateService,
    private searchService: SearchService
  ) {}

  get totalPages(): number {
    return Math.ceil(this.totalPhotos / this.limit) || 1;
  }

  ngOnInit(): void {
    const savedPage = this.galleryState.getCurrentPage();
    this.currentPage = savedPage;
    this.offset = (savedPage - 1) * this.limit;
    
    this.searchSubscription = this.searchService.searchTerm$.subscribe(term => {
      this.currentSearchTerm = term;
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
    this.subscription = this.photoService.listPhotos(this.limit, this.offset).subscribe({
      next: (response: ListPhotosResponse) => {
        const allPhotos = response.photos;
        
        if (this.currentSearchTerm.trim() !== '') {
          const term = this.currentSearchTerm.toLowerCase();
          this.photos = allPhotos.filter(p => 
            p.filename.toLowerCase().includes(term)
          );
          this.totalPhotos = this.photos.length;
        } else {
          this.photos = allPhotos;
          this.totalPhotos = response.total;
        }

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
  }

  onPhotoClick(id: string): void {
    this.router.navigate(['/photos', id]);
  }
}
