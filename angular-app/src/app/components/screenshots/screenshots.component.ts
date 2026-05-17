import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';

@Component({
  selector: 'app-screenshots',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div class="placeholder-container">
      <h1>Screenshots</h1>
      <p class="placeholder-text">Screenshots view coming soon...</p>
    </div>
  `,
  styles: [`
    .placeholder-container { padding: 20px; }
    h1 { font-size: 24px; color: #e5e7eb; margin-bottom: 16px; }
    .placeholder-text { color: #9ca3af; font-size: 14px; }
  `]
})
export class ScreenshotsComponent {}
