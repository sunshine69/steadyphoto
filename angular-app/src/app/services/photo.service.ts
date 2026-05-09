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
      path: id ? `${this.API_BASE_URL}/photos/${id}/file` : '',
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

  /**
   * Fetches a paginated list of photos.
   * @param limit Number of photos to fetch
   * @param offset Number of photos to skip
   */
  listPhotos(limit: number = 20, offset: number = 0): Observable<ListPhotosResponse> {
    return this.http.get<any>(`${this.API_BASE_URL}/photos?limit=${limit}&offset=${offset}`)
      .pipe(
        map(response => {
          // 1. Determine if the response is the new object or the old array
          const isArray = Array.isArray(response);
          
          // 2. Extract photos array
          // We check for 'photos' or 'Photos' to handle potential case differences in JSON keys
          const photosArray = isArray 
            ? response 
            : (response?.media || response?.photos || response?.Photos || []);
          
          // 3. Extract total count
          const total = isArray 
            ? photosArray.length 
            : (response?.total ?? response?.Total ?? photosArray.length);

          // 4. Return the standardized ListPhotosResponse
          return {
            photos: photosArray.map((p: any) => this.normalizePhoto(p)),
            total: total
          };
        }),
        catchError(this.handleError)
      );
  }

  getPhoto(id: string): Observable<Photo> {
    return this.http.get<any>(`${this.API_BASE_URL}/photos/${id}`)
      .pipe(
        map(response => {
          const photoData = response?.Photo || response?.photo || response;
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
