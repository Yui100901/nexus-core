<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { Braces, Check, Wand2 } from 'lucide-vue-next';
import { prettyJson } from '../utils/status';

const props = defineProps<{
  modelValue: string;
  rows?: number;
  placeholder?: string;
}>();

const emit = defineEmits<{
  'update:modelValue': [value: string];
}>();

const localValue = ref(props.modelValue);
const error = ref('');

watch(
  () => props.modelValue,
  (value) => {
    if (value !== localValue.value) localValue.value = value;
  },
);

watch(localValue, (value) => {
  emit('update:modelValue', value);
});

const isValid = computed(() => {
  try {
    JSON.parse(localValue.value || '{}');
    return true;
  } catch {
    return false;
  }
});

function format() {
  try {
    localValue.value = prettyJson(JSON.parse(localValue.value || '{}'));
    error.value = '';
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'JSON 格式错误';
  }
}

function validate() {
  try {
    JSON.parse(localValue.value || '{}');
    error.value = '';
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'JSON 格式错误';
  }
}
</script>

<template>
  <div class="json-editor">
    <div class="json-editor__bar">
      <span><Braces :size="15" /> JSON</span>
      <div class="icon-row">
        <button class="icon-button" type="button" title="格式化 JSON" @click="format">
          <Wand2 :size="15" />
        </button>
        <button class="icon-button" type="button" title="校验 JSON" @click="validate">
          <Check :size="15" />
        </button>
      </div>
    </div>
    <textarea
      v-model="localValue"
      class="json-editor__textarea"
      :class="{ invalid: !isValid }"
      :rows="rows || 8"
      spellcheck="false"
      :placeholder="placeholder || '{}'"
    />
    <p v-if="error" class="form-error">{{ error }}</p>
  </div>
</template>
