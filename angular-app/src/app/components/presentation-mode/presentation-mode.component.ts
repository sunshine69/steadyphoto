import { Component, OnInit, OnDestroy, AfterViewInit, inject, ChangeDetectionStrategy } from '@angular/core';
import { Router, ActivatedRoute } from '@angular/router';
import { CommonModule } from '@angular/common';
import { PresentationService, MediaItem } from '../../services/presentation.service';
import { PhotoService } from '../../services/photo.service';
import { Subscription } from 'rxjs';

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
          aria-label="Previous item">
          ‹
        </button>
        <button
          class="nav-btn nav-next"
          (click)="nextItem()"
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
              @for (item of items; track item.id; let i = $index) {
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
      align-items: center;
    }

    /* Landscape orientation — media left, controls right */
    @media (orientation: landscape) and (max-width: 768px) {
      .presentation-overlay {
        flex-direction: row;
        align-items: stretch;
      }

      .media-container {
        flex: 1;
        width: auto;
        padding: 10px;
        min-width: 0;
      }

      .presentation-media {
        width: 100%;
        height: 100%;
      }

      .bottom-controls {
        width: 120px;
        flex-shrink: 0;
        flex-direction: column;
        padding: 10px 10px 10px 8px;
        gap: 6px;
        overflow-y: auto;
      }

      .thumbnail-strip {
        max-width: 100%;
        overflow-x: hidden;
        overflow-y: auto;
        flex-direction: column;
        gap: 4px;
      }

      .thumb-btn {
        width: 100%;
        height: 50px;
      }

      .thumb-img {
        width: 100%;
        height: 100%;
        object-fit: cover;
      }

      .progress-info {
        font-size: 11px;
        padding: 3px 8px;
      }

      .file-name {
        max-width: 100%;
        font-size: 11px;
        text-align: center;
      }
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
      flex: 1;
      width: 100%;
      display: flex;
      align-items: center;
      justify-content: center;
      overflow: hidden;
      min-width: 0;
      min-height: 0;
    }

    .presentation-media {
      max-width: 100%;
      max-height: 100%;
      object-fit: contain;
      display: block;
      width: 100%;
      height: 100%;
    }

    video.presentation-media {
      background-color: #000;
      max-width: 100%;
      max-height: 100%;
    }

    .bottom-controls {
      flex-shrink: 0;
      width: 100%;
      display: flex;
      flex-direction: column;
      align-items: center;
      gap: 10px;
      padding: 10px 20px 20px 20px;
      box-sizing: border-box;
      z-index: 10000;
    }

    .bottom-controls .thumbnail-strip {
      max-width: 100%;
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

    /* ===== Mobile Responsive Styles ===== */
    @media (max-width: 768px) {
      .btn-close-btn {
        top: 10px;
        right: 10px;
        width: 40px;
        height: 40px;
        font-size: 28px;
      }

      .nav-btn {
        width: 48px;
        height: 48px;
        font-size: 36px;
      }

      .nav-prev {
        left: 10px;
      }

      .nav-next {
        right: 10px;
      }

      .media-container {
        flex: 1;
        width: 100%;
        padding: 10px;
        display: flex;
        align-items: center;
        justify-content: center;
        overflow: hidden;
      }

      .bottom-controls {
        gap: 6px;
        padding: 10px 16px 16px 16px;
      }

      .progress-info {
        font-size: 12px;
        padding: 4px 10px;
      }

      .thumbnail-strip {
        max-width: 100vw;
        overflow-x: auto;
        -webkit-overflow-scrolling: touch;
      }

      .thumb-btn {
        width: 50px;
        height: 35px;
        flex-shrink: 0;
      }

      .thumb-icon {
        font-size: 16px;
        line-height: 35px;
      }

      .file-name {
        max-width: 90vw;
        font-size: 11px;
      }
    }

    /* Small mobile devices */
    @media (max-width: 480px) {
      .btn-close-btn {
        width: 36px;
        height: 36px;
        font-size: 24px;
      }

      .nav-btn {
        width: 40px;
        height: 40px;
        font-size: 32px;
      }

      .media-container {
        flex: 1;
        width: 100%;
        padding: 8px;
        display: flex;
        align-items: center;
        justify-content: center;
        overflow: hidden;
      }

      .bottom-controls {
        gap: 4px;
        padding: 6px 12px 12px 12px;
      }

      .file-name {
        font-size: 10px;
      }
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
  
  private subscription?: Subscription;

  ngOnInit(): void {
    this.loadPresentationData();
    
    // Subscribe to state changes to update view
    this.subscription = this.presentationService.stateChanges$.subscribe(() => {
      this.items = this.presentationService.getItems();
      this.currentIndex = this.presentationService.getCurrentIndex();
      this.totalItems = this.items.length;
      this.currentItem = this.presentationService.getCurrentItem();
      this.isLoading = false;
    });
  }

  ngAfterViewInit(): void {
    document.addEventListener('keydown', this.handleKeyDown);
  }

  ngOnDestroy(): void {
    this.subscription?.unsubscribe();
    document.removeEventListener('keydown', this.handleKeyDown);
  }

  /**
   * Check if we're at the beginning of the loaded items
   */
  isAtStart(): boolean {
    return this.presentationService.isAtStart();
  }

  /**
   * Check if we're at the end of the loaded items
   */
  isAtEnd(): boolean {
    return this.presentationService.isAtEnd();
  }

  private loadPresentationData(): void {
    this.items = this.presentationService.getItems();
    this.currentIndex = this.presentationService.getCurrentIndex();
    this.totalItems = this.items.length;
    this.currentItem = this.presentationService.getCurrentItem();
    this.isLoading = false;
  }

  async nextItem(): Promise<void> {
    console.log('[PresentationComponent] nextItem called, isAtEnd:', this.presentationService.isAtEnd(), 'currentIndex:', this.currentIndex, 'items.length:', this.items.length);
    // If at the end, try to load more items first
    if (this.presentationService.isAtEnd()) {
      console.log('[PresentationComponent] nextItem - at end, calling loadNextPage');
      const loaded = await this.presentationService.loadNextPage();
      console.log('[PresentationComponent] nextItem - loadNextPage returned:', loaded, 'loaded.items.length:', loaded ? this.presentationService.getItems().length : 0);
      if (loaded) {
        // Update state after loading
        this.items = this.presentationService.getItems();
        this.currentIndex = this.presentationService.getCurrentIndex();
        this.totalItems = this.items.length;
        this.currentItem = this.presentationService.getCurrentItem();
        console.log('[PresentationComponent] nextItem - new items loaded, new currentIndex:', this.currentIndex);
        return; // Don't navigate further, just show the new items
      }
      return; // No more items to load
    }
    
    // Navigate to next item normally
    if (this.presentationService.next()) {
      this.currentIndex = this.presentationService.getCurrentIndex();
      this.currentItem = this.presentationService.getCurrentItem();
    }
  }

  async previousItem(): Promise<void> {
    console.log('[PresentationComponent] previousItem called, isAtStart:', this.presentationService.isAtStart(), 'currentIndex:', this.currentIndex, 'items.length:', this.items.length);
    // If at the start, try to load previous items first
    if (this.presentationService.isAtStart()) {
      console.log('[PresentationComponent] previousItem - at start, calling loadPreviousPage');
      const loaded = await this.presentationService.loadPreviousPage();
      console.log('[PresentationComponent] previousItem - loadPreviousPage returned:', loaded, 'loaded.items.length:', loaded ? this.presentationService.getItems().length : 0);
      if (loaded) {
        // Update state after loading
        this.items = this.presentationService.getItems();
        this.currentIndex = this.presentationService.getCurrentIndex();
        this.totalItems = this.items.length;
        this.currentItem = this.presentationService.getCurrentItem();
        console.log('[PresentationComponent] previousItem - new items loaded, new currentIndex:', this.currentIndex);
        return; // Don't navigate further, just show the new items
      }
      return; // No more items to load
    }
    
    // Navigate to previous item normally
    if (this.presentationService.previous()) {
      this.currentIndex = this.presentationService.getCurrentIndex();
      this.currentItem = this.presentationService.getCurrentItem();
    }
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
