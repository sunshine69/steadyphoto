import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';

@Component({
    selector: 'app-map',
    imports: [CommonModule],
    template: `
    <div class="map-container">
      <h1>Map</h1>
      <p class="placeholder-text">Interactive map view coming soon...</p>
    </div>
  `,
    styles: [`
    .map-container {
      padding: 20px;
    }

    h1 {
      font-size: 24px;
      color: #e5e7eb;
      margin-bottom: 16px;
    }

    .placeholder-text {
      color: #9ca3af;
      font-size: 14px;
    }
  `]
})
export class MapComponent {}
