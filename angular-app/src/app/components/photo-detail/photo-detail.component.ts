import { Component, OnInit, OnDestroy, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router, RouterModule } from '@angular/router';
import { Subscription } from 'rxjs';
import { PhotoService } from '../../services/photo.service';
import { Photo } from '../../models/photo.model';

@Component({
  selector: 'app-photo-detail',
  standalone: true,
  imports: [CommonModule, RouterModule],
  template: `
    <div class="container mt-4">
      <div class="row">
        <div class="col-md-8">
          <div class="photo-detail" *ngIf="photo; else loading">
            <img [src]="photo.thumbnailUrl || photo.path" [alt]="photo.filename" class="img-fluid rounded">
            <h3 class="mt-3">{{ photo.filename }}</h3>
            <p class="text-muted">Captured: {{ photo.captured_at | date:'medium' }}</p>
            <div class="mt-3">
              <a [routerLink]="['/']" class="btn btn-primary">Back to Gallery</a>
            </div>
          </div>
          <ng-template #loading>
            <div class="text-center py-5">
              <div class="spinner-border text-primary" role="status">
                <span class="visually-hidden">Loading...</span>
              </div>
              <p class="mt-2">Loading photo details...</p>
            </div>
          </ng-template>
        </div>
        <div class="col-md-4">
          <div class="card">
            <div class="card-header">
              <h5 class="mb-0">Photo Details</h5>
            </div>
            <ul class="list-group list-group-flush">
              <li class="list-group-item d-flex justify-content-between align-items-center">
                <span>Filename</span>
                <code class="bg-light px-2 py-1 rounded">{{ photo?.filename }}</code>
              </li>
              <li class="list-group-item d-flex justify-content-between align-items-center">
                <span>Path</span>
                <code class="bg-light px-2 py-1 rounded">{{ photo?.path }}</code>
              </li>
              <li class="list-group-item d-flex justify-content-between align-items-center">
                <span>Captured</span>
                <span>{{ photo?.captured_at | date:'fullDate' }}</span>
              </li>
              <li class="list-group-item d-flex justify-content-between align-items-center">
                <span>Dimensions</span>
                <span>{{ photo?.width }}x{{ photo?.height }}</span>
              </li>
              <li class="list-group-item d-flex justify-content-between align-items-center">
                <span>Type</span>
                <span>{{ photo?.type || 'Unknown' }}</span>
              </li>
            </ul>
          </div>
        </div>
      </div>
    </div>
  `
})
export class PhotoDetailComponent implements OnInit, OnDestroy {
  photo: Photo | null = null;
  private route = inject(ActivatedRoute);
  private router = inject(Router);
  private photoService = inject(PhotoService);
  private subscription?: Subscription;

  ngOnInit(): void {
    const id = this.route.snapshot.paramMap.get('id');
    if (id) {
      this.subscription = this.photoService.getPhoto(id).subscribe({
        next: (photo) => {
          this.photo = photo;
        },
        error: (err) => {
          console.error('Error fetching photo', err);
          this.router.navigate(['/']);
        }
      });
    } else {
      this.router.navigate(['/']);
    }
  }

  ngOnDestroy(): void {
    this.subscription?.unsubscribe();
  }
}
