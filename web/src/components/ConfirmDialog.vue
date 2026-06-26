<script setup lang="ts">
defineProps<{
  open: boolean;
  title: string;
  message: string;
  confirmLabel?: string;
  cancelLabel?: string;
}>();

const emit = defineEmits<{
  confirm: [];
  cancel: [];
}>();
</script>

<template>
  <div v-if="open" class="modal-backdrop" @click.self="emit('cancel')">
    <section class="modal-panel confirm-dialog" role="dialog" aria-modal="true" :aria-label="title">
      <div class="modal-head">
        <h2>{{ title }}</h2>
      </div>
      <p>{{ message }}</p>
      <div class="button-row">
        <button class="danger-button" type="button" @click="emit('confirm')">{{ confirmLabel || '确认执行' }}</button>
        <button class="secondary-button" type="button" @click="emit('cancel')">{{ cancelLabel || '取消' }}</button>
      </div>
    </section>
  </div>
</template>
