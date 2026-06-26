<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue';
import { Copy, Edit3, KeyRound, Plus, RefreshCw, Save, Trash2 } from 'lucide-vue-next';
import { api } from '../api/client';
import type { LicenseData } from '../api/types';
import StatusBadge from '../components/StatusBadge.vue';
import ResultPanel from '../components/ResultPanel.vue';
import { licenseStatusLabel, statusTone } from '../utils/status';

const loading = ref(false);
const error = ref('');
const result = ref<unknown>({});
const licenses = ref<LicenseData[]>([]);
const selected = ref<LicenseData | null>(null);
const showCreate = ref(false);

const filters = reactive({
  product_id: undefined as number | undefined,
  status: undefined as number | undefined,
  license_key: '',
  page: 1,
  page_size: 20,
});
const createForm = reactive({ product_id: 1, validity_hours: 720, max_nodes: 2, max_concurrent: 1, remark: '' });
const batchForm = reactive({ product_id: 1, validity_hours: 720, max_nodes: 2, max_concurrent: 1, count: 10, remark: '' });
const editForm = reactive({ id: 0, max_nodes: 0, max_concurrent: 0, feature_mask: '', remark: '' });
const renewForm = reactive({ id: 0, extra_hours: 168 });

async function run(action: () => Promise<unknown>, refresh = false) {
  error.value = '';
  try {
    result.value = await action();
    if (refresh) await loadLicenses();
  } catch (err) {
    error.value = err instanceof Error ? err.message : '操作失败';
  }
}

async function loadLicenses() {
  loading.value = true;
  error.value = '';
  try {
    licenses.value = await api.listLicenses(filters);
  } catch (err) {
    error.value = err instanceof Error ? err.message : '加载失败';
  } finally {
    loading.value = false;
  }
}

function selectLicense(license: LicenseData) {
  selected.value = license;
  editForm.id = license.id;
  editForm.max_nodes = license.max_nodes ?? 0;
  editForm.max_concurrent = license.max_concurrent ?? 0;
  editForm.feature_mask = license.feature_mask || '';
  editForm.remark = license.remark || '';
  renewForm.id = license.id;
}

async function copyKey(key: string) {
  await navigator.clipboard.writeText(key);
}

async function createLicense() {
  await run(() => api.createLicense({ ...createForm, remark: createForm.remark || null }), true);
  showCreate.value = false;
}

onMounted(loadLicenses);
</script>

