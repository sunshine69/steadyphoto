<template>
  <div class="flex h-screen bg-zinc-950 text-zinc-100 font-sans selection:bg-indigo-500/30">
    <!-- Sidebar -->
    <aside class="w-64 border-r border-zinc-800 flex flex-col bg-zinc-950/50 backdrop-blur-xl">
      <div class="p-6">
        <h1 class="text-xl font-bold tracking-tight text-white flex items-center gap-2">
          <div class="w-6 h-6 bg-indigo-600 rounded-lg"></div>
          SteadyPhoto
        </h1>
      </div>
      
      <nav class="flex-1 px-4 space-y-1">
        <a href="#" class="flex items-center px-3 py-2 rounded-lg text-sm font-medium bg-zinc-800 text-white transition-colors">
          <span>All Photos</span>
        </a>
        <a href="#" class="flex items-center px-3 py-2 rounded-lg text-sm font-medium text-zinc-400 hover:bg-zinc-900 hover:text-white transition-colors">
          <span>Albums</span>
        </a>
        <a href="#" class="flex items-center px-3 py-2 rounded-lg text-sm font-medium text-zinc-400 hover:bg-zinc-900 hover:text-white transition-colors">
          <span>People</span>
        </a>
        <a href="#" class="flex items-center px-3 py-2 rounded-lg text-sm font-medium text-zinc-400 hover:bg-zinc-900 hover:text-white transition-colors">
          <span>Places</span>
        </a>
      </nav>

      <div class="p-4 border-t border-zinc-800 text-[10px] uppercase tracking-widest text-zinc-600 text-center">
        v0.1.0-alpha
      </div>
    </aside>

    <!-- Main Content -->
    <main class="flex-1 overflow-y-auto relative">
      <div class="min-h-full">
        <slot />
      </div>
    </main>
  </div>
</template>

<script setup>
// Layout component logic
</script>
