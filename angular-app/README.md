# Angular App - SteadyPhoto Frontend

This Angular application is the frontend for SteadyPhoto, a filesystem-first photo management system.

## Features

- **Photo Grid**: Responsive masonry-style grid displaying photo thumbnails
- **Photo Detail View**: Full-screen modal with detailed EXIF information
- **Responsive Design**: Works on mobile, tablet, and desktop
- **Image Optimization**: Lazy loading and responsive image handling
- **API Integration**: Connects to the SteadyPhoto backend API

## Project Structure

```
angular-app/
├── src/
│   ├── app/
│   │   ├── components/      # UI components
│   │   │   ├── photo-list/
│   │   │   ├── photo-card/
│   │   │   └── photo-detail/
│   │   ├── models/          # TypeScript interfaces
│   │   │   └── photo.model.ts
│   │   ├── services/        # API services
│   │   │   └── photo.service.ts
│   │   ├── app.component.ts # Root component
│   │   └── app.module.ts    # Angular module
│   ├── environments/        # Environment configurations
│   ├── main.ts              # Application entry point
│   └── styles.scss          # Global styles
├── angular.json             # Angular workspace configuration
├── package.json             # Dependencies
├── tsconfig.json            # TypeScript configuration
└── README.md                # This file
```

## Getting Started

### Prerequisites

- Node.js 18+ 
- Angular CLI 18+
- Access to the SteadyPhoto backend API running on `http://localhost:8081/api/v1`

### Installation

1. Navigate to the `angular-app` directory:
   ```bash
   cd angular-app
   ```

2. Install dependencies:
   ```bash
   npm install
   ```

3. Configure environment variables:
   - Copy `src/environments/environment.ts` and set your API base URL:
   ```bash
   API_BASE_URL=http://localhost:8081/api/v1
   ```

4. Start the development server:
   ```bash
   ng serve --host 0.0.0.0 --port 4200
   ```

5. Open your browser and navigate to `http://localhost:4200`

## Available Commands

| Command | Description |
|---------|-------------|
| `ng serve` | Start development server |
| `ng build` | Build for production |
| `ng test` | Run unit tests |
| `ng lint` | Lint the code |
| `ng e2e` | Run end-to-end tests |

## Components

### PhotoListComponent
- Displays the main photo grid
- Handles loading states and errors
- Shows empty state when no photos are available
- Supports refresh functionality

### PhotoCardComponent
- Individual photo card with hover effects
- Displays thumbnail and capture date
- Triggers photo detail view on click

### PhotoDetailComponent
- Full-screen modal for photo viewing
- Shows metadata in a side panel
- Supports keyboard navigation (ESC to close)
- Responsive layout for mobile/desktop

## API Integration

The app uses Angular's HttpClient to communicate with the SteadyPhoto backend API:

```typescript
// Fetch all photos
this.photoService.getPhotos().subscribe({
  next: (response) => console.log(response),
  error: (error) => console.error(error)
});

// Get a specific photo
this.photoService.getPhoto(id).subscribe({
  next: (photo) => console.log(photo),
  error: (error) => console.error(error)
});
```

## Styling

The application uses a combination of:
- **Bootstrap 5**: For layout and base components
- **Custom SCSS**: For theme variables and app-specific styling
- **Tailwind CSS**: For utility classes (configurable)

## Responsive Design

The app is fully responsive:
- **Mobile**: Single column layout
- **Tablet**: 2-3 columns
- **Desktop**: 4-5 columns grid
- **Large Screens**: Optimized viewing

## Future Enhancements

- [ ] Add search functionality
- [ ] Implement infinite scrolling
- [ ] Add photo album management
- [ ] Support for facial recognition data
- [ ] Advanced filtering options
- [ ] Dark/Light theme toggle
- [ ] Keyboard shortcuts for common actions

## Troubleshooting

### Common Issues

1. **CORS Errors**: Ensure your backend API allows CORS requests from `http://localhost:4200`

2. **API Connection Failures**: Verify the API_BASE_URL in environment configuration

3. **Build Errors**: Clean node_modules and reinstall:
   ```bash
   rm -rf node_modules package-lock.json
   npm install
   ```

## Contributing

1. Create a feature branch
2. Make your changes
3. Run tests: `ng test`
4. Submit a pull request

## License

This project is part of the SteadyPhoto ecosystem.
