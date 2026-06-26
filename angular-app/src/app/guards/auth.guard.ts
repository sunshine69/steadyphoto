import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { AuthService } from '../services/auth.service';

/**
 * AuthGuard protects routes by checking if the user is authenticated via AuthService.
 */

// === DEBUG: Track route navigation attempts ===
const ROUTE_DEBUG_MAP = new Map<string, string>([
  ['photos', 'photo-detail'],
  ['albums', 'album-detail'],
]);

export const authGuard: CanActivateFn = (route, state) => {
  const authService = inject(AuthService);
  const router = inject(Router);

  // === DEBUG: Log which route is being guarded ===
  const requestedPath = state.url;
  console.group('🔒 AuthGuard - Route Guard Check');
  console.log('📍 Requested path:', requestedPath);
  console.log('🔑 Authenticated?', authService.isAuthenticated());

  // Check if this is a shared resource access attempt via query param
  const isSharedResource = route.queryParamMap.get('source') === 'shared';
  
  // Detect if this is a shared resource access attempt (legacy param detection)
  const isSharedPhoto = requestedPath.includes('/photos/') && route.paramMap.has('id');
  const isSharedAlbum = requestedPath.includes('/albums/') && route.paramMap.has('id');
  
  // Check if this is a presentation mode route with shareToken in the query params
  const isPresentationMode = requestedPath.includes('/presentation') && route.queryParamMap.has('shareToken');
  
  if (isSharedResource) {
    console.log('🔗 Shared resource access detected via query param, allowing access');
    console.groupEnd();
    return true;
  }
  
  if (isPresentationMode) {
    console.log('🖥️ Presentation mode with shareToken detected, allowing access');
    console.groupEnd();
    return true;
  }

  if (isSharedPhoto) {
    console.log('📷 Accessing shared photo, param ID:', route.paramMap.get('id'));
  } else if (isSharedAlbum) {
    console.log('📁 Accessing shared album, param ID:', route.paramMap.get('id'));
  }

  console.groupEnd();

  if (authService.isAuthenticated()) {
    console.log('✅ AuthGuard: User authenticated, allowing access');
    return true;
  } else {
    // === DEBUG: Log detailed redirect reason ===
    console.group('🚨 AuthGuard: Redirecting to login');
    console.log('Reason: User not authenticated');
    console.log('Requested path:', requestedPath);
    
    // Check if this is a shared resource - might need different handling
    if (isSharedPhoto || isSharedAlbum) {
      console.warn('⚠️ This appears to be a shared resource access attempt!');
      console.warn('The user may have been redirected because their session expired.');
      console.warn('They should see a login dialog, not be kicked out of the app entirely.');
    }
    
    console.groupEnd();

    return router.parseUrl('/login');
  }
};
