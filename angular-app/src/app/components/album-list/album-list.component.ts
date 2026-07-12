import { Component, inject, OnInit, OnDestroy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule, Router } from '@angular/router';
import { AlbumService } from '../../services/album.service';
import { ShareService } from '../../services/share.service';
import { Album } from '../../models/album.model';
import { SharedAlbumItem } from '../../models/share.model';
import { Subscription } from 'rxjs';

@Component({
    selector: 'app-album-list',
    imports: [CommonModule, RouterModule],
    template: `
    <div class="container mt-4">
      <!-- Tab navigation -->
      <ul class="nav nav-tabs mb-4">
        <li class="nav-item">
          <a class="nav-link" [class.active]="activeTab === 'myAlbums'" (click)="setTab('myAlbums')">My Albums</a>
        </li>
        <li class="nav-item">
          <a class="nav-link" [class.active]="activeTab === 'sharedAlbums'" (click)="setTab('sharedAlbums')">Shared Albums</a>
        </li>
      </ul>

      <!-- My Albums View -->
      <div *ngIf="activeTab === 'myAlbums'">
        <div class="d-flex justify-content-between align-items-center mb-4">
          <h2 class="mb-0">My Albums</h2>

          <div class="d-flex gap-2">
            <!-- Bulk delete button - only shown when albums are selected -->
            <button *ngIf="selectedAlbums.length > 0" class="btn btn-danger btn-sm" (click)="openBulkDeleteModal()">
              Delete Selected ({{ selectedAlbums.length }})
            </button>

            <button class="btn btn-primary" (click)="openCreateAlbumModal()">
              <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" class="bi bi-plus-lg me-2" viewBox="0 0 16 16">
                <path fill-rule="evenodd" d="M8 2a.5.5 0 0 1 .5.5v5h5a.5.5 0 0 1 0 1h-5v5a.5.5 0 0 1 -1 0v-5h-5a.5.5 0 0 1 0-1h5v-5A.5.5 0 0 1 8 2"/>
              </svg> New Album
            </button>
          </div>
        </div>

        <!-- Albums grid -->
        <div class="row" *ngIf="albums && albums.length > 0; else noMyAlbums">
          <div class="col-md-4 col-lg-3 mb-4" *ngFor="let album of albums">
            <div 
              class="card h-100 shadow-sm album-card"
              [class.selected]="isAlbumSelected(album)"
              (click)="!deleting && navigateToAlbum(album)">

              <!-- Delete button - only visible on hover -->
              <button 
                *ngIf="album.id"
                class="btn btn-danger delete-btn" 
                (click)="$event.stopPropagation(); openDeleteModal(album)"
                title="Delete album">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <polyline points="3 6 5 6 21 6"/>
                  <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
                </svg>
              </button>

              <!-- Selection checkbox - visible on hover OR when selected -->
              <div class="selection-checkbox" *ngIf="album.id">
                <input 
                  type="checkbox" 
                  [checked]="isAlbumSelected(album)" 
                  (click)="$event.stopPropagation(); toggleSelection(album)"
                  title="Select album for bulk delete"
                />
              </div>

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

        <!-- No albums template -->
        <ng-template #noMyAlbums>
          <div class="empty-state py-5 text-center border rounded bg-light" *ngIf="!loading">
            <svg xmlns="http://www.w3.org/2000/svg" width="64" height="64" fill="currentColor" class="bi bi-images text-muted mb-3" viewBox="0 0 16 16">
              <path d="M4.502 9a1.5 1.5 0 0 1 .5-.866 1.5 1.5 0 0 0-.798-2.403 1.5 1.5 0 0 0-1.17 1.95L3.37 6.5H.5A1.5 1.5 0 0 0 0 7.5v1A1.5 1.5 0 0 0 1.5 10h2.87l-.4-1.9a1.5 1.5 0 0 0-1.17-1.95 1.5 1.5 0 0 0-.5.866zm.5 3a1.5 1.5 0 0 1 .5-.866 1.5 1.5 0 0 0-.798-2.403 1.5 1.5 0 0 0-1.17 1.95L3.37 10.5H.5A1.5 1.5 0 0 0 0 11.5v1A1.5 1.5 0 0 0 1.5 13h2.87l-.4-1.9a1.5 1.5 0 0 0-1.17-1.95 1.5 1.5 0 0 0-.5.866zm4.5 0a1.5 1.5 0 0 1 .5-.866 1.5 1.5 0 0 0-.798-2.403 1.5 1.5 0 0 0-1.17 1.95L7.37 10.5H4.5A1.5 1.5 0 0 0 3 11.5v1A1.5 1.5 0 0 0 4.5 13h2.87l-.4-1.9a1.5 1.5 0 0 0-1.17-1.95 1.5 1.5 0 0 0-.5.866zm4.5 0a1.5 1.5 0 0 1 .5-.866 1.5 1.5 0 0 0-.798-2.403 1.5 1.5 0 0 0-1.17 1.95L11.37 10.5H8.5A1.5 1.5 0 0 0 7 11.5v1A1.5 1.5 0 0 0 8.5 13h2.87l-.4-1.9a1.5 1.5 0 0 0-1.17-1.95 1.5 1.5 0 0 0-.5.866z"/>
            </svg>
            <p class="lead">You haven't created any albums yet.</p>
          </div>
        </ng-template>

        <!-- Loading spinner -->
        <div *ngIf="loading" class="d-flex justify-content-center my-5">
          <div class="spinner-border text-primary" role="status">
            <span class="visually-hidden">Loading...</span>
          </div>
        </div>
      </div>

      <!-- Shared Albums View -->
      <div *ngIf="activeTab === 'sharedAlbums'">
        <div class="d-flex justify-content-between align-items-center mb-4">
          <h2 class="mb-0">Shared Albums</h2>
        </div>

        <!-- Shared albums grid -->
        <div class="row" *ngIf="sharedAlbums && sharedAlbums.length > 0; else noSharedAlbums">
          <div class="col-md-4 col-lg-3 mb-4" *ngFor="let album of sharedAlbums">
            <div 
              class="card h-100 shadow-sm album-card"
              (click)="navigateToSharedAlbum(album)">
              
              <div class="card-body text-center d-flex flex-column justify-content-center align-items-center py-5">
                <div class="album-icon mb-3">
                  <svg xmlns="http://www.w3.org/2000/svg" width="48" height="48" fill="currentColor" class="bi bi-collection-play text-success" viewBox="0 0 16 16">
                    <path fill-rule="evenodd" d="M6.5 9.5a.5.5 0 0 0 0 1h5a.5.5 0 0 0 0-1h-5zM1.5 2a1.5 1.5 0 0 0-1.5 1.5v9a1.5 1.5 0 0 0 1.5 1.5h9a1.5 1.5 0 0 0 1.5-1.5v-9A1.5 1.5 0 0 0 10.5 1h-9zm-1 1.5a1 1 0 0 1 1-1h9a1 1 0 0 1 1 1v9a1 1 0 0 1-1 1h-9a1 1 0 0 1-1-1v-9zm5.5 3a.5.5 0 0 0 0 1h5a.5.5 0 0 0 0-1h-5zm0 2a.5.5 0 0 0 0 1h5a.5.5 0 0 0 0-1h-5z"/>
                    <path d="M5.5 4l4 2.5-4 2.5V4z"/>
                  </svg>
                </div>
                <h5 class="card-title mb-1">{{ album.name }}</h5>
                <p class="card-text text-muted small" *ngIf="album.description">{{ album.description }}</p>
                <p class="card-text text-muted small" *ngIf="!album.description">Shared album</p>
              </div>
            </div>
          </div>
        </div>

        <!-- No shared albums template -->
        <ng-template #noSharedAlbums>
          <div class="empty-state py-5 text-center border rounded bg-light" *ngIf="!loadingSharedAlbums">
            <svg xmlns="http://www.w3.org/2000/svg" width="64" height="64" fill="currentColor" class="bi bi-collection-play text-muted mb-3" viewBox="0 0 16 16">
              <path fill-rule="evenodd" d="M6.5 9.5a.5.5 0 0 0 0 1h5a.5.5 0 0 0 0-1h-5zM1.5 2a1.5 1.5 0 0 0-1.5 1.5v9a1.5 1.5 0 0 0 1.5 1.5h9a1.5 1.5 0 0 0 1.5-1.5v-9A1.5 1.5 0 0 0 10.5 1h-9zm-1 1.5a1 1 0 0 1 1-1h9a1 1 0 0 1 1 1v9a1 1 0 0 1-1 1h-9a1 1 0 0 1-1-1v-9zm5.5 3a.5.5 0 0 0 0 1h5a.5.5 0 0 0 0-1h-5zm0 2a.5.5 0 0 0 0 1h5a.5.5 0 0 0 0-1h-5z"/>
              <path d="M5.5 4l4 2.5-4 2.5V4z"/>
            </svg>
            <p class="lead">No albums have been shared with you yet.</p>
          </div>
        </ng-template>

        <!-- Loading spinner for shared albums -->
        <div *ngIf="loadingSharedAlbums" class="d-flex justify-content-center my-5">
          <div class="spinner-border text-primary" role="status">
            <span class="visually-hidden">Loading...</span>
          </div>
        </div>
      </div>

    </div>

    <!-- Delete confirmation modal - always visible when showDeleteModal is true -->
    <div *ngIf="showDeleteModal" class="delete-modal-backdrop" (click)="closeDeleteModal()">
      <div class="delete-modal-content" (click)="$event.stopPropagation()">
        <h3>Delete Album</h3>

        <!-- Single album delete message -->
        <p *ngIf="albumToDelete && !isBulkDelete">Are you sure you want to delete this album?</p>

        <!-- Bulk delete info - shows when multiple albums are selected -->
        <div *ngIf="selectedAlbums.length > 0" class="bulk-delete-info">
          <p class="text-danger mb-1"><strong>{{ selectedAlbums.length }} albums will be deleted:</strong></p>
          <ul class="list-group list-group-flush small">
            <li *ngFor="let a of selectedAlbums" class="list-group-item bg-light">{{ a.name }}</li>
          </ul>
        </div>

        <!-- Single album name display -->
        <p *ngIf="albumToDelete && isBulkDelete" class="fw-bold text-muted">Album: {{ albumToDelete.name }}</p>

        <p class="text-muted small mt-3 mb-0">This action cannot be undone.</p>

        <div class="d-flex gap-2 mt-4">
          <button class="btn btn-secondary flex-fill" (click)="closeDeleteModal()" [disabled]="deleting">Cancel</button>
          <button class="btn btn-danger flex-fill" (click)="confirmDeletion()" [disabled]="deleting">
            {{ deleting ? 'Deleting...' : 'Delete' }}
          </button>
        </div>
      </div>
    </div>

  `,
    styles: [`
    .album-card { 
      cursor: pointer; 
      transition: transform 0.2s ease, box-shadow 0.2s ease; 
      border-radius: 1rem; 
      overflow: hidden; 
      position: relative;
    }
    .album-card:hover { 
      transform: translateY(-5px); 
      box-shadow: 0 10px 20px rgba(0,0,0,0.1) !important; 
    }
    .album-card.selected {
      border: 2px solid #6366f1;
      background-color: rgba(99, 102, 241, 0.05);
    }

    /* Delete button - only visible on hover */
    .delete-btn {
      position: absolute;
      top: 8px;
      right: 8px;
      width: 32px;
      height: 32px;
      padding: 0;
      display: flex;
      align-items: center;
      justify-content: center;
      border-radius: 50%;
      opacity: 0;
      transition: all 0.2s ease;
    }
    .album-card:hover .delete-btn {
      opacity: 1;
    }

    /* Selection checkbox - visible on hover OR when selected */
    .selection-checkbox {
      position: absolute;
      top: 8px;
      left: 8px;
      z-index: 2;
      opacity: 0;
      transition: all 0.2s ease;
    }
    /* Show checkbox when hovering over the card OR when it's selected */
    .album-card:hover .selection-checkbox,
    .album-card.selected .selection-checkbox {
      opacity: 1;
    }
    .selection-checkbox input[type="checkbox"] {
      width: 18px;
      height: 18px;
      cursor: pointer;
      accent-color: #6366f1;
    }

    /* Delete modal */
    .delete-modal-backdrop {
      position: fixed;
      top: 0; left: 0; width: 100%; height: 100%;
      background-color: rgba(0, 0, 0, 0.5);
      display: flex;
      align-items: center;
      justify-content: center;
      z-index: 9999;
    }

    .delete-modal-content {
      background: white;
      padding: 24px;
      border-radius: 8px;
      max-width: 500px;
      width: 90%;
      box-shadow: 0 4px 16px rgba(0,0,0,0.3);
    }

    .delete-modal-content h3 {
      margin-bottom: 8px;
    }

    .bulk-delete-info {
      background-color: #f8d7da;
      border-radius: 4px;
      padding: 12px;
      margin-top: 12px;
    }

    .list-group-item {
      border-left: none;
      border-right: none;
      border-bottom: 1px solid #f5c6cb;
    }
    .list-group-item:first-child {
      border-top: none;
    }
    .list-group-item:last-child {
      border-bottom: none;
    }

    /* Empty state */
    .empty-state {
        margin-top: 2rem;
    }
  `]
})
export class AlbumListComponent implements OnInit, OnDestroy {
  albums: Album[] = [];
  sharedAlbums: SharedAlbumItem[] = [];
  loading = true;
  loadingSharedAlbums = false;
  activeTab: 'myAlbums' | 'sharedAlbums' = 'myAlbums';
  
