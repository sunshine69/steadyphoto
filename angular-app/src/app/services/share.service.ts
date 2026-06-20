import { Injectable, inject } from '@angular/core';
import { HttpClient, HttpHeaders } from '@angular/common/http';
import { BehaviorSubject, Observable, Subject, throwError, of } from 'rxjs';
import { catchError, tap, switchMap, finalize, map } from 'rxjs/operators';
import { environment } from '../../environments/environment';

// Import types for internal use in this file AND re-export them for consumers
import {
  ShareRequest, PublicShareRequest,
  SharedMediaItem, SharedAlbumItem,
  CreateShareResponseFull, PublicShareLinkResponse,
  PublicShareListItem, SharedItemsResponse, SearchUser,
  ShareGroupListItem
} from '../models/share.model';

// Re-export types so components can import them from share.service.ts
export type {
  ShareRequest, PublicShareRequest,
  SharedMediaItem, SharedAlbumItem,
  CreateShareResponseFull, PublicShareLinkResponse,
  PublicShareListItem, SharedItemsResponse, SearchUser,
  ShareGroupListItem
};

// Extended shared media item with full details (for single-item view)
export interface SharedMediaDetail {
  id: string;
  userId: string; // owner of the media (not necessarily current user)
  filename: string;
  path: string;   // file path for thumbnail/original serving
  mediaType: 'photo' | 'video';
  capturedAt: string;
  sharerUserId: string; // who shared it with current user
}

// Extended shared album detail
export interface SharedAlbumDetail {
  id: string;
  name: string;
  description?: string;
  userId: string;      // owner of the album (not necessarily current user)
  createdAt: string;
  sharerUserId: string; // who shared it with current user
}

@Injectable({ providedIn: 'root' })
export class ShareService {
  private http = inject(HttpClient);
  private API_BASE_URL = environment.apiBaseUrl;

  // Subject for auth state changes (same pattern as AuthService)
  private _isAuthenticatedSubject = new BehaviorSubject<boolean>(false);
  isAuthenticated$ = this._isAuthenticatedSubject.asObservable();

  /** Flag indicating logout is in progress — prevents refresh attempts during logout */
  private _isLoggingOutSubject = new BehaviorSubject<boolean>(false);
  isLoggingOut$ = this._isLoggingOutSubject.asObservable();

  /** Subject for deduplicating concurrent 401 requests (only one refresh happens) */
  private _refreshInFlightSubject = new Subject<void>();

  // --- User-to-User Sharing API ---

  /** Create a share with specific users (user-to-user sharing) */
  createShare(request: ShareRequest): Observable<CreateShareResponseFull> {
    return this.http.post<CreateShareResponseFull>(`${this.API_BASE_URL}/shares`, request, { withCredentials: true })
      .pipe(
        tap((res) => console.log('ShareService: Share created', res)),
        catchError(err => {
          console.error('ShareService: Failed to create share', err);
          return throwError(() => err);
        })
      );
  }

  /** Search users by email for sharing purposes (returns only id and email) */
  searchUsers(query: string): Observable<SearchUser[]> {
    return this.http.get<SearchUser[]>(`${this.API_BASE_URL}/auth/users/search?query=${encodeURIComponent(query)}`, { withCredentials: true })
      .pipe(
        tap((res) => console.log('ShareService: Users searched', res)),
        catchError(err => {
          console.error('ShareService: Failed to search users', err);
          return throwError(() => err);
        })
      );
  }

  // --- Public Share Links API ---

  /** Create a public share link for media or albums */
  createPublicShareLink(request: PublicShareRequest): Observable<PublicShareLinkResponse> {
    return this.http.post<PublicShareLinkResponse>(`${this.API_BASE_URL}/public-shares`, request, { withCredentials: true })
      .pipe(
        tap((res) => console.log('ShareService: Public share link created', res)),
        catchError(err => {
          console.error('ShareService: Failed to create public share link', err);
          return throwError(() => err);
        })
      );
  }

  /** Revoke a public share link by ID */
  revokePublicShareLink(id: string): Observable<{ deleted: boolean }> {
    return this.http.delete<{ deleted: boolean }>(`${this.API_BASE_URL}/public-shares/${id}`, { withCredentials: true })
      .pipe(
        tap((res) => console.log('ShareService: Public share link revoked', res)),
        catchError(err => {
          console.error('ShareService: Failed to revoke public share link', err);
          return throwError(() => err);
        })
      );
  }

