import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Observable, tap, catchError, throwError } from 'rxjs';
import { environment } from '../../environments/environment';

export interface LoginResponse {
  user: {
    id: string;
    email: string;
  };
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
  login(email: string, password: string): Observable<LoginResponse> {
    return this.http.post<LoginResponse>(`${this.API_BASE_URL}/auth/login`, { email, password }, { withCredentials: true })
      .pipe(
        tap(() => console.log('AuthService: Login successful')),
        catchError(err => {
          console.error('AuthService: Login failed', err);
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
        catchError(err => {
          console.error('AuthService: Logout failed', err);
          return throwError(() => err);
        })
      );
  }

  /**
   * Attempts to refresh the access token using the stored refresh cookie.
   */
  refreshToken(): Observable<any> {
    // This endpoint is specifically designed for silent renewal via HttpOnly cookies
    return this.http.post(`${this.API_BASE_URL}/auth/refresh`, {}, { withCredentials: true });
  }

  /**
   * Checks if the user has an active session (basic check).
   * Note: In a production app, you might call /api/auth/me to verify server-side state.
   */
  isAuthenticated(): boolean {
    // For now, we rely on the Interceptor and 401 errors to drive auth state logic
    // A real implementation would check local storage or an 'isLoggedIn' signal/subject.
    return true; 
  }
}
