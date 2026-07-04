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
  metadata?: PhotoMetadata;
  videoMetadata?: VideoMetadata;
}

export interface PhotoMetadata {
  // Camera info
  make?: string;
  model?: string;
  lens_model?: string;
  modifydate?: string;
  ModifyDate?: string;
  // Exposure settings
  exposure_time?: string;
  f_number?: string;
  iso?: string;
  focal_length?: string;
  exposure_program?: string;
  white_balance?: string;
  flash?: string;
  color_space?: string;
  // DateTime
  datetime_original?: string;
  DateTimeOriginal?: string;
  datetime?: string;
  datetime_digitized?: string;
  // Dimensions from EXIF
  image_width?: string;
  image_length?: string;
  // Orientation
  orientation?: string;
  // GPS
  gps_latitude?: string;
  gps_longitude?: string;
  gps_altitude?: string;
  gps_latitude_ref?: string;
  gps_longitude_ref?: string;
  // Software
  software?: string;
  artist?: string;
  image_description?: string;
}

export interface UpdateTagsRequest {
  tags: string;
}

export interface ListPhotosResponse {
  photos: Photo[];
  total: number;
}