  /** List all public share links for current user */
  listMyPublicShares(): Observable<PublicShareListItem[]> {
    return this.http.get<any>(`${this.API_BASE_URL}/public-shares`, { withCredentials: true })
      .pipe(
        tap((res) => console.log('ShareService: Public shares listed', res)),
        map(response => response || []),
        catchError(err => {
          console.error('ShareService: Failed to list public shares', err);
          return throwError(() => err);
        })
      );
  }

  // --- Incoming Shares API (items shared with the current user) ---

  /** List media items shared with the current user */
  listSharedMedia(limit = 20, offset = 0): Observable<SharedItemsResponse<SharedMediaItem>> {
    return this.http.get<any>(`${this.API_BASE_URL}/media/shared?limit=${limit}&offset=${offset}`, { withCredentials: true })
      .pipe(
        tap((res) => console.log('ShareService: Shared media listed', res)),
        map(response => ({
          items: response?.items || [],
          total: response?.total || 0,
          limit: response?.limit || limit,
          offset: response?.offset || offset
        })),
        catchError(err => {
          console.error('ShareService: Failed to list shared media', err);
          return throwError(() => err);
        })
      );
  }

  /** List albums shared with the current user */
  listSharedAlbums(limit = 20, offset = 0): Observable<SharedItemsResponse<SharedAlbumItem>> {
    return this.http.get<any>(`${this.API_BASE_URL}/albums/shared?limit=${limit}&offset=${offset}`, { withCredentials: true })
      .pipe(
        tap((res) => console.log('ShareService: Shared albums listed', res)),
        map(response => ({
          items: response?.items || [],
          total: response?.total || 0,
          limit: response?.limit || limit,
          offset: response?.offset || offset
        })),
        catchError(err => {
          console.error('ShareService: Failed to list shared albums', err);
          return throwError(() => err);
        })
      );
  }

  // --- Shared Media Detail Endpoints (checks sharee access, not ownership) ---

  /** Get full metadata for a single media item that was shared with current user */
  getSharedMediaDetail(id: string): Observable<SharedMediaDetail> {
    return this.http.get<any>(`${this.API_BASE_URL}/media/shared/${id}`, { withCredentials: true })
      .pipe(
        tap((res) => console.log('ShareService: Shared media detail fetched', res)),
        map(response => ({
          id: response.ID ?? response.id,
          userId: response.UserID ?? response.userId,     // owner of the media (may not be current user!)
          filename: response.Filename ?? response.filename,
          path: response.Path ?? response.path,            // file path for thumbnail/original serving
          mediaType: response.MediaType ?? response.mediaType as 'photo' | 'video',
          capturedAt: response.CapturedAt ?? response.capturedAt,
          sharerUserId: response.SharerUserID ?? response.sharerUserId
        })),
        catchError(err => {
          console.error('ShareService: Failed to fetch shared media detail:', err);
          return throwError(() => err);
        })
      );
  }

  /** Get thumbnail URL for a shared media item */
  getSharedMediaThumbnailUrl(id: string): Observable<string | null> {
    // First try the dedicated thumbnail endpoint
    const thumbPath = `${this.API_BASE_URL}/media/shared/${id}/thumb`;
    
    return this.http.head(thumbPath, { observe: 'response', responseType: 'text' }).pipe(
      map(response => {
        if (response.status === 200) {
          // Thumbnail exists — use it
          console.log('ShareService: Shared media thumbnail available at:', thumbPath);
          return thumbPath;
        } else {
          // No thumbnail, fall back to original as fallback
          const originalPath = `${this.API_BASE_URL}/media/shared/${id}/original`;
          return originalPath;
        }
      }),
      catchError(() => {
        // If even the thumbnail request fails entirely (401/403/etc), 
        // try fetching via /original instead — this endpoint should succeed if sharee access is valid
        console.log('ShareService: Thumbnail check failed, using original as fallback');
        return of(`${this.API_BASE_URL}/media/shared/${id}/original`);
      })
    );
  }

