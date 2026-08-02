import { Injectable, signal } from '@angular/core';
import { BehaviorSubject, Observable, map, firstValueFrom } from 'rxjs';
import { PhotoService } from './photo.service';
import { AlbumService } from './album.service';
import { GalleryStateService } from './gallery-state.service';

export interface MediaItem {
  id: string;
  path: string;
  filename?: string;
  mediaType?: 'photo' | 'video';
  capturedAt?: string; // epoch seconds for sorting
}

interface PresentationState {
  isOpen: boolean;
  items: MediaItem[];
  currentIndex: number;
}

export interface AlbumContext {
  albumId?: string;
  albumIds?: string;
  source?: string;
  shareToken?: string;
  isSharedAlbumView?: boolean;
  galleryPage?: number;
}

@Injectable({ providedIn: 'root' })
export class PresentationService {
  private albumContext = signal<AlbumContext | null>(null);

  constructor(
    private photoService: PhotoService,
    private albumService: AlbumService,
    private galleryState: GalleryStateService
  ) {}

  private state = signal<PresentationState>({
    isOpen: false,
    items: [],
    currentIndex: 0
  });

  private _stateSubject = new BehaviorSubject(this.state());

  // Public observable for components to subscribe to
  stateChanges$: Observable<PresentationState> = this._stateSubject.asObservable();

  getState(): PresentationState { return this.state(); }
  getItems(): MediaItem[] { return this.state().items; }
  getCurrentIndex(): number { return this.state().currentIndex; }
  getCurrentItem(): MediaItem | undefined { 
    const items = this.state().items;
    if (items.length === 0) return undefined;
    return items[this.state().currentIndex];
  }
  
  isOpen$: Observable<boolean> = this._stateSubject.pipe(map(s => s.isOpen));

  private notify(): void {
    const currentState = this.state();
    this._stateSubject.next(currentState);
  }

  open(items: MediaItem[], startIndex: number, ctx?: AlbumContext): void {
    // Sort items by capturedAt ascending (oldest first) for presentation
    // This ensures clicking RIGHT (ArrowRight) goes to later dates (newer items)
    const sortedItems = [...items].sort((a, b) => {
      if (a.capturedAt && b.capturedAt) {
        return Number(a.capturedAt) - Number(b.capturedAt); // ascending = oldest first
      }
      // Fallback: if no capturedAt, compare by filename
      return (a.filename || '').localeCompare(b.filename || '');
    });

    const clampedIndex = Math.max(0, Math.min(startIndex, sortedItems.length - 1));
    this.state.update(prev => ({ ...prev, isOpen: true, items: sortedItems, currentIndex: clampedIndex }));
    // Store the album context so loadNextPage/loadPreviousPage can use it
    if (ctx) {
      this.setAlbumContext(ctx);
    }
    this.notify();
  }

  close(): void {
    this.state.update(prev => ({ ...prev, isOpen: false, items: [], currentIndex: 0 }));
    this.notify();
  }

  next(): boolean {
    if (this.state().currentIndex < this.state().items.length - 1) {
      this.state.update(prev => ({ ...prev, currentIndex: prev.currentIndex + 1 }));
      this.notify();
      return true;
    }
    return false;
  }

  previous(): boolean {
    if (this.state().currentIndex > 0) {
      this.state.update(prev => ({ ...prev, currentIndex: prev.currentIndex - 1 }));
      this.notify();
      return true;
    }
    return false;
  }

  goTo(index: number): boolean {
    if (index >= 0 && index < this.state().items.length) {
      this.state.update(prev => ({ ...prev, currentIndex: index }));
      this.notify();
      return true;
    }
    return false;
  }

  isOpen(): boolean {
    return this.state().isOpen;
  }

  setAlbumContext(ctx: AlbumContext | null): void {
    this.albumContext.set(ctx);
  }

  getAlbumContext(): AlbumContext | null {
    return this.albumContext();
  }

  clearAlbumContext(): void {
    this.albumContext.set(null);
  }

