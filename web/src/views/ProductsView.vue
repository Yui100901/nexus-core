<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import {
  CalendarClock,
  CheckCircle2,
  ChevronDown,
  ChevronRight,
  Edit3,
  Plus,
  RefreshCw,
  Save,
  ShieldCheck,
  Trash2,
  UploadCloud,
  X,
} from 'lucide-vue-next';
import { api } from '../api/client';
import type { ProductData, ProductVersionData } from '../api/types';
import ConfirmDialog from '../components/ConfirmDialog.vue';
import StatusBadge from '../components/StatusBadge.vue';
import { errorMessage, notifyError, notifySuccess } from '../composables/useToast';
import { formatDate, statusTone } from '../utils/status';

type ProductDialogMode = 'create' | 'edit';

const loading = ref(false);
const error = ref('');
const products = ref<ProductData[]>([]);
const expandedProductId = ref<number | null>(null);
const productDialogMode = ref<ProductDialogMode | null>(null);
const versionDialogOpen = ref(false);
const releaseDialogOpen = ref(false);

const filters = reactive({ name: '', status: undefined as number | undefined, page: 1, page_size: 20 });
const productForm = reactive({ id: 0, name: '', description: '' });
const versionForm = reactive({ product_id: 0, version_code: '1.0.0', description: '', release_method: 0, release_date: '' });
const releaseForm = reactive({ product_id: 0, version_id: 0, release_date: '' });
const confirmDialog = reactive({
  open: false,
  title: '',
  message: '',
  confirmLabel: '确认执行',
  action: null as null | (() => Promise<unknown>),
});

const activeProduct = computed(() => products.value.find((item) => item.id === expandedProductId.value) || null);

function toRfc3339(value: string) {
  return value ? new Date(value).toISOString() : null;
}

function versionStatusLabel(status: number) {
  return ({ 0: '未发布', 1: '可用', 2: '已废弃' } as Record<number, string>)[status] || `未知 ${status}`;
}

function versionTone(status: number) {
  return status === 1 ? 'good' : status === 0 ? 'idle' : 'bad';
}

function canRelease(version: ProductVersionData) {
  return version.status === 0;
}

function canSetMinVersion(version: ProductVersionData) {
  return version.status === 1;
}

function resetProductForm() {
  productForm.id = 0;
  productForm.name = '';
  productForm.description = '';
}

function openCreateProduct() {
  resetProductForm();
  productDialogMode.value = 'create';
}

function openEditProduct(product: ProductData) {
  productForm.id = product.id;
  productForm.name = product.name;
  productForm.description = product.description || '';
  productDialogMode.value = 'edit';
}

function closeProductDialog() {
  productDialogMode.value = null;
  resetProductForm();
}

function openCreateVersion(product: ProductData) {
  expandedProductId.value = product.id;
  versionForm.product_id = product.id;
  versionForm.version_code = '1.0.0';
  versionForm.description = '';
  versionForm.release_method = 0;
  versionForm.release_date = '';
  versionDialogOpen.value = true;
}

function closeVersionDialog() {
  versionDialogOpen.value = false;
}

function openScheduleRelease(product: ProductData, version: ProductVersionData) {
  releaseForm.product_id = product.id;
  releaseForm.version_id = version.id;
  releaseForm.release_date = version.release_date ? version.release_date.slice(0, 16) : '';
  releaseDialogOpen.value = true;
}

function closeReleaseDialog() {
  releaseDialogOpen.value = false;
}

function toggleProduct(product: ProductData) {
  expandedProductId.value = expandedProductId.value === product.id ? null : product.id;
}

async function run(action: () => Promise<unknown>, refresh = false) {
  error.value = '';
  try {
    await action();
    if (refresh) await loadProducts();
    notifySuccess();
    return true;
  } catch (err) {
    notifyError(errorMessage(err));
    return false;
  }
}

function askConfirm(title: string, message: string, confirmLabel: string, action: () => Promise<unknown>) {
  confirmDialog.title = title;
  confirmDialog.message = message;
  confirmDialog.confirmLabel = confirmLabel;
  confirmDialog.action = action;
  confirmDialog.open = true;
}

function closeConfirm() {
  confirmDialog.open = false;
  confirmDialog.action = null;
}

async function confirmDangerAction() {
  const action = confirmDialog.action;
  closeConfirm();
  if (action) {
    await run(action, true);
  }
}

async function loadProducts() {
  loading.value = true;
  error.value = '';
  try {
    products.value = await api.listProducts(filters);
    if (expandedProductId.value && !products.value.some((item) => item.id === expandedProductId.value)) {
      expandedProductId.value = null;
    }
  } catch (err) {
    error.value = err instanceof Error ? err.message : '加载失败';
  } finally {
    loading.value = false;
  }
}

