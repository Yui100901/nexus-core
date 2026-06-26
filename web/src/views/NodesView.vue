<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue';
import { Ban, Edit3, Link2, Plus, PowerOff, RefreshCw, RotateCcw, Save, ShieldCheck, Trash2, X } from 'lucide-vue-next';
import { api } from '../api/client';
import type { NodeData } from '../api/types';
import ConfirmDialog from '../components/ConfirmDialog.vue';
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
const nodeDialogOpen = ref(false);

const filters = reactive({ device_code: '', status: undefined as number | undefined, page: 1, page_size: 20 });
const createForm = reactive({ device_code: 'demo-node-001', metadata: '{\n  "os": "windows"\n}' });
const editForm = reactive({ id: 0, device_code: '', metadata: '{}' });
const bindingForm = reactive({ node_id: 0, license_id: 1 });
const banForm = reactive({ node_id: 0, reason: '' });
const forceOfflineForm = reactive({ node_id: 0, reason: '' });
const confirmDialog = reactive({
  open: false,
  title: '',
  message: '',
  confirmLabel: '确认执行',
  action: null as null | (() => Promise<unknown>),
});

async function run(action: () => Promise<unknown>, refresh = false) {
  error.value = '';
  try {
    result.value = await action();
    if (refresh) await loadNodes();
  } catch (err) {
    error.value = err instanceof Error ? err.message : '操作失败';
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
  if (action) await run(action, true);
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

function fillNodeForms(node: NodeData) {
  selected.value = node;
  editForm.id = node.id;
  editForm.device_code = node.device_code;
  editForm.metadata = prettyJson(node.metadata || '{}');
  bindingForm.node_id = node.id;
  banForm.node_id = node.id;
  forceOfflineForm.node_id = node.id;
}

function openNodeDialog(node: NodeData) {
  fillNodeForms(node);
  nodeDialogOpen.value = true;
}

function closeNodeDialog() {
  nodeDialogOpen.value = false;
}

async function createNode() {
  await run(() => api.createNode({ device_code: createForm.device_code, metadata: createForm.metadata }), true);
  showCreate.value = false;
}

async function saveNode() {
  await run(() => api.updateNode(editForm.id, { device_code: editForm.device_code || null, metadata: editForm.metadata }), true);
  closeNodeDialog();
}

function confirmBanNode(nodeID: number, deviceCode?: string, reason?: string | null) {
  askConfirm('封禁节点', `确认封禁节点${deviceCode ? `「${deviceCode}」` : ` #${nodeID}`}？封禁后该节点无法注册、心跳或执行控制命令。`, '确认封禁', () => api.banNode(nodeID, reason || null));
}

function confirmForceOfflineNode(nodeID: number, deviceCode?: string, reason?: string | null) {
  askConfirm('强制下线节点', `确认强制下线节点${deviceCode ? `「${deviceCode}」` : ` #${nodeID}`}？下线后心跳会被拒绝，节点重新注册后才允许上线。`, '确认下线', () => api.forceOfflineNode(nodeID, reason || null));
}

function confirmDeleteNode(node: NodeData) {
  askConfirm('删除节点', `确认删除节点「${node.device_code}」？该操作会删除节点及其绑定关系。`, '确认删除', () => api.deleteNode(node.id));
}

function confirmUnbindNode(nodeID: number, licenseID: number) {
  askConfirm('解绑节点', `确认解除节点 #${nodeID} 与 License #${licenseID} 的绑定？`, '确认解绑', () => api.unbindNode(nodeID, licenseID));
}

function confirmCleanUnboundNodes() {
  askConfirm('清理无绑定节点', '确认删除所有无绑定关系的节点？该操作不可撤销。', '确认清理', () => api.cleanUnboundNodes());
}

onMounted(loadNodes);
</script>

<template>
  <section class="page">
    <div class="page-head">
      <div>
        <h1>节点管理</h1>
        <p>围绕节点列表维护元数据、绑定关系、封禁状态和强制下线状态。</p>
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
            <option :value="3">强制下线</option>
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
          <tr v-for="node in nodes" :key="node.id">
            <td>{{ node.id }}</td>
            <td>{{ node.device_code }}</td>
            <td><StatusBadge :label="nodeStatusLabel(node.status)" :tone="statusTone(node.status, 'node')" /></td>
            <td><code>{{ node.metadata || '{}' }}</code></td>
            <td>
              <div class="button-row wrap">
                <button class="secondary-button" type="button" @click="openNodeDialog(node)"><Edit3 :size="15" /> 编辑</button>
                <button class="danger-button" type="button" @click="confirmBanNode(node.id, node.device_code, null)"><Ban :size="15" /> 封禁</button>
                <button class="primary-button" type="button" @click="run(() => api.unbanNode(node.id), true)"><ShieldCheck :size="15" /> 解封</button>
                <button class="danger-button" type="button" @click="confirmForceOfflineNode(node.id, node.device_code, null)"><PowerOff :size="15" /> 强制下线</button>
                <button class="primary-button" type="button" @click="run(() => api.restoreOnlineNode(node.id), true)"><RotateCcw :size="15" /> 恢复</button>
                <button class="danger-button" type="button" @click="confirmDeleteNode(node)"><Trash2 :size="15" /> 删除</button>
              </div>
            </td>
          </tr>
          <tr v-if="!nodes.length">
            <td colspan="5" class="empty-cell">{{ loading ? '加载中' : '暂无节点' }}</td>
          </tr>
        </tbody>
      </table>
    </section>

    <div v-if="nodeDialogOpen" class="modal-backdrop" @click.self="closeNodeDialog">
      <form class="modal-panel form-panel" @submit.prevent="saveNode">
        <div class="modal-head">
          <h2>编辑节点 #{{ editForm.id }}</h2>
          <button class="icon-button" type="button" title="关闭" @click="closeNodeDialog"><X :size="16" /></button>
        </div>
        <label>设备码<input v-model="editForm.device_code" /></label>
        <JsonEditor v-model="editForm.metadata" :rows="7" />

        <div class="section-divider"></div>
        <h2>绑定与状态</h2>
        <label>License ID<input v-model.number="bindingForm.license_id" type="number" min="1" /></label>
        <label>封禁原因<input v-model="banForm.reason" /></label>
        <label>强制下线原因<input v-model="forceOfflineForm.reason" /></label>
        <div class="button-row wrap">
          <button class="primary-button" type="submit"><Save :size="16" /> 保存修改</button>
          <button class="primary-button" type="button" @click="run(() => api.bindNode(bindingForm.node_id, bindingForm.license_id), true)"><Link2 :size="16" /> 绑定</button>
          <button class="danger-button" type="button" @click="confirmUnbindNode(bindingForm.node_id, bindingForm.license_id)">解绑</button>
          <button class="danger-button" type="button" @click="confirmBanNode(banForm.node_id, selected?.device_code, banForm.reason || null)"><Ban :size="16" /> 封禁</button>
          <button class="primary-button" type="button" @click="run(() => api.unbanNode(banForm.node_id), true)"><ShieldCheck :size="16" /> 解封</button>
          <button class="danger-button" type="button" @click="confirmForceOfflineNode(forceOfflineForm.node_id, selected?.device_code, forceOfflineForm.reason || null)"><PowerOff :size="16" /> 强制下线</button>
          <button class="primary-button" type="button" @click="run(() => api.restoreOnlineNode(forceOfflineForm.node_id), true)"><RotateCcw :size="16" /> 恢复上线</button>
          <button class="danger-button" type="button" @click="confirmCleanUnboundNodes">清理无绑定节点</button>
        </div>
      </form>
    </div>

    <ResultPanel :value="result" />
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
