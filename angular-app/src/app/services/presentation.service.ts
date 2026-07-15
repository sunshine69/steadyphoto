import { Injectable, signal } from '@angular/core';
import { BehaviorSubject, Observable, map } from 'rxjs';

export interface MediaItem {
  id: string;
  path: string;
  filename?: string;
  mediaType?: 'photo' | 'video';
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
}

@Injectable({ providedIn: 'root' })
export class PresentationService {
  private albumContext = signal<AlbumContext | null>(null);

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

  open(items: MediaItem[], startIndex: number): void {
    const clampedIndex = Math.max(0, Math.min(startIndex, items.length - 1));
    this.state.update(prev => ({ ...prev, isOpen: true, items, currentIndex: clampedIndex }));
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
}