async function submitProduct() {
  const ok = productDialogMode.value === 'create'
    ? await run(() => api.createProduct({ name: productForm.name, description: productForm.description || null }), true)
    : await run(() => api.updateProduct(productForm.id, { name: productForm.name || null, description: productForm.description || null }), true);
  if (ok) closeProductDialog();
}

async function submitVersion() {
  const ok = await run(() => api.createProductVersion({
    product_id: versionForm.product_id,
    version_code: versionForm.version_code,
    description: versionForm.description || null,
    release_method: versionForm.release_method,
    release_date: toRfc3339(versionForm.release_date),
  }), true);
  if (ok) closeVersionDialog();
}

async function releaseNow(product: ProductData, version: ProductVersionData) {
  await run(() => api.releaseVersion({ product_id: product.id, version_id: version.id, release_date: null }), true);
}

async function submitScheduledRelease() {
  const ok = await run(() => api.releaseVersion({
    product_id: releaseForm.product_id,
    version_id: releaseForm.version_id,
    release_date: toRfc3339(releaseForm.release_date),
  }), true);
  if (ok) closeReleaseDialog();
}

async function setMinVersion(product: ProductData, version: ProductVersionData) {
  await run(() => api.setMinVersion({ product_id: product.id, version_id: version.id }), true);
}

function confirmDeleteProduct(product: ProductData) {
  askConfirm(
    '删除产品',
    `确认删除产品「${product.name}」？该操作会同时删除该产品下的版本，且不可撤销。`,
    '确认删除',
    () => api.deleteProduct(product.id),
  );
}

function confirmDeprecateVersion(product: ProductData, version: ProductVersionData) {
  askConfirm(
    '废弃版本',
    `确认废弃产品「${product.name}」的版本 ${version.version_code}？废弃后客户端不能再用该版本注册或心跳。`,
    '确认废弃',
    () => api.deprecateVersion({ product_id: product.id, version_id: version.id }),
  );
}

function confirmDeleteVersion(product: ProductData, version: ProductVersionData) {
  askConfirm(
    '删除版本',
    `确认删除产品「${product.name}」的版本 ${version.version_code}？删除后该版本不会再出现在版本列表中。`,
    '确认删除',
    () => api.deleteProductVersion({ product_id: product.id, version_id: version.id }),
  );
}

onMounted(loadProducts);
</script>

