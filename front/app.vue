<template>
  <NuxtLayout>
    <NuxtPage/>
    <Toaster richColors position="top-center"/>
  </NuxtLayout>
</template>

<script lang="ts" setup>
import { Toaster } from 'vue-sonner'
import "@fancyapps/ui/dist/fancybox/fancybox.css";
import '~/assets/simple-markdown.scss'
</script>

<style>
html,
body,
#__nuxt {
  width: 100%;
  max-width: 100%;
  overflow-x: clip;
}

@media (max-width: 767px) {
  html,
  body {
    overscroll-behavior-x: none;
    touch-action: pan-y pinch-zoom;
  }
}

div.grecaptcha-badge{
  display: none !important;
}
</style>
