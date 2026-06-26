import { HttpClient, HttpErrorResponse, HttpParams } from '@angular/common/http';
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
    
    // Determine media type - check multiple possible field names (MediaType, MediaType, media_type, mediaType)
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
    // Use /media/{id} endpoint which works for both photos and videos (ownership check)
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

  /**
   * Fetches a shared media item using /media/shared/{id} endpoint.
   * This does NOT check ownership — it checks sharee access instead.
   */
  getSharedMedia(id: string): Observable<Photo> {
    return this.http.get<any>(`${this.API_BASE_URL}/media/shared/${id}`)
      .pipe(
        map(response => {
          // Backend returns media object directly, not wrapped
          const mediaData = response?.Media || response;
          return this.normalizeSharedMedia(mediaData);
        }),
        catchError(this.handleError)
      );
  }

  /**
   * Fetches media detail for a public share using the album token and media path.
   * This is the PUBLIC endpoint - no authentication required.
   */
  getPublicShareMedia(token: string, mediaPath: string): Observable<Photo> {
    const password = sessionStorage.getItem(`share_password_${token}`);
    let params = new HttpParams().set('path', encodeURIComponent(mediaPath));
    if (password) {
      params = params.set('password', password);
    }
    return this.http.get<any>(`${this.API_BASE_URL}/public/shares/album/${token}/media`, { params: params })
      .pipe(
        map(response => this.normalizePublicShareMedia(response)),
        catchError(this.handleError)
      );
  }

  /**
   * Fetches media detail for a public share using just the media ID (path).
   * This is the PUBLIC endpoint - no authentication required.
   */
  getPublicShareMediaByPath(token: string, mediaPath: string): Observable<Photo> {
    const password = sessionStorage.getItem(`share_password_${token}`);
    let params = new HttpParams().set('path', encodeURIComponent(mediaPath));
    if (password) {
      params = params.set('password', password);
    }
    return this.http.get<any>(`${this.API_BASE_URL}/public/shares/album/${token}/media`, { params: params })
      .pipe(
        map(response => this.normalizePublicShareMedia(response)),
        catchError(this.handleError)
      );
  }

  /**
   * Fetches a paginated list of media items from a public share album.
   * This is the PUBLIC endpoint - no authentication required.
   */
  listPublicShareMedia(token: string, limit: number = 20, offset: number = 0, password?: string | null): Observable<any> {
    let url = `${this.API_BASE_URL}/public/shares/album/${token}/media?limit=${limit}&offset=${offset}`;
    if (password) {
      url += `&password=${encodeURIComponent(password)}`;
    }
    return this.http.get<any>(url)
      .pipe(
        map(response => ({
          media: response.media || [],
          totalItems: response.totalItems || 0,
          limit: response.limit || limit,
          offset: response.offset || offset
        })),
        catchError(this.handleError)
      );
  }

  /**
   * Gets the thumbnail URL for a public share media item.
   */
  getPublicShareThumbnailUrl(token: string, mediaPath: string): string {
    const password = sessionStorage.getItem(`share_password_${token}`);
    let url = `${this.API_BASE_URL}/public/shares/album/${token}/media/thumb?path=${encodeURIComponent(mediaPath)}`;
    if (password) {
      url += `&password=${encodeURIComponent(password)}`;
    }
    return url;
  }

  /**
   * Gets the original file URL for a public share media item.
   */
  getPublicShareOriginalUrl(token: string, mediaPath: string): string {
    const password = sessionStorage.getItem(`share_password_${token}`);
    let url = `${this.API_BASE_URL}/public/shares/album/${token}/media/original?path=${encodeURIComponent(mediaPath)}`;
    if (password) {
      url += `&password=${encodeURIComponent(password)}`;
    }
    return url;
  }

  /**
   * Fetches a shared album detail using /albums/shared/{id} endpoint.
   * This does NOT check ownership — it checks sharee access instead.
   */
  getSharedAlbum(id: string): Observable<any> {
    return this.http.get<any>(`${this.API_BASE_URL}/albums/shared/${id}`)
      .pipe(
        map(response => response),
        catchError(this.handleError)
      );
  }

  /**
   * Fetches a paginated list of media items shared with the current user.
   * @param limit Number of items to fetch (default: 20)
   * @param offset Number of items to skip (default: 0)
   */
  listSharedMedia(limit: number = 20, offset: number = 0): Observable<any> {
    return this.http.get<any>(`${this.API_BASE_URL}/media/shared?limit=${limit}&offset=${offset}`)
      .pipe(
        map(response => ({
          items: response.items || [],
          total: response.total || 0,
          limit: response.limit || limit,
          offset: response.offset || offset
        })),
        catchError(this.handleError)
      );
  }

  /**
   * Fetches media items belonging to a shared album.
   */
  getSharedAlbumMedia(albumId: string, limit = 50, offset = 0): Observable<any> {
    return this.http.get<any>(`${this.API_BASE_URL}/albums/shared/${albumId}/media?limit=${limit}&offset=${offset}`)
      .pipe(
        map(response => response),
        catchError(this.handleError)
      );
  }

  /**
   * Normalizes shared media data from /media/shared/{id} endpoint.
   * The backend may return the path as either "path" or "Path", and file type info similarly.
   */
  private normalizeSharedMedia(p: any): Photo {
    const id = p.ID ?? p.id ?? '';
    
    // Determine media type - check multiple possible field names
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
    
    const filePath = id ? `${this.API_BASE_URL}/media/shared/${id}/original` : '';
    
    return {
      id: id,
      path: filePath,
      thumbnailUrl: id ? `${this.API_BASE_URL}/media/shared/${id}/thumb` : '',
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

  /**
   * Deletes a media item by ID
   */
  deleteMedia(id: string): Observable<any> {
    return this.http.delete(`${this.API_BASE_URL}/media/${id}`)
      .pipe(
        catchError(this.handleError)
      );
  }

  /**
   * Normalizes media data from the public share album media endpoint.
   * Returns a Photo with thumbnailUrl and path pointing to the public share endpoints.
   */
  private normalizePublicShareMedia(p: any): Photo {
    const id = p.ID ?? p.id ?? '';
    const mediaPath = p.Path ?? p.path ?? '';
    
    // Determine media type
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
    
    // Build URLs using the public share endpoints
    const token = p.Token ?? p.token ?? '';
    const password = sessionStorage.getItem(`share_password_${token}`);
    
    const filePath = mediaPath && token
      ? this.buildPublicShareUrl(token, mediaPath, 'original', password)
      : '';
    
    return {
      id: id,
      path: filePath,
      thumbnailUrl: mediaPath && token
        ? this.buildPublicShareUrl(token, mediaPath, 'thumb', password)
        : '',
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

  private buildPublicShareUrl(token: string, mediaPath: string, type: 'thumb' | 'original', password?: string | null): string {
    let url = `${this.API_BASE_URL}/public/shares/album/${token}/media/${type}?path=${encodeURIComponent(mediaPath)}`;
    if (password) {
      url += `&password=${encodeURIComponent(password)}`;
    }
    return url;
  }
  private handleError(error: HttpErrorResponse) {
    console.error('API Error:', error);
    return throwError(() => new Error(error.message || 'An error occurred'));
  }
}