  /**
   * Check if we're at the end of the current items list (right edge)
   */
  isAtEnd(): boolean {
    return this.state().currentIndex >= this.state().items.length - 1;
  }

  /**
   * Check if we're at the beginning of the current items list (left edge)
   */
  isAtStart(): boolean {
    return this.state().currentIndex <= 0;
  }

  /**
   * Load the next page of items from the album.
   * Since items are sorted oldest-first, "next page" means NEWER items (further in the future).
   * We use timestamp-based pagination with "after" to get items newer than the last item's capturedAt.
   * Returns true if items were loaded successfully.
   */
  async loadNextPage(): Promise<boolean> {
    const ctx = this.albumContext();
    console.log('[PresentationService] loadNextPage - ctx:', ctx);

    // NO CONTEXT: Create a fresh context and fetch a brand new items list
    // Treat this like another click on "Presentation" with the current media as starting point
    if (!ctx) {
      console.log('[PresentationService] loadNextPage - no context, creating fresh fetch');
      const currentItem = this.getCurrentItem();
      if (!currentItem || this.state().items.length === 0) {
        console.log('[PresentationService] loadNextPage - no current item, returning false');
        return false;
      }

      // Clear context and use gallery state to fetch the NEXT page
      this.clearAlbumContext();
      const galleryPage = this.galleryState.getCurrentPage() || 1;
      const limit = 20;
      // Next page offset: move forward from current page
      const offset = galleryPage * limit;

      try {
        const response = await firstValueFrom<any>(
          this.photoService.listMedia(limit, offset)
        );
        const rawItems = response.photos || response.items || [];
        console.log('[PresentationService] loadNextPage - fresh fetch rawItems.length:', rawItems.length);

        if (rawItems.length > 0) {
          const newItems = this.normalizeMediaItems(rawItems, false);
          // Sort by capturedAt ascending (oldest first)
          newItems.sort((a, b) => {
            if (a.capturedAt && b.capturedAt) {
              return Number(a.capturedAt) - Number(b.capturedAt);
            }
            return (a.filename || '').localeCompare(b.filename || '');
          });

          // Find current item in new list to set it as starting index
          const foundIndex = newItems.findIndex(item => item.id === currentItem.id);
          const startIndex = foundIndex !== -1 ? foundIndex : 0;

          this.state.update(prev => ({
            ...prev,
            items: newItems,
            currentIndex: startIndex
          }));
          this.notify();
          console.log('[PresentationService] loadNextPage - SUCCESS, fresh page loaded, startIndex:', startIndex);
          return true;
        }
        console.log('[PresentationService] loadNextPage - no new items');
        return false;
      } catch (e) {
        console.error('[PresentationService] loadNextPage - EXCEPTION:', e);
        return false;
      }
    }

    // WITH CONTEXT: Normal append/prepend flow
    const currentItems = this.state().items;
    console.log('[PresentationService] loadNextPage - currentItems.length:', currentItems.length);
    if (currentItems.length === 0) { console.log('[PresentationService] loadNextPage - no items, returning false'); return false; }

    const limit = 20;

    try {
      let rawItems: any[] = [];
      
      if (ctx.isSharedAlbumView) {
        // Shared albums use offset-based pagination (no cursor support in public share API)
        const offset = currentItems.length; // offset = number of items already shown
        console.log('[PresentationService] loadNextPage - shared album, offset=', offset, ', token=', ctx.shareToken);
        const response = await firstValueFrom<any>(
          this.photoService.listPublicShareMedia(ctx.shareToken!, limit, offset)
        );
        console.log('[PresentationService] loadNextPage - shared album response:', response);
        rawItems = response.media || response.items || [];
      } else {
        // Normal albums use cursor-based pagination with capturedAt
        const lastItem = currentItems[currentItems.length - 1];
        console.log('[PresentationService] loadNextPage - lastItem:', lastItem);
        if (!lastItem?.capturedAt) { console.log('[PresentationService] loadNextPage - no capturedAt, returning false'); return false; }

        const capturedAtVal = lastItem.capturedAt;
        console.log('[PresentationService] loadNextPage - calling API with albumId=', ctx.albumId, ', limit=', limit, ', capturedAt=', capturedAtVal, ', type=', ctx.isSharedAlbumView ? 'shared' : 'normal');
        const response = await firstValueFrom<any>(
          this.albumService.getAlbumMediaPaginated(ctx.albumId!, limit, 0, capturedAtVal)
        );
        console.log('[PresentationService] loadNextPage - normal album response:', response);
        rawItems = response.media || response.items || [];
      }

      console.log('[PresentationService] loadNextPage - rawItems.length:', rawItems.length);
      if (rawItems.length > 0) {
        // The backend returns items already sorted by captured_at ASC (oldest first)
        // Since our list is oldest-first, we APPEND the new items to the end
        const newItems = this.normalizeMediaItems(rawItems, ctx.isSharedAlbumView);
        console.log('[PresentationService] loadNextPage - newItems.length:', newItems.length, 'newItems:', newItems.slice(0, 2));
        const prevItems = this.state().items;
        const newCurrentIndex = this.state().currentIndex + prevItems.length; // adjust index for appended items
        
        this.state.update(prev => ({
          ...prev,
          items: [...prevItems, ...newItems],
          currentIndex: Math.min(newCurrentIndex, prevItems.length + newItems.length - 1)
        }));
        this.notify();
        console.log('[PresentationService] loadNextPage - SUCCESS, new items appended');
        return true;
      }
      console.log('[PresentationService] loadNextPage - no new items, returning false');
      return false;
    } catch (e) {
      console.error('[PresentationService] loadNextPage - EXCEPTION:', e);
      return false;
    }
  }

