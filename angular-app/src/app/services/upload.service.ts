import { Injectable } from '@angular/core';
import { HttpClient, HttpEventType, HttpResponse } from '@angular/common/http';
import { Observable, Subject } from 'rxjs';
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
   * Uploads files to the backend. Emits progress events for each file and a final result event.  
   */
  uploadFiles(files: any[]): Observable<any> { // Changed from File[] to handle SelectableFile wrapper objects too
    const formData = new FormData();
    
    // Append all selected files under 'files' field (matches backend expectation)
    for (const file of files) {
      // Handle both raw Files and our SelectableFile wrapper objects  
      const actualFile = ('nativeFile' in file && typeof file !== 'string') ? file.nativeFile : file;
      
      if (!(actualFile instanceof File)) continue; // Skip non-Files
      
      formData.append('files', actualFile, actualFile.name);
    }

    return this.http.post(`${this.API_BASE_URL}/media/upload`, formData, {
      reportProgress: true, // Enable progress events from HttpClient  
      observe: 'events'     // Observe all event types (upload and response)
    });
  }

  /**
   * Helper to check if a file is an allowed media type based on extension/MIME.
   */
  static isValidMediaType(file: File): boolean {
    const validTypes = [
      'image/jpeg', 'image/png', 
      'video/mp4', 'video/quicktime' // .mov often maps to quicktime or mp4 depending on OS/browser
    ];
    
    if (validTypes.includes(file.type)) return true;

    // Fallback check by extension for browsers that don't set MIME correctly
    const ext = file.name.split('.').pop()?.toLowerCase();
    return ['jpg', 'jpeg', 'png', 'mp4', 'mov'].includes(ext || '');
  }
}
