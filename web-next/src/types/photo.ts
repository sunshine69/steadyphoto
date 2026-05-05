export interface Photo {
  id: string;
  path: string;
  filename: string;
  captured_at: string;
  width: number;
  height: number;
  size_bytes: number;
  metadata: Record<string, any>;
}

export interface ListPhotosResponse {
  photos: Photo[];
  total: number;
}
