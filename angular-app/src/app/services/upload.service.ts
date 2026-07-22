import { Injectable } from '@angular/core';
import { HttpClient, HttpEventType } from '@angular/common/http';
import { Observable, Subject, firstValueFrom } from 'rxjs';
import { environment } from '../../environments/environment';

export interface UploadProgressEvent {
  fileName: string;
  progress: number; // percentage (0-100)
  status: 'uploading' | 'completed' | 'error';
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
   * Uploads multiple files one by one via the single-file endpoint.
   * Returns combined progress + final result.
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
    const results: any[] = [];
    const errors: any[] = [];
    let completed = 0;
    const total = actualFiles.length;

    return new Observable(observer => {
      actualFiles.forEach((file, idx) => {
        const formData = new FormData();
        formData.append('file', file, file.name);
        
        if (file.lastModified) {
          const date = new Date(file.lastModified);
          const formatted = `${date.getFullYear()}/${String(date.getMonth() + 1).padStart(2, '0')}/${String(date.getDate()).padStart(2, '0')} ${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}:${String(date.getSeconds()).padStart(2, '0')}`;
          formData.append('fileCreatedAt', formatted);
        }

        this.http.post(
          `${this.API_BASE_URL}/media/upload/single`,
          formData,
          { reportProgress: true, observe: 'events' }
        ).subscribe({
          next: (event: any) => {
            if (event.type === HttpEventType.UploadProgress && event.total) {
              const percent = Math.round((event.loaded / event.total) * 100);
              const overallProgress = (percent / total) * (idx + 1);
              
              observer.next({
                fileName: file.name,
                progress: overallProgress,
                status: 'uploading' as const
              });
            } else if (event.type === HttpEventType.Response && event.body) {
              const resData = event.body as any;
              if (resData && resData.uploaded) {
                results.push(...resData.uploaded);
              }
              if (resData && resData.skipped_duplicates) {
                errors.push(...resData.skipped_duplicates);
              }
            }
          },
          error: (err: any) => {
            errors.push({ file: file.name, error: err.message });
          },
          complete: () => {
            completed++;
            if (completed === total) {
              observer.next({ 
                uploaded: results,
                skipped_duplicates: errors
              });
              observer.complete();
            }
          }
        });
      });
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
