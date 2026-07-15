import { Injectable } from '@angular/core';

@Injectable({
  providedIn: 'root'
})
export class GalleryStateService {
  private STORAGE_KEY = 'gallery_current_page';
  private PRESENTATION_KEY = 'presentation_current_item';
  private ALBUM_CONTEXT_KEY = 'gallery_album_context';

  constructor() {}

  saveCurrentPage(page: number): void {
    localStorage.setItem(this.STORAGE_KEY, page.toString());
  }

  getCurrentPage(): number {
    const stored = localStorage.getItem(this.STORAGE_KEY);
    return stored ? parseInt(stored, 10) : 1;
  }

  savePresentationItem(itemId: string): void {
    if (itemId) {
      localStorage.setItem(this.PRESENTATION_KEY, itemId);
    }
  }

  getPresentationItem(): string | null {
    return localStorage.getItem(this.PRESENTATION_KEY);
  }

  clearPresentationItem(): void {
    localStorage.removeItem(this.PRESENTATION_KEY);
  }

  /**
   * Save album context so we can navigate back to the album view after viewing a photo.
   */
  saveAlbumContext(albumId: string, offset: number, totalPhotos: number): void {
    const context = { albumId, offset, totalPhotos };
    localStorage.setItem(this.ALBUM_CONTEXT_KEY, JSON.stringify(context));
  }

  getAlbumContext(): { albumId: string; offset: number; totalPhotos: number } | null {
    const stored = localStorage.getItem(this.ALBUM_CONTEXT_KEY);
    if (!stored) return null;
    try {
      const parsed = JSON.parse(stored);
      if (parsed && parsed.albumId && parsed.offset !== undefined && parsed.totalPhotos !== undefined) {
        return parsed;
      }
    } catch {
      // Corrupted data, return null
    }
    return null;
  }

  clearAlbumContext(): void {
    localStorage.removeItem(this.ALBUM_CONTEXT_KEY);
  }

  clearState(): void {
    localStorage.removeItem(this.STORAGE_KEY);
    localStorage.removeItem(this.PRESENTATION_KEY);
    localStorage.removeItem(this.ALBUM_CONTEXT_KEY);
  }
}
