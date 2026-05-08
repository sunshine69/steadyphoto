import { HttpClient, HttpErrorResponse } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { catchError, map, Observable, throwError } from 'rxjs';
import { Photo } from '../models/photo.model';
import { environment } from '../../environments/environment';

@Injectable({
  providedIn: 'root'
})
export class PhotoService {
  private http = inject(HttpClient);
  private API_BASE_URL = environment.apiBaseUrl;

  /**
   * Maps backend properties to frontend model and strictly uses 
   * the API endpoints defined in internal/api/server.go:
   * 
   * r.HandleFunc("/photos/{id}/original", s.handleGetOriginal).Methods(http.MethodGet)
   * r.HandleFunc("/photos/{id}/thumb", s.handleGetThumbnail).Methods(http.MethodGet)
   */
  private normalizePhoto(p: any): Photo {
    const id = p.ID ?? p.id ?? '';
    
    return {
      id: id,
      path: id ? `${this.API_BASE_URL}/photos/${id}/original` : '',
      thumbnailUrl: id ? `${this.API_BASE_URL}/photos/${id}/thumb` : '',
      filename: p.Filename ?? p.filename ?? '',
      captured_at: p.CapturedAt ?? p.captured_at ?? '',
      width: p.Width ?? p.width,
      height: p.Height ?? p.height,
      size: p.Size ?? p.size,
      type: p.Type ?? p.type,
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
