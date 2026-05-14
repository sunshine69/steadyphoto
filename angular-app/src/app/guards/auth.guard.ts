import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { AuthService } from '../services/auth.service';

/**
 * AuthGuard protects routes by checking if the user is authenticated via AuthService.
 */
export const authGuard: CanActivateFn = (route, state) => {
  const authService = inject(AuthService);
  const router = inject(Router);

  if (authService.isAuthenticated()) {
    return true;
  } else {
    console.warn('AuthGuard: User not authenticated, redirecting to /login');
    return router.parseUrl('/login');
  }
};
