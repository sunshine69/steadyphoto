import { Injectable } from '@angular/core';
import { BehaviorSubject, Observable } from 'rxjs';

@Injectable({
  providedIn: 'root'
})
export class ExifService {
  private exifDataSubject = new BehaviorSubject<any>(null);
  private isOpenSubject = new BehaviorSubject<boolean>(false);

  exifData$: Observable<any> = this.exifDataSubject.asObservable();
  isOpen$: Observable<boolean> = this.isOpenSubject.asObservable();

  open(exifData: any): void {
    this.exifDataSubject.next(exifData);
    this.isOpenSubject.next(true);
  }

  close(): void {
    this.exifDataSubject.next(null);
    this.isOpenSubject.next(false);
  }

  getData(): any {
    return this.exifDataSubject.value;
  }

  getIsOpen(): boolean {
    return this.isOpenSubject.value;
  }
}
