import { Component, Input, Output, EventEmitter, OnInit, OnDestroy, Inject, Optional } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Subscription } from 'rxjs';
import { PhotoService } from '../../services/photo.service';
import { GalleryStateService } from '../../services/gallery-state.service';
import { Photo, ListPhotosResponse } from '../../models/photo.model';
import { Router, RouterModule } from '@angular/router';
import { PhotoCardComponent } from '../photo-card/photo-card.component';

@Component({
  selector: 'app-photo-list',
  standalone: true,
  imports: [CommonModule, RouterModule, PhotoCardComponent],
  template: `
    <div class="photo-list-container">
      <div class="empty-state" *ngIf="!loading && (!photos || photos.length === 0)">
        <p>No photos found. Start by importing your photo library.</p>
      </div>
      
      <div class="grid-container" *ngIf="!loading && photos && photos.length > 0">
        <div class="grid-item" *ngFor="let photo of photos">
          <app-photo-card [photo]="photo" (cardClick)="onPhotoClick(photo.id)"></app-photo-card>
        </div >
      </div >

      <div class="pagination-controls" *ngIf="!loading && totalPhotos > photos.length">
        <button class="btn btn-outline-primary me-2" 
                [disabled]="offset === 0" 
                (click)="changePage(-1)">
          Previous
        </button>
        <span class="mx-3">Page {{ currentPage }}</span>
        <button class="btn btn-outline-primary ms-2" 
                [disabled]="offset + limit >= totalPhotos" 
                (click)="changePage(1)">
          Next
        </button>
      </div>

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
  `]
})
export class PhotoListComponent implements OnInit, OnDestroy {
  photos: Photo[] = [];
  totalPhotos = 0;
  limit = 20;
  offset = 0;
  currentPage = 1;

  loading = true;
  private subscription?: Subscription;

  constructor(
    @Optional() @Inject(PhotoService) private photoService: PhotoService,
    private router: Router,
    private galleryState: GalleryStateService
  ) {}

  ngOnInit(): void {
    // Restore page state from localStorage to prevent reset to page 1
    const savedPage = this.galleryState.getCurrentPage();
    this.currentPage = savedPage;
    this.offset = (savedPage - 1) * this.limit;
    
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
        this.photos = response.photos;
        this.totalPhotos = response.total;
        this.loading = false;
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
    
    // Save page state to localStorage
    this.galleryState.saveCurrentPage(this.currentPage);
    
    this.loadPhotos();
    window.scrollTo(0, 0);
  }

  ngOnDestroy(): void {
    this.subscription?.unsubscribe();
  }

  onPhotoClick(id: string): void {
    this.router.navigate(['/photos', id]);
  }
}
