import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';

@Component({
    selector: 'app-sharing',
    imports: [CommonModule],
    template: `
    <div class="sharing-container">
      <h1>Sharing</h1>
      <p class="placeholder-text">Sharing view coming soon...</p>
    </div>
  `,
    styles: [`
    .sharing-container {
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
export class SharingComponent {}