  // Selection state for bulk delete
  selectedAlbums: Album[] = [];

  // Delete modal state
  showDeleteModal = false;
  albumToDelete: Album | null = null;
  deleting = false;

  // Subscriptions for cleanup
  private subscriptions = new Subscription();

  private albumService = inject(AlbumService);
  private shareService = inject(ShareService);
  public router = inject(Router);

  ngOnInit(): void {
    this.loadAlbums();
  }

  ngOnDestroy(): void {
    this.subscriptions.unsubscribe();
  }

  setTab(tab: 'myAlbums' | 'sharedAlbums'): void {
    this.activeTab = tab;
    
    // Load shared albums when switching to that tab
    if (tab === 'sharedAlbums') {
      this.loadSharedAlbums();
    }
  }

  loadAlbums(): void {
    this.loading = true;
    const sub = this.albumService.getAlbums().subscribe({
      next: (albums: Album[]) => {
        this.albums = albums;
        this.selectedAlbums = []; // Clear selection when loading
        this.loading = false;
      },
      error: (err: any) => {
        console.error('Error loading albums', err);
        this.loading = false;
      }
    });
    this.subscriptions.add(sub);
  }

  loadSharedAlbums(): void {
    this.loadingSharedAlbums = true;
    const sub = this.shareService.listSharedAlbums(100, 0).subscribe({
      next: (response) => {
        this.sharedAlbums = response.items;
        this.loadingSharedAlbums = false;
      },
      error: (err: any) => {
        console.error('Error loading shared albums', err);
        this.loadingSharedAlbums = false;
      }
    });
    this.subscriptions.add(sub);
  }