  /** Get full metadata for a single album that was shared with current user */
  getSharedAlbumDetail(id: string): Observable<SharedAlbumDetail> {
    return this.http.get<any>(`${this.API_BASE_URL}/albums/shared/${id}`, { withCredentials: true })
      .pipe(
        tap((res) => console.log('ShareService: Shared album detail fetched', res)),
        map(response => ({
          id: response.ID ?? response.id,
          name: response.Name ?? response.name,
          description: response.Description ?? response.description, // note: may be null in DB
          userId: response.UserID ?? response.userId,     // owner of the album (may not be current user!)
          createdAt: response.CreatedAt ?? response.createdAt,
          sharerUserId: response.SharerUserID ?? response.sharerUserId
        })),
        catchError(err => {
          console.error('ShareService: Failed to fetch shared album detail:', err);
          return throwError(() => err);
        })
      );
  }

  // --- Auth state management (same pattern as AuthService) ---

  /** Attempts to refresh the access token using the stored refresh cookie. */
  // --- Outgoing Shares API (items shared BY the current user) ---

  /** List outgoing share groups created by the current user */
  listMyOutgoingShares(): Observable<ShareGroupListItem[]> {
    return this.http.get<any>(`${this.API_BASE_URL}/shares`, { withCredentials: true })
      .pipe(
        tap((res) => console.log('ShareService: Outgoing shares listed', res)),
        map(response => response || []),
        catchError(err => {
          console.error('ShareService: Failed to list outgoing shares', err);
          return throwError(() => err);
        })
      );
  }

  /** Revoke an outgoing share group by ID */
  revokeOutgoingShare(id: string): Observable<{ message: string }> {
    return this.http.delete<{ message: string }>(`${this.API_BASE_URL}/shares/${id}`, { withCredentials: true })
      .pipe(
        tap((res) => console.log('ShareService: Outgoing share revoked', res)),
        catchError(err => {
          console.error('ShareService: Failed to revoke outgoing share', err);
          return throwError(() => err);
        })
      );
  }

  refreshToken(): Observable<any> {
    return this.http.post(`${this.API_BASE_URL}/auth/refresh`, {}, { withCredentials: true })
      .pipe(
        tap(() => console.log('ShareService: Token refreshed')),
        catchError(err => {
          console.error('ShareService: Refresh failed', err);
          // If refresh fails, we must assume the user is truly logged out
          this.setAuthenticated(false);
          return throwError(() => err);
        })
      );
  }

  /** 
   * Handles a 401 error — either triggers a fresh refresh or waits for an in-flight one.
   */
  handle401(): Observable<boolean> {
    // If logout is in progress, don't try to refresh at all — just fail with 401 immediately
    if (this._isLoggingOutSubject.value) {
      console.warn('ShareService: Skipping refresh during logout');
      return throwError(() => new Error('Logout in progress'));
    }

    // If a refresh is already in flight, wait for it and then emit to signal completion
    const _isRefreshing = this._refreshInFlightSubject.observed;
    
    if (_isRefreshing) {
      console.log('ShareService: Refresh already in progress, waiting...');
      return throwError(() => new Error('Refresh in progress'));
    }

    // Mark refresh as in-flight (only the first caller does this)
    console.log('ShareService: Starting token refresh due to 401');
    
    return this.refreshToken().pipe(
      switchMap(() => {
        // Refresh succeeded — emit and signal completion
        this._refreshInFlightSubject.next();
        // Return of(true) so interceptor's switchMap executes for retry
        return of(true);
      }),
      catchError((err) => {
        console.error('ShareService: Token refresh failed', err);
        this.setAuthenticated(false);
        
        // Emit to wake up any waiting requests (they'll fail with 401)
        this._refreshInFlightSubject.next();
        return throwError(() => err);
      }),
      finalize(() => {})
    );
  }

  /** Returns true if logout is currently in progress. */
  isLoggingOut(): boolean {
    return this._isLoggingOutSubject.value;
  }

  /** Returns the synchronous value of the auth state. */
  isAuthenticated(): boolean {
    return this._isAuthenticatedSubject.value;
  }

  /** Internal method used by login/logout handlers to update the auth state stream. */
  setAuthenticated(status: boolean): void {
    this._isAuthenticatedSubject.next(status);
  }
}
