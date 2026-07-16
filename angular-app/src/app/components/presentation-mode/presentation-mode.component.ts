import { Component, OnInit, OnDestroy, AfterViewInit, inject, ChangeDetectionStrategy } from '@angular/core';
import { Router, ActivatedRoute } from '@angular/router';
import { CommonModule } from '@angular/common';
import { PresentationService, MediaItem } from '../../services/presentation.service';
import { PhotoService } from '../../services/photo.service';

@Component({
    selector: 'app-presentation-mode',
    imports: [CommonModule],
    template: `
    @if (presentationService.isOpen$ | async) {
      <div class="presentation-overlay">
        <!-- Close Button -->
        <button
          class="btn-close-btn"
          (click)="closePresentation()"
          aria-label="Close presentation mode">
          ×
        </button>
        <!-- Navigation Buttons -->
        <button
          class="nav-btn nav-prev"
          (click)="previousItem()"
          [class.disabled]="currentIndex <= 0"
          aria-label="Previous item">
          ‹
        </button>
        <button
          class="nav-btn nav-next"
          (click)="nextItem()"
          [class.disabled]="currentIndex >= totalItems - 1"
          aria-label="Next item">
          ›
        </button>
        <!-- Media Display -->
        <div class="media-container">
          <!-- Video Player for videos -->
          @if (currentItem?.mediaType === 'video') {
            <video
              [attr.src]="currentItem.path"
              controls
              preload="auto"
              autoplay
              class="presentation-media"
              (error)="onMediaError($event)"
              >
              Your browser does not support the video tag.
            </video>
          }
          <!-- Image display for photos -->
          @if (currentItem?.mediaType !== 'video') {
            <img
              [src]="currentItem?.path"
              [alt]="currentItem?.filename || 'Presentation media'"
              class="presentation-media"
              (error)="onMediaError($event)"
              >
          }
        </div>
        <!-- Bottom Controls -->
        <div class="bottom-controls">
          <!-- Progress Info -->
          <div class="progress-info">
            <span>{{ currentIndex + 1 }} of {{ totalItems }}</span>
          </div>
          <!-- Thumbnail Strip (optional, can be expanded later) -->
          @if (totalItems > 5) {
            <div class="thumbnail-strip">
              @for (item of items; track item; let i = $index) {
                <button
                  [class.active]="i === currentIndex"
                  (click)="goToItem(i)"
                  class="thumb-btn"
                  [style.width.px]="60"
                  [style.height.px]="45">
                  @if (item.mediaType !== 'video') {
                    <img
                      [src]="getThumbnailUrl(item.id)"
                      alt=""
                      class="thumb-img">
                  }
                  @if (item.mediaType === 'video') {
                    <span class="thumb-icon">🎥</span>
                  }
                </button>
              }
            </div>
          }
          <!-- File Name -->
          @if (currentItem?.filename) {
            <div class="file-name">
              {{ currentItem.filename }}
            </div>
          }
        </div>
        <!-- Loading State -->
        @if (isLoading) {
          <div class="loading-overlay">
            <div class="spinner-border text-white" role="status"></div>
          </div>
        }
      </div>
    } @else {
    }
    
    <!-- Close button when presentation is closed (for testing) -->
    `,
    changeDetection: ChangeDetectionStrategy.Eager,
    styles: [`
    .presentation-overlay {
      position: fixed;
      top: 0;
      left: 0;
      width: 100vw;
      height: 100vh;
      background-color: #000;
      z-index: 9999;
      display: flex;
      flex-direction: column;
      justify-content: center;
      align-items: center;
    }

    .btn-close-btn {
      position: absolute;
      top: 20px;
      right: 20px;
      background: rgba(255, 255, 255, 0.1);
      border: none;
      color: #fff;
      font-size: 32px;
      width: 48px;
      height: 48px;
      cursor: pointer;
      z-index: 10001;
      display: flex;
      align-items: center;
      justify-content: center;
      border-radius: 50%;
      transition: background-color 0.2s;
    }

    .btn-close-btn:hover {
      background: rgba(255, 255, 255, 0.2);
    }

    .nav-btn {
      position: absolute;
      top: 50%;
      transform: translateY(-50%);
      background: rgba(255, 255, 255, 0.1);
      border: none;
      color: #fff;
      font-size: 48px;
      width: 64px;
      height: 64px;
      cursor: pointer;
      z-index: 10000;
      display: flex;
      align-items: center;
      justify-content: center;
      border-radius: 50%;
      transition: background-color 0.2s, opacity 0.2s;
    }

    .nav-btn:hover:not(.disabled) {
      background: rgba(255, 255, 255, 0.2);
    }

    .nav-btn.disabled {
      opacity: 0.3;
      cursor: not-allowed;
    }

    .nav-prev {
      left: 20px;
    }

    .nav-next {
      right: 20px;
    }

    .media-container {
      width: 100%;
      height: 100%;
      display: flex;
      align-items: center;
      justify-content: center;
      padding: 80px 100px;
    }

    .presentation-media {
      max-width: 100%;
      max-height: 100%;
      object-fit: contain;
      display: block;
    }

    video.presentation-media {
      background-color: #000;
    }

    .bottom-controls {
      position: absolute;
      bottom: 20px;
      left: 50%;
      transform: translateX(-50%);
      display: flex;
      flex-direction: column;
      align-items: center;
      gap: 10px;
      z-index: 10000;
    }

    .progress-info {
      color: #fff;
      font-size: 14px;
      opacity: 0.8;
      background: rgba(0, 0, 0, 0.5);
      padding: 6px 12px;
      border-radius: 16px;
    }

    .thumbnail-strip {
      display: flex;
      gap: 8px;
      overflow-x: auto;
      max-width: 90vw;
      padding: 4px;
    }

    .thumb-btn {
      border: 2px solid transparent;
      background: rgba(255, 255, 255, 0.1);
      cursor: pointer;
      overflow: hidden;
      transition: border-color 0.2s;
      flex-shrink: 0;
    }

    .thumb-btn.active {
      border-color: #fff;
    }

    .thumb-btn:hover:not(.active) {
      background: rgba(255, 255, 255, 0.2);
    }

    .thumb-img {
      width: 100%;
      height: 100%;
      object-fit: cover;
    }

    .thumb-icon {
      font-size: 20px;
      line-height: 45px;
      text-align: center;
      color: #fff;
    }

    .file-name {
      color: #fff;
      font-size: 12px;
      opacity: 0.6;
      max-width: 300px;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    .loading-overlay {
      position: absolute;
      top: 50%;
      left: 50%;
      transform: translate(-50%, -50%);
      z-index: 10002;
    }
  `]
})
export class PresentationModeComponent implements OnInit, AfterViewInit, OnDestroy {
  public presentationService = inject(PresentationService);
  private photoService = inject(PhotoService);
  private router = inject(Router);
  private route = inject(ActivatedRoute);

