
import { Calendar, Info } from 'lucide-react';
import type { Photo } from '@/types/photo';

interface PhotoGridProps {
  photos: Photo[];
  onPhotoClick?: (photoId: string) => void;
}

export default function PhotoGrid({ photos, onPhotoClick }: PhotoGridProps) {
  if (!photos || photos.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center h-64 text-zinc-500">
        <p>No photos found</p>
      </div>
    );
  }

  return (
    <div className="columns-2 md:columns-3 lg:columns-4 xl:columns-5 gap-2 space-y-2">
      {photos.map((photo) => (
        <PhotoCard 
          key={photo.id} 
          photo={photo} 
          onClick={() => onPhotoClick?.(photo.id)} 
        />
      ))}
    </div>
  );
}

function PhotoCard({ photo, onClick }: { photo: Photo; onClick?: () => void }) {
  return (
    <div 
      className="break-inside-avoid group relative rounded-sm overflow-hidden bg-zinc-900 cursor-pointer"
      onClick={onClick}
    >
      {/* Thumbnail Image */}
      <img
        src={photo.thumbnailUrl || `/api/v1/photos/${photo.id}/thumbnail`}
        alt={photo.filename}
        className="w-full h-auto object-cover transition-transform duration-300 group-hover:scale-105"
        loading="lazy"
      />

      {/* Hover Overlay */}
      <div className="absolute inset-0 bg-black/40 opacity-0 group-hover:opacity-100 transition-opacity duration-200 flex flex-col justify-between p-2">
        <div className="flex justify-end">
          <button 
            className="p-1.5 rounded-full hover:bg-white/20 text-white transition-colors"
            onClick={(e) => { e.stopPropagation(); }} // Prevent opening lightbox on info click for now
          >
            <Info size={16} />
          </button>
        </div>
        
        <div className="flex items-center gap-2 text-xs text-white font-medium bg-black/30 px-2 py-1 rounded-md w-fit backdrop-blur-sm">
          <Calendar size={14} />
          {new Date(photo.captured_at).toLocaleDateString()}
        </div>
      </div>
    </div>
  );
}
