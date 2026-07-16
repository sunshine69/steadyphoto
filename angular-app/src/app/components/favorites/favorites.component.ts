import { Component, ChangeDetectionStrategy } from '@angular/core';


@Component({
    selector: 'app-favorites',
    imports: [],
    template: `
    <div class="favorites-container">
      <h1>Favorites</h1>
      <p class="placeholder-text">Favorites view coming soon...</p>
    </div>
  `,
    changeDetection: ChangeDetectionStrategy.Eager,
    styles: [`
    .favorites-container {
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
export class FavoritesComponent {}
