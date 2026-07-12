import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';

@Component({
    selector: 'app-camera',
    imports: [CommonModule],
    template: `
    <div class="placeholder-container">
      <h1>Camera</h1>
      <p class="placeholder-text">Camera uploads view coming soon...</p>
    </div>
  `,
    styles: [`
    .placeholder-container { padding: 20px; }
    h1 { font-size: 24px; color: #e5e7eb; margin-bottom: 16px; }
    .placeholder-text { color: #9ca3af; font-size: 14px; }
  `]
})
export class CameraComponent {}
