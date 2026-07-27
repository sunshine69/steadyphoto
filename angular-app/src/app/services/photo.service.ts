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
  public normalizeMetadata(p: any): Photo['metadata'] {
    // Handle both API response styles: PascalCase (Metadata) and camelCase (metadata)
    const m = p.Metadata || p.metadata;
    if (!m) return undefined;
    
    // If Metadata is a flat object with string keys (from JSONB), map directly
    if (typeof m === 'object' && !Array.isArray(m) && Object.values(m).every(v => typeof v === 'string' || typeof v === 'number' || v == null)) {
      return {
        make: m.Make || m.make,
        model: m.Model || m.model,
        lens_model: m.LensModel || m.lens_model || m.Lens,
        exposure_time: m.ExposureTime || m.exposure_time,
        f_number: m.FNumber || m.f_number,
        iso: m.ISO || m.iso || m.ISOSpeedRatings || m.iso_speed,
        focal_length: m.FocalLength || m.focal_length || m.FocalLenmm,
        exposure_program: m.ExposureProgram || m.exposure_program,
        white_balance: m.WhiteBalance || m.white_balance,
        flash: m.Flash || m.flash || m.FlashFired,
        color_space: m.ColorSpace || m.color_space,
        datetime_original: m.DateTimeOriginal || m.datetime_original || m.DateTime,
        datetime: m.DateTime || m.datetime,
        datetime_digitized: m.DateTimeDigitized || m.datetime_digitized,
        image_width: m.ImageWidth || m.image_width || m.PixelXDimension,
        image_length: m.ImageLength || m.image_length || m.PixelYDimension,
        orientation: m.Orientation || m.orientation,
        gps_latitude: m.GPSLatitude || m.gps_latitude,
        gps_longitude: m.GPSLongitude || m.gps_longitude,
        gps_altitude: m.GPSAltitude || m.gps_altitude,
        gps_latitude_ref: m.GPSLatitudeRef || m.gps_latitude_ref,
        gps_longitude_ref: m.GPSLongitudeRef || m.gps_longitude_ref,
        software: m.Software || m.software,
        artist: m.Artist || m.artist,
        image_description: m.ImageDescription || m.image_description,
      };
    }
    
    // Legacy nested object format
    return {
      make: m.Make || m.make,
      model: m.Model || m.model,
      lens_model: m.LensModel || m.lens_model,
      exposure_time: m.ExposureTime || m.exposure_time,
      f_number: m.FNumber || m.f_number,
      iso: m.ISO || m.iso || m.ISOSpeedRatings,
      focal_length: m.FocalLength || m.focal_length,
    } as Photo['metadata'];
  }

  private normalizePhoto(p: any): Photo {
    const id = p.ID ?? p.id ?? '';
    
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
    
    const filePath = id ? `${this.API_BASE_URL}/media/${id}/original` : '';
    
    return {
      id: id,
      path: filePath,
      thumbnailUrl: id ? `${this.API_BASE_URL}/media/${id}/thumb` : '',
      filename: p.Filename ?? p.filename ?? '',
      captured_at: p.capturedAt ?? p.capturedAt ?? '',
      width: p.Width ?? p.width,
      height: p.Height ?? p.height,
      size: p.Size ?? p.size,
      type: p.Type ?? p.type,
      mediaType: mediaType,
      metadata: this.normalizeMetadata(p),
      videoMetadata: p.videoMetadata ? {
        duration: p.videoMetadata.duration,
        bitrate: p.videoMetadata.bitrate,
        video_codec: p.videoMetadata.video_codec,
        audio_codec: p.videoMetadata.audio_codec,
        frame_rate: p.videoMetadata.frame_rate,
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
        map(response => this.normalizePublicShareMedia(response, token)),
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
        map(response => this.normalizePublicShareMedia(response, token)),
        catchError(this.handleError)
      );
  }

  /**
   * Fetches a paginated list of media items from a public share album.
   * This is the PUBLIC endpoint - no authentication required.
   */
  listPublicShareMedia(token: string, limit: number = 20, offset: number = 0, password?: string | null): Observable<any> {
    const resolvedPassword = password ?? sessionStorage.getItem(`share_password_${token}`);
    let url = `${this.API_BASE_URL}/public/shares/album/${token}/media?limit=${limit}&offset=${offset}`;
    if (resolvedPassword) {
      url += `&password=${encodeURIComponent(resolvedPassword)}`;
    }
    return this.http.get<any>(url)
      .pipe(
        map(response => ({
          media: (response.media || []).map((p: any) => this.normalizePublicShareMedia(p, token)),
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
   * Fetches the original file as a Blob for download.
   * This is needed because browsers don't honor the 'download' attribute for images/videos.
   */
  downloadOriginal(id: string): Observable<Blob> {
    return this.http.get(`${this.API_BASE_URL}/media/${id}/original`, { responseType: 'blob' });
  }

  /**
   * Fetches the shared original file as a Blob for download.
   */
  downloadSharedOriginal(id: string): Observable<Blob> {
    return this.http.get(`${this.API_BASE_URL}/media/shared/${id}/original`, { responseType: 'blob' });
  }

  /**
   * Fetches the public share original file as a Blob for download.
   */
  downloadPublicShareOriginal(token: string, mediaPath: string): Observable<Blob> {
    const password = sessionStorage.getItem(`share_password_${token}`);
    let url = `${this.API_BASE_URL}/public/shares/album/${token}/media/original?path=${encodeURIComponent(mediaPath)}`;
    if (password) {
      url += `&password=${encodeURIComponent(password)}`;
    }
    return this.http.get(url, { responseType: 'blob' });
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
      captured_at: p.capturedAt ?? p.capturedAt ?? '',
      width: p.Width ?? p.width,
      height: p.Height ?? p.height,
      size: p.Size ?? p.size,
      type: p.Type ?? p.type,
      mediaType: mediaType,
      metadata: this.normalizeMetadata(p),
      videoMetadata: p.videoMetadata ? {
        duration: p.videoMetadata.duration,
        bitrate: p.videoMetadata.bitrate,
        video_codec: p.videoMetadata.video_codec,
        audio_codec: p.videoMetadata.audio_codec,
        frame_rate: p.videoMetadata.frame_rate,
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
  normalizePublicShareMedia(p: any, token?: string): Photo {
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
    
    // Use the token parameter (passed from the caller), falling back to response
    const resolveToken = token ?? '';
    const password = sessionStorage.getItem(`share_password_${resolveToken}`);
    
    const filePath = mediaPath && resolveToken
      ? this.buildPublicShareUrl(resolveToken, mediaPath, 'original', password)
      : '';
    
    return {
      id: id,
      path: filePath,
      thumbnailUrl: mediaPath && resolveToken
        ? this.buildPublicShareUrl(resolveToken, mediaPath, 'thumb', password)
        : '',
      filename: p.Filename ?? p.filename ?? '',
      captured_at: p.capturedAt ?? p.capturedAt ?? '',
      width: p.Width ?? p.width,
      height: p.Height ?? p.height,
      size: p.Size ?? p.size,
      type: p.Type ?? p.type,
      mediaType: mediaType,
      metadata: this.normalizeMetadata(p),
      videoMetadata: p.videoMetadata ? {
        duration: p.videoMetadata.duration,
        bitrate: p.videoMetadata.bitrate,
        video_codec: p.videoMetadata.video_codec,
        audio_codec: p.videoMetadata.audio_codec,
        frame_rate: p.videoMetadata.frame_rate,
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
