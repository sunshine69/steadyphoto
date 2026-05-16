import { HttpClient } from '@angular/common/http';
import { inject, Injectable } from '@angular/core';
import { catchError, throwError, Observable } from 'rxjs';
import { environment } from '../../environments/environment';
import { Album, CreateAlbumRequest, UpdateAlbumRequest, AddMediaToAlbumRequest, BulkRemoveMediaRequest } from '../models/album.model';
import { Photo } from '../models/photo.model';

@Injectable({
  providedIn: 'root'
})
export class AlbumService {
  private http = inject(HttpClient);
  private API_BASE_URL = environment.apiBaseUrl;

  /**
   * Fetches all albums for the authenticated user.
   */
  getAlbums(): Observable<Album[]> {
    return this.http.get<Album[]>(`${this.API_BASE_URL}/albums`)
      .pipe(catchError(this.handleError));
  }

  /**
   * Fetches a specific album by ID.
   */
  getAlbum(id: string): Observable<Album> {
    return this.http.get<Album>(`${this.API_BASE_URL}/albums/${id}`)
      .pipe(catchError(this.handleError));
  }

  /**
   * Creates a new album.
   */
  createAlbum(request: CreateAlbumRequest): Observable<Album> {
    return this.http.post<Album>(`${this.API_BASE_URL}/albums`, request)
      .pipe(catchError(this.handleError));
  }

  /**
   * Updates an existing album's metadata.
   */
  updateAlbum(id: string, request: UpdateAlbumRequest): Observable<Album> {
    return this.http.put<Album>(`${this.API_BASE_URL}/albums/${id}`, request)
      .pipe(catchError(this.handleError));
  }

  /**
   * Deletes an album.
   */
  deleteAlbum(id: string): Observable<void> {
    return this.http.delete<void>(`${this.API_BASE_URL}/albums/${id}`)
      .pipe(catchError(this.handleError));
  }

  /**
   * Adds multiple media assets to an album.
   */
  addMediaToAlbum(albumId: string, mediaIds: string[]): Observable<void> {
    const request: AddMediaToAlbumRequest = { mediaIds };
    return this.http.post<void>(`${this.API_BASE_URL}/albums/${albumId}/media`, request)
      .pipe(catchError(this.handleError));
  }

  /**
   * Removes a specific media asset from an album.
   */
  removeMediaFromAlbum(albumId: string, mediaId: string): Observable<void> {
    return this.http.delete<void>(`${this.API_BASE_URL}/albums/${albumId}/media/${mediaId}`)
      .pipe(catchError(this.handleError));
  }

  /**
   * Bulk removes multiple media assets from an album.
   */
  bulkRemoveMediaFromAlbum(albumId: string, mediaIds: string[]): Observable<void> {
    const request: BulkRemoveMediaRequest = { mediaIds };
    return this.http.delete<void>(`${this.API_BASE_URL}/albums/${albumId}/media`, { body: request })
      .pipe(catchError(this.handleError));
  }

  /**
   * Fetches all media assets belonging to a specific album.
   */
  getAlbumMedia(albumId: string): Observable<Photo[]> {
    return this.http.get<Photo[]>(`${this.API_BASE_URL}/albums/${albumId}/media`)
      .pipe(catchError(this.handleError));
  }

  private handleError(error: any) {
    console.error('AlbumService Error:', error);
    return throwError(() => new Error(error.message || 'An error occurred with the album service'));
  }
}
