<template>
  <section class="settings-card rounded-2xl bg-white p-4 shadow-sm ring-1 ring-black/5 dark:bg-neutral-800 dark:ring-white/10">
    <div>
      <h2 class="font-semibold text-neutral-800 dark:text-white">{{ title }}</h2>
      <p v-if="description" class="mt-1 text-xs leading-5 text-gray-500 dark:text-gray-400">{{ description }}</p>
    </div>
    <slot />
  </section>
</template>

<script setup lang="ts">
defineProps<{
  title: string;
  description?: string;
}>();
</script>
