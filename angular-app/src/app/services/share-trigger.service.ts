import { Injectable } from '@angular/core';
import { BehaviorSubject, Observable } from 'rxjs';

export interface ShareModalData {
  itemId: string;
  itemType?: 'media' | 'album';
}

@Injectable({ providedIn: 'root' })
export class ShareTriggerService {
  private isOpenSubject = new BehaviorSubject<boolean>(false);
  private shareDataSubject = new BehaviorSubject<ShareModalData | null>(null);

  /** Observable to track modal visibility state */
  get isShareModalOpen$(): Observable<boolean> { return this.isOpenSubject.asObservable(); }
  
  /** Observable to get the current share data (itemId, itemType) */
  get shareData$(): Observable<ShareModalData | null> { return this.shareDataSubject.asObservable(); }

  /** Open the share modal for a specific item */
  open(itemId: string, itemType: 'media' | 'album' = 'media'): void { 
    console.log('Opening share modal...', itemId, itemType);
    this.shareDataSubject.next({ itemId, itemType });
    this.isOpenSubject.next(true); 
  }

  close(): void { 
    console.log('Closing share modal...');
    this.isOpenSubject.next(false); 
    // Clear data after a short delay to allow any animations
    setTimeout(() => {
      this.shareDataSubject.next(null);
    }, 200); 
  }

  isOpen(): boolean { return this.isOpenSubject.value; }
}
