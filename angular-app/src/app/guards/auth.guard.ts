import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { AuthService } from '../services/auth.service';

/**
 * AuthGuard protects routes by checking if the user is authenticated via AuthService.
 */
export const authGuard: CanActivateFn = (route, state) => {
  const authService = inject(AuthService);
  const router = inject(Router);

  // Check if this is a shared resource access attempt via query param
  const isSharedResource = route.queryParamMap.get('source') === 'shared';
  
  // Detect if this is a shared resource access attempt (legacy param detection)
  const isSharedPhoto = state.url.includes('/photos/') && route.paramMap.has('id');
  const isSharedAlbum = state.url.includes('/albums/') && route.paramMap.has('id');
  
  // Check if this is a presentation mode route with shareToken in the query params
  const isPresentationMode = state.url.includes('/presentation') && route.queryParamMap.has('shareToken');
  
  if (isSharedResource || isPresentationMode || isSharedPhoto || isSharedAlbum) {
    return true;
  }

  if (authService.isAuthenticated()) {
    return true;
  } else {
    return router.parseUrl('/login');
  }
};
