import { bootstrapApplication } from '@angular/platform-browser';
import { provideZoneChangeDetection, importProvidersFrom } from '@angular/core';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { provideHttpClient, withInterceptors, withXhr } from '@angular/common/http';
import { provideRouter } from '@angular/router'; // Modern router provider
import { AppComponent } from './app/app.component';
import { routes } from './app/app.routes';
import { authInterceptor } from './app/interceptors/auth.interceptor';

bootstrapApplication(AppComponent, {
  providers: [
    // 1. Explicitly restore Zone.js behavior to fix the async image-loading issue
    provideZoneChangeDetection({ eventCoalescing: true }),
    
    // 2. Modern routing registration replacing importProvidersFrom(RouterModule)
    provideRouter(routes), 
    
    // 3. Kept for legacy animation modules
    importProvidersFrom(BrowserAnimationsModule),
    
    provideHttpClient(withXhr(), withInterceptors([authInterceptor]))
  ]
}).catch(err => console.error(err));

