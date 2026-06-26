<script setup lang="ts">
import { reactive, ref } from 'vue';
import { Plus, Search, Save, Trash2, UploadCloud } from 'lucide-vue-next';
import { api } from '../api/client';
import type { ProductData } from '../api/types';
import ResultPanel from '../components/ResultPanel.vue';

const result = ref<unknown>({});
const error = ref('');
const current = ref<ProductData | null>(null);

const createForm = reactive({ name: '', description: '' });
const queryForm = reactive({ id: 1 });
const updateForm = reactive({ id: 1, name: '', description: '' });
const versionForm = reactive({ product_id: 1, version_code: '1.0.0', description: '', release_method: 0, release_date: '' });
const versionAction = reactive({ product_id: 1, version_id: 1, release_date: '' });

async function run(action: () => Promise<unknown>) {
  error.value = '';
  try {
    result.value = await action();
  } catch (err) {
    error.value = err instanceof Error ? err.message : '操作失败';
  }
}

function loadProduct() {
  run(async () => {
    const data = await api.getProduct(Number(queryForm.id));
    current.value = data;
    updateForm.id = data.id;
    updateForm.name = data.name;
    updateForm.description = data.description || '';
    return data;
  });
}
</script>

<template>
  <section class="page">
    <div class="page-head">
      <div>
        <h1>产品管理</h1>
        <p>创建产品、维护版本发布和最低支持版本。</p>
      </div>
    </div>

    <p v-if="error" class="alert bad">{{ error }}</p>

    <div class="grid three">
      <form class="panel form-panel" @submit.prevent="run(() => api.createProduct({ name: createForm.name, description: createForm.description || null }))">
        <h2>创建产品</h2>
        <label>名称<input v-model="createForm.name" required /></label>
        <label>描述<textarea v-model="createForm.description" rows="4" /></label>
        <button class="primary-button" type="submit"><Plus :size="16" /> 创建</button>
      </form>

      <form class="panel form-panel" @submit.prevent="loadProduct">
        <h2>查询产品</h2>
        <label>产品 ID<input v-model.number="queryForm.id" type="number" min="1" required /></label>
        <button class="primary-button" type="submit"><Search :size="16" /> 查询</button>
        <div v-if="current" class="detail-list">
          <span>ID</span><strong>{{ current.id }}</strong>
          <span>名称</span><strong>{{ current.name }}</strong>
          <span>描述</span><strong>{{ current.description || '-' }}</strong>
        </div>
      </form>

      <form class="panel form-panel" @submit.prevent="run(() => api.updateProduct(updateForm.id, { name: updateForm.name || null, description: updateForm.description || null }))">
        <h2>编辑产品</h2>
        <label>产品 ID<input v-model.number="updateForm.id" type="number" min="1" required /></label>
        <label>名称<input v-model="updateForm.name" /></label>
        <label>描述<textarea v-model="updateForm.description" rows="3" /></label>
        <div class="button-row">
          <button class="primary-button" type="submit"><Save :size="16" /> 保存</button>
          <button class="danger-button" type="button" @click="run(() => api.deleteProduct(updateForm.id))"><Trash2 :size="16" /> 删除</button>
        </div>
      </form>
    </div>

    <div class="grid two">
      <form class="panel form-panel" @submit.prevent="run(() => api.createProductVersion({
        product_id: versionForm.product_id,
        version_code: versionForm.version_code,
        description: versionForm.description || null,
        release_method: versionForm.release_method,
        release_date: versionForm.release_date || null
      }))">
        <h2>创建版本</h2>
        <label>产品 ID<input v-model.number="versionForm.product_id" type="number" min="1" required /></label>
        <label>版本号<input v-model="versionForm.version_code" required /></label>
        <label>发布方式
          <select v-model.number="versionForm.release_method">
            <option :value="0">立即发布</option>
            <option :value="1">定时发布</option>
            <option :value="2">暂不发布</option>
          </select>
        </label>
        <label>发布时间<input v-model="versionForm.release_date" type="datetime-local" /></label>
        <label>描述<textarea v-model="versionForm.description" rows="3" /></label>
        <button class="primary-button" type="submit"><UploadCloud :size="16" /> 创建版本</button>
      </form>

      <form class="panel form-panel">
        <h2>版本操作</h2>
        <label>产品 ID<input v-model.number="versionAction.product_id" type="number" min="1" required /></label>
        <label>版本 ID<input v-model.number="versionAction.version_id" type="number" min="1" required /></label>
        <label>定时发布时间<input v-model="versionAction.release_date" type="datetime-local" /></label>
        <div class="button-row wrap">
          <button class="primary-button" type="button" @click="run(() => api.releaseVersion({ product_id: versionAction.product_id, version_id: versionAction.version_id, release_date: versionAction.release_date || null }))">发布</button>
          <button class="secondary-button" type="button" @click="run(() => api.setMinVersion({ product_id: versionAction.product_id, version_id: versionAction.version_id }))">设为最低版本</button>
          <button class="danger-button" type="button" @click="run(() => api.deprecateVersion({ product_id: versionAction.product_id, version_id: versionAction.version_id }))">废弃版本</button>
        </div>
      </form>
    </div>

    <ResultPanel :value="result" />
  </section>
</template>
