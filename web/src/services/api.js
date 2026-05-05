const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api/v1';

export const api = {
  async getPhotos(params = {}) {
    const query = new URLSearchParams(params).toString();
    const url = `${API_BASE_URL}/photos${query ? '?' + query : ''}`;
    
    console.log("DEBUG: Fetching photos from URL:", url);
    
    const response = await fetch(url);
    if (!response.ok) {
        console.error(`Fetch error: ${response.status} ${response.statusText} for URL: ${url}`);
        throw new Error('Failed to fetch photos');
    }
    const data = await response.json();
    console.log("DEBUG: Received photos data:", data);
    return data;
  },

  async getPhoto(id) {
    const url = `${API_BASE_URL}/photos/${id}`;
    console.log("DEBUG: Fetching photo metadata from URL:", url);
    
    const response = await fetch(url);
    if (!response.ok) throw new Error('Failed to fetch photo metadata');
    return response.json();
  },

  async getPhotoOriginal(id) {
    return `${API_BASE_URL}/photos/${id}/original`;
  },

  async getPhotoThumbnail(id) {
    return `${API_BASE_URL}/photos/${id}/thumb`;
  }
};
