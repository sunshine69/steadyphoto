import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';

@Component({
  selector: 'app-explore',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div class="explore-container">
      <h1>Explore</h1>
      <p class="placeholder-text">Explore view coming soon...</p>
    </div>
  `,
  styles: [`
    .explore-container {
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
export class ExploreComponent {}
