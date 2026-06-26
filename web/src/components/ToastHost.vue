<script setup lang="ts">
import { AlertCircle, CheckCircle2, Info, X, XCircle } from 'lucide-vue-next';
import { dismissToast, useToasts, type ToastTone } from '../composables/useToast';

const toasts = useToasts();

function iconFor(tone: ToastTone) {
  return tone === 'success' ? CheckCircle2 : tone === 'error' ? XCircle : tone === 'warning' ? AlertCircle : Info;
}
</script>

<template>
  <Teleport to="body">
    <TransitionGroup name="toast" tag="div" class="toast-stack">
      <article v-for="toast in toasts" :key="toast.id" class="toast-card" :class="toast.tone" role="status">
        <component :is="iconFor(toast.tone)" class="toast-icon" :size="20" />
        <div class="toast-body">
          <strong>{{ toast.title }}</strong>
          <p v-if="toast.message">{{ toast.message }}</p>
        </div>
        <button class="toast-close" type="button" title="关闭提示" @click="dismissToast(toast.id)">
          <X :size="15" />
        </button>
      </article>
    </TransitionGroup>
  </Teleport>
</template>