  items: MediaItem[] = [];
  currentIndex = 0;
  totalItems = 0;
  currentItem?: MediaItem;
  isLoading = false;

  ngOnInit(): void {
    this.presentationService.isOpen$.subscribe((isOpen: boolean) => {
      if (isOpen) {
        this.loadPresentationData();
      } else {
        this.resetState();
      }
    });
  }

  ngAfterViewInit(): void {
    document.addEventListener('keydown', this.handleKeyDown);
  }

  ngOnDestroy(): void {
    document.removeEventListener('keydown', this.handleKeyDown);
  }

  private loadPresentationData(): void {
    this.items = this.presentationService.getItems();
    this.currentIndex = this.presentationService.getCurrentIndex();
    this.totalItems = this.items.length;
    this.currentItem = this.presentationService.getCurrentItem();
    this.isLoading = false;
  }

  private resetState(): void {
    this.items = [];
    this.currentIndex = 0;
    this.totalItems = 0;
    this.currentItem = undefined;
    this.isLoading = false;
  }

  nextItem(): boolean {
    if (this.presentationService.next()) {
      this.currentIndex = this.presentationService.getCurrentIndex();
      this.currentItem = this.presentationService.getCurrentItem();
      return true;
    }
    return false;
  }

  previousItem(): boolean {
    if (this.presentationService.previous()) {
      this.currentIndex = this.presentationService.getCurrentIndex();
      this.currentItem = this.presentationService.getCurrentItem();
      return true;
    }
    return false;
  }

  goToItem(index: number): void {
    if (this.presentationService.goTo(index)) {
      this.currentIndex = this.presentationService.getCurrentIndex();
      this.currentItem = this.presentationService.getCurrentItem();
    }
  }

  closePresentation(): void {
    // Capture the last viewed item BEFORE closing (close() resets state)
    const lastItem = this.presentationService.getCurrentItem();
    this.presentationService.close();

    if (lastItem?.id) {
      // Preserve album context for proper goBack() behavior in PhotoDetail
      const queryParams: any = {};
      
      // Read album context from current presentation route
      const currentAlbumId = this.route.snapshot.queryParams['currentAlbumId'];
      const albumIds = this.route.snapshot.queryParams['albumIds'];
      const source = this.route.snapshot.queryParams['source'];
      const shareToken = this.route.snapshot.queryParams['shareToken'];
      
      if (currentAlbumId) {
        queryParams.currentAlbumId = currentAlbumId;
      }
      if (albumIds) {
        queryParams.albumIds = albumIds;
      }
      if (source === 'shared') {
        queryParams.source = 'shared';
        if (shareToken) {
          queryParams.shareToken = shareToken;
        }
      }
      
      this.router.navigate(['/photos', lastItem.id], { queryParams });
    } else {
      window.history.back();
    }
  }

  getThumbnailUrl(id: string): string {
    return `${this.photoService['API_BASE_URL']}/media/${id}/thumb`;
  }

  onMediaError(event: Event): void {
    console.error('Presentation media error:', event);
  }

  private handleKeyDown = (event: KeyboardEvent): void => {
    if (!this.presentationService.isOpen()) return;

    switch (event.key) {
      case 'Escape':
        this.closePresentation();
        break;
      case 'ArrowRight':
        this.nextItem();
        break;
      case 'ArrowLeft':
        this.previousItem();
        break;
    }
  };
}
