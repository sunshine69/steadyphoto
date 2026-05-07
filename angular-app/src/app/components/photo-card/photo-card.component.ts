import { Component, Input, Output, EventEmitter, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Photo } from '../../models/photo.model';

@Component({
  selector: 'app-photo-card',
  standalone: true,
  imports: [CommonModule],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="photo-card">
      <div class="photo-wrapper" (click)="onCardClick()">
        <img 
          [src]="photo.thumbnailUrl || photo.path" 
          [alt]="photo.filename"
          class="photo-thumb"
          loading="lazy"
        >
      </div>
      <div class="photo-info">
        <p class="photo-filename">{{ photo.filename }}</p>
        <p class="photo-date">{{ photo.captured_at | date:'medium' }}</p>
      </div>
    </div>
  `
})
export class PhotoCardComponent {
  @Input() photo!: Photo;
  @Output() cardClick = new EventEmitter<string>();

  onCardClick() {
    this.cardClick.emit(this.photo.id);
  }
}
