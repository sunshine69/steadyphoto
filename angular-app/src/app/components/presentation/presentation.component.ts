import { Component, OnInit, OnDestroy, HostListener, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router } from '@angular/router';
import { Subscription } from 'rxjs';
import { PresentationService, MediaItem } from '../../services/presentation.service';
import { PhotoService } from '../../services/photo.service';
import { GalleryStateService } from '../../services/gallery-state.service';

@Component({
  selector: 'app-presentation',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div class="presentation-container" (click)="close()">
      <div class="overlay"></div>

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
          *ngIf="currentIndex < allItems.length - 1"
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
          {{ currentIndex + 1 }} / {{ allItems.length }}
        </div>

        <!-- Thumbnail Strip -->
        <div class="thumbnail-strip" (click)="$event.stopPropagation()">
          <div 
            *ngFor="let item of allItems; let i = index"
            class="thumbnail-item"
            [class.active]="i === currentIndex"
            (click)="goTo(i)">
            <img 
              [src]="getThumbnailUrl(item)" 
              [alt]="item.filename || ''"
              class="thumbnail-image"
              loading="lazy"
            >
            <div *ngIf="item.mediaType === 'video'" class="video-icon-overlay">
              ▶
            </div>
          </div>
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
      height: calc(100% - 72px);
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
      max-height: calc(100vh - 72px);
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
      z-index: 10;
    }

    .thumbnail-strip {
      position: absolute;
      bottom: 0;
      left: 0;
      right: 0;
      height: 72px;
      display: flex;
      align-items: center;
      justify-content: center;
      gap: 4px;
      padding: 0 0.75rem;
      background: linear-gradient(to top, rgba(0,0,0,0.7) 0%, transparent 100%);
      z-index: 10;
      overflow-x: hidden;
      cursor: pointer;
    }

    .thumbnail-item {
      position: relative;
      width: 56px;
      height: 56px;
      border-radius: 4px;
      overflow: hidden;
      cursor: pointer;
      flex-shrink: 0;
      border: 2px solid transparent;
      transition: border-color 0.15s, transform 0.15s, opacity 0.15s;
      opacity: 0.6;
    }

    .thumbnail-item:hover {
      opacity: 0.9;
      transform: scale(1.08);
    }

    .thumbnail-item.active {
      border-color: #6366f1;
      opacity: 1;
      transform: scale(1.12);
      box-shadow: 0 0 8px rgba(99, 102, 241, 0.6);
    }

    .thumbnail-image {
      width: 100%;
      height: 100%;
      object-fit: cover;
      display: block;
      pointer-events: none;
    }

    .video-icon-overlay {
      position: absolute;
      top: 50%;
      left: 50%;
      transform: translate(-50%, -50%);
      color: white;
      font-size: 16px;
      text-shadow: 0 1px 3px rgba(0,0,0,0.7);
      pointer-events: none;
    }

    @media (max-width: 768px) {
      .nav-btn { font-size: 2rem; padding: 0.5rem 1rem; }
      .prev-btn { left: 0.5rem; }
      .next-btn { right: 0.5rem; }
      .thumbnail-strip { height: 56px; gap: 3px; padding: 0 0.5rem; }
      .thumbnail-item { width: 44px; height: 44px; }
    }
  `]
})
export class PresentationComponent implements OnInit, OnDestroy {
  private route = inject(ActivatedRoute);
  private router = inject(Router);
  private presentationService = inject(PresentationService);
  private photoService = inject(PhotoService);
  private galleryState = inject(GalleryStateService);
  private subscriptions = new Subscription();

  ngOnInit(): void {
    const state = this.presentationService.getState();
    console.log('🎬 [COMP] ngOnInit - service items:', state.items.length, 'index:', state.currentIndex);
    
    // If no items yet, load from gallery
    if (state.items.length === 0) {
      this.loadGalleryItems();
    }
  }

  private loadGalleryItems(): void {
    const shareToken = this.route.snapshot.queryParamMap.get('shareToken');
    const isSharedMedia = this.route.snapshot.queryParamMap.get('source') === 'shared';
    
    const load = (response: any, mapFn: (p: any) => MediaItem) => {
      const items = response.media?.map(mapFn) || response.items?.map(mapFn) || [];
      console.log('📥 [COMP] Loaded', items.length, 'items from gallery');
      if (items.length > 0) {
        this.presentationService.open(items, 0);
        console.log('🔓 [COMP] Opened presentation with items, first id:', items[0].id);
      }
    };

    if (shareToken) {
      this.photoService.listPublicShareMedia(shareToken, 50, 0).subscribe({
        next: (response: any) => load(response, (p: any) => ({
          id: p.id, path: p.path, filename: p.filename, mediaType: p.mediaType
        })),
        error: (err) => console.error('Failed to load public share media', err)
      });
    } else if (isSharedMedia) {
      this.photoService.listSharedMedia(50, 0).subscribe({
        next: (response: any) => load(response, (p: any) => ({
          id: p.id || p.ID,
          path: `${this.photoService['API_BASE_URL']}/media/shared/${(p.id || p.ID)}/original`,
          filename: p.filename || '',
          mediaType: (p.mediaType || 'photo')
        })),
        error: (err) => console.error('Failed to load shared media', err)
      });
    } else {
      this.photoService.listMedia(50, 0).subscribe({
        next: (response: any) => load(response, (p: any) => ({
          id: p.id, path: p.path, filename: p.filename, mediaType: p.mediaType
        })),
        error: (err) => console.error('Failed to load media', err)
      });
    }
  }

  getThumbnailUrl(item: MediaItem): string {
    if (item.id) {
      if (item.path?.includes('/media/shared/')) {
        return `${this.photoService['API_BASE_URL']}/media/shared/${item.id}/thumb`;
      }
      return `${this.photoService['API_BASE_URL']}/media/${item.id}/thumb`;
    }
    return '';
  }

  // Read directly from service - no sync needed
  get allItems(): MediaItem[] {
    return this.presentationService.getItems();
  }

  get currentIndex(): number {
    const idx = this.presentationService.getCurrentIndex();
    const item = this.presentationService.getCurrentItem();
    console.log('📊 [COMP] currentIndex getter:', idx, '| id:', item?.id, '| file:', item?.filename);
    return idx;
  }

  get currentItem(): MediaItem | undefined {
    const item = this.presentationService.getCurrentItem();
    console.log('🖼️ [COMP] currentItem getter:', item?.id, '| file:', item?.filename);
    return item;
  }

  next(): void {
    console.log('➡️ [COMP] next() called');
    this.presentationService.next();
  }

  previous(): void {
    console.log('⬅️ [COMP] previous() called');
    this.presentationService.previous();
  }

  goTo(index: number): void {
    console.log('🖱️ [COMP] goTo(' + index + ') called');
    this.presentationService.goTo(index);
  }

  close(): void {
    console.log('❌ [COMP] close() called');
    // CRITICAL: Save current item to localStorage BEFORE closing (close() clears items)
    const currentItem = this.presentationService.getCurrentItem();
    console.log('   Current item id before close:', currentItem?.id);
    
    if (currentItem?.id) {
      // Store in localStorage like gallery_current_page
      this.galleryState.savePresentationItem(currentItem.id);
      console.log('   💾 Saved presentation_current_item to localStorage:', currentItem.id);
    }
    
    this.presentationService.close();
    
    if (currentItem?.id) {
      console.log('🚀 [COMP] Navigating to /photos/', currentItem.id);
      this.router.navigate(['/photos', currentItem.id]);
    } else {
      console.log('⚠️ [COMP] No item, going back');
      window.history.back();
    }
  }

  @HostListener('document:keydown', ['$event'])
  handleKeyboard(event: KeyboardEvent): void {
    if (!this.allItems.length) return;
    switch (event.key) {
      case 'ArrowRight': this.next(); break;
      case 'ArrowLeft': this.previous(); break;
      case 'Escape': this.close(); break;
    }
  }

  ngOnDestroy(): void {
    this.subscriptions.unsubscribe();
  }
}
