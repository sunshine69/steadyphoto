import { Routes } from '@angular/router';
import { PhotoListComponent } from './components/photo-list/photo-list.component';
import { PhotoDetailComponent } from './components/photo-detail/photo-detail.component';
import { AlbumListComponent } from './components/album-list/album-list.component';
import { AlbumDetailComponent } from './components/album-detail/album-detail.component';
import { LoginComponent } from './components/login/login.component';
import { RegisterComponent } from './components/register/register.component';
import { PresentationComponent } from './components/presentation/presentation.component';
import { ExploreComponent } from './components/explore/explore.component';
import { MapComponent } from './components/map/map.component';
import { SharingComponent } from './components/sharing/sharing.component';
import { FavoritesComponent } from './components/favorites/favorites.component';
import { CameraComponent } from './components/camera/camera.component';
import { ScreenshotsComponent } from './components/screenshots/screenshots.component';
import { MessengerComponent } from './components/messenger/messenger.component';
import { UtilitiesComponent } from './components/utilities/utilities.component';
import { ArchiveComponent } from './components/archive/archive.component';
import { LockedComponent } from './components/locked/locked.component';
import { TrashComponent } from './components/trash/trash.component';
import { authGuard } from './guards/auth.guard';

export const routes: Routes = [
  // Main App Routes (Protected)
  { path: '', component: PhotoListComponent, canActivate: [authGuard] },
  { path: 'photos/:id', component: PhotoDetailComponent, canActivate: [authGuard] },
  
  // New Immich-style Views
  { path: 'explore', component: ExploreComponent, canActivate: [authGuard] },
  { path: 'map', component: MapComponent, canActivate: [authGuard] },
  { path: 'sharing', component: SharingComponent, canActivate: [authGuard] },
  
  // Library Routes
  { path: 'favorites', component: FavoritesComponent, canActivate: [authGuard] },
  { path: 'albums', component: AlbumListComponent, canActivate: [authGuard] },
  { path: 'albums/:id', component: AlbumDetailComponent, canActivate: [authGuard] },
  
  // Library Sub-sections
  { path: 'camera', component: CameraComponent, canActivate: [authGuard] },
  { path: 'screenshots', component: ScreenshotsComponent, canActivate: [authGuard] },
  { path: 'messenger', component: MessengerComponent, canActivate: [authGuard] },
  
  // Utilities & System Routes
  { path: 'utilities', component: UtilitiesComponent, canActivate: [authGuard] },
  { path: 'archive', component: ArchiveComponent, canActivate: [authGuard] },
  { path: 'locked', component: LockedComponent, canActivate: [authGuard] },
  { path: 'trash', component: TrashComponent, canActivate: [authGuard] },
  
  // Special Routes
  { path: 'presentation', component: PresentationComponent, canActivate: [authGuard] },
  
  // Auth Routes (Public)
  { path: 'login', component: LoginComponent },
  { path: 'register', component: RegisterComponent },
  
  // Redirect unknown routes to Photos
  { path: '**', redirectTo: '' }
];
