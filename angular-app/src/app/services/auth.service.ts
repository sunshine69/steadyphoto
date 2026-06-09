import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { BehaviorSubject, Observable, throwError, of } from 'rxjs';
import { catchError, tap, map } from 'rxjs/operators';
import { environment } from '../../environments/environment';

export interface AuthResponse {
  access_token: string; // Included for backward compatibility if needed by legacy code, but not stored locally.
  refresh_token: string; 
  user_id: string;
  role: string;
  status: string;
}

export interface CurrentUser {
  id: string;
  email: string;
  role: string;
  status: string;
  username?: string;
}

// Simplified User type for general UI usage
export interface User {
  email: string;
  username?: string;
}

@Injectable({
  providedIn: 'root'
})
export class AuthService {
  private http = inject(HttpClient);
  private API_BASE_URL = environment.apiBaseUrl;

  /** 
   * Authentication state tracking using in-memory subjects only.
   * This prevents XSS attacks from accessing sensitive identity or session data stored in localStorage.
   */
  private _isAuthenticatedSubject = new BehaviorSubject<boolean>(false);
  isAuthenticated$ = this._isAuthenticatedSubject.asObservable();

  private _currentUser = new BehaviorSubject<CurrentUser | null>(null);
  currentUser$ = this._currentUser.asObservable();

  /**
   * Fetches the current user's profile from the server using secure HttpOnly cookies.
   * This should be called on application initialization to restore session state in memory.
   */
  getProfile(): Observable<CurrentUser> {
    return this.http.get<CurrentUser>(`${this.API_BASE_URL}/auth/profile`, { withCredentials: true })
      .pipe(
        tap((user) => {
          console.log('AuthService: Profile fetched', user);
          this._currentUser.next(user);
          this._isAuthenticatedSubject.next(true);
        }),
        catchError(err => {
          console.error('AuthService: Failed to fetch profile, session may be invalid.', err);
          this.clearUser(); // Reset state if the server rejects our credentials
          return throwError(() => err);
        })
      );
  }

  /**
   * Updates the user's email address via secure API call.
   */
  updateEmail(newEmail: string): Observable<any> {
    return this.http.patch(`${this.API_BASE_URL}/auth/profile/email`, { new_email: newEmail }, { withCredentials: true })
      .pipe(
        tap(() => console.log('AuthService: Email updated')),
        catchError(err => {
          console.error('AuthService: Failed to update email', err);
          return throwError(() => err);
        })
      );
  }

  /**
   * Changes the user's password via secure API call.
   */
  changePassword(currentPassword: string, newPassword: string): Observable<any> {
    return this.http.patch(`${this.API_BASE_URL}/auth/profile/password`, { 
      current_password: currentPassword,
      new_password: newPassword
    }, { withCredentials: true })
      .pipe(
        tap(() => console.log('AuthService: Password changed')),
        catchError(err => {
          console.error('AuthService: Failed to change password', err);
          return throwError(() => err);
        })
      );
  }

  /**
   * Logs in a user and relies on the backend setting secure HttpOnly cookies for authentication.
   */
  login(email: string, password: string): Observable<AuthResponse> {
    return this.http.post<AuthResponse>(`${this.API_BASE_URL}/auth/login`, { email, password }, { withCredentials: true })
      .pipe(
        tap((res) => {
          console.log('AuthService: Login successful');
          // SECURITY FIX: We NO LONGER store access_token or refresh_token in localStorage. 
          // The browser handles the HttpOnly cookies automatically from this response.

          const newUser: CurrentUser = {
            id: res.user_id,
            email: email,
            role: res.role,
            status: res.status
          };

          this._currentUser.next(newUser);
          this._isAuthenticatedSubject.next(true);
        }),
        catchError(err => {
          console.error('AuthService: Login failed', err);
          this.clearUser();
          return throwError(() => err);
        })
      );
  }

  /**
   * Registers a new user via secure API call.
   */
  register(email: string, password: string): Observable<any> {
    return this.http.post(`${this.API_BASE_URL}/auth/register`, { email, password }, { withCredentials: true })
      .pipe(
        catchError(err => {
          console.error('AuthService: Registration failed', err);
          return throwError(() => err);
        }),
        tap(() => console.log('AuthService: Registration request completed'))
      );
  }

  /**
   * Logs out the user by notifying the backend and clearing local in-memory state.
   */
  logout(): Observable<any> {
    return this.http.post(`${this.API_BASE_URL}/auth/logout`, {}, { withCredentials: true })
      .pipe(
        tap(() => console.log('AuthService: Logout successful')),
        catchError((err) => {
          console.error('AuthService: Logout error (server might have already invalidated session)', err);
          // We proceed to clear local state even if the network call fails for safety/UX
          return of(null); 
        }),
        tap(() => this.clearUser()) // Clear memory after successful or failed logout attempt
      );
  }

  /**
   * Attempts a silent token refresh using only secure HttpOnly cookies.
   */
  refreshToken(): Observable<any> {
    return this.http.post(`${this.API_BASE_URL}/auth/refresh`, {}, { withCredentials: true })
      .pipe(
        tap(() => console.log('AuthService: Token refreshed')),
        catchError(err => {
          console.error('AuthService: Refresh failed', err);
          this.clearUser();
          return throwError(() => err);
        })
      );
  }

  /**
   * Returns the current authentication status synchronously. 
   */
  isAuthenticated(): boolean {
    return this._isAuthenticatedSubject.value;
  }

  // Internal state update methods (not for public use)
  private setAuthenticated(status: boolean): void {
    this._isAuthenticatedSubject.next(status);
  }

  /** 
   * Updates current user in-memory only. Use this when profile changes or after login/refresh.
   */
  setCurrentUser(user: CurrentUser): void {
    this._currentUser.next(user);
  }

  getCurrentUser(): CurrentUser | null {
    return this._currentUser.value;
  }

  /** 
   * Retrieves the username from in-memory state, falling back to email part if needed for UI display only.
   */
  getUsername(): string {
    const user = this.getCurrentUser();
    if (user?.username) return user.username;
    if (user?.email) {
      // Fallback: show the prefix of their email as a username in the UI header
      return user.email.split('@')[0] || 'User';
    }
    return 'Guest';
  }

  /** 
   * Clears all sensitive identity and session data from application memory.
   */
  clearUser(): void {
    console.log('AuthService: Clearing in-memory authentication state.');
    this._currentUser.next(null);
    this._isAuthenticatedSubject.next(false);
  }

}