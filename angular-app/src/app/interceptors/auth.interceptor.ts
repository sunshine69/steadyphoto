import { Injectable, inject } from '@angular/core';
import { HttpInterceptorFn, HttpRequest, HttpHandlerFn, HttpEvent, HttpErrorResponse } from '@angular/common/http';
import { Observable, throwError, catchError, switchMap } from 'rxjs';
import { AuthService } from '../services/auth.service';

// Debug prefix for console logs - filter by this to see all debug output
const DEBUG_PREFIX = '[Auth Interceptor Debug]';

/**
 * AuthInterceptor automatically:
 * 1. Adds `withCredentials: true` to every outgoing request so HttpOnly cookies are sent.
 * 2. Injects the access_token as a Bearer header for API calls that require it (e.g., Android scanner).
 *    The token is read from sessionStorage (cleared on tab close) — NOT localStorage, to reduce XSS risk.
 * 3. Listens for 401 Unauthorized errors, attempts a silent token refresh via /api/auth/refresh.
 * 
 * CRITICAL: During logout or when a refresh is already in-flight, no refresh attempt will be made.
 * This prevents the "refresh avalanche" that causes rate limiter (429) issues during logout.
 */
export const authInterceptor: HttpInterceptorFn = (req: HttpRequest<unknown>, next: HttpHandlerFn): Observable<HttpEvent<unknown>> => {
  const authService = inject(AuthService);

  // Build the initial request with Authorization header from sessionStorage and credentials
  let authReq = req;
  
  // SECURITY FIX: Read access_token from sessionStorage instead of localStorage to reduce XSS risk.
  // The Bearer token is a fallback for API clients that can't use HttpOnly cookies (e.g., Android scanner).
  // For normal browser requests, the cookie is sent automatically via withCredentials.
  const token = sessionStorage.getItem('access_token');
  if (token) {
    authReq = req.clone({
      setHeaders: {
        Authorization: `Bearer ${token}`
      }
    });
  }

  // Also ensure credentials are sent for HttpOnly cookies/refresh logic
  if (!authReq.withCredentials) {
    authReq = authReq.clone({ withCredentials: true });
  }

  console.log(`${DEBUG_PREFIX} Request: ${req.method} ${req.url}`);
  if (token) {
    console.log(`${DEBUG_PREFIX} Access token present (first 10 chars): ${token.substring(0, 10)}...`);
  } else {
    console.log(`${DEBUG_PREFIX} WARNING: No access token in sessionStorage for request: ${req.url}`);
  }

  return next(authReq).pipe(
    catchError((error: HttpErrorResponse) => {
      // Only attempt refresh for actual 401 errors — NOT for 403, 500, etc.
      // Public share endpoints return 403 (not 401) when password is required/incorrect.
      // If we try to refresh on a 403, the refresh fails and swallows the original error data.
      if (!authReq.headers.has('X-Auth-Retry') && error.status === 401) {
        console.log(`${DEBUG_PREFIX} 401 Unauthorized detected for: ${req.method} ${req.url}`);
        
        // LAYER 1: Check if we're in the middle of logout — skip ALL refresh attempts during logout.
        // This prevents the "refresh avalanche" where concurrent requests fail with 401 and each tries to refresh,
        // hitting the rate limiter (429) and hanging the browser.
        if (authService.isLoggingOut()) {
          console.warn(`${DEBUG_PREFIX} Skipping refresh — logout in progress`);
          return throwError(() => error);
        }

 

        // LAYER 2: Use the single-refresh mechanism from AuthService to prevent concurrent refreshes.
        // This ensures only ONE refresh happens even if multiple requests fail simultaneously.
        return authService.handle401().pipe(
          switchMap(() => {
            console.log(`${DEBUG_PREFIX} Refresh completed, retrying original request: ${req.method} ${req.url}`);
            
            let retryReq = authReq;
            const newToken = sessionStorage.getItem('access_token');
            if (newToken) {
              // Clone the request with the updated Bearer token for the retry
              retryReq = authReq.clone({
                setHeaders: {
                  Authorization: `Bearer ${newToken}`
                }
              });
            }

            // Ensure withCredentials is set on the retry so HttpOnly cookie is sent
            if (!retryReq.withCredentials) {
              retryReq = retryReq.clone({ withCredentials: true });
            }

            // 3. Retry the original failed request — now authorized via refreshed HttpOnly cookie (and optionally new Bearer header)
            return next(retryReq);
          }),
          catchError((refreshErr) => {
            console.error(`${DEBUG_PREFIX} Refresh failed, user must log in again.`, refreshErr);
            console.error(`${DEBUG_PREFIX} Error status: ${refreshErr.status}, message: ${refreshErr.message}`);
            // If refresh fails too, we are truly unauthorized (session expired/revoked)
            // In a real app, you'd redirect to /login here using the Router
            return throwError(() => refreshErr);
          })
        );
      }

      // For any other error code (403, 500, etc.), just pass it through unchanged
      if (error.status !== 0) { // Not a network error (status 0 means offline/abort)
        console.log(`${DEBUG_PREFIX} Non-401 error: ${req.method} ${req.url} - Status: ${error.status}`);
      }

      return throwError(() => error);
    })
  );
};
