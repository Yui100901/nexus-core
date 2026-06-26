<script setup lang="ts">
import { computed } from 'vue';
import { Copy } from 'lucide-vue-next';
import { prettyJson } from '../utils/status';

const props = defineProps<{
  title?: string;
  value: unknown;
}>();

const text = computed(() => prettyJson(props.value));

async function copy() {
  await navigator.clipboard.writeText(text.value);
}
</script>

<template>
  <section class="result-panel">
    <div class="result-panel__head">
      <strong>{{ title || '响应结果' }}</strong>
      <button class="icon-button" type="button" title="复制结果" @click="copy">
        <Copy :size="15" />
      </button>
    </div>
    <pre>{{ text }}</pre>
  </section>
</template>
