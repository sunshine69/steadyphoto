// User-to-user share (group of media/albums shared between two users)
export interface Share {
  id: string;
  sharerUserId: string;
  shareeUserId: string;
  sharedAt: string;
}

// Incoming shared media item (received from another user)
export interface SharedMediaItem {
  id: string;
  filename: string;
  mediaType: 'photo' | 'video';
  sharerUserId: string;
}

// Incoming shared album (received from another user)
export interface SharedAlbumItem {
  id: string;
  name: string;
  description?: string;
  sharerUserId: string;
}

// Public share link response after creation
export interface PublicShareLinkResponse {
  id: string;
  token: string;
  sharerUserId: string;
  resourceType: 'media' | 'album';
  password_protected: boolean;
  expires_at?: string;
  created_at: string;
}

// Public share link in list view (includes access count)
export interface PublicShareListItem {
  id: string;
  token: string;
  sharerUserId: string;
  resourceType: 'media' | 'album';
  password_protected: boolean;
  expires_at?: string;
  created_at: string;
  access_count: number;
}

// Request body for user-to-user share creation
export interface ShareRequest {
  sharerUserId?: string; // Set automatically from auth context
  sharee_user_ids: string[];
  media_ids?: string[];
  album_ids?: string[];
}

// Response after creating a share
export interface CreateShareResponseFull {
  shares_created: Share[];
  media_shared_count: number;
  albums_shared_count: number;
}

// User search result for sharing (only id and email returned)
export interface SearchUser {
  id: string;
  email: string;
}

// Request body for public share link creation
export interface PublicShareRequest {
  resource_type: 'media' | 'album';
  resource_id: string;
  password?: string; // Optional password protection
  expires_at?: string; // Optional expiration (ISO datetime)
}

// Paginated response for shared media/albums lists
export interface SharedItemsResponse<T> {
  items: T[];
  total: number;
  limit: number;
  offset: number;
}

// Outgoing share group (group of media/albums the current user has shared with someone)
export interface ShareGroupListItem {
  id: string;
  sharerName?: string;   // Name of current user (displayed on their side, not needed for outgoing)
  shareeName?: string;   // Name of person receiving shares
  mediaCount: number;    // Number of media items in this share group
  albumsCount: number;   // Number of albums in this share group  
  sharedAt: string;      // ISO datetime when shares were created
}

