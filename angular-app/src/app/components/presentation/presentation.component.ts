import { Component, OnInit, OnDestroy, HostListener, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute } from '@angular/router';
import { PresentationService, MediaItem } from '../../services/presentation.service';
import { PhotoService } from '../../services/photo.service';

@Component({
  selector: 'app-presentation',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div class="presentation-container" (click)="close()">
      <!-- Background dimmer -->
      <div class="overlay"></div>

      <!-- Main Content Area -->
      <div class="content-wrapper" (click)="$event.stopPropagation()">
        
        <!-- Previous Button -->
        <button 
          *ngIf="currentIndex > 0"
          class="nav-btn prev-btn" 
          (click)="previous()"
          aria-label="Previous">
          ❮
        </button>

        <!-- Media Display -->
        <div class="media-display">
          <img 
            *ngIf="currentItem?.mediaType !== 'video'"
            [src]="currentItem?.path" 
            [alt]="currentItem?.filename || 'Presentation Image'"
            class="main-media-image"
          >

          <video 
            *ngIf="currentItem?.mediaType === 'video'"
            [attr.src]="currentItem?.path"
            controls
            autoplay
            preload="metadata"
            class="main-media-video"
          ></video>
        </div>

        <!-- Next Button -->
        <button 
          *ngIf="currentIndex < items.length - 1"
          class="nav-btn next-btn" 
          (click)="next()"
          aria-label="Next">
          ❯
        </button>

        <!-- Close Button -->
        <button class="close-btn" (click)="close()">
          ✕
        </button>

        <!-- Counter / Progress -->
        <div class="progress-indicator">
          {{ currentIndex + 1 }} / {{ items.length }}
        </div>
      </div>
    </div>
  `,
  styles: [`
    .presentation-container {
      position: fixed;
      top: 0;
      left: 0;
      width: 100%;
      height: 100%;
      z-index: 9999;
      background-color: #000;
    }

    .overlay {
      position: absolute;
      top: 0;
      left: 0;
      width: 100%;
      height: 100%;
      cursor: pointer;
    }

    .content-wrapper {
      position: relative;
      width: 100%;
      height: 100%;
      display: flex;
      justify-content: center;
      align-items: center;
    }

    .media-display {
      max-width: 95vw;
      max-height: 95vh;
      display: flex;
      justify-content: center;
      align-items: center;
    }

    .main-media-image, 
    .main-media-video {
      max-width: 100%;
      max-height: 100%;
      object-fit: contain;
      box-shadow: 0 0 20px rgba(0,0,0,0.5);
    }

    .nav-btn {
      position: absolute;
      top: 50%;
      transform: translateY(-50%);
      background-color: rgba(255, 255, 255, 0.1);
      color: white;
      border: none;
      font-size: 3rem;
      padding: 1rem 1.5rem;
      cursor: pointer;
      transition: background-color 0.2s;
      z-index: 10;
    }

    .nav-btn:hover {
      background-color: rgba(255, 255, 255, 0.3);
    }

    .prev-btn { left: 1rem; }
    .next-btn { right: 1rem; }

    .close-btn {
      position: absolute;
      top: 1rem;
      right: 1rem;
      background-color: rgba(0, 0, 0, 0.5);
      color: white;
      border: none;
      font-size: 2rem;
      width: 3rem;
      height: 3rem;
      cursor: pointer;
      z-index: 10;
    }

    .progress-indicator {
      position: absolute;
      bottom: 1rem;
      left: 50%;
      transform: translateX(-50%);
      color: rgba(255, 255, 255, 0.7);
      font-size: 1.2rem;
      background-color: rgba(0, 0, 0, 0.5);
      padding: 0.5rem 1rem;
      border-radius: 4px;
    }

    @media (max-width: 768px) {
      .nav-btn { font-size: 2rem; padding: 0.5rem 1rem; }
      .prev-btn { left: 0.5rem; }
      .next-btn { right: 0.5rem; }
    }
  `]
})
export class PresentationComponent implements OnInit, OnDestroy {
  items: MediaItem[] = [];
  currentIndex: number = 0;

  private route = inject(ActivatedRoute);

  constructor(
    private presentationService: PresentationService,
    private photoService: PhotoService
  ) {}

  ngOnInit(): void {
    // Load the current state from service if available (e.g. navigated to route)
    const state = this.presentationService.getState();
    
    if (state.items.length > 0 && state.currentIndex >= 0) {
      this.items = state.items;
      this.currentIndex = state.currentIndex;
    } else {
      // Fallback: Fetch main gallery items if no state provided
      this.loadGalleryItems();
    }

    // Subscribe to service changes in case we navigate back and forth without reloading component
    // Though usually route navigation destroys/recreates, it's good practice for single-page apps
  }

  loadGalleryItems(): void {
    // Check if we're in public share presentation mode
    const shareToken = this.route.snapshot.queryParamMap.get('shareToken');
    const isSharedMedia = this.route.snapshot.queryParamMap.get('source') === 'shared';
    
    if (shareToken) {
      // Fetch from public share endpoint (no auth required)
      // listPublicShareMedia already normalizes items with proper URLs and password
      this.photoService.listPublicShareMedia(shareToken, 50, 0).subscribe({
        next: (response: any) => {
          this.items = response.media.map((p: any) => ({
            id: p.id,
            path: p.path,
            filename: p.filename,
            mediaType: p.mediaType
          }));
          this.currentIndex = 0;
        },
        error: (err) => {
          console.error('Failed to load public share media for presentation', err);
        }
      });
    } else if (isSharedMedia) {
      // Fetch from authenticated shared media endpoint
      this.photoService.listSharedMedia(50, 0).subscribe({
        next: (response: any) => {
          this.items = response.items.map((p: any) => ({
            id: p.id || p.ID,
            path: `${this.photoService['API_BASE_URL']}/media/shared/${(p.id || p.ID)}/original`,
            filename: p.filename || '',
            mediaType: (p.mediaType || 'photo')
          }));
          this.currentIndex = 0;
        },
        error: (err) => {
          console.error('Failed to load shared media for presentation', err);
        }
      });
    } else {
      // Fetch from authenticated endpoint
      this.photoService.listMedia(50, 0).subscribe({
        next: (response) => {
          this.items = response.photos;
          this.currentIndex = 0;
        },
        error: (err) => {
          console.error('Failed to load media for presentation', err);
        }
      });
    }
  }

  get currentItem(): MediaItem | undefined {
    return this.items[this.currentIndex];
  }

  next(): void {
    if (this.presentationService.next()) {
      // Service updated state, we just need to ensure our local index matches
      this.currentIndex = this.presentationService.getCurrentIndex();
    } else {
      // Optional: Loop back to start or stop
      console.log('End of presentation');
    }
  }

  previous(): void {
    if (this.presentationService.previous()) {
      this.currentIndex = this.presentationService.getCurrentIndex();
    }
  }

  close(): void {
    this.presentationService.close();
    // Navigate back to home or previous route
    window.history.back(); 
  }

  @HostListener('document:keydown', ['$event'])
  handleKeyboard(event: KeyboardEvent): void {
    if (!this.items.length) return;

    switch (event.key) {
      case 'ArrowRight':
        this.next();
        break;
      case 'ArrowLeft':
        this.previous();
        break;
      case 'Escape':
        this.close();
        break;
    }
  }

  ngOnDestroy(): void {
    // Clean up if needed
  }
}
