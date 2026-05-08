import { Injectable } from '@angular/core';

@Injectable({
  providedIn: 'root'
})
export class GalleryStateService {
  private STORAGE_KEY = 'gallery_current_page';

  constructor() {}

  /**
   * Save the current page number to localStorage
   */
  saveCurrentPage(page: number): void {
    localStorage.setItem(this.STORAGE_KEY, page.toString());
  }

  /**
   * Retrieve the current page number from localStorage
   * Returns 1 as default if no value found
   */
  getCurrentPage(): number {
    const stored = localStorage.getItem(this.STORAGE_KEY);
    return stored ? parseInt(stored, 10) : 1;
  }

  /**
   * Clear the stored page state (useful for logout or reset)
   */
  clearState(): void {
    localStorage.removeItem(this.STORAGE_KEY);
  }
}