<template>
  <section class="page">
    <div class="page-head">
      <div>
        <h1>产品管理</h1>
        <p>管理产品基础信息、版本发布和最低支持版本。</p>
      </div>
      <button class="primary-button" type="button" @click="openCreateProduct">
        <Plus :size="16" />
        新增产品
      </button>
    </div>

    <p v-if="error" class="alert bad">{{ error }}</p>

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
            <th class="cell-tight"></th>
            <th>ID</th>
            <th>名称</th>
            <th>状态</th>
            <th>版本</th>
            <th>描述</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <template v-for="product in products" :key="product.id">
            <tr :class="{ selected: expandedProductId === product.id }">
              <td class="cell-tight">
                <button class="icon-button" type="button" :title="expandedProductId === product.id ? '收起版本' : '展开版本'" @click="toggleProduct(product)">
                  <ChevronDown v-if="expandedProductId === product.id" :size="16" />
                  <ChevronRight v-else :size="16" />
                </button>
              </td>
              <td>{{ product.id }}</td>
              <td><strong>{{ product.name }}</strong></td>
              <td><StatusBadge :label="product.status === 1 ? '启用' : product.status === 2 ? '禁用' : '废弃'" :tone="statusTone(product.status, 'enabled')" /></td>
              <td>{{ product.versions?.length || 0 }}</td>
              <td>{{ product.description || '-' }}</td>
              <td>
                <div class="button-row wrap">
                  <button class="secondary-button" type="button" @click="openEditProduct(product)"><Edit3 :size="15" /> 编辑</button>
                  <button class="primary-button" type="button" @click="openCreateVersion(product)"><UploadCloud :size="15" /> 新增版本</button>
                  <button class="danger-button" type="button" @click="confirmDeleteProduct(product)"><Trash2 :size="15" /> 删除</button>
                </div>
              </td>
            </tr>
            <tr v-if="expandedProductId === product.id" class="nested-row">
              <td colspan="7">
                <div class="nested-head">
                  <strong>{{ product.name }} 的版本</strong>
                  <button class="primary-button" type="button" @click="openCreateVersion(product)"><Plus :size="15" /> 新增版本</button>
                </div>
                <table class="nested-table">
                  <thead>
                    <tr>
                      <th>ID</th>
                      <th>版本号</th>
                      <th>状态</th>
                      <th>发布时间</th>
                      <th>最低支持</th>
                      <th>描述</th>
                      <th>操作</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr v-for="version in product.versions || []" :key="version.id">
                      <td>{{ version.id }}</td>
                      <td><code>{{ version.version_code }}</code></td>
                      <td><StatusBadge :label="versionStatusLabel(version.status)" :tone="versionTone(version.status)" /></td>
                      <td>{{ formatDate(version.release_date) }}</td>
                      <td>
                        <StatusBadge
                          v-if="product.min_supported_version_id === version.id"
                          label="是"
                          tone="good"
                        />
                        <span v-else>-</span>
                      </td>
                      <td>{{ version.description || '-' }}</td>
                      <td>
                        <div class="button-row wrap">
                          <button class="primary-button" type="button" :disabled="!canRelease(version)" @click="releaseNow(product, version)"><CheckCircle2 :size="15" /> 发布</button>
                          <button class="secondary-button" type="button" :disabled="!canRelease(version)" @click="openScheduleRelease(product, version)"><CalendarClock :size="15" /> 定时</button>
                          <button class="secondary-button" type="button" :disabled="!canSetMinVersion(version)" @click="setMinVersion(product, version)"><ShieldCheck :size="15" /> 最低</button>
                          <button class="danger-button" type="button" :disabled="version.status === 2" @click="confirmDeprecateVersion(product, version)"><Trash2 :size="15" /> 废弃</button>
                          <button class="danger-button" type="button" @click="confirmDeleteVersion(product, version)"><Trash2 :size="15" /> 删除</button>
                        </div>
                      </td>
                    </tr>
                    <tr v-if="!(product.versions || []).length">
                      <td colspan="7" class="empty-cell">暂无版本</td>
                    </tr>
                  </tbody>
                </table>
              </td>
            </tr>
          </template>
          <tr v-if="!products.length">
            <td colspan="7" class="empty-cell">{{ loading ? '加载中' : '暂无产品' }}</td>
          </tr>
        </tbody>
      </table>
    </section>

    <div v-if="productDialogMode" class="modal-backdrop" @click.self="closeProductDialog">
      <form class="modal-panel form-panel" @submit.prevent="submitProduct">
        <div class="modal-head">
          <h2>{{ productDialogMode === 'create' ? '新增产品' : `编辑产品 #${productForm.id}` }}</h2>
          <button class="icon-button" type="button" title="关闭" @click="closeProductDialog"><X :size="16" /></button>
        </div>
        <label>名称<input v-model="productForm.name" required /></label>
        <label>描述<textarea v-model="productForm.description" rows="4" /></label>
        <div class="button-row">
          <button class="primary-button" type="submit"><Save :size="16" /> 保存</button>
          <button class="secondary-button" type="button" @click="closeProductDialog">取消</button>
        </div>
      </form>
    </div>

    <div v-if="versionDialogOpen" class="modal-backdrop" @click.self="closeVersionDialog">
      <form class="modal-panel form-panel" @submit.prevent="submitVersion">
        <div class="modal-head">
          <h2>新增版本 · {{ activeProduct?.name || `产品 #${versionForm.product_id}` }}</h2>
          <button class="icon-button" type="button" title="关闭" @click="closeVersionDialog"><X :size="16" /></button>
        </div>
        <label>版本号<input v-model="versionForm.version_code" required /></label>
        <label>发布方式
          <select v-model.number="versionForm.release_method">
            <option :value="0">立即发布</option>
            <option :value="1">定时发布</option>
            <option :value="2">暂不发布</option>
          </select>
        </label>
        <label v-if="versionForm.release_method === 1">发布时间<input v-model="versionForm.release_date" type="datetime-local" required /></label>
        <label>描述<textarea v-model="versionForm.description" rows="3" /></label>
        <div class="button-row">
          <button class="primary-button" type="submit"><UploadCloud :size="16" /> 创建版本</button>
          <button class="secondary-button" type="button" @click="closeVersionDialog">取消</button>
        </div>
      </form>
    </div>

    <div v-if="releaseDialogOpen" class="modal-backdrop" @click.self="closeReleaseDialog">
      <form class="modal-panel form-panel" @submit.prevent="submitScheduledRelease">
        <div class="modal-head">
          <h2>定时发布版本 #{{ releaseForm.version_id }}</h2>
          <button class="icon-button" type="button" title="关闭" @click="closeReleaseDialog"><X :size="16" /></button>
        </div>
        <label>发布时间<input v-model="releaseForm.release_date" type="datetime-local" required /></label>
        <div class="button-row">
          <button class="primary-button" type="submit"><CalendarClock :size="16" /> 保存</button>
          <button class="secondary-button" type="button" @click="closeReleaseDialog">取消</button>
        </div>
      </form>
    </div>

    <ConfirmDialog
      :open="confirmDialog.open"
      :title="confirmDialog.title"
      :message="confirmDialog.message"
      :confirm-label="confirmDialog.confirmLabel"
      @confirm="confirmDangerAction"
      @cancel="closeConfirm"
    />
  </section>
</template>