<template>
  <section class="page">
    <div class="page-head">
      <div>
        <h1>License 管理</h1>
        <p>围绕授权列表执行创建、筛选、编辑、续期和状态操作。</p>
      </div>
      <button class="primary-button" type="button" @click="showCreate = !showCreate">
        <Plus :size="16" />
        新增授权
      </button>
    </div>

    <p v-if="error" class="alert bad">{{ error }}</p>

    <div v-if="showCreate" class="grid two">
      <form class="panel form-panel" @submit.prevent="createLicense">
        <h2>新增 License</h2>
        <div class="grid two compact">
          <label>产品 ID<input v-model.number="createForm.product_id" type="number" min="1" required /></label>
          <label>有效小时<input v-model.number="createForm.validity_hours" type="number" min="1" required /></label>
          <label>最大节点<input v-model.number="createForm.max_nodes" type="number" min="0" /></label>
          <label>最大并发<input v-model.number="createForm.max_concurrent" type="number" min="0" /></label>
        </div>
        <label>备注<input v-model="createForm.remark" /></label>
        <button class="primary-button" type="submit"><KeyRound :size="16" /> 创建</button>
      </form>

      <form class="panel form-panel" @submit.prevent="run(() => api.batchCreateLicenses({ ...batchForm, remark: batchForm.remark || null }), true)">
        <h2>批量创建</h2>
        <div class="grid two compact">
          <label>产品 ID<input v-model.number="batchForm.product_id" type="number" min="1" required /></label>
          <label>数量<input v-model.number="batchForm.count" type="number" min="1" max="1000" required /></label>
          <label>有效小时<input v-model.number="batchForm.validity_hours" type="number" min="1" required /></label>
          <label>最大节点<input v-model.number="batchForm.max_nodes" type="number" min="0" /></label>
          <label>最大并发<input v-model.number="batchForm.max_concurrent" type="number" min="0" /></label>
        </div>
        <label>备注<input v-model="batchForm.remark" /></label>
        <button class="primary-button" type="submit">批量创建</button>
      </form>
    </div>

    <section class="panel">
      <div class="panel-head">
        <h2>License 列表</h2>
        <button class="icon-button" title="刷新 License 列表" @click="loadLicenses"><RefreshCw :size="16" /></button>
      </div>
      <div class="filters">
        <label>产品 ID<input v-model.number="filters.product_id" type="number" min="1" /></label>
        <label>状态
          <select v-model.number="filters.status">
            <option :value="undefined">全部</option>
            <option :value="0">未激活</option>
            <option :value="1">已激活</option>
            <option :value="2">已过期</option>
            <option :value="3">已吊销</option>
          </select>
        </label>
        <label>License Key<input v-model="filters.license_key" placeholder="模糊搜索" /></label>
        <label>页码<input v-model.number="filters.page" type="number" min="1" /></label>
        <label>每页<input v-model.number="filters.page_size" type="number" min="1" max="200" /></label>
        <button class="primary-button" type="button" @click="loadLicenses">筛选</button>
      </div>
      <table>
        <thead>
          <tr>
            <th>ID</th>
            <th>产品</th>
            <th>License Key</th>
            <th>状态</th>
            <th>节点</th>
            <th>并发</th>
            <th>有效小时</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="license in licenses" :key="license.id" :class="{ selected: selected?.id === license.id }">
            <td>{{ license.id }}</td>
            <td>{{ license.product_id }}</td>
            <td>
              <button class="text-copy" type="button" @click="copyKey(license.license_key)">
                <Copy :size="14" /> {{ license.license_key }}
              </button>
            </td>
            <td><StatusBadge :label="licenseStatusLabel(license.status)" :tone="statusTone(license.status, 'license')" /></td>
            <td>{{ license.max_nodes }}</td>
            <td>{{ license.max_concurrent }}</td>
            <td>{{ license.validity_hours }}</td>
            <td>
              <div class="button-row wrap">
                <button class="secondary-button" type="button" @click="selectLicense(license)"><Edit3 :size="15" /> 编辑</button>
                <button class="danger-button" type="button" @click="run(() => api.revokeLicense(license.id), true)">吊销</button>
                <button class="secondary-button" type="button" @click="run(() => api.restoreLicense(license.id), true)">恢复</button>
                <button class="danger-button" type="button" @click="run(() => api.deleteLicense(license.id), true)"><Trash2 :size="15" /> 删除</button>
              </div>
            </td>
          </tr>
          <tr v-if="!licenses.length">
            <td colspan="8" class="empty-cell">{{ loading ? '加载中' : '暂无 License' }}</td>
          </tr>
        </tbody>
      </table>
    </section>

    <div v-if="selected" class="grid two">
      <form class="panel form-panel" @submit.prevent="run(() => api.updateLicense(editForm.id, { max_nodes: editForm.max_nodes, max_concurrent: editForm.max_concurrent, feature_mask: editForm.feature_mask, remark: editForm.remark || null }), true)">
        <h2>编辑 License #{{ editForm.id }}</h2>
        <div class="grid two compact">
          <label>最大节点<input v-model.number="editForm.max_nodes" type="number" min="0" /></label>
          <label>最大并发<input v-model.number="editForm.max_concurrent" type="number" min="0" /></label>
        </div>
        <label>功能掩码<input v-model="editForm.feature_mask" placeholder="control" /></label>
        <label>备注<input v-model="editForm.remark" /></label>
        <button class="primary-button" type="submit"><Save :size="16" /> 保存修改</button>
      </form>

      <form class="panel form-panel">
        <h2>续期与清理 #{{ renewForm.id }}</h2>
        <label>续期小时<input v-model.number="renewForm.extra_hours" type="number" required /></label>
        <div class="button-row wrap">
          <button class="primary-button" type="button" @click="run(() => api.renewLicense(renewForm.id, renewForm.extra_hours), true)">续期</button>
          <button class="danger-button" type="button" @click="run(() => api.cleanLicenseBindings(renewForm.id), true)">清理绑定</button>
          <button class="danger-button" type="button" @click="run(() => api.cleanInvalidLicenses(), true)">清理无效 License</button>
        </div>
      </form>
    </div>

    <ResultPanel :value="result" />
  </section>
</template>
