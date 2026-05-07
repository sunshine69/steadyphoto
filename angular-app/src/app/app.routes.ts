import { Routes } from '@angular/router';
import { PhotoListComponent } from './components/photo-list/photo-list.component';
import { PhotoDetailComponent } from './components/photo-detail/photo-detail.component';

export const routes: Routes = [
  { path: '', component: PhotoListComponent },
  { path: 'photos/:id', component: PhotoDetailComponent },
  { path: '**', redirectTo: '' }
];
