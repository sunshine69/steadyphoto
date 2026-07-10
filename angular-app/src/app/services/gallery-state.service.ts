import { Injectable } from '@angular/core';

@Injectable({
  providedIn: 'root'
})
export class GalleryStateService {
  private STORAGE_KEY = 'gallery_current_page';
  private PRESENTATION_KEY = 'presentation_current_item';

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

  clearState(): void {
    localStorage.removeItem(this.STORAGE_KEY);
    localStorage.removeItem(this.PRESENTATION_KEY);
  }
}
