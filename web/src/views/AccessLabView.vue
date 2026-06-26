<script setup lang="ts">
import { reactive } from 'vue';
import { HeartPulse, PlugZap } from 'lucide-vue-next';
import { api } from '../api/client';
import { errorMessage, notifyError, notifySuccess } from '../composables/useToast';

const form = reactive({
  device_code: 'demo-node-001',
  license_key: '',
  product_id: 1,
  version_code: '1.0.0',
});

async function run(action: () => Promise<unknown>, successMessage: string) {
  try {
    await action();
    notifySuccess(successMessage);
  } catch (err) {
    notifyError(errorMessage(err));
  }
}
</script>

<template>
  <section class="page">
    <div class="page-head">
      <div>
        <h1>接入调试</h1>
        <p>模拟客户端注册和心跳，快速验证授权链路。</p>
      </div>
    </div>

    <form class="panel form-panel wide-form">
      <h2>客户端参数</h2>
      <div class="grid two compact">
        <label>设备码<input v-model="form.device_code" required /></label>
        <label>License Key<input v-model="form.license_key" required /></label>
        <label>产品 ID<input v-model.number="form.product_id" type="number" min="1" required /></label>
        <label>版本号<input v-model="form.version_code" required /></label>
      </div>
      <div class="button-row">
        <button class="primary-button" type="button" @click="run(() => api.registerAccess(form), '注册成功')"><PlugZap :size="16" /> 注册</button>
        <button class="secondary-button" type="button" @click="run(() => api.heartbeat(form), '心跳成功')"><HeartPulse :size="16" /> 心跳</button>
      </div>
    </form>
  </section>
</template>