  /**
   * Load the previous page of items from the album.
   * Since items are sorted oldest-first, "previous page" means OLDER items (further in the past).
   * We use timestamp-based pagination with "before" to get items older than the first item's capturedAt.
   * Returns true if items were loaded successfully.
   */
  async loadPreviousPage(): Promise<boolean> {
    const ctx = this.albumContext();
    console.log('[PresentationService] loadPreviousPage - ctx:', ctx);

    // NO CONTEXT: Create a fresh context and fetch a brand new items list
    // Treat this like another click on "Presentation" with the current media as starting point
    if (!ctx) {
      console.log('[PresentationService] loadPreviousPage - no context, creating fresh fetch');
      const currentItem = this.getCurrentItem();
      if (!currentItem || this.state().items.length === 0) {
        console.log('[PresentationService] loadPreviousPage - no current item, returning false');
        return false;
      }

      this.clearAlbumContext();
      const galleryPage = this.galleryState.getCurrentPage() || 1;
      const limit = 20;
      // Go back one FULL page (current page minus one)
      const offset = (galleryPage - 2) * limit;

      try {
        const response = await firstValueFrom<any>(
          this.photoService.listMedia(limit, Math.max(0, offset))
        );
        const rawItems = response.photos || response.items || [];
        console.log('[PresentationService] loadPreviousPage - fresh fetch rawItems.length:', rawItems.length);

        if (rawItems.length > 0) {
          const newItems = this.normalizeMediaItems(rawItems, false);
          newItems.sort((a, b) => {
            if (a.capturedAt && b.capturedAt) {
              return Number(a.capturedAt) - Number(b.capturedAt);
            }
            return (a.filename || '').localeCompare(b.filename || '');
          });

          const foundIndex = newItems.findIndex(item => item.id === currentItem.id);
          const startIndex = foundIndex !== -1 ? foundIndex : 0;

          this.state.update(prev => ({
            ...prev,
            items: newItems,
            currentIndex: startIndex
          }));
          this.notify();
          console.log('[PresentationService] loadPreviousPage - SUCCESS, fresh page loaded, startIndex:', startIndex);
          return true;
        }
        console.log('[PresentationService] loadPreviousPage - no new items');
        return false;
      } catch (e) {
        console.error('[PresentationService] loadPreviousPage - EXCEPTION:', e);
        return false;
      }
    }

    // WITH CONTEXT: Normal prepend flow
    const currentItems = this.state().items;
    console.log('[PresentationService] loadPreviousPage - currentItems.length:', currentItems.length);
    if (currentItems.length === 0) { console.log('[PresentationService] loadPreviousPage - no items, returning false'); return false; }

    const limit = 20;

    try {
      let rawItems: any[] = [];
      
      if (ctx.isSharedAlbumView) {
        const offset = 0;
        console.log('[PresentationService] loadPreviousPage - shared album, offset=', offset, ', token=', ctx.shareToken);
        const response = await firstValueFrom<any>(
          this.photoService.listPublicShareMedia(ctx.shareToken!, limit, offset)
        );
        rawItems = response.media || response.items || [];
      } else {
        const firstItem = currentItems[0];
        console.log('[PresentationService] loadPreviousPage - firstItem:', firstItem);
        if (!firstItem?.capturedAt) { console.log('[PresentationService] loadPreviousPage - no capturedAt, returning false'); return false; }

        const capturedAtVal = firstItem.capturedAt;
        console.log('[PresentationService] loadPreviousPage - calling API with albumId=', ctx.albumId, ', limit=', limit, ', capturedAt=', capturedAtVal);
        const response = await firstValueFrom<any>(
          this.albumService.getAlbumMediaPaginated(ctx.albumId!, limit, 0, undefined, capturedAtVal)
        );
        rawItems = response.media || response.items || [];
      }

      console.log('[PresentationService] loadPreviousPage - rawItems.length:', rawItems.length);
      if (rawItems.length > 0) {
        const newItems = this.normalizeMediaItems(rawItems, ctx.isSharedAlbumView);
        console.log('[PresentationService] loadPreviousPage - newItems.length:', newItems.length, 'newItems:', newItems.slice(0, 2));
        const prevItems = this.state().items;
        const newCurrentIndex = this.state().currentIndex + newItems.length;
        
        this.state.update(prev => ({
          ...prev,
          items: [...newItems, ...prevItems],
          currentIndex: Math.min(newCurrentIndex, newItems.length + prevItems.length - 1)
        }));
        this.notify();
        console.log('[PresentationService] loadPreviousPage - SUCCESS, items prepended');
        return true;
      }
      console.log('[PresentationService] loadPreviousPage - no new items, returning false');
      return false;
    } catch (e) {
      console.error('[PresentationService] loadPreviousPage - EXCEPTION:', e);
      return false;
    }
  }

