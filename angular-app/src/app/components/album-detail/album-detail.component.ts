import { Component, inject, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, RouterModule, Router } from '@angular/router';
import { AlbumService } from '../../services/album.service';
import { PhotoService } from '../../services/photo.service';
import { Photo } from '../../models/photo.model';
import { PhotoCardComponent } from '../photo-card/photo-card.component';

@Component({
  selector: 'app-album-detail',
  standalone: true,
  imports: [CommonModule, RouterModule, PhotoCardComponent],
  template: `
    <div class="container mt-4">
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

      <div class="grid-container" *ngIf="!loading && photos.length > 0">
        <div class="grid-item" *ngFor="let photo of photos">
          <app-photo-card [photo]="photo" (cardClick)="onPhotoClick(photo.id)"></app-photo-card>
        </div>
      </div>

      <div class="empty-state py-5 text-center border rounded bg-light" *ngIf="!loading && photos.length === 0">
        <p class="lead text-muted">This album is empty.</p>
      </div>

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
      display: flex;
      height: 100%;
    }
    .empty-state {
        margin-top: 2rem;
    }
  `]
})
export class AlbumDetailComponent implements OnInit {
  private route = inject(ActivatedRoute);
  private router = inject(Router);

  private albumService = inject(AlbumService);
  private photoService = inject(PhotoService);

  albumName: string = 'Loading...';
  photos: Photo[] = [];
  loading = true;

  ngOnInit(): void {
    const albumId = this.route.snapshot.paramMap.get('id');
    if (albumId) {
      this.loadAlbumContent(albumId);
    } else {
      this.router.navigate(['/albums']);
    }
  }

  private loadAlbumContent(albumId: string): void {
    this.loading = true;
    // Fetching album media content
    this.albumService.getAlbumMedia(albumId).subscribe({
      next: (photos) => {
        this.photos = photos;
        this.loading = false;
        // Note: In a complete implementation, you'd also call 
        // this.albumService.getAlbum(albumId) to get the actual album name.
        this.albumName = 'My Album'; // Fallback for now as we are focusing on media retrieval logic
      },
      error: (err) => {
        console.error('Error loading album media', err);
        this.loading = false;
        this.albumName = 'Error loading album';
      }
    });
  }

  goBack(): void {
    this.router.navigate(['/albums']);
  }

  onPhotoClick(id: string): void {
    this.router.navigate(['/photos', id]);
  }
}
