<template>
  <Layout>
    <PhotoGrid />
  </Layout>
</template>

<script setup>
import Layout from './components/Layout.vue';
import PhotoGrid from './components/PhotoGrid.vue';
</script>

<style>
/* Basic reset to ensure full-screen layout works correctly */
html, body {
  margin: 0;
  padding: 0;
  height: 100%;
  width: 100%;
  overflow: hidden;
  background-color: #09090b; /* zinc-950 */
}

#app {
  height: 100%;
  width: 100%;
}
</style>
