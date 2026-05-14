import { Routes } from '@angular/router';
import { PhotoListComponent } from './components/photo-list/photo-list.component';
import { PhotoDetailComponent } from './components/photo-detail/photo-detail.component';
import { LoginComponent } from './components/login/login.component';
import { RegisterComponent } from './components/register/register.component';
import { authGuard } from './guards/auth.guard';

export const routes: Routes = [
  { path: '', component: PhotoListComponent, canActivate: [authGuard] },
  { path: 'photos/:id', component: PhotoDetailComponent, canActivate: [authGuard] },
  { path: 'login', component: LoginComponent },
  { path: 'register', component: RegisterComponent },
  { path: '**', redirectTo: '' }
];
