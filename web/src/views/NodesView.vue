<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue';
import { Ban, Edit3, Link2, Plus, RefreshCw, Save, ShieldCheck, Trash2 } from 'lucide-vue-next';
import { api } from '../api/client';
import type { NodeData } from '../api/types';
import JsonEditor from '../components/JsonEditor.vue';
import ResultPanel from '../components/ResultPanel.vue';
import StatusBadge from '../components/StatusBadge.vue';
import { nodeStatusLabel, prettyJson, statusTone } from '../utils/status';

const loading = ref(false);
const error = ref('');
const result = ref<unknown>({});
const nodes = ref<NodeData[]>([]);
const selected = ref<NodeData | null>(null);
const showCreate = ref(false);

const filters = reactive({ device_code: '', status: undefined as number | undefined, page: 1, page_size: 20 });
const createForm = reactive({ device_code: 'demo-node-001', metadata: '{\n  "os": "windows"\n}' });
const editForm = reactive({ id: 0, device_code: '', metadata: '{}' });
const bindingForm = reactive({ node_id: 0, license_id: 1 });
const banForm = reactive({ node_id: 0, reason: '' });

async function run(action: () => Promise<unknown>, refresh = false) {
  error.value = '';
  try {
    result.value = await action();
    if (refresh) await loadNodes();
  } catch (err) {
    error.value = err instanceof Error ? err.message : '操作失败';
  }
}

async function loadNodes() {
  loading.value = true;
  error.value = '';
  try {
    nodes.value = await api.listNodes(filters);
  } catch (err) {
    error.value = err instanceof Error ? err.message : '加载失败';
  } finally {
    loading.value = false;
  }
}

function selectNode(node: NodeData) {
  selected.value = node;
  editForm.id = node.id;
  editForm.device_code = node.device_code;
  editForm.metadata = prettyJson(node.metadata || '{}');
  bindingForm.node_id = node.id;
  banForm.node_id = node.id;
}

async function createNode() {
  await run(() => api.createNode({ device_code: createForm.device_code, metadata: createForm.metadata }), true);
  showCreate.value = false;
}

onMounted(loadNodes);
</script>

<template>
  <section class="page">
    <div class="page-head">
      <div>
        <h1>节点管理</h1>
        <p>围绕节点列表维护元数据、绑定关系和封禁状态。</p>
      </div>
      <button class="primary-button" type="button" @click="showCreate = !showCreate">
        <Plus :size="16" />
        新增节点
      </button>
    </div>

    <p v-if="error" class="alert bad">{{ error }}</p>

    <section v-if="showCreate" class="panel form-panel drawer-panel">
      <h2>新增节点</h2>
      <div class="grid two compact">
        <label>设备码<input v-model="createForm.device_code" required /></label>
        <JsonEditor v-model="createForm.metadata" :rows="5" />
      </div>
      <div class="button-row">
        <button class="primary-button" type="button" @click="createNode"><Save :size="16" /> 保存</button>
        <button class="secondary-button" type="button" @click="showCreate = false">取消</button>
      </div>
    </section>

    <section class="panel">
      <div class="panel-head">
        <h2>节点列表</h2>
        <button class="icon-button" title="刷新节点列表" @click="loadNodes"><RefreshCw :size="16" /></button>
      </div>
      <div class="filters">
        <label>设备码<input v-model="filters.device_code" placeholder="模糊搜索" /></label>
        <label>状态
          <select v-model.number="filters.status">
            <option :value="undefined">全部</option>
            <option :value="0">正常</option>
            <option :value="1">离线</option>
            <option :value="2">封禁</option>
          </select>
        </label>
        <label>页码<input v-model.number="filters.page" type="number" min="1" /></label>
        <label>每页<input v-model.number="filters.page_size" type="number" min="1" max="200" /></label>
        <button class="primary-button" type="button" @click="loadNodes">筛选</button>
      </div>

      <table>
        <thead>
          <tr>
            <th>ID</th>
            <th>设备码</th>
            <th>状态</th>
            <th>元数据</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="node in nodes" :key="node.id" :class="{ selected: selected?.id === node.id }">
            <td>{{ node.id }}</td>
            <td>{{ node.device_code }}</td>
            <td><StatusBadge :label="nodeStatusLabel(node.status)" :tone="statusTone(node.status, 'node')" /></td>
            <td><code>{{ node.metadata || '{}' }}</code></td>
            <td>
              <div class="button-row wrap">
                <button class="secondary-button" type="button" @click="selectNode(node)"><Edit3 :size="15" /> 编辑</button>
                <button class="danger-button" type="button" @click="run(() => api.banNode(node.id, banForm.reason || null), true)"><Ban :size="15" /> 封禁</button>
                <button class="primary-button" type="button" @click="run(() => api.unbanNode(node.id), true)"><ShieldCheck :size="15" /> 解封</button>
                <button class="danger-button" type="button" @click="run(() => api.deleteNode(node.id), true)"><Trash2 :size="15" /> 删除</button>
              </div>
            </td>
          </tr>
          <tr v-if="!nodes.length">
            <td colspan="5" class="empty-cell">{{ loading ? '加载中' : '暂无节点' }}</td>
          </tr>
        </tbody>
      </table>
    </section>

    <div v-if="selected" class="grid two">
      <form class="panel form-panel" @submit.prevent="run(() => api.updateNode(editForm.id, { device_code: editForm.device_code || null, metadata: editForm.metadata }), true)">
        <h2>编辑节点 #{{ editForm.id }}</h2>
        <label>设备码<input v-model="editForm.device_code" /></label>
        <JsonEditor v-model="editForm.metadata" :rows="7" />
        <button class="primary-button" type="submit"><Save :size="16" /> 保存修改</button>
      </form>

      <form class="panel form-panel">
        <h2>绑定与状态 #{{ bindingForm.node_id }}</h2>
        <label>License ID<input v-model.number="bindingForm.license_id" type="number" min="1" /></label>
        <label>封禁原因<input v-model="banForm.reason" /></label>
        <div class="button-row wrap">
          <button class="primary-button" type="button" @click="run(() => api.bindNode(bindingForm.node_id, bindingForm.license_id), true)"><Link2 :size="16" /> 绑定</button>
          <button class="danger-button" type="button" @click="run(() => api.unbindNode(bindingForm.node_id, bindingForm.license_id), true)">解绑</button>
          <button class="danger-button" type="button" @click="run(() => api.banNode(banForm.node_id, banForm.reason || null), true)"><Ban :size="16" /> 封禁</button>
          <button class="primary-button" type="button" @click="run(() => api.unbanNode(banForm.node_id), true)"><ShieldCheck :size="16" /> 解封</button>
          <button class="danger-button" type="button" @click="run(() => api.cleanUnboundNodes(), true)">清理无绑定节点</button>
        </div>
      </form>
    </div>

    <ResultPanel :value="result" />
  </section>
</template>
