import { Component, inject, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule, Router } from '@angular/router';
import { AlbumService } from '../../services/album.service';
import { Album } from '../../models/album.model';

@Component({
  selector: 'app-album-list',
  standalone: true,
  imports: [CommonModule, RouterModule],
  template: `
    <div class="container mt-4">
      <div class="d-flex justify-content-between align-items-center mb-4">
        <h2 class="mb-0">My Albums</h2>
        <button class="btn btn-primary" (click)="openCreateAlbumModal()">
          <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" class="bi bi-plus-lg me-2" viewBox="0 0 16 16">
            <path fill-rule="evenodd" d="M8 2a.5.5 0 0 1 .5.5v5h5a.5.5 0 0 1 0 1h-5v5a.5.5 0 0 1-1 0v-5h-5a.5.5 0 0 1 0-1h5v-5A.5.5 0 0 1 8 2"/>
          </svg> New Album
        </button>
      </div>

      <div class="row" *ngIf="albums && albums.length > 0; else noAlbums">
        <div class="col-md-4 col-lg-3 mb-4" *ngFor="let album of albums">
          <div class="card h-100 shadow-sm album-card" (click)="album.id ? router.navigate(['/albums', album.id]) : null">
            <div class="card-body text-center d-flex flex-column justify-content-center align-items-center py-5">
              <div class="album-icon mb-3">
                <svg xmlns="http://www.w3.org/2000/svg" width="48" height="48" fill="currentColor" class="bi bi-collection text-primary" viewBox="0 0 16 16">
                  <path d="M11 2H9v2h2V2zM5 2h2v2H5V2zM1 3.5A1.5 1.5 0 0 1 2.5 2h11a1.5 1.5 0 0 1 1.5 1.5v9a1.5 1.5 0 0 1-1.5 1.5h-11A1.5 1.5 0 0 1 1 12.5v-9zm1.5-.5a.5.5 0 0 0-.5.5v9a.5.5 0 0 0 .5.5h11a.5.5 0 0 0 .5-.5v-9a.5.5 0 0 0-.5-.5h-11z"/>
                </svg>
              </div>
              <h5 class="card-title mb-1">{{ album.name }}</h5>
              <p class="card-text text-muted small">Created {{ album.createdAt | date:'mediumDate' }}</p>
            </div>
          </div>
        </div>
      </div>

      <ng-template #noAlbums>
        <div class="empty-state py-5 text-center border rounded bg-light" *ngIf="!loading">
          <svg xmlns="http://www.w3.org/2000/svg" width="64" height="64" fill="currentColor" class="bi bi-images text-muted mb-3" viewBox="0 0 16 16">
            <path d="M4.502 9a1.5 1.5 0 0 1 .5-.866 1.5 1.5 0 0 0-.798-2.403 1.5 1.5 0 0 0-1.17 1.95L3.37 6.5H.5A1.5 1.5 0 0 0 0 7.5v1A1.5 1.5 0 0 0 1.5 10h2.87l-.4-1.9a1.5 1.5 0 0 0-1.17-1.95 1.5 1.5 0 0 0-.5.866zm.5 3a1.5 1.5 0 0 1 .5-.866 1.5 1.5 0 0 0-.798-2.403 1.5 1.5 0 0 0-1.17 1.95L3.37 10.5H.5A1.5 1.5 0 0 0 0 11.5v1A1.5 1.5 0 0 0 1.5 13h2.87l-.4-1.9a1.5 1.5 0 0 0-1.17-1.95 1.5 1.5 0 0 0-.5.866zm4.5 0a1.5 1.5 0 0 1 .5-.866 1.5 1.5 0 0 0-.798-2.403 1.5 1.5 0 0 0-1.17 1.95L7.37 10.5H4.5A1.5 1.5 0 0 0 3 11.5v1A1.5 1.5 0 0 0 4.5 13h2.87l-.4-1.9a1.5 1.5 0 0 0-1.17-1.95 1.5 1.5 0 0 0-.5.866zm4.5 0a1.5 1.5 0 0 1 .5-.866 1.5 1.5 0 0 0-.798-2.403 1.5 1.5 0 0 0-1.17 1.95L11.37 10.5H8.5A1.5 1.5 0 0 0 7 11.5v1A1.5 1.5 0 0 0 8.5 13h2.87l-.4-1.9a1.5 1.5 0 0 0-1.17-1.95 1.5 1.5 0 0 0-.5.866z"/>
                  </svg>
                  <p class="lead">You haven't created any albums yet.</p>
                </div>
              </ng-template>

      <div *ngIf="loading" class="d-flex justify-content-center my-5">
        <div class="spinner-border text-primary" role="status">
          <span class="visually-hidden">Loading...</span>
        </div>
      </div>
    </div>
  `,
  styles: [`
    .album-card { cursor: pointer; transition: transform 0.2s ease, box-shadow 0.2s ease; border-radius: 1rem; overflow: hidden; }
    .album-card:hover { transform: translateY(-5px); box-shadow: 0 10px 20px rgba(0,0,0,0.1) !important; }
  `]
})
export class AlbumListComponent implements OnInit {
  albums: Album[] = [];
  loading = true;

  private albumService = inject(AlbumService);
  public router = inject(Router);

  ngOnInit(): void {
    this.loadAlbums();
  }

  loadAlbums(): void {
    this.loading = true;
    this.albumService.getAlbums().subscribe({
      next: (albums: Album[]) => {
        this.albums = albums;
        this.loading = false;
      },
      error: (err: any) => {
        console.error('Error loading albums', err);
        this.loading = false;
      }
    });
  }

  openCreateAlbumModal(): void {
    const name = prompt('Enter album name:');
    if (name) {
      this.albumService.createAlbum({ name }).subscribe({
        next: () => this.loadAlbums(),
        error: (err: any) => alert('Failed to create album')
      });
    }
  }
}
