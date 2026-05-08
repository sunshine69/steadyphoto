import { Component, Input, Output, EventEmitter, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Photo } from '../../models/photo.model';

@Component({
  selector: 'app-photo-card',
  standalone: true,
  imports: [CommonModule],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="photo-card" (click)="onCardClick()">
      <div class="photo-wrapper">
        <img 
          [src]="photo.thumbnailUrl || photo.path" 
          [alt]="photo.filename"
          class="photo-thumb"
          loading="lazy"
        >
      </div>
      <div class="photo-info">
        <p class="photo-filename" [title]="photo.filename">{{ photo.filename }}</p>
        <p class="photo-date">{{ photo.captured_at | date:'shortDate' }}</p>
      </div>
    </div>
  `,
  styles: [`
    :host {
      display: block;
      height: 100%;
    }
    .photo-card {
      display: flex;
      flex-direction: column;
      height: 100%;
      background: #fff;
      border-radius: 8px;
      overflow: hidden;
      box-shadow: 0 2px 4px rgba(0,0,0,0.1);
      cursor: pointer;
      transition: transform 0.2s, box-shadow 0.2s;
    }
    .photo-card:hover {
      transform: translateY(-4px);
      box-shadow: 0 4px 8px rgba(0,0,0,0.15);
    }
    .photo-wrapper {
      position: relative;
      width: 100%;
      padding-top: 100%; /* 1:1 Aspect Ratio */
      overflow: hidden;
      background-color: #f0f0f0;
    }
    .photo-thumb {
      position: absolute;
      top: 0;
      left: 0;
      width: 100%;
      height: 100%;
      object-fit: cover; /* This ensures images fill the area without distortion */
    }
    .photo-info {
      padding: 0.75rem;
      flex-grow: 1;
      display: flex;
      flex-direction: column;
      justify-content: space-between;
      background: white;
    }
    .photo-filename {
      margin: 0;
      font-size: 0.9rem;
      font-weight: 500;
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
    }
    .photo-date {
      margin: 0.25rem 0 0 0;
      font-size: 0.8rem;
      color: #6c757d;
    }
  `]
})
export class PhotoCardComponent {
  @Input() photo!: Photo;
  @Output() cardClick = new EventEmitter<string>();

  onCardClick() {
    this.cardClick.emit(this.photo.id);
  }
}
