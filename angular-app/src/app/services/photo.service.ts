import { HttpClient, HttpErrorResponse } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { catchError, map, Observable, throwError } from 'rxjs';
import { Photo, ListPhotosResponse } from '../models/photo.model';
import { environment } from '../../environments/environment';

@Injectable({
  providedIn: 'root'
})
export class PhotoService {
  private http = inject(HttpClient);
  
  private API_BASE_URL = environment.apiBaseUrl;
  // The base URL for serving static files/images
  // Since API_BASE_URL is http://localhost:8081/api/v1, we can derive MEDIA_BASE_URL
  // or just use the same base if we want to be safe. 
  // However, API_BASE_URL contains /api/v1. 
  // Let's assume MEDIA_BASE_URL is the same base but without /api/v1 if needed.
  // But the original code had MEDIA_BASE_URL = 'http://localhost:8081'.
  private MEDIA_BASE_URL = environment.apiBaseUrl.replace('/api/v1', '');

  /**
   * Maps backend PascalCase properties to frontend camelCase properties
   * and ensures image paths point to the backend server.
   */
  private normalizePhoto(p: any): Photo {
    const formatPath = (path: string | undefined): string | undefined => {
      if (!path) return undefined;
      // If it's already an absolute URL (starts with http), return as is
      if (path.startsWith('http')) return path;
      // Otherwise, prepend the media base URL
      // Ensure we don't end up with double slashes if path starts with /
      const cleanPath = path.startsWith('/') ? path : `/${path}`;
      return `${this.MEDIA_BASE_URL}${cleanPath}`;
    };

    const id = p.ID ?? '';
    return {
      id: id,
      path: id ? `${this.API_BASE_URL}/photos/${id}/file` : '',
      filename: p.Filename ?? '',
      captured_at: p.CapturedAt ?? '',
      width: p.Width,
      height: p.Height,
      size: p.Size,
      type: p.Type,
      thumbnailUrl: id ? `${this.API_BASE_URL}/photos/${id}/thumb` : '',
      metadata: p.Metadata ? {
        camera: p.Metadata.Camera,
        iso: p.Metadata.Iso,
        aperture: p.Metadata.Aperture,
        focal_length: p.Metadata.FocalLength,
        gps_lat: p.Metadata.GpsLat,
        gps_lon: p.Metadata.GpsLon,
      } : undefined
    };
  }

  listPhotos(): Observable<Photo[]> {
    return this.http.get<any>(`${this.API_BASE_URL}/photos`)
      .pipe(
        map(response => {
          const photosArray = Array.isArray(response) ? response : (response?.Photos || response?.photos || []);
          return photosArray.map((p: any) => this.normalizePhoto(p));
        }),
        catchError(this.handleError)
      );
  }

  getPhoto(id: string): Observable<Photo> {
    return this.http.get<any>(`${this.API_BASE_URL}/photos/${id}`)
      .pipe(
        map(response => {
          const photoData = response?.Photo || response;
          return this.normalizePhoto(photoData);
        }),
        catchError(this.handleError)
      );
  }

  private handleError(error: HttpErrorResponse) {
    console.error('API Error:', error);
    return throwError(() => new Error(error.message || 'An error occurred'));
  }
}
