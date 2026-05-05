<template>
  <div class="p-6">
    <div class="flex items-center justify-between mb-8">
      <h2 class="text-2xl font-bold text-white">All Photos</h2>
      <div class="text-zinc-500 text-sm">
        {{ photos.length }} photos found
      </div>
    </div>

    <div v-if="loading" class="flex items-center justify-center h-64 text-zinc-500">
      <div class="animate-pulse">Loading your library...</div>
    </div>

    <div v-else-if="photos.length === 0" class="flex flex-col items-center justify-center h-64 text-zinc-500">
      <p>No photos found in your library.</p>
    </div>

    <!-- Photo Grid -->
    <div v-else class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-4">
      <div 
        v-for="photo in photos" 
        :key="photo.id" 
        @click="openLightbox(photo)"
        class="group relative aspect-square overflow-hidden rounded-xl bg-zinc-900 cursor-pointer transition-all hover:ring-2 hover:ring-indigo-500"
      >
        <img 
          :src="photoOriginalUrl(photo.id)" 
          :alt="photo.filename"
          class="h-full w-full object-cover transition-transform duration-500 group-hover:scale-110"
          loading="lazy"
          @error="handleImageError"
        />
        
        <!-- Loading placeholder for large files -->
        <div v-if="photo.loading" class="absolute inset-0 flex items-center justify-center bg-zinc-900">
          <div class="w-6 h-6 border-2 border-indigo-500 border-t-transparent rounded-full animate-spin"></div>
        </div>

        <!-- Overlay on hover -->
        <div class="absolute inset-0 bg-black/40 opacity-0 group-hover:opacity-100 transition-opacity flex items-end p-3">
          <p class="text-xs text-white truncate">{{ photo.filename }}</p>
        </div>
      </div>
    </div>

    <!-- Lightbox Modal -->
    <div v-if="selectedPhoto" class="fixed inset-0 z-50 flex items-center justify-center bg-black/90 backdrop-blur-sm p-4 md:p-10">
      <!-- Close Button -->
      <button 
        @click="selectedPhoto = null" 
        class="absolute top-6 right-6 text-white/50 hover:text-white text-3xl transition-colors z-[60]"
      >
        &times;
      </button>

      <!-- Image Container -->
      <div class="relative max-w-full max-h-full flex flex-col items-center">
        <img 
          :src="photoOriginalUrl(selectedPhoto.id)" 
          class="max-w-full max-h-[85vh] object-contain rounded-sm shadow-2xl"
          @click="selectedPhoto = null"
        />
        <div class="mt-4 text-center">
          <p class="text-white font-medium text-lg">{{ selectedPhoto.filename }}</p>
          <p class="text-zinc-500 text-sm">{{ formatDate(selectedPhoto.captured_at) }}</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue';
import { api } from '../services/api';

const photos = ref([]);
const loading = ref(true);
const selectedPhoto = ref(null);

onMounted(async () => {
  try {
    const data = await api.getPhotos();
    // Map data to add a loading state property
    photos.value = data.map(p => ({ ...p, loading: true }));
  } catch (err) {
    console.error("Error fetching photos:", err);
  } finally {
    loading.value = false;
  }
});

const photoOriginalUrl = (id) => {
  return api.getPhotoOriginal(id);
};

const openLightbox = (photo) => {
  selectedPhoto.value = photo;
};

const handleImageError = (event) => {
  // If image fails to load, we could show a placeholder icon
  event.target.classList.add('opacity-50');
};

const formatDate = (dateStr) => {
  if (!dateStr) return '';
  const date = new Date(dateStr);
  return date.toLocaleDateString('en-US', { 
    month: 'long', 
    day: 'numeric', 
    year: 'numeric' 
  });
};
</script>

<style scoped>
/* Smooth transition for the grid items */
.grid > div {
  transition: transform 0.2s ease-out;
}
</style>
