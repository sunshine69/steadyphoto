import { Injectable, inject } from '@angular/core';
import { HttpInterceptorFn, HttpRequest, HttpHandlerFn, HttpEvent, HttpErrorResponse } from '@angular/common/http';
import { Observable, throwError, catchError, switchMap } from 'rxjs';
import { AuthService } from '../services/auth.service';

/**
 * AuthInterceptor automatically:
 * 1. Adds `withCredentials: true` to every outgoing request so HttpOnly cookies are sent.
 * 2. Listens for 401 Unauthorized errors, attempts a silent token refresh via /api/auth/refresh.
 *    If successful, it retries the original failed request once.
 */
export const authInterceptor: HttpInterceptorFn = (req: HttpRequest<unknown>, next: HttpHandlerFn): Observable<HttpEvent<unknown>> => {
  const authService = inject(AuthService);

  // 1. Ensure every single request includes credentials so cookies are sent
  let authReq = req;
  if (!req.withCredentials) {
    authReq = req.clone({ withCredentials: true });
  }

  return next(authReq).pipe(
    catchError((error: HttpErrorResponse) => {
      // 2. Detect Unauthorized (401) error
      if (error.status === 401) {
        console.warn('AuthInterceptor: Detected 401, attempting silent refresh...');

        // 3. Attempt to use the Refresh Token via our AuthService
        return authService.refreshToken().pipe(
          switchMap(() => {
            console.log('AuthInterceptor: Refresh successful! Retrying original request.');
            // 4. Retry the original failed request (which is now authorized with new access cookie)
            return next(authReq);
          }),
          catchError((refreshErr) => {
            console.error('AuthInterceptor: Refresh failed, user must log in again.', refreshErr);
            // If refresh fails too, we are truly unauthorized (session expired/revoked)
            // In a real app, you'd redirect to /login here using the Router
            return throwError(() => refreshErr);
          })
        );
      }

      // For any other error code (403, 500, etc.), just pass it through
      return throwError(() => error);
    })
  );
};
