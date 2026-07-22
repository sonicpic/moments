<template>
  <div :class="disabled ? 'opacity-50' : ''" class="flex items-center justify-between gap-4 rounded-xl bg-gray-50 px-3 py-2.5 dark:bg-neutral-700/40">
    <span class="text-sm text-neutral-700 dark:text-neutral-100">{{ label }}</span>
    <UToggle :model-value="modelValue" :disabled="disabled" @update:model-value="emit('update:modelValue', $event)" />
  </div>
</template>

<script setup lang="ts">
defineProps<{
  label: string;
  modelValue: boolean;
  disabled?: boolean;
}>();

const emit = defineEmits<{
  "update:modelValue": [value: boolean];
}>();
</script>
