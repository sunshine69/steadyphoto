import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { BehaviorSubject, Observable, Subject, throwError, of } from 'rxjs';
import { catchError, tap, switchMap, finalize } from 'rxjs/operators';
import { environment } from '../../environments/environment';

export interface AuthResponse {
  access_token: string;
  refresh_token: string; // kept for backward compatibility (Android scanner), but NOT stored in localStorage
  user_id: string;
  role: string;
  status: string;
}

export interface CurrentUser {
  id: string;
  email: string;
  username?: string;
  role: string;
  status: string;
}

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

  /** Flag indicating logout is in progress — prevents refresh attempts during logout */
  private _isLoggingOutSubject = new BehaviorSubject<boolean>(false);
  isLoggingOut$ = this._isLoggingOutSubject.asObservable();

  /** Subject for deduplicating concurrent 401 requests (only one refresh happens) */
  private _refreshInFlightSubject = new Subject<void>();
  
  /** Observable for auth state changes */
  private _isAuthenticatedSubject = new BehaviorSubject<boolean>(false);
  isAuthenticated$ = this._isAuthenticatedSubject.asObservable();

  /** Observable for admin status */
  private _isAdminSubject = new BehaviorSubject<boolean>(false);
  isAdmin$ = this._isAdminSubject.asObservable();

  /** Gets the current user's profile information. */
  getProfile(): Observable<any> {
    return this.http.get(`${this.API_BASE_URL}/auth/profile`, { withCredentials: true })
      .pipe(
        tap((res) => res),
        catchError(err => {
          console.error('AuthService: Failed to fetch profile', err);
          return throwError(() => err);
        })
      );
  }

  /** Updates the user's email address. */
  updateEmail(newEmail: string): Observable<any> {
    return this.http.patch(`${this.API_BASE_URL}/auth/profile/email`, { new_email: newEmail }, { withCredentials: true })
      .pipe(
        tap((res) => res),
        catchError(err => {
          console.error('AuthService: Failed to update email', err);
          return throwError(() => err);
        })
      );
  }

  /** Changes the user's password. */
  changePassword(currentPassword: string, newPassword: string): Observable<any> {
    return this.http.patch(`${this.API_BASE_URL}/auth/profile/password`, { 
      current_password: currentPassword,
      new_password: newPassword
    }, { withCredentials: true })
      .pipe(
        tap((res) => res),
        catchError(err => {
          console.error('AuthService: Failed to change password', err);
          return throwError(() => err);
        })
      );
  }

  /** Logs in a user and sets session cookies via backend response. */
  login(email: string, password: string): Observable<AuthResponse> {
    return this.http.post<AuthResponse>(`${this.API_BASE_URL}/auth/login`, { email, password }, { withCredentials: true })
      .pipe(
        tap((res) => {
          // Store access_token in sessionStorage (cleared on tab close) for Bearer token usage.
          if (res.access_token) {
            sessionStorage.setItem('access_token', res.access_token);
          }
          
          localStorage.setItem('email', email);
          
          const currentUser: CurrentUser = {
            id: res.user_id,
            email: email,
            role: res.role,
            status: res.status
          };
          localStorage.setItem('currentUser', JSON.stringify(currentUser));
          
          // Set the current user and admin status via the BehaviorSubject
          this.setCurrentUser(currentUser);
          this.setAuthenticated(true);
        }),
        catchError(err => {
          console.error('AuthService: Login failed', err);
          // Don't clear user on login failure — the user might retry with correct credentials
          return throwError(() => err);
        })
      );
  }

  /** Registers a new user. */
  register(email: string, password: string): Observable<any> {
    return this.http.post(`${this.API_BASE_URL}/auth/register`, { email, password }, { withCredentials: true })
      .pipe(
        catchError(err => {
          console.error('AuthService: Registration failed', err);
          // Ensure we don't trigger token refresh for registration errors
          return throwError(() => err);
        }),
        tap(() => {})
      );
  }

  /** Logs out the user and clears session on backend. */
  logout(): Observable<any> {
    // IMPORTANT: Set flag FIRST before any concurrent requests can fire — prevents avalanche
    this._isLoggingOutSubject.next(true);
    
    return this.http.post(`${this.API_BASE_URL}/auth/logout`, {}, { withCredentials: true })
      .pipe(
        tap(() => {
          // Logout successful
        }),
        catchError(err => {
          console.error('AuthService: Logout failed', err);
          // Even if server fails, we should clear local state on client side for security/UX
          return throwError(() => err);
        }),
        finalize(() => {
          // Reset the flag after logout completes (success or error) — allows future 401s to refresh again
          this._isLoggingOutSubject.next(false);
          
          // Always clear local auth state on logout (success or error)
          this.clearUser();
          this.setAuthenticated(false);
        })
      );
  }

  /** Attempts to refresh the access token using the stored refresh cookie. */
  refreshToken(): Observable<any> {
    return this.http.post(`${this.API_BASE_URL}/auth/refresh`, {}, { withCredentials: true })
      .pipe(
        tap(() => {}),
        catchError(err => {
          console.error('AuthService: Refresh failed', err);
          // If refresh fails, we must assume the user is truly logged out
          this.setAuthenticated(false);
          return throwError(() => err);
        })
      );
  }

  /** 
   * Handles a 401 error — either triggers a fresh refresh or waits for an in-flight one.
   * Returns true if waiting (another request is already refreshing), false if we triggered it ourselves.
   */
  handle401(): Observable<boolean> {
    // If logout is in progress, don't try to refresh at all — just fail with 401 immediately
    if (this._isLoggingOutSubject.value) {
      return throwError(() => new Error('Logout in progress'));
    }

    // If a refresh is already in flight, wait for it and then emit to signal completion
    if (!this._refreshInFlightSubject.observed && this._isRefreshing) {
      return throwError(() => new Error('Refresh in progress'));
    }

    // Mark refresh as in-flight (only the first caller does this)
    if (!this._refreshInFlightSubject.observed && !this._isRefreshing) {
      this._isRefreshing = true;
      
      return this.refreshToken().pipe(
        switchMap(() => {
          // Refresh succeeded — emit and signal completion
          this._refreshInFlightSubject.next();
          // Return of(true) so interceptor's switchMap executes for retry
          return of(true);
        }),
        catchError((err) => {
          console.error('AuthService: Token refresh failed', err);
          this.setAuthenticated(false);
          
          // Emit to wake up any waiting requests (they'll fail with 401)
          this._refreshInFlightSubject.next();
          return throwError(() => err);
        }),
        finalize(() => {
          // Reset the in-flight flag when done
          this._isRefreshing = false;
        })
      );
    }

    // A refresh is already in progress — wait for it to complete, then fail with 401
    return throwError(() => new Error('Refresh in progress'));
  }

  /** Returns true if logout is currently in progress. */
  isLoggingOut(): boolean {
    return this._isLoggingOutSubject.value;
  }

  /** Returns the synchronous value of the auth state. */
  isAuthenticated(): boolean {
    // First check if the BehaviorSubject was set (e.g., after login)
    if (this._isAuthenticatedSubject.value) {
      return true;
    }
    // Fallback: check if there's a stored currentUser in localStorage
    const currentUser = localStorage.getItem('currentUser');
    return currentUser !== null;
  }

  /** Internal method used by login/logout handlers to update the auth state stream. */
  setAuthenticated(status: boolean): void {
    this._isAuthenticatedSubject.next(status);
  }

  /** Sets the admin status based on the current user's role. */
  setAdminStatus(isAdmin: boolean): void {
    this._isAdminSubject.next(isAdmin);
  }

  /** Initializes auth state by checking if the user has a valid session cookie. */
  initializeAuth(): void {
    this.getProfile().subscribe({
      next: () => {
        this.setAuthenticated(true);
        // Also set the current user if not already set
        if (!this.getCurrentUser()) {
          const currentUser = localStorage.getItem('currentUser');
          if (currentUser) {
            try {
              const user = JSON.parse(currentUser);
              this._currentUser.next(user);
            } catch (e) {
              console.error('Failed to parse current user from localStorage', e);
            }
          }
        }
      },
      error: (err) => {
        console.warn('AuthService: Session invalid on page load', err);
        this.setAuthenticated(false);
        // Clear localStorage to prevent stale data on page reload
        this.clearUser();
      }
    });
  }

  private _currentUser = new BehaviorSubject<CurrentUser | null>(null);
  currentUser$ = this._currentUser.asObservable();

  setCurrentUser(user: CurrentUser): void {
    localStorage.setItem('username', user.username || '');
    this._currentUser.next(user);
    // Update admin status based on role
    this.setAdminStatus(user.role === 'admin');
  }

  getCurrentUser(): CurrentUser | null {
    return this._currentUser.value;
  }

  getUsername(): string {
    const user = this.getCurrentUser();
    if (user?.username) return user.username;
    
    const storedUsername = localStorage.getItem('username');
    if (storedUsername && storedUsername.trim()) return storedUsername;
    
    const email = localStorage.getItem('email') || '';
    if (email) {
      return email.charAt(0).toUpperCase();
    }
    
    return 'U';
  }

  getEmailUsername(): string {
    const email = localStorage.getItem('email') || '';
    if (!email) return 'User';
    
    const parts = email.split('@');
    return parts[0] || 'User';
  }

  clearUser(): void {
    this._currentUser.next(null);
    this.setAdminStatus(false);
    // Clear ALL local storage items to prevent data contamination between users
    localStorage.removeItem('username');
    localStorage.removeItem('email');
    localStorage.removeItem('currentUser');
    // access_token is in sessionStorage, not localStorage — clear it separately
    sessionStorage.removeItem('access_token');
  }

  private _isRefreshing = false;

}