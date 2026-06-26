<script setup lang="ts">
import { reactive, ref } from 'vue';
import { Copy, KeyRound, Search, Trash2 } from 'lucide-vue-next';
import { api } from '../api/client';
import type { LicenseData } from '../api/types';
import StatusBadge from '../components/StatusBadge.vue';
import ResultPanel from '../components/ResultPanel.vue';
import { licenseStatusLabel, statusTone } from '../utils/status';

const result = ref<unknown>({});
const error = ref('');
const current = ref<LicenseData | null>(null);
const createForm = reactive({ product_id: 1, validity_hours: 720, max_nodes: 2, max_concurrent: 1, remark: '' });
const batchForm = reactive({ product_id: 1, validity_hours: 720, max_nodes: 2, max_concurrent: 1, count: 10, remark: '' });
const queryForm = reactive({ id: 1, key: '' });
const updateForm = reactive({ id: 1, max_nodes: 2, max_concurrent: 1, feature_mask: '', remark: '' });
const renewForm = reactive({ id: 1, extra_hours: 168 });

async function run(action: () => Promise<unknown>) {
  error.value = '';
  try {
    result.value = await action();
  } catch (err) {
    error.value = err instanceof Error ? err.message : '操作失败';
  }
}

function setCurrent(data: LicenseData) {
  current.value = data;
  updateForm.id = data.id;
  updateForm.remark = data.remark || '';
}

async function copyKey() {
  if (current.value?.license_key) await navigator.clipboard.writeText(current.value.license_key);
}
</script>

<template>
  <section class="page">
    <div class="page-head">
      <div>
        <h1>License 管理</h1>
        <p>创建授权、查询密钥、续期、吊销和清理。</p>
      </div>
    </div>

    <p v-if="error" class="alert bad">{{ error }}</p>

    <div class="grid three">
      <form class="panel form-panel" @submit.prevent="run(() => api.createLicense({ ...createForm, remark: createForm.remark || null }))">
        <h2>创建 License</h2>
        <label>产品 ID<input v-model.number="createForm.product_id" type="number" min="1" required /></label>
        <label>有效小时<input v-model.number="createForm.validity_hours" type="number" min="1" required /></label>
        <label>最大节点<input v-model.number="createForm.max_nodes" type="number" min="0" /></label>
        <label>最大并发<input v-model.number="createForm.max_concurrent" type="number" min="0" /></label>
        <label>备注<input v-model="createForm.remark" /></label>
        <button class="primary-button" type="submit"><KeyRound :size="16" /> 创建</button>
      </form>

      <form class="panel form-panel" @submit.prevent="run(() => api.batchCreateLicenses({ ...batchForm, remark: batchForm.remark || null }))">
        <h2>批量创建</h2>
        <label>产品 ID<input v-model.number="batchForm.product_id" type="number" min="1" required /></label>
        <label>数量<input v-model.number="batchForm.count" type="number" min="1" max="1000" required /></label>
        <label>有效小时<input v-model.number="batchForm.validity_hours" type="number" min="1" required /></label>
        <label>最大节点<input v-model.number="batchForm.max_nodes" type="number" min="0" /></label>
        <label>最大并发<input v-model.number="batchForm.max_concurrent" type="number" min="0" /></label>
        <label>备注<input v-model="batchForm.remark" /></label>
        <button class="primary-button" type="submit"><KeyRound :size="16" /> 批量创建</button>
      </form>

      <form class="panel form-panel">
        <h2>查询</h2>
        <label>License ID<input v-model.number="queryForm.id" type="number" min="1" /></label>
        <button class="primary-button" type="button" @click="run(async () => { const data = await api.getLicense(queryForm.id); setCurrent(data); return data; })"><Search :size="16" /> 按 ID 查询</button>
        <label>License Key<input v-model="queryForm.key" /></label>
        <button class="secondary-button" type="button" @click="run(async () => { const data = await api.getLicenseByKey(queryForm.key); setCurrent(data); return data; })"><Search :size="16" /> 按 Key 查询</button>
        <div v-if="current" class="detail-list">
          <span>状态</span>
          <StatusBadge :label="licenseStatusLabel(current.status)" :tone="statusTone(current.status, 'license')" />
          <span>Key</span>
          <button class="text-copy" type="button" @click="copyKey"><Copy :size="14" /> {{ current.license_key }}</button>
        </div>
      </form>
    </div>

    <div class="grid two">
      <form class="panel form-panel" @submit.prevent="run(() => api.updateLicense(updateForm.id, { ...updateForm, remark: updateForm.remark || null }))">
        <h2>编辑 License</h2>
        <label>ID<input v-model.number="updateForm.id" type="number" min="1" required /></label>
        <label>最大节点<input v-model.number="updateForm.max_nodes" type="number" min="0" /></label>
        <label>最大并发<input v-model.number="updateForm.max_concurrent" type="number" min="0" /></label>
        <label>功能掩码<input v-model="updateForm.feature_mask" placeholder="control" /></label>
        <label>备注<input v-model="updateForm.remark" /></label>
        <button class="primary-button" type="submit">保存</button>
      </form>

      <form class="panel form-panel">
        <h2>状态与清理</h2>
        <label>ID<input v-model.number="renewForm.id" type="number" min="1" required /></label>
        <label>续期小时<input v-model.number="renewForm.extra_hours" type="number" required /></label>
        <div class="button-row wrap">
          <button class="primary-button" type="button" @click="run(() => api.renewLicense(renewForm.id, renewForm.extra_hours))">续期</button>
          <button class="secondary-button" type="button" @click="run(() => api.restoreLicense(renewForm.id))">恢复</button>
          <button class="danger-button" type="button" @click="run(() => api.revokeLicense(renewForm.id))">吊销</button>
          <button class="danger-button" type="button" @click="run(() => api.cleanLicenseBindings(renewForm.id))">清理绑定</button>
          <button class="danger-button" type="button" @click="run(() => api.deleteLicense(renewForm.id))"><Trash2 :size="16" /> 删除</button>
          <button class="danger-button" type="button" @click="run(() => api.cleanInvalidLicenses())">清理无效</button>
        </div>
      </form>
    </div>

    <ResultPanel :value="result" />
  </section>
</template>
