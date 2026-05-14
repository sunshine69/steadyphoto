import { Routes } from '@angular/router';
import { PhotoListComponent } from './components/photo-list/photo-list.component';
import { PhotoDetailComponent } from './components/photo-detail/photo-detail.component';
import { authGuard } from './guards/auth.guard';

export const routes: Routes = [
  { path: '', component: PhotoListComponent, canActivate: [authGuard] },
  { path: 'photos/:id', component: PhotoDetailComponent, canActivate: [authGuard] },
  // Login and Register are public (no guard)
  // { path: 'login', component: LoginComponent }, // To be implemented in Phase 7
  // { path: 'register', component: RegisterComponent }, // To be implemented in Phase 7
  { path: '**', redirectTo: '' }
];
