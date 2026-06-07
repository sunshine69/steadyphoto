import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { BehaviorSubject, Observable, throwError } from 'rxjs';
import { catchError, tap } from 'rxjs/operators';
import { environment } from '../../environments/environment';

export interface AuthResponse {
  access_token: string;
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

  /**
   * Gets the current user's profile information.
   */
  getProfile(): Observable<any> {
    return this.http.get(`${this.API_BASE_URL}/auth/profile`, { withCredentials: true })
      .pipe(
        tap((res) => console.log('AuthService: Profile fetched', res)),
        catchError(err => {
          console.error('AuthService: Failed to fetch profile', err);
          return throwError(() => err);
        })
      );
  }

  /**
   * Updates the user's email address.
   */
  updateEmail(newEmail: string): Observable<any> {
    return this.http.patch(`${this.API_BASE_URL}/auth/profile/email`, { new_email: newEmail }, { withCredentials: true })
      .pipe(
        tap((res) => console.log('AuthService: Email updated', res)),
        catchError(err => {
          console.error('AuthService: Failed to update email', err);
          return throwError(() => err);
        })
      );
  }

  /**
   * Changes the user's password.
   */
  changePassword(currentPassword: string, newPassword: string): Observable<any> {
    return this.http.patch(`${this.API_BASE_URL}/auth/profile/password`, { 
      current_password: currentPassword,
      new_password: newPassword
    }, { withCredentials: true })
      .pipe(
        tap((res) => console.log('AuthService: Password changed', res)),
        catchError(err => {
          console.error('AuthService: Failed to change password', err);
          return throwError(() => err);
        })
      );
  }

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
          localStorage.setItem('email', email);
          
          // Store current user with role info for admin checks
          const currentUser: CurrentUser = {
            id: res.user_id,
            email: email,
            role: res.role,
            status: res.status
          };
          localStorage.setItem('currentUser', JSON.stringify(currentUser));
          
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
          // Ensure we don't trigger token refresh for registration errors
          return throwError(() => err);
        }),
        tap(() => {
          console.log('AuthService: Registration request completed');
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
          this.clearUser(); // Ensure all local storage data is wiped on logout
        }),
        catchError(err => {
          console.error('AuthService: Logout failed', err);
          // Even if server fails, we should clear local state on client side for security/UX
          this.setAuthenticated(false); 
          this.clearUser(); // Clear local storage even if logout request fails
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

  private _currentUser = new BehaviorSubject<User | null>(null);
  currentUser$ = this._currentUser.asObservable();

  setCurrentUser(user: User): void {
    localStorage.setItem('username', user.username || '');
    this._currentUser.next(user);
  }

  getCurrentUser(): User | null {
    return this._currentUser.value;
  }

  getUsername(): string {
    // First try to get username from current user object
    const user = this.getCurrentUser();
    if (user?.username) return user.username;
    
    // Then try stored username
    const storedUsername = localStorage.getItem('username');
    if (storedUsername && storedUsername.trim()) return storedUsername;
    
    // Fallback to first letter of email
    const email = localStorage.getItem('email') || '';
    if (email) {
      return email.charAt(0).toUpperCase();
    }
    
    // Ultimate fallback - just 'U' for User
    return 'U';
  }

  getEmailUsername(): string {
    // Extract the username part from email (before @ symbol)
    const email = localStorage.getItem('email') || '';
    if (!email) return 'User';
    
    const parts = email.split('@');
    return parts[0] || 'User';
  }

  clearUser(): void {
    this._currentUser.next(null);
    // Clear ALL local storage items to prevent data contamination between users
    localStorage.removeItem('username');
    localStorage.removeItem('email');
    localStorage.removeItem('currentUser');
    localStorage.removeItem('access_token');
    localStorage.removeItem('refresh_token');
  }

}
