<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue';
import { Edit3, Plus, RefreshCw, Save, Trash2, UploadCloud } from 'lucide-vue-next';
import { api } from '../api/client';
import type { ProductData } from '../api/types';
import ResultPanel from '../components/ResultPanel.vue';

const loading = ref(false);
const error = ref('');
const result = ref<unknown>({});
const products = ref<ProductData[]>([]);
const selected = ref<ProductData | null>(null);
const showCreate = ref(false);

const filters = reactive({ name: '', status: undefined as number | undefined, page: 1, page_size: 20 });
const createForm = reactive({ name: '', description: '' });
const editForm = reactive({ id: 0, name: '', description: '' });
const versionForm = reactive({ product_id: 0, version_code: '1.0.0', description: '', release_method: 0, release_date: '' });
const versionAction = reactive({ product_id: 0, version_id: 0, release_date: '' });

function toRfc3339(value: string) {
  return value ? new Date(value).toISOString() : null;
}

async function run(action: () => Promise<unknown>, refresh = false) {
  error.value = '';
  try {
    result.value = await action();
    if (refresh) await loadProducts();
  } catch (err) {
    error.value = err instanceof Error ? err.message : '操作失败';
  }
}

async function loadProducts() {
  loading.value = true;
  error.value = '';
  try {
    products.value = await api.listProducts(filters);
  } catch (err) {
    error.value = err instanceof Error ? err.message : '加载失败';
  } finally {
    loading.value = false;
  }
}

function selectProduct(product: ProductData) {
  selected.value = product;
  editForm.id = product.id;
  editForm.name = product.name;
  editForm.description = product.description || '';
  versionForm.product_id = product.id;
  versionAction.product_id = product.id;
}

async function createProduct() {
  await run(() => api.createProduct({ name: createForm.name, description: createForm.description || null }), true);
  createForm.name = '';
  createForm.description = '';
  showCreate.value = false;
}

onMounted(loadProducts);
</script>

<template>
  <section class="page">
    <div class="page-head">
      <div>
        <h1>产品管理</h1>
        <p>以列表为中心维护产品、版本发布和最低支持版本。</p>
      </div>
      <button class="primary-button" type="button" @click="showCreate = !showCreate">
        <Plus :size="16" />
        新增产品
      </button>
    </div>

    <p v-if="error" class="alert bad">{{ error }}</p>

    <section v-if="showCreate" class="panel form-panel drawer-panel">
      <h2>新增产品</h2>
      <div class="grid two compact">
        <label>名称<input v-model="createForm.name" required /></label>
        <label>描述<input v-model="createForm.description" /></label>
      </div>
      <div class="button-row">
        <button class="primary-button" type="button" @click="createProduct"><Save :size="16" /> 保存</button>
        <button class="secondary-button" type="button" @click="showCreate = false">取消</button>
      </div>
    </section>

    <section class="panel">
      <div class="panel-head">
        <h2>产品列表</h2>
        <button class="icon-button" title="刷新产品列表" @click="loadProducts"><RefreshCw :size="16" /></button>
      </div>
      <div class="filters">
        <label>名称<input v-model="filters.name" placeholder="模糊搜索" /></label>
        <label>状态
          <select v-model.number="filters.status">
            <option :value="undefined">全部</option>
            <option :value="1">启用</option>
            <option :value="2">禁用</option>
            <option :value="3">废弃</option>
          </select>
        </label>
        <label>页码<input v-model.number="filters.page" type="number" min="1" /></label>
        <label>每页<input v-model.number="filters.page_size" type="number" min="1" max="200" /></label>
        <button class="primary-button" type="button" @click="loadProducts">筛选</button>
      </div>
      <table>
        <thead>
          <tr>
            <th>ID</th>
            <th>名称</th>
            <th>描述</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="product in products" :key="product.id" :class="{ selected: selected?.id === product.id }">
            <td>{{ product.id }}</td>
            <td>{{ product.name }}</td>
            <td>{{ product.description || '-' }}</td>
            <td>
              <div class="button-row wrap">
                <button class="secondary-button" type="button" @click="selectProduct(product)"><Edit3 :size="15" /> 编辑</button>
                <button class="danger-button" type="button" @click="run(() => api.deleteProduct(product.id), true)"><Trash2 :size="15" /> 删除</button>
              </div>
            </td>
          </tr>
          <tr v-if="!products.length">
            <td colspan="4" class="empty-cell">{{ loading ? '加载中' : '暂无产品' }}</td>
          </tr>
        </tbody>
      </table>
    </section>

    <div v-if="selected" class="grid two">
      <form class="panel form-panel" @submit.prevent="run(() => api.updateProduct(editForm.id, { name: editForm.name || null, description: editForm.description || null }), true)">
        <h2>编辑产品 #{{ editForm.id }}</h2>
        <label>名称<input v-model="editForm.name" /></label>
        <label>描述<textarea v-model="editForm.description" rows="3" /></label>
        <button class="primary-button" type="submit"><Save :size="16" /> 保存修改</button>
      </form>

      <form class="panel form-panel" @submit.prevent="run(() => api.createProductVersion({
        product_id: versionForm.product_id,
        version_code: versionForm.version_code,
        description: versionForm.description || null,
        release_method: versionForm.release_method,
        release_date: toRfc3339(versionForm.release_date)
      }))">
        <h2>为该产品新增版本</h2>
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
    </div>

    <section v-if="selected" class="panel form-panel">
      <h2>版本操作</h2>
      <div class="grid three compact">
        <label>产品 ID<input v-model.number="versionAction.product_id" type="number" min="1" /></label>
        <label>版本 ID<input v-model.number="versionAction.version_id" type="number" min="1" /></label>
        <label>定时发布时间<input v-model="versionAction.release_date" type="datetime-local" /></label>
      </div>
      <div class="button-row wrap">
        <button class="primary-button" type="button" @click="run(() => api.releaseVersion({ product_id: versionAction.product_id, version_id: versionAction.version_id, release_date: toRfc3339(versionAction.release_date) }))">发布</button>
        <button class="secondary-button" type="button" @click="run(() => api.setMinVersion({ product_id: versionAction.product_id, version_id: versionAction.version_id }))">设为最低版本</button>
        <button class="danger-button" type="button" @click="run(() => api.deprecateVersion({ product_id: versionAction.product_id, version_id: versionAction.version_id }))">废弃版本</button>
      </div>
    </section>

    <ResultPanel :value="result" />
  </section>
</template>