  /**
   * Normalize raw media data to MediaItem format.
   * Note: Items should already be sorted by captured_at ascending from the backend.
   * For shared album items, the 'path' is already a public share URL from normalizePublicShareMedia.
   */
  private normalizeMediaItems(rawItems: any[], isShared = false): MediaItem[] {
    const apiBaseUrl = this.photoService['API_BASE_URL'];
    
    return rawItems.map((item: any) => {
      const mediaType: 'photo' | 'video' = (item.MediaType || item.mediaType || 'photo').toLowerCase() === 'video' ? 'video' : 'photo';
      const capturedAtTimestamp = item.CapturedAt ?? item.captured_at ?? item.capturedAt ?? '';
      const capturedAt = capturedAtTimestamp !== ''
        ? String(new Date(capturedAtTimestamp).getTime())
        : '';
      
      // For shared albums, the path is already a public share URL from normalizePublicShareMedia
      let path = item.path || '';
      if (!isShared && !path) {
        // For normal albums, build the path from the media ID
        path = `${apiBaseUrl}/media/${item.ID ?? item.id ?? ''}/original`;
      }
      
      return {
        id: item.ID ?? item.id ?? '',
        path,
        filename: item.Filename ?? item.filename ?? '',
        mediaType,
        capturedAt
      };
    });
  }
}
