import { Injectable, inject } from '@angular/core';
import { HttpInterceptorFn, HttpRequest, HttpHandlerFn, HttpEvent, HttpErrorResponse } from '@angular/common/http';
import { Observable, throwError, catchError, switchMap } from 'rxjs';
import { AuthService } from '../services/auth.service';

/**
 * AuthInterceptor automatically:
 * 1. Adds `withCredentials: true` to every outgoing request so HttpOnly cookies are sent and received.
 *    (Removed the redundant Authorization Bearer header logic).
 * 2. Listens for 401 Unauthorized errors, attempts a silent token refresh via /api/auth/refresh using secure cookies.
 *    If successful, it retries the original failed request once.
 */
export const authInterceptor: HttpInterceptorFn = (req: HttpRequest<unknown>, next: HttpHandlerFn): Observable<HttpEvent<unknown>> => {
  const authService = inject(AuthService);

  // SECURITY FIX: Removed manual Authorization header attachment from localStorage. 
  // We now rely strictly on Secure, HttpOnly cookies sent via withCredentials.

  // Ensure credentials (cookies) are sent for every request to allow the backend session check.
  let authReq = req;
  if (!authReq.withCredentials) {
    authReq = authReq.clone({ withCredentials: true });
  }

  return next(authReq).pipe(
    catchError((error: HttpErrorResponse) => {
      // 2. Detect Unauthorized (401) error from the server session check.
      if (error.status === 401) {
        console.warn('AuthInterceptor: Detected 401, attempting silent refresh via secure cookies...');

        // 3. Attempt to use the Refresh Token mechanism via our AuthService (which uses HttpOnly Cookies).
        return authService.refreshToken().pipe(
          switchMap(() => {
            console.log('AuthInterceptor: Silent refresh successful! Retrying original request with new access cookie.');
            // 4. Retry the original failed request, which will now include the updated session cookies.
            return next(authReq);
          }),
          catchError((refreshErr) => {
            console.error('AuthInterceptor: Refresh failed or session expired. User must re-authenticate.', refreshErr);
            // If refresh fails too, we are truly unauthorized (session has ended). 
            // The app state will be cleared by the AuthService catch block in refreshToken().
            return throwError(() => refreshErr);
          })
        );
      }

      // For any other error code (403 Forbidden, 500 Server Error, etc.), just pass it through.
      return throwError(() => error);
    })
  );
};