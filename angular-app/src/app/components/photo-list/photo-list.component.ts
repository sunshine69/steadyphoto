import { Component, Input, Output, EventEmitter, OnInit, OnDestroy, Inject, Optional } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Subscription } from 'rxjs';
import { PhotoService } from '../../services/photo.service';
import { Photo } from '../../models/photo.model';
import { Router, RouterModule } from '@angular/router';
import { PhotoCardComponent } from '../photo-card/photo-card.component';

@Component({
  selector: 'app-photo-list',
  standalone: true,
  imports: [CommonModule, RouterModule, PhotoCardComponent],
  template: `
    <div class="photo-list">
      <div class="empty-state" *ngIf="!loading && (!photos || photos.length === 0)">
        <p>No photos found. Start by importing your photo library.</p>
      </div>
      <div class="grid-container" *ngIf="!loading && photos && photos.length > 0">
        <div class="grid-item" *ngFor="let photo of photos">
          <app-photo-card [photo]="photo" (cardClick)="onPhotoClick(photo.id)"></app-photo-card>
        </div >
      </div >
      <div class="text-center py-5" *ngIf="loading">
        <div class="spinner-border text-primary" role="status">
          <span class="visually-hidden">Loading...</span>
        </div>
      </div>
    </div>
  `,
  styles: [`
    .grid-container {
      display: grid;
      grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
      gap: 1rem;
    }
    .empty-state {
      text-align: center;
      padding: 2rem;
      color: #6c757d;
    }
  `]
})
export class PhotoListComponent implements OnInit, OnDestroy {
  @Input() photos: Photo[] = [];
  @Output() photoClick = new EventEmitter<string>();

  loading = true;
  private subscription?: Subscription;

  constructor(
    @Optional() @Inject(PhotoService) private photoService: PhotoService,
    private router: Router
  ) {}

  ngOnInit(): void {
    if (this.photoService) {
      this.subscription = this.photoService.listPhotos().subscribe({
        next: (photos) => {
          this.photos = photos;
          this.loading = false;
        },
        error: (err) => {
          console.error('Error fetching photos', err);
          this.photos = [];
          this.loading = false;
        }
      });
    } else {
      this.loading = false;
    }
  }

  ngOnDestroy(): void {
    this.subscription?.unsubscribe();
  }

  onPhotoClick(id: string): void {
    this.router.navigate(['/photos', id]);
  }
}
