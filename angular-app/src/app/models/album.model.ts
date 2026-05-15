export interface Album {
  id: string;
  userId: string;
  name: string;
  description?: string;
  createdAt: string;
  updatedAt: string;
}

export interface CreateAlbumRequest {
  name: string;
  description?: string;
}

export interface UpdateAlbumRequest {
  name: string;
  description?: string;
}

export interface AddMediaToAlbumRequest {
  mediaIds: string[];
}
