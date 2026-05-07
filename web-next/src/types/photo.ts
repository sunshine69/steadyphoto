export interface Photo {
  id: string;
  path: string;
  filename: string;
  captured_at: string;
  width?: number;
  height?: number;
  size?: number;
  type?: string;
  thumbnailUrl?: string;
  metadata?: {
    camera?: string;
    iso?: string | number;
    aperture?: string;
    focal_length?: string;
    gps_lat?: number;
    gps_lon?: number;
    [key: string]: any;
  };
}

export interface ListPhotosResponse {
  photos: Photo[];
  total: number;
  page: number;
  per_page: number;
}