  openCreateAlbumModal(): void {
    const name = prompt('Enter album name:');
    if (name) {
      const sub = this.albumService.createAlbum({ name }).subscribe({
        next: () => this.loadAlbums(),
        error: (err: any) => alert('Failed to create album')
      });
      this.subscriptions.add(sub);
    }
  }

  // Navigation
  navigateToAlbum(album: Album): void {
    if (!album.id) return;
    this.router.navigate(['/albums', album.id]);
  }

  navigateToSharedAlbum(album: SharedAlbumItem): void {
    if (!album.id) return;
    this.router.navigate(['/albums', album.id], { queryParams: { source: 'shared' } });
  }

  // Selection methods for bulk delete
  isAlbumSelected(album: Album): boolean {
    return !!album.id && this.selectedAlbums.some(a => a.id === album.id);
  }

  toggleSelection(album: Album): void {
    if (!album.id) return;
    
    const index = this.selectedAlbums.findIndex(a => a.id === album.id);
    if (index > -1) {
      // Deselect
      this.selectedAlbums.splice(index, 1);
    } else {
      // Select
      this.selectedAlbums.push(album);
    }
  }

  // Delete methods
  openDeleteModal(album: Album): void {
    if (!album.id) return;
    
    if (this.selectedAlbums.length === 0) {
      // Single delete mode - no albums selected, just this one  
      this.albumToDelete = album;
      this.showDeleteModal = true;
    } else {
      // Bulk delete mode - some albums already selected
      // Add to selection if not already selected (don't use toggleSelection)
      if (!this.isAlbumSelected(album)) {
        this.selectedAlbums.push(album);
      }
      
      // Now open bulk delete modal with all selected albums including this one
      this.albumToDelete = null; // Not a single album delete
      this.showDeleteModal = true;
    }
  }

