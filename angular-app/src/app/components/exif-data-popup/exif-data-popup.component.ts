import { Component, Input, Output, EventEmitter, ChangeDetectionStrategy } from '@angular/core';
import { CommonModule } from '@angular/common';

interface ExifData {
  make?: string;
  model?: string;
  lens_model?: string;
  exposure_time?: string | number;
  f_number?: number;
  focal_length?: number;
  iso_speed_ratings?: number;
  timestamp?: string;
  dimensions?: string;
  DateTimeOriginal?: string;
  [key: string]: any;
}

@Component({
    selector: 'app-exif-data-popup',
    imports: [CommonModule],
    template: `
    <div class="exif-popup-overlay" (click)="onOverlayClick($event)">
      <div class="exif-popup-panel" (click)="$event.stopPropagation()">
        <div class="exif-popup-header">
          <h3 class="exif-popup-title">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="exif-icon">
              <circle cx="12" cy="12" r="3"/>
              <path d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42"/>
            </svg>
            EXIF Data
          </h3>
          <button class="exif-close-btn" (click)="close.emit()" title="Close">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="18" y1="6" x2="6" y2="18"/>
              <line x1="6" y1="6" x2="18" y2="18"/>
            </svg>
          </button>
        </div>
        
        <div class="exif-popup-content">
          <div *ngIf="!hasExifData" class="exif-empty-state">
            <p class="text-muted">No EXIF data available for this photo.</p>
          </div>
          
          <div *ngIf="hasExifData" class="exif-data-list">
            <div class="exif-data-item" *ngFor="let item of formattedExifData">
              <span class="exif-data-label">{{ item.label }}</span>
              <span class="exif-data-value">{{ item.value }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  `,
    styles: [`
    .exif-popup-overlay {
      position: fixed;
      top: 0;
      left: 0;
      width: 100%;
      height: 100%;
      background: rgba(0, 0, 0, 0.6);
      backdrop-filter: blur(4px);
      display: flex;
      justify-content: flex-end;
      z-index: 1000;
      animation: fadeIn 0.2s ease-out;
    }

    .exif-popup-panel {
      width: 100%;
      max-width: 400px;
      height: 100%;
      background: linear-gradient(180deg, #1e293b 0%, #0f172a 100%);
      border-left: 1px solid #334155;
      box-shadow: -4px 0 20px rgba(0, 0, 0, 0.5);
      display: flex;
      flex-direction: column;
      animation: slideInRight 0.3s ease-out;
    }

    .exif-popup-header {
      display: flex;
      align-items: center;
      justify-content: space-between;
      padding: 20px 24px;
      border-bottom: 1px solid #334155;
      background: rgba(30, 41, 59, 0.95);
    }

    .exif-popup-title {
      display: flex;
      align-items: center;
      gap: 10px;
      margin: 0;
      font-size: 18px;
      font-weight: 600;
      color: #f1f5f9;
      letter-spacing: 0.5px;
    }

    .exif-icon {
      color: #60a5fa;
    }

    .exif-close-btn {
      width: 32px;
      height: 32px;
      border-radius: 8px;
      background: transparent;
      border: 1px solid #475569;
      color: #94a3b8;
      cursor: pointer;
      display: flex;
      align-items: center;
      justify-content: center;
      transition: all 0.2s ease;
    }

    .exif-close-btn:hover {
      background: rgba(239, 68, 68, 0.1);
      border-color: #ef4444;
      color: #ef4444;
    }

    .exif-popup-content {
      flex: 1;
      overflow-y: auto;
      padding: 24px;
    }

    .exif-empty-state {
      text-align: center;
      padding: 40px 20px;
    }

    .exif-data-list {
      display: flex;
      flex-direction: column;
      gap: 16px;
    }

    .exif-data-item {
      display: flex;
      flex-direction: column;
      gap: 6px;
      padding: 12px 16px;
      background: rgba(51, 65, 85, 0.4);
      border-radius: 10px;
      border: 1px solid #334155;
      transition: all 0.2s ease;
    }

    .exif-data-item:hover {
      background: rgba(51, 65, 85, 0.6);
      border-color: #475569;
    }

    .exif-data-label {
      font-size: 11px;
      font-weight: 600;
      color: #94a3b8;
      text-transform: uppercase;
      letter-spacing: 0.8px;
    }

    .exif-data-value {
      font-size: 15px;
      font-weight: 500;
      color: #e2e8f0;
      line-height: 1.4;
    }

    @keyframes fadeIn {
      from {
        opacity: 0;
      }
      to {
        opacity: 1;
      }
    }

    @keyframes slideInRight {
      from {
        transform: translateX(100%);
      }
      to {
        transform: translateX(0);
      }
    }

    /* Scrollbar styling */
    .exif-popup-content::-webkit-scrollbar {
      width: 8px;
    }

    .exif-popup-content::-webkit-scrollbar-track {
      background: rgba(15, 23, 42, 0.5);
    }

    .exif-popup-content::-webkit-scrollbar-thumb {
      background: #475569;
      border-radius: 4px;
    }

    .exif-popup-content::-webkit-scrollbar-thumb:hover {
      background: #64748b;
    }
  `]
})
export class ExifDataPopupComponent {
  @Input() exifData: ExifData | null = null;
  @Output() close = new EventEmitter<void>();


