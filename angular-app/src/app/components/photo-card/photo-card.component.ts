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
        <!-- Show thumbnail for both photos and videos -->
        <div *ngIf="photo.thumbnailUrl; else fallback" class="thumb-container">
          <img 
            [src]="photo.thumbnailUrl" 
            [alt]="photo.filename"
            class="photo-thumb"
            loading="lazy"
            crossorigin="use-credentials"
          >
          <!-- Video icon overlay for videos -->
          <div *ngIf="isVideo()" class="video-overlay">
            <span class="play-icon">▶</span>
          </div>
        </div>
        <!-- Fallback when no thumbnail available -->
        <ng-template #fallback>
          <div [class]="getFallbackClass()">
            <span *ngIf="isVideo()" class="video-icon">▶</span>
            <span *ngIf="!isVideo()" class="photo-icon">📷</span>
            <span class="filename-overlay">{{ photo.filename }}</span>
          </div>
        </ng-template>
      </div >
      <div class="photo-info">
        <p class="photo-filename" [title]="photo.filename">{{ photo.filename }}</p>
        <p class="photo-date">{{ isVideo() ? formatDuration(photo.videoMetadata?.duration) : (photo.captured_at | date:'shortDate') }}</p>
      </div >
    </div >
  `,
  styles: [`
    :host {
      display: block;
      width: 100%;
      max-width: 100%;
      min-width: 0; /* Prevents the component from pushing the grid cell width */
    }
    .photo-card {
      display: flex;
      flex-direction: column;
      width: 100%;
      height: 100%;
      background: #fff;
      border-radius: 8px;
      overflow: hidden;
      box-shadow: 0 2px 4px rgba(0,0,0,0.1);
      cursor: pointer;
      transition: transform 0.2s, box-shadow 0.2s;
      box-sizing: border-box;
    }
    .photo-card:hover {
      transform: translateY(-4px);
      box-shadow: 0 4px 8px rgba(0,0,0,0.15);
    }
    .photo-wrapper {
      position: relative;
      width: 100%;
      aspect-ratio: 1 / 1; /* Modern way to force square aspect ratio */
      overflow: hidden;
      background-color: #f0f0f0;
      flex-shrink: 0;
    }
    /* Fallback for older browsers that don't support aspect-ratio */
    @supports not (aspect-ratio: 1 / 1) {
      .photo-wrapper {
        padding-top: 100%;
      }
      .photo-thumb {
        position: absolute;
        top: 0;
        left: 0;
      }
    }
    .photo-thumb {
      width: 100%;
      height: 100%;
      object-fit: cover; /* Ensures the image covers the square area without distortion */
      display: block;
    }

    .thumb-container {
      width: 100%;
      height: 100%;
      position: relative;
    }

    /* Video overlay icon on top of thumbnail */
    .video-overlay {
      position: absolute;
      top: 50%;
      left: 50%;
      transform: translate(-50%, -50%);
      width: 48px;
      height: 48px;
      background-color: rgba(0, 0, 0, 0.7);
      border-radius: 50%;
      display: flex;
      align-items: center;
      justify-content: center;
      transition: transform 0.2s ease-in-out;
    }

    .photo-card:hover .video-overlay {
      transform: translate(-50%, -50%) scale(1.1);
      background-color: rgba(0, 0, 0, 0.85);
    }

    .play-icon {
      color: white;
      font-size: 20px;
      margin-left: 3px; /* Slight offset to visually center the triangle */
    }

    .fallback {
      width: 100%;
      height: 100%;
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      color: white;
      text-align: center;
      padding: 0.5rem;
      box-sizing: border-box;
    }

    .fallback.video {
      background-color: #2c3e50;
    }

    .fallback.photo {
      background-color: #7f8c8d;
    }

    .video-icon, .photo-icon {
      font-size: 2rem;
      margin-bottom: 0.5rem;
    }

    .filename-overlay {
      font-size: 0.75rem;
      word-break: break-all;
      line-height: 1.2;
    }
    .photo-info {
      padding: 0.75rem;
      flex-grow: 1;
      display: flex;
      flex-direction: column;
      justify-content: space-between;
      background: white;
      min-height: 60px;
      box-sizing: border-box;
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

  isVideo(): boolean {
    return this.photo.mediaType === 'video';
  }

  getFallbackClass(): string {
    if (this.isVideo()) {
      return 'fallback video';
    }
    return 'fallback photo';
  }

  formatDuration(seconds?: number): string {
    if (!seconds || seconds <= 0) return '';
    const hrs = Math.floor(seconds / 3600);
    const mins = Math.floor((seconds % 3600) / 60);
    const secs = Math.floor(seconds % 60);
    
    if (hrs > 0) {
      return `${hrs}:${mins.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`;
    }
    return `${mins}:${secs.toString().padStart(2, '0')}`;
  }
}
