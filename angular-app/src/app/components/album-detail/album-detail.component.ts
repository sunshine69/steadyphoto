import { Component, inject, OnInit, OnDestroy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, RouterModule, Router } from '@angular/router';
import { AlbumService } from '../../services/album.service';
import { PhotoService } from '../../services/photo.service';
import { Photo } from '../../models/photo.model';
import { Album } from '../../models/album.model';
import { PhotoCardComponent } from '../photo-card/photo-card.component';
import { FormsModule } from '@angular/forms';

@Component({
  selector: 'app-album-detail',
  standalone: true,
  imports: [CommonModule, RouterModule, PhotoCardComponent, FormsModule],
  template: `
    <div class="container mt-4">
      <!-- Bulk Action Toolbar (Only visible when items are selected) -->
      <div class="bulk-action-toolbar mb-3" *ngIf="selectedPhotoIds.size > 0">
        <div class="d-flex align-items-center justify-content-between bg-white p-2 rounded shadow-sm border">
          <div>
            <span class="badge bg-primary me-2">{{ selectedPhotoIds.size }}</span> items selected
          </div>
          <div class="actions d-flex gap-3 align-items-center">
             <!-- Removal Actions -->
             <div class="d-flex align-items-center gap-2 border-end pe-3">
                <span class="text-muted small me-1 text-nowrap">Remove from this album:</span>
                <button class="btn btn-sm btn-danger" (click)="bulkRemoveFromAlbum()">Remove Selected</button>
             </div>

            <!-- Global Actions -->
            <div class="d-flex align-items-center gap-2 ps-3 border-start ms-2">
              <button class="btn btn-sm btn-secondary" (click)="clearSelection()">Cancel</button>
            </div>
          </div>
        </div>
      </div>

      <div class="d-flex justify-content-between align-items-center mb-4">
        <div>
          <nav aria-label="breadcrumb">
            <ol class="breadcrumb mb-1">
              <li class="breadcrumb-item"><a routerLink="/albums" class="text-decoration-none">Albums</a></li>
              <li class="breadcrumb-item active" aria-current="page">{{ albumName }}</li>
            </ol>
          </nav>
          <h2 class="mb-0">{{ albumName }}</h2>
        </div>
        <div>
           <button class="btn btn-outline-secondary me-2" (click)="goBack()">Back to Albums</button>
        </div>
      </div>

      <!-- Media Grid -->
      <div class="grid-container" *ngIf="!loading && photos.length > 0">
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

      <!-- Empty State -->
      <div class="empty-state py-5 text-center border rounded bg-light" *ngIf="!loading && photos.length === 0">
        <p class="lead text-muted">This album is empty.</p>
      </div>

      <!-- Loading Spinner -->
      <div class="loading-spinner d-flex justify-content-center my-5" *ngIf="loading">
        <div class="spinner-border text-primary" role="status">
          <span class="visually-hidden">Loading...</span>
        </div>
      </div>
    </div>
  `,
  styles: [`
    .grid-container {
      display: grid;
      grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
      gap: 1.5rem;
      width: 100%;
    }
    .grid-item {
      position: relative;
      display: flex;
      height: 100%;
    }

    .selection-checkbox-btn {
      position: absolute; top: 8px; left: 8px; z-index: 20; width: 26px; height: 26px; border-radius: 50%; background: rgba(255, 255, 255, 0.8); backdrop-filter: blur(4px); border: 1px solid rgba(0,0,0,0.1); display: flex; align-items: center; justify-content: center; padding: 0; cursor: pointer; transition: all 0.2s ease; box-shadow: 0 2px 4px rgba(0,0,0,0.15);
    }

    .selection-checkbox-btn:hover { transform: scale(1.1); background: #fff; }
    .selection-checkbox-btn.selected { background: #dc3545; border-color: #a71d2a; } /* Red for removal mode */

    .empty-state {
        margin-top: 2rem;
    }
  `]
})
export class AlbumDetailComponent implements OnInit {
  private route = inject(ActivatedRoute);
  private router = inject(Router);
  private albumService = inject(AlbumService);

  albumName: string = 'Loading...';
  photos: Photo[] = [];
  loading = true;
  albumId!: string;

  selectedPhotoIds: Set<string> = new Set();

  ngOnInit(): void {
    const id = this.route.snapshot.paramMap.get('id');
    if (id) {
      this.albumId = id;
      this.loadAlbumContent(id);
    } else {
      this.router.navigate(['/albums']);
    }
  }

  private loadAlbumContent(id: string): void {
    this.loading = true;
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

  private fetchMedia(id: string): void {
    this.albumService.getAlbumMedia(id).subscribe({
      next: (photos: Photo[]) => {
        this.photos = photos;
        this.loading = false;
      },
      error: (err: any) => {
        console.error('Error loading album media', err);
        this.loading = false;
      }
    });
  }

  toggleSelection(id: string): void {
    if (this.selectedPhotoIds.has(id)) { this.selectedPhotoIds.delete(id); } 
    else { this.selectedPhotoIds.add(id); }
  }

  isPhotoSelected(id: string): boolean { return this.selectedPhotoIds.has(id); }

  clearSelection(): void { this.selectedPhotoIds.clear(); }

  bulkRemoveFromAlbum(): void {
    if (this.selectedPhotoIds.size === 0) return;
    
    const mediaIds = Array.from(this.selectedPhotoIds);
    const confirmMsg = `Are you sure you want to remove ${mediaIds.length} items from "${this.albumName}"?`;

    if (!confirm(confirmMsg)) return;

    // We use the existing bulkRemoveMediaFromAlbum service method which targets 1 album ID and many media IDs
    this.albumService.bulkRemoveMediaFromAlbum(this.albumId, mediaIds).subscribe({
      next: () => {
        alert('Removed from album successfully!');
        this.clearSelection();
        this.fetchMedia(this.albumId); // Refresh list
      },
      error: (err: any) => alert(`Error removing items - ${err.message || 'Unknown error'}`)
    });
  }

  goBack(): void { this.router.navigate(['/albums']); }

  onPhotoClick(id: string): void { this.router.navigate(['/photos', id]); }
}
