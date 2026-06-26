<script setup lang="ts">
defineProps<{
  label: string;
  tone?: 'good' | 'warn' | 'bad' | 'idle';
}>();
</script>

<template>
  <span class="status-badge" :class="tone || 'idle'">{{ label }}</span>
</template>