  get hasExifData(): boolean {
    if (!this.exifData) return false;
    return Object.keys(this.exifData).length > 0;
  }
  get formattedExifData(): Array<{ label: string; value: string }> {
    if (!this.exifData) return [];

    const formatted: Array<{ label: string; value: string }> = [];

    // Camera Make & Model
    if (this.exifData.make || this.exifData.model) {
      const cameraName = [this.exifData.make, this.exifData.model]
        .filter(Boolean)
        .join(' ');
      formatted.push({
        label: 'Camera',
        value: cameraName
      });
    }

    // Lens
    if (this.exifData.lens_model) {
      formatted.push({
        label: 'Lens',
        value: this.exifData.lens_model
      });
    }

    // Exposure Time
    if (this.exifData.exposure_time) {
      formatted.push({
        label: 'Exposure Time',
        value: this.formatExposureTime(this.exifData.exposure_time)
      });
    }

    // Aperture
    if (this.exifData.f_number) {
      formatted.push({
        label: 'Aperture',
        value: `f/${this.exifData.f_number}`
      });
    }

    // Focal Length
    if (this.exifData.focal_length) {
      formatted.push({
        label: 'Focal Length',
        value: `${this.exifData.focal_length}mm`
      });
    }

    // ISO
    if (this.exifData.iso_speed_ratings) {
      formatted.push({
        label: 'ISO',
        value: this.exifData.iso_speed_ratings.toString()
      });
    }

    // DateTimeOriginal (EXIF capture time) - check both PascalCase and lowercase
    const dtOriginal = this.exifData.DateTimeOriginal || this.exifData['datetime_original'];
    if (dtOriginal) {
      // Convert EXIF date format "2026:04:25 11:13:43" to proper Date
      const exifDateStr = dtOriginal.replace(/(\d{4}):(\d{2}):(\d{2})\s+(\d{2}):(\d{2}):(\d{2})/, '$1-$2-$3T$4:$5:$6');
      const date = new Date(exifDateStr);
      if (!isNaN(date.getTime())) {
        formatted.push({
          label: 'Date Taken',
          value: date.toLocaleString()
        });
      }
    }
    
    // Fallback: timestamp field
    if (this.exifData.timestamp && !dtOriginal) {
      const date = new Date(this.exifData.timestamp);
      if (!isNaN(date.getTime())) {
        formatted.push({
          label: 'Date Taken',
          value: date.toLocaleString()
        });
      }
    }

    // Dimensions
    if (this.exifData.dimensions) {
      formatted.push({
        label: 'Dimensions',
        value: this.exifData.dimensions
      });
    }

    // Add any additional custom fields
    const excludedKeys = [
      'make', 'model', 'lens_model', 'exposure_time', 'f_number',
      'focal_length', 'iso_speed_ratings', 'timestamp', 'dimensions'
    ];

    for (const [key, value] of Object.entries(this.exifData)) {
      if (!excludedKeys.includes(key) && value !== undefined && value !== null) {
        formatted.push({
          label: this.formatLabel(key),
          value: String(value)
        });
      }
    }

    return formatted;
  }

  private formatExposureTime(exposureTime: string | number): string {
    if (!exposureTime) return 'N/A';
    
    // Handle string format like "1/250"
    if (typeof exposureTime === 'string' && exposureTime.includes('/')) {
      return exposureTime;
    }
    
    // Handle numeric value (fraction of a second)
    const value = typeof exposureTime === 'string' ? parseFloat(exposureTime) : exposureTime;
    if (isNaN(value) || value <= 0) return 'N/A';
    
    // Convert to fraction if it's a decimal
    if (value < 1) {
      const denominator = Math.round(1 / value);
      return `1/${denominator}`;
    }
    
    return `${value}s`;
  }

  private formatLabel(key: string): string {
    // Convert camelCase or snake_case to Title Case with spaces
    return key
      .replace(/([A-Z])/g, ' $1')
      .replace(/_/g, ' ')
      .replace(/^\w/, (c) => c.toUpperCase())
      .trim();
  }

  onOverlayClick(event: Event): void {
    // Close when clicking on the overlay (not the panel)
    if (event.target === event.currentTarget) {
      this.close.emit();
    }
  }
}
