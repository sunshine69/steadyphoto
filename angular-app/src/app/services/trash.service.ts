import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { catchError, map, Observable, throwError } from 'rxjs';
import { Photo, ListPhotosResponse } from '../models/photo.model';
import { environment } from '../../environments/environment';

@Injectable({
  providedIn: 'root'
})
export class TrashService {
  private http = inject(HttpClient);
  private API_BASE_URL = environment.apiBaseUrl;

  /**
   * Lists trashed (soft-deleted) media items for the authenticated user.
   */
  listTrash(limit: number = 20, offset: number = 0): Observable<ListPhotosResponse> {
    return this.http.get<any>(`${this.API_BASE_URL}/trash?limit=${limit}&offset=${offset}`)
      .pipe(
        map(response => {
          const mediaArray = response?.media || [];
          
          // Normalize each item using the PhotoService's normalize logic inline
          const photos: Photo[] = mediaArray.map((p: any) => this.normalizePhoto(p));

          return {
            photos,
            total: response.total ?? 0
          };
        }),
        catchError(this.handleError)
      );
  }

  /**
   * Restores a soft-deleted media item back to the active library.
   */
  restoreMedia(id: string): Observable<any> {
    return this.http.patch(`${this.API_BASE_URL}/media/${id}/restore`, {})
      .pipe(
        catchError(this.handleError)
      );
  }

  /**
   * Permanently deletes a media item from the database and storage.
   */
  permanentlyDelete(id: string): Observable<any> {
    return this.http.delete(`${this.API_BASE_URL}/trash/${id}`)
      .pipe(
        catchError(this.handleError)
      );
  }

  private normalizePhoto(p: any): Photo {
    const id = p.ID ?? p.id ?? '';
    
    let mediaType: 'photo' | 'video' | undefined;
    const rawMediaType = p.MediaType || p.mediaType || p.media_type;
    if (rawMediaType) {
      const mt = String(rawMediaType).toLowerCase();
      if (mt === 'video') {
        mediaType = 'video';
      } else {
        mediaType = 'photo';
      }
    }

    return {
      id: id,
      path: p.Path ?? '',
      thumbnailUrl: id ? `${this.API_BASE_URL}/media/${id}/thumb` : '',
      filename: p.Filename ?? p.filename ?? '',
      captured_at: p.CapturedAt ?? p.captured_at ?? '',
      width: p.Width ?? p.width,
      height: p.Height ?? p.height,
      size: p.Size ?? p.size,
      type: p.Type ?? p.type,
      mediaType: mediaType,
      metadata: undefined, // Not typically needed for trash view
      videoMetadata: undefined,
      tags: p.Tags ?? p.tags ?? ''
    };
  }

  private handleError(error: any) {
    console.error('Trash API Error:', error);
    return throwError(() => new Error(error.message || 'An error occurred'));
  }
}
