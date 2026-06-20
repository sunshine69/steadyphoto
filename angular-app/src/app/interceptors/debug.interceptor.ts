import { Injectable, inject } from '@angular/core';
import { HttpInterceptorFn, HttpRequest, HttpHandlerFn, HttpEvent, HttpResponse, HttpErrorResponse } from '@angular/common/http';
import { Observable, catchError, throwError, tap, map } from 'rxjs';

/**
 * DEBUGGING INTERCEPTOR - Logs ALL HTTP requests and responses to the console.
 * This helps track what's being sent/received for shared media endpoints.
 */
export const debugInterceptor: HttpInterceptorFn = (req: HttpRequest<unknown>, next: HttpHandlerFn): Observable<HttpEvent<unknown>> => {

  const startTimestamp = Date.now();
  let isLoggable = false;

  // Only log requests to API or shared media endpoints
  if (req.url.includes('api') || req.url.includes('media/shared')) {
    isLoggable = true;
    console.group(`🔍 DEBUG: [${req.method.toUpperCase()}] ${req.url}`);
    console.log('⏰ Time:', new Date().toISOString(), '(ms since start):', startTimestamp - performance.now());

    // Log request headers (without credentials/sensitive data)
    req.headers.keys().forEach(key => {
      const value = req.headers.get(key);
      let displayValue = value;
      if (key.toLowerCase() === 'authorization') {
        displayValue = `${value?.substring(0, 15)}...`;
      }
      console.log(`   ${key}: ${displayValue}`);
    });

    // Log request body if present
    if (req.body) {
      let bodyPreview: string;
      try {
        const parsed = typeof req.body === 'string' ? JSON.parse(req.body as string) : req.body;
        console.log('📨 Request Body:', parsed);
      } catch (e) {
        // Body might not be parseable as JSON, just log type and size
        console.log('📨 Request Body Type:', typeof req.body === 'string' ? 'String' : 'Blob/ArrayBuffer');
        if (typeof req.body === 'string') {
          console.log('📨 Request Body Preview:', req.body.substring(0, 200));
        } else if (req.body instanceof Blob) {
          const blob = req.body as Blob;
          console.log(`📨 Request Body: ${blob.size} bytes (${blob.type})`);
        } else if (req.body instanceof ArrayBuffer) {
          console.log(`📨 Request Body: ${(req.body as ArrayBuffer).byteLength} bytes`);
        }
      }
    }

    // Log credentials info
    console.log('withCredentials:', req.withCredentials);
    console.log('responseType:', req.responseType);
  }

  return next(req).pipe(
    tap(event => {
      if (isLoggable && event instanceof HttpResponse) {
        const duration = Date.now() - startTimestamp;
        console.group(`✅ RESPONSE [${event.status}] (${duration}ms): ${req.url}`);
        console.log('📋 Status:', event.status, event.statusText);

        // Log response headers (partial - mask CORS/wildcard)
        event.headers.keys().forEach(key => {
          const value = event.headers.get(key);
          let displayValue = value;
          if (key.toLowerCase() === 'access-control-allow-origin') {
            displayValue = `${value?.substring(0, 25)}...`;
          }
          console.log(`   ${key}: ${displayValue}`);
        });

        // Log response body
        let responseBody: string;
        try {
          responseBody = JSON.stringify(event.body);
          if (responseBody.length > 1000) {
            responseBody = responseBody.substring(0, 1000) + '...';
          }
          console.log('📥 Response Body:', responseBody);
        } catch (e) {
          // Response body might not be JSON-parseable
          if (typeof event.body === 'string') {
            console.log('📥 Response Body:', event.body.substring(0, 200));
          } else if (event.body instanceof Blob) {
            const blob = event.body as Blob;
            console.log(`📥 Response: ${blob.size} bytes (${blob.type})`);
          } else if (typeof event.body === 'object') {
            console.log('📥 Response Body:', JSON.stringify(event.body).substring(0, 500));
          }
        }

        console.groupEnd();
      }
    }),
    catchError(error => {
      const duration = Date.now() - startTimestamp;
      if (isLoggable) {
        console.group(`❌ ERROR [${(error as HttpErrorResponse).status} ${((error as HttpErrorResponse).statusText || 'Error')}] (${duration}ms): ${req.url}`);

        let errorMsg: string = '';
        const httpErr = error as HttpErrorResponse;

        // Try to parse error body if it's JSON
        if (httpErr.error) {
          try {
            const parsedBody = typeof httpErr.error === 'string' ? {} :
              (typeof httpErr.error === 'object') ? JSON.stringify(httpErr.error).substring(0, 500) : String(httpErr.error);
            errorMsg = `Response: ${parsedBody}`;
          } catch (e) {
            errorMsg = `Raw error response`;
          }
        }

        console.log('📋 Status:', httpErr.status, httpErr.statusText || '');
        console.log('📨 Request URL:', req.url);
        console.log('📨 Request Method:', req.method.toUpperCase());

        // For 401 errors - this is the critical case for shared media!
        if (httpErr.status === 401) {
          console.error('⚠️ AUTHORIZATION ERROR: Shared item access denied');
          console.error('   This could be caused by:');
          console.error('   1. Missing or expired session cookie');
          console.error('   2. Different auth mechanism for shared items vs owner items');
          console.error('   3. Shared item not found in the share table');

          // Check if there's a different response on retry (after interceptor refresh)
          if (req.headers.has('X-Auth-Retry')) {
            console.error('   This is a RE-TRY request that also failed! The auth interceptor\'s refresh did NOT fix it.');
          }
        }

        // Check for 403
        if (httpErr.status === 403) {
          console.error('⚠️ FORBIDDEN: Access denied to shared item');
        }

        if (errorMsg) {
          console.error(errorMsg);
        }

        console.groupEnd();
      }
      return throwError(() => error);
    })
  );
};
