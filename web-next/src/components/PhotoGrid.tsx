"use client";

import React, { useState, useEffect } from 'react';
import { useQuery } from '@tanstack/react-query';
import { fetchPhotos } from '@/lib/api';
import { Photo } from '@/types/photo';
import { getPhotoThumbnailUrl } from '@/lib/api';
import { Lightbox } from '@/components/Lightbox';

export default function PhotoGrid() {
  const { data, isLoading, error } = useQuery({
    queryKey: ['photos'],
    queryFn: fetchPhotos,
  });

  // DEBUGGING LOGS
  useEffect(() => {
    console.log('[PhotoGrid] Query status:', { isLoading, error, data });
  }, [data, isLoading, error]);

  const [selectedPhoto, setSelectedPhoto] = useState<Photo | null>(null);
  const [isLightboxOpen, setIsLightboxOpen] = useState(false);

  const handlePhotoClick = (photo: Photo) => {
    setSelectedPhoto(photo);
    setIsLightboxOpen(true);
  };

  if (isLoading) {
    return (
      <div className="flex h-screen w-full items-center justify-center">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-zinc-700 border-t-white"></div>
      </div>
    );
  }

  if (error) {
    console.error('[PhotoGrid] Error details:', error);
    return (
      <div className="flex h-screen w-full items-center justify-center text-red-500">
        Error loading photos. Please check if the backend is running.
      </div>
    );
  }

  // Determine if data is the array directly or an object containing the array
  const photos = Array.isArray(data) ? data : (data as any)?.photos || [];

  return (
    <main className="p-4">
      <div className="mb-8 text-center">
        <h1 className="text-3xl font-bold text-white">SteadyPhoto</h1>
        <p className="text-zinc-500">{photos.length} photos found</p>
      </div>

      <div className="columns-1 gap-4 sm:columns-2 md:columns-3 lg:columns-4 xl:columns-5">
        {photos.map((photo: Photo) => (
          <div 
            key={photo.id} 
            className="mb-4 break-inside-avoid cursor-pointer"
            onClick={() => handlePhotoClick(photo)}
          >
            <div className="group relative overflow-hidden rounded-lg bg-zinc-900">
              <img
                src={getPhotoThumbnailUrl(photo.id)}
                alt={photo.filename}
                className="w-full object-cover transition-transform duration-300 group-hover:scale-105"
                loading="lazy"
                onError={(e) => {
                  console.warn(`[PhotoGrid] Failed to load thumbnail for ${photo.id}`);
                  (e.target as HTMLImageElement).src = 'https://via.placeholder.com/400x300?text=Error+Loading+Image';
                }}
              />
              <div className="absolute inset-0 bg-black/40 opacity-0 transition-opacity duration-300 group-hover:opacity-100">
                <div className="absolute bottom-0 left-0 p-3 text-xs text-white">
                  {photo.filename}
                </div>
              </div>
            </div>
          </div>
        ))}
      </div>

      <Lightbox 
        photo={selectedPhoto} 
        isOpen={isLightboxOpen} 
        onClose={() => setIsLightboxOpen(false)} 
      />
    </main>
  );
}
