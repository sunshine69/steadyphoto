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

    <div v-else class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-4">
      <div 
        v-for="photo in photos" 
        :key="photo.id" 
        class="group relative aspect-square overflow-hidden rounded-xl bg-zinc-900 transition-all hover:ring-2 hover:ring-indigo-500"
      >
        <img 
          :src="photoOriginalUrl(photo.id)" 
          :alt="photo.filename"
          class="h-full w-full object-cover transition-transform duration-500 group-hover:scale-110"
          loading="lazy"
        />
        
        <!-- Overlay on hover -->
        <div class="absolute inset-0 bg-black/40 opacity-0 group-hover:opacity-100 transition-opacity flex items-end p-3">
          <p class="text-xs text-white truncate">{{ photo.filename }}</p>
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

onMounted(async () => {
  try {
    // In a real environment, this calls your Go backend
    // For now, we'll handle potential errors gracefully
    const data = await api.getPhotos();
    photos.value = data;
  } catch (err) {
    console.error("Error fetching photos:", err);
  } finally {
    loading.value = false;
  }
});

const photoOriginalUrl = (id) => {
  return api.getPhotoOriginal(id);
};
</script>
