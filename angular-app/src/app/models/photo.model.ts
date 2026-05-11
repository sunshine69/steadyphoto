export interface VideoMetadata {
  duration?: number;
  bitrate?: number;
  video_codec?: string;
  audio_codec?: string;
  frame_rate?: number;
}

export interface Photo {
  id: string;
  path: string;
  filename: string;
  captured_at: string;
  width?: number;
  height?: number;
  size?: number;
  type?: string;
  mediaType?: 'photo' | 'video';
  thumbnailUrl?: string;
  tags?: string; // Comma-separated tags from backend
  metadata?: {
    camera?: string;
    iso?: string | number;
    aperture?: string;
    focal_length?: string;
    gps_lat?: number;
    gps_lon?: number;
    [key: string]: any;
  };
  videoMetadata?: VideoMetadata;
}

export interface UpdateTagsRequest {
  tags: string;
}

export interface ListPhotosResponse {
  photos: Photo[];
  total: number;
}