  get isBulkDelete(): boolean {
    return !this.albumToDelete || this.selectedAlbums.length > 1;
  }

  openBulkDeleteModal(): void {
    if (this.selectedAlbums.length === 0) return;
    
    this.albumToDelete = null; // Not a single album delete
    this.showDeleteModal = true;
  }

  closeDeleteModal(): void {
    this.showDeleteModal = false;
    this.albumToDelete = null;
    if (this.deleting) {
      // Abort ongoing deletion by unsubscribing from all pending observables
      this.deleteSubscriptions.forEach(sub => sub.unsubscribe());
      this.deleteSubscriptions = [];
      this.deleting = false;
    }
  }

  private deleteSubscriptions: any[] = [];

  confirmDeletion(): void {
    if (this.selectedAlbums.length === 0) return;

    this.deleting = true;

    // Delete the album(s) one by one
    const albumsToDelete = [...this.selectedAlbums].filter(a => a.id);
    
    let completedCount = 0;
    const totalToComplete = albumsToDelete.length;
    let hasError = false;

    for (const album of albumsToDelete) {
      if (!album.id) continue;
      
      const sub = this.albumService.deleteAlbum(album.id).subscribe({
        next: () => {
          completedCount++;
          
          // If all done, reload and close modal - DON'T show another modal!
          if (completedCount === totalToComplete && !hasError) {
            // Close the delete confirmation modal immediately
            this.showDeleteModal = false;
            this.albumToDelete = null;
            
            // Clear selection and reload
            this.selectedAlbums = [];
            this.loadAlbums();
          }
        },
        error: (err: any) => {
          console.error(`Failed to delete album ${album.id}`, err);
          hasError = true;
          alert(`Failed to delete album "${album.name}". Please try again.`);
        }
      });
      
      this.deleteSubscriptions.push(sub);
    }

    // Safety timeout - clear interval after 30 seconds
    setTimeout(() => {
      if (this.deleting) {
        this.closeDeleteModal();
        this.deleting = false;
        this.loadAlbums();
      }
    }, 30000);
  }
}
