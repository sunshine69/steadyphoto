import axios from 'axios';
import { Photo, ListPhotosResponse } from '../types/photo';

const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL || 'http://localhost:8081/api/v1';

const api = axios.create({
  baseURL: API_BASE_URL,
});

// Map backend PascalCase keys to frontend camelCase keys
function normalizePhoto(p: any): Photo {
  return {
    id: p.ID ?? '',
    path: p.Path ?? '',
    filename: p.Filename ?? '',
    captured_at: p.CapturedAt ?? '',
    width: p.Width ?? 0,
    height: p.Height ?? 0,
    size_bytes: p.SizeBytes ?? 0,
    metadata: p.Metadata ?? {},
  };
}

export const fetchPhotos = async (): Promise<ListPhotosResponse> => {
  const { data } = await api.get('/photos');
  
  // Backend returns photos as a bare array in ListPhotosResponse.Photos field
  if (data.photos && Array.isArray(data.photos)) {
    return {
      photos: data.photos.map(normalizePhoto),
      total: data.totalCount ?? data.photos.length,
    };
  }

  // Fallback: treat the response itself as an array of photo objects
  const rawPhotos = Array.isArray(data) ? data : [];
  
  return {
    photos: rawPhotos.map(normalizePhoto),
    total: rawPhotos.length,
  };
};

export const fetchPhoto = async (id: string): Promise<Photo> => {
  const { data } = await api.get(`/photos/${id}`);
  return normalizePhoto(data);
};

export const getPhotoThumbnailUrl = (id: string): string => {
  return `${API_BASE_URL}/photos/${id}/thumb`;
};

export const getPhotoOriginalUrl = (id: string): string => {
  return `${API_BASE_URL}/photos/${id}/file`;
};
