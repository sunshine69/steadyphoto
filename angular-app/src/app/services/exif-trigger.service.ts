import { Injectable } from '@angular/core';
import { Subject } from 'rxjs';

@Injectable({
  providedIn: 'root'
})
export class ExifTriggerService {
  private trigger$ = new Subject<void>();
  
  /** Observable to subscribe to when you want to know when the EXIF popup should open */
  public exifTrigger$ = this.trigger$.asObservable();
  
  /** Call this method to trigger opening the EXIF popup */
  triggerExifPopup(): void {
    console.log('[ExifTrigger] EXIF popup trigger called');
    this.trigger$.next();
  }
}
