"use client";

import { useState, useEffect } from 'react';
import PhotoGrid from '@/components/PhotoGrid';
import Lightbox from '@/components/Lightbox';
import type { Photo } from '@/types/photo';

// Mock data using picsum.photos for realistic placeholder images
const MOCK_PHOTOS: Photo[] = [
  {
    id: '1',
    path: '/2024/01/15/sunset.jpg',
    filename: 'sunset.jpg',
    captured_at: '2024-01-15T18:30:00Z',
    width: 4000,
    height: 3000,
    size: 5000000,
    metadata: { camera: 'Canon EOS R5', iso: 200, aperture: 'f/2.8' },
    thumbnailUrl: "https://picsum.photos/seed/sunset1/600/400" 
  },
  {
    id: '2',
    path: '/2024/01/16/mountain.jpg',
    filename: 'mountain.jpg',
    captured_at: '2024-01-16T10:15:00Z',
    width: 3000,
    height: 4000,
    size: 6000000,
    metadata: { camera: 'Sony A7IV', iso: 100, aperture: 'f/8' },
    thumbnailUrl: "https://picsum.photos/seed/mountain2/400/600" 
  },
  {
    id: '3',
    path: '/2024/01/17/city.jpg',
    filename: 'city.jpg',
    captured_at: '2024-01-17T20:45:00Z',
    width: 5000,
    height: 3333,
    size: 7000000,
    metadata: { camera: 'Nikon Z8', iso: 400, aperture: 'f/1.8' },
    thumbnailUrl: "https://picsum.photos/seed/city3/600/400" 
  },
];

export default function Home() {
  const [photos, setPhotos] = useState<Photo[]>([]);
  const [selectedPhotoId, setSelectedPhotoId] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    console.log('[DEBUG HOME] Component mounted');
    
    const loadPhotos = async () => {
      setIsLoading(true);
      console.log('[DEBUG HOME] Loading photos...');
      
      try {
        // Using a Promise for the delay to ensure correct async behavior
        await new Promise(resolve => setTimeout(resolve, 500));
        console.log('[DEBUG HOME] Setting photos state', MOCK_PHOTOS.length, 'photos');
        setPhotos(MOCK_PHOTOS);
        setError(null);
      } catch (err: any) {
        console.error('[DEBUG HOME] Error loading photos:', err);
        setError(err.message);
      } finally {
        setIsLoading(false);
      }
    };

    loadPhotos();
    
    return () => {
      console.log('[DEBUG HOME] Component unmounting');
    };
  }, []);

  // Debug: log when state changes
  useEffect(() => {
    if (photos.length > 0) {
      console.log('[DEBUG HOME] Photos rendered:', photos.length, 'selected:', selectedPhotoId);
    }
  }, [photos, selectedPhotoId]);

  return (
    <div className="min-h-full">
      {/* Error Display */}
      {error && (
        <div className="p-4 bg-red-900/50 text-red-200 rounded-lg m-4">
          <strong>Error:</strong> {error}
        </div>
      )}

      {/* Empty State */}
      {isLoading ? (
        <div className="flex flex-col items-center justify-center h-64 text-zinc-500 space-y-4">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-500"></div>
          <p>Loading photos...</p>
        </div>
      ) : photos.length === 0 ? (
        <div className="flex flex-col items-center justify-center h-96 text-zinc-500">
          <svg xmlns="http://www.w3.org/2000/svg" width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1" strokeLinecap="round" strokeLinejoin="round" className="mb-4 opacity-50">
            <rect x="3" y="3" width="18" height="18" rx="2" ry="2"></rect>
            <circle cx="8.5" cy="8.5" r="1.5"></circle>
            <polyline points="21 15 16 10 5 21"></polyline>
          </svg>
          <p className="text-lg font-medium">No photos yet</p>
          <p className="text-sm mt-2">Import some photos to get started</p>
        </div>
      ) : (
        <>
          <PhotoGrid 
            photos={photos} 
            onPhotoClick={(photoId) => setSelectedPhotoId(photoId)} 
          />
          
          {selectedPhotoId && (
            <Lightbox 
              isOpen={!!selectedPhotoId}
              onClose={() => setSelectedPhotoId(null)}
              photoUrl={photos.find(p => p.id === selectedPhotoId)?.thumbnailUrl || `/api/v1/photos/${selectedPhotoId}/thumbnail`} 
              photoData={photos.find(p => p.id === selectedPhotoId)}
            />
          )}
        </>
      )}
    </div>
  );
}
