import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { BehaviorSubject, Observable, tap, catchError, throwError } from 'rxjs';
import { environment } from '../../environments/environment';

export interface AuthResponse {
  access_token: string;
  refresh_token: string;
  user_id: string;
}

@Injectable({
  providedIn: 'root'
})
export class AuthService {
  private http = inject(HttpClient);
  private API_BASE_URL = environment.apiBaseUrl;

  /**
   * Logs in a user and sets session cookies via backend response.
   */
  login(email: string, password: string): Observable<AuthResponse> {
    return this.http.post<AuthResponse>(`${this.API_BASE_URL}/auth/login`, { email, password }, { withCredentials: true })
      .pipe(
        tap((res) => {
          console.log('AuthService: Login successful', res);
          localStorage.setItem('access_token', res.access_token);
          localStorage.setItem('refresh_token', res.refresh_token);
          this.setAuthenticated(true);
        }),
        catchError(err => {
          console.error('AuthService: Login failed', err);
          this.setAuthenticated(false);
          return throwError(() => err);
        })
      );
  }

  /**
   * Registers a new user.
   */
  register(email: string, password: string): Observable<any> {
    return this.http.post(`${this.API_BASE_URL}/auth/register`, { email, password }, { withCredentials: true })
      .pipe(
        catchError(err => {
          console.error('AuthService: Registration failed', err);
          return throwError(() => err);
        })
      );
  }

  /**
   * Logs out the user and clears session on backend.
   */
  logout(): Observable<any> {
    return this.http.post(`${this.API_BASE_URL}/auth/logout`, {}, { withCredentials: true })
      .pipe(
        tap(() => {
          console.log('AuthService: Logout successful');
          this.setAuthenticated(false);
        }),
        catchError(err => {
          console.error('AuthService: Logout failed', err);
          // Even if server fails, we should clear local state on client side for security/UX
          this.setAuthenticated(false); 
          return throwError(() => err);
        })
      );
  }

  /**
   * Attempts to refresh the access token using the stored refresh cookie.
   */
  refreshToken(): Observable<any> {
    // This endpoint is specifically designed for silent renewal via HttpOnly cookies
    return this.http.post(`${this.API_BASE_URL}/auth/refresh`, {}, { withCredentials: true })
      .pipe(
        tap(() => console.log('AuthService: Token refreshed')),
        catchError(err => {
          console.error('AuthService: Refresh failed', err);
          // If refresh fails, we must assume the user is truly logged out
          this.setAuthenticated(false);
          return throwError(() => err);
        })
      );
  }

  /**
   * An observable stream representing the user's current authentication state.
   */
  private _isAuthenticatedSubject = new BehaviorSubject<boolean>(false);
  isAuthenticated$ = this._isAuthenticatedSubject.asObservable();

  /**
   * Returns the synchronous value of the auth state. 
   * Note: In a production app, you might call /api/auth/me to verify server-side state first.
   */
  isAuthenticated(): boolean {
    return this._isAuthenticatedSubject.value;
  }

  /**
   * Internal method used by login/logout handlers to update the auth state stream.
   */
  setAuthenticated(status: boolean): void {
    this._isAuthenticatedSubject.next(status);
  }
}
