import { Injectable } from '@angular/core';
import { BehaviorSubject, Observable } from 'rxjs';

@Injectable({ providedIn: 'root' })
export class UploadTriggerService {
  private isOpenSubject = new BehaviorSubject<boolean>(false);
  
  /** Observable to track modal visibility state */
  get isUploadModalOpen$(): Observable<boolean> { return this.isOpenSubject.asObservable(); }
  
  open(): void { 
    console.log('Opening upload modal...');
    // Reset any previous error states when opening new session
     if (this.isOpenSubject.value === false) {
       localStorage.removeItem('_upload_error_state');
      }
    
    this.isOpenSubject.next(true); 
  }

  close(): void { 
    console.log('Closing upload modal...');
    setTimeout(() => { // Delay hiding to allow backdrop transition if we added one, or just clean up state
      this.isOpenSubject.next(false); 
     }, 100); 
   }

  isOpen(): boolean { return this.isOpenSubject.value; }
}
