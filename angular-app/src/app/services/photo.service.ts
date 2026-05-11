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
    
    // Determine media type - check multiple possible field names
    let mediaType: 'photo' | 'video' | undefined;
    if (p.MediaType || p.media_type) {
      const mt = (p.MediaType || p.media_type).toLowerCase();
      if (mt === 'video') {
        mediaType = 'video';
      } else {
        mediaType = 'photo';
      }
    }
    
        // Use /media/{id}/original for all files to support both photos and videos with Range requests
    const filePath = id ? `${this.API_BASE_URL}/media/${id}/original` : '';
    
    return {
      id: id,
      path: filePath,
      thumbnailUrl: id ? `${this.API_BASE_URL}/media/${id}/thumb` : '',
      filename: p.Filename ?? p.filename ?? '',
      captured_at: p.CapturedAt ?? p.captured_at ?? '',
      width: p.Width ?? p.width,
      height: p.Height ?? p.height,
      size: p.Size ?? p.size,
      type: p.Type ?? p.type,
      mediaType: mediaType,
      metadata: p.Metadata ? {
        camera: p.Metadata.Camera,
        iso: p.Metadata.Iso,
        aperture: p.Metadata.Aperture,
        focal_length: p.Metadata.FocalLength,
        gps_lat: p.Metadata.GpsLat,
        gps_lon: p.Metadata.GpsLon,
      } : undefined,
      videoMetadata: p.VideoMetadata ? {
        duration: p.VideoMetadata.Duration ?? p.VideoMetadata.duration,
        bitrate: p.VideoMetadata.Bitrate ?? p.VideoMetadata.bitrate,
        video_codec: p.VideoMetadata.VideoCodec ?? p.VideoMetadata.video_codec,
        audio_codec: p.VideoMetadata.AudioCodec ?? p.VideoMetadata.audio_codec,
        frame_rate: p.VideoMetadata.FrameRate ?? p.VideoMetadata.frame_rate,
      } : undefined,
      tags: p.Tags ?? p.tags ?? ''
    };
  }

  /**
   * Fetches a paginated list of photos.
   * @param limit Number of photos to fetch
   * @param offset Number of photos to skip
   */
  /**
   * Fetches a paginated list of all media (photos and videos).
   * @param limit Number of items to fetch
   * @param offset Number of items to skip
   */
  listMedia(limit: number = 20, offset: number = 0): Observable<ListPhotosResponse> {
    return this.http.get<any>(`${this.API_BASE_URL}/media?limit=${limit}&offset=${offset}`)
      .pipe(
        map(response => {
          const isArray = Array.isArray(response);
          const mediaArray = isArray 
            ? response 
            : (response?.photos || response?.media || response?.Photos || []);
          
          const total = isArray 
            ? mediaArray.length 
            : (response?.total ?? response?.Total ?? 0);

          return {
            photos: mediaArray.map((p: any) => this.normalizePhoto(p)),
            total: total
          };
        }),
        catchError(this.handleError)
      );
  }

  /**
   * Fetches a paginated list of photos only.
   * @param limit Number of photos to fetch
   * @param offset Number of photos to skip
   */
  listPhotos(limit: number = 20, offset: number = 0): Observable<ListPhotosResponse> {
    return this.http.get<any>(`${this.API_BASE_URL}/photos?limit=${limit}&offset=${offset}`)
      .pipe(
        map(response => {
          const isArray = Array.isArray(response);
          const photosArray = isArray 
            ? response 
            : (response?.photos || response?.media || response?.Photos || []);
          
          const total = isArray 
            ? photosArray.length 
            : (response?.total ?? response?.Total ?? 0);

          return {
            photos: photosArray.map((p: any) => this.normalizePhoto(p)),
            total: total
          };
        }),
        catchError(this.handleError)
      );
  }

  getMedia(id: string): Observable<Photo> {
    // Use /media/{id} endpoint which works for both photos and videos
    return this.http.get<any>(`${this.API_BASE_URL}/media/${id}`)
      .pipe(
        map(response => {
          // Backend returns media object directly, not wrapped
          const mediaData = response?.Media || response;
          return this.normalizePhoto(mediaData);
        }),
        catchError(this.handleError)
      );
  }

  getPhoto(id: string): Observable<Photo> {
    return this.http.get<any>(`${this.API_BASE_URL}/photos/${id}`)
      .pipe(
        map(response => {
          // Backend returns media object directly, not wrapped
          const photoData = response?.Photo || response;
          return this.normalizePhoto(photoData);
        }),
        catchError(this.handleError)
      );
  }

  /**
   * Updates tags for a media item
   */
  updateTags(id: string, tags: string): Observable<any> {
    return this.http.patch(`${this.API_BASE_URL}/media/${id}/tags`, { tags })
      .pipe(
        catchError(this.handleError)
      );
  }

  /**
   * Searches media by tag
   */
  searchByTags(tag: string, limit: number = 50): Observable<ListPhotosResponse> {
    return this.http.get<any>(`${this.API_BASE_URL}/media/search?tags=${encodeURIComponent(tag)}&limit=${limit}`)
      .pipe(
        map(response => {
          const isArray = Array.isArray(response);
          const mediaArray = isArray 
            ? response 
            : (response?.photos || response?.media || response?.Photos || []);
          
          const total = isArray 
            ? mediaArray.length 
            : (response?.total ?? response?.Total ?? 0);

          return {
            photos: mediaArray.map((p: any) => this.normalizePhoto(p)),
            total: total
          };
        }),
        catchError(this.handleError)
      );
  }

  private handleError(error: HttpErrorResponse) {
    console.error('API Error:', error);
    return throwError(() => new Error(error.message || 'An error occurred'));
  }
}
