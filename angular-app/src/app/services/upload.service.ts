import { Injectable } from '@angular/core';
import { HttpClient, HttpEventType } from '@angular/common/http';
import { Observable, Subject, firstValueFrom, throwError } from 'rxjs';
import { environment } from '../../environments/environment';

export interface UploadProgressEvent {
  fileName: string;
  progress: number; // percentage (0-100)
  status: 'uploading' | 'completed' | 'error';
  error?: string;
}

@Injectable({
  providedIn: 'root'
})
export class UploadService {
  private API_BASE_URL = environment.apiBaseUrl;
  
  /**
   * Emits progress events during upload. 
   */
  private uploadProgress$ = new Subject<UploadProgressEvent>();
  
  getUploadProgress(): Observable<UploadProgressEvent> {
    return this.uploadProgress$.asObservable();
  }

  constructor(private http: HttpClient) {}

  /**
   * Uploads a single file to the /media/upload/single endpoint.
   * Sends file.mtime as fileCreatedAt for filesystem creation time tracking.
   */
  uploadSingleFile(file: File): Observable<any> {
    const formData = new FormData();
    
    formData.append('file', file, file.name);
    
    // Send mtime so server populates file_created_at field
    // Go server expects "2006/01/02 15:04:05" format (YYYY/MM/DD HH:MM:SS)
    if (file.lastModified) {
      const date = new Date(file.lastModified);
      const formatted = `${date.getFullYear()}/${String(date.getMonth() + 1).padStart(2, '0')}/${String(date.getDate()).padStart(2, '0')} ${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}:${String(date.getSeconds()).padStart(2, '0')}`;
      formData.append('fileCreatedAt', formatted);
    }

    return this.http.post(`${this.API_BASE_URL}/media/upload/single`, formData, {
      reportProgress: true,
      observe: 'events'
    });
  }

  /**
   * Uploads multiple files SERIALLY — one file at a time.
   * This is critical for mobile stability: parallel uploads on mobile networks
   * cause connection drops, "stream closed" errors, and flaky results.
   * 
   * Emits UploadProgressEvent for each file as it uploads, with a cumulative
   * progress percentage based on how many files have been uploaded.
   */
  uploadFiles(files: any[]): Observable<any> {
    const actualFiles: File[] = [];
    
    for (const file of files) {
      const f = ('nativeFile' in file && typeof file !== 'string') ? file.nativeFile : file;
      if (f instanceof File) {
        actualFiles.push(f);
      }
    }

    if (actualFiles.length === 0) {
      return new Observable(observer => observer.next({ uploaded: [], skipped_duplicates: [], errors: [] }));
    }

    // Track per-file results
    const uploadedItems: any[] = [];
    const skippedDuplicates: any[] = [];
    const errors: any[] = [];
    
    return new Observable(observer => {
      const total = actualFiles.length;

      // Process files one at a time in order
      let currentIndex = 0;
      
      const processNext = () => {
        if (currentIndex >= total) {
          // All files processed
          observer.next({ 
            uploaded: uploadedItems,
            skipped_duplicates: skippedDuplicates,
            errors: errors
          });
          observer.complete();
          return;
        }

        const file = actualFiles[currentIndex];
        currentIndex++;

        // Build form data
        const formData = new FormData();
        formData.append('file', file, file.name);
        
        if (file.lastModified) {
          const date = new Date(file.lastModified);
          const formatted = `${date.getFullYear()}/${String(date.getMonth() + 1).padStart(2, '0')}/${String(date.getDate()).padStart(2, '0')} ${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}:${String(date.getSeconds()).padStart(2, '0')}`;
          formData.append('fileCreatedAt', formatted);
        }

        // Send upload progress event (start)
        observer.next({ 
          fileName: file.name, 
          status: 'uploading' as const,
          progress: 0 
        });

        this.http.post(
          `${this.API_BASE_URL}/media/upload/single`,
          formData,
          { reportProgress: true, observe: 'events' }
        ).subscribe({
          next: (event: any) => {
            if (event.type === HttpEventType.UploadProgress && event.total) {
              const percent = Math.round((event.loaded / event.total) * 100);
              // Overall progress = percentage of files completed (not per-file %)
              // This is cleaner: 0% → 33% → 66% → 100% for 3 files
              const fileProgress = (currentIndex / total) * 100;
              // For in-progress file, blend the file upload % into overall
              // e.g., file 2 of 3 at 50% upload = overall ~50%
              
              observer.next({ 
                fileName: file.name, 
                progress: fileProgress, 
                status: 'uploading' as const 
              });
            } else if (event.type === HttpEventType.Response && event.body) {
              const resData = event.body as any;
              if (resData && resData.uploaded) {
                uploadedItems.push(...resData.uploaded);
              }
              if (resData && resData.skipped_duplicates) {
                skippedDuplicates.push(...resData.skipped_duplicates);
              }
              
              // Mark this file as completed
              observer.next({ 
                fileName: file.name, 
                progress: (currentIndex / total) * 100, 
                status: 'completed' as const 
              });
            }
          },
          error: (err: any) => {
            errors.push({ file: file.name, error: err.message });
            
            observer.next({ 
              fileName: file.name, 
              progress: (currentIndex / total) * 100, 
              status: 'error' as const,
              error: err.message
            });
          },
          complete: () => {
            // Process the next file
            processNext();
          }
        });
      };

      // Start processing
      processNext();
    });
  }

  /**
   * Helper to check if a file is an allowed media type based on extension/MIME.
   */
  static isValidMediaType(file: File): boolean {
    const validTypes = [
      'image/jpeg', 'image/png', 
      'video/mp4', 'video/quicktime'
    ];
    
    if (validTypes.includes(file.type)) return true;

    const ext = file.name.split('.').pop()?.toLowerCase();
    return ['jpg', 'jpeg', 'png', 'mp4', 'mov'].includes(ext || '');
  }
}
