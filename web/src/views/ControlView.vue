<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue';
import { Cpu, Edit3, Play, Plus, RefreshCw, Save, Search, Trash2, X } from 'lucide-vue-next';
import { api } from '../api/client';
import type { ControlCommandData, ControlServiceData, NodeCapabilityData } from '../api/types';
import ConfirmDialog from '../components/ConfirmDialog.vue';
import JsonEditor from '../components/JsonEditor.vue';
import StatusBadge from '../components/StatusBadge.vue';
import { errorMessage, notifyError, notifySuccess } from '../composables/useToast';
import { controlCommandStatusLabel, enabledStatusLabel, prettyJson, statusTone } from '../utils/status';

const activeTab = ref<'services' | 'capabilities' | 'commands'>('services');
const error = ref('');
const showCreateService = ref(false);
const showCreateCapability = ref(false);
const showCreateCommand = ref(false);
const serviceEditDialogOpen = ref(false);

const services = ref<ControlServiceData[]>([]);
const capabilities = ref<NodeCapabilityData[]>([]);
const commands = ref<ControlCommandData[]>([]);
const selectedService = ref<ControlServiceData | null>(null);
const confirmDialog = reactive({
  open: false,
  title: '',
  message: '',
  confirmLabel: '确认执行',
  action: null as null | (() => Promise<unknown>),
});

const serviceQuery = reactive({ product_id: undefined as number | undefined, page: 1, page_size: 20 });
const serviceForm = reactive({
  product_id: undefined as number | undefined,
  identifier: 'restart_process',
  name: 'Restart Process',
  description: '',
  service_type: 'command',
  input_schema: '{\n  "type": "object",\n  "properties": {\n    "process_name": { "type": "string" },\n    "delay_seconds": { "type": "integer" }\n  },\n  "required": ["process_name"]\n}',
  output_schema: '{\n  "type": "object"\n}',
});
const serviceEdit = reactive({
  id: 0,
  product_id: undefined as number | undefined,
  name: '',
  description: '',
  service_type: 'command',
  input_schema: '{}',
  output_schema: '{}',
});

const capabilityQuery = reactive({ node_id: undefined as number | undefined, page: 1, page_size: 20 });
const capabilityForm = reactive({
  node_id: 1,
  service_identifier: 'restart_process',
  protocol: 'http',
  endpoint: 'http://127.0.0.1:19090/control/restart',
  schema: '{\n  "fields": {\n    "proc": { "source": "process_name", "type": "string", "required": true },\n    "delay": { "source": "delay_seconds", "type": "integer", "default": 0 }\n  }\n}',
});

const commandQuery = reactive({
  node_id: undefined as number | undefined,
  service_identifier: '',
  status: undefined as number | undefined,
  page: 1,
  page_size: 20,
});
const commandForm = reactive({
  node_id: 1,
  service_identifier: 'restart_process',
  payload: '{\n  "process_name": "worker",\n  "delay_seconds": "3"\n}',
});
const commandLookup = reactive({ id: 1 });
const completeForm = reactive({ id: 1, status: 'success', result: '{\n  "ok": true\n}', error_message: '' });

function parseJson(text: string) {
  return JSON.parse(text || '{}');
}

async function run(action: () => Promise<unknown>, refresh?: () => Promise<void>) {
  error.value = '';
  try {
    await action();
    if (refresh) await refresh();
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
    await run(action, loadServices);
  }
}

async function loadServices() {
  error.value = '';
  try {
    services.value = await api.listControlServices(serviceQuery);
  } catch (err) {
    error.value = err instanceof Error ? err.message : '加载服务失败';
  }
}

async function loadCapabilities() {
  error.value = '';
  try {
    capabilities.value = await api.listNodeCapabilities(capabilityQuery);
  } catch (err) {
    error.value = err instanceof Error ? err.message : '加载能力失败';
  }
}

async function loadCommands() {
  error.value = '';
  try {
    commands.value = await api.listControlCommands(commandQuery);
  } catch (err) {
    error.value = err instanceof Error ? err.message : '加载指令失败';
  }
}

function selectService(service: ControlServiceData) {
  selectedService.value = service;
  serviceEdit.id = service.id;
  serviceEdit.product_id = service.product_id || undefined;
  serviceEdit.name = service.name;
  serviceEdit.description = service.description || '';
  serviceEdit.service_type = service.service_type;
  serviceEdit.input_schema = prettyJson(service.input_schema);
  serviceEdit.output_schema = prettyJson(service.output_schema);
  serviceEditDialogOpen.value = true;
}

function closeServiceEditDialog() {
  serviceEditDialogOpen.value = false;
}

async function saveControlService() {
  const ok = await run(() => api.updateControlService(serviceEdit.id, {
    product_id: serviceEdit.product_id || null,
    name: serviceEdit.name,
    description: serviceEdit.description || null,
    service_type: serviceEdit.service_type,
    input_schema: parseJson(serviceEdit.input_schema),
    output_schema: parseJson(serviceEdit.output_schema),
  }), loadServices);
  if (ok) closeServiceEditDialog();
}

function useCapability(capability: NodeCapabilityData) {
  capabilityForm.node_id = capability.node_id;
  capabilityForm.service_identifier = capability.service_identifier;
  capabilityForm.protocol = capability.protocol;
  capabilityForm.endpoint = capability.endpoint || '';
  capabilityForm.schema = prettyJson(capability.schema);
  showCreateCapability.value = true;
}

function confirmDeleteControlService(service: ControlServiceData) {
  askConfirm('删除控制服务', `确认删除控制服务「${service.name}」？已有能力或指令关联时服务端会拒绝删除。`, '确认删除', () => api.deleteControlService(service.id));
}

onMounted(() => {
  loadServices();
  loadCapabilities();
  loadCommands();
});
</script>

<template>
  <section class="page">
    <div class="page-head">
      <div>
        <h1>节点控制</h1>
        <p>控制服务、节点能力和控制指令都以列表进行维护。</p>
      </div>
      <div class="segmented">
        <button :class="{ active: activeTab === 'services' }" @click="activeTab = 'services'">服务</button>
        <button :class="{ active: activeTab === 'capabilities' }" @click="activeTab = 'capabilities'">能力</button>
        <button :class="{ active: activeTab === 'commands' }" @click="activeTab = 'commands'">指令</button>
      </div>
    </div>

    <p v-if="error" class="alert bad">{{ error }}</p>

    <template v-if="activeTab === 'services'">
      <div class="page-actions">
        <button class="primary-button" type="button" @click="showCreateService = !showCreateService"><Plus :size="16" /> 新增服务</button>
      </div>

      <form v-if="showCreateService" class="panel form-panel drawer-panel" @submit.prevent="run(() => api.createControlService({
        product_id: serviceForm.product_id || null,
        identifier: serviceForm.identifier,
        name: serviceForm.name,
        description: serviceForm.description || null,
        service_type: serviceForm.service_type,
        input_schema: parseJson(serviceForm.input_schema),
        output_schema: parseJson(serviceForm.output_schema)
      }), loadServices)">
        <h2>新增控制服务</h2>
        <div class="grid three compact">
          <label>产品 ID<input v-model.number="serviceForm.product_id" type="number" min="1" placeholder="空为通用服务" /></label>
          <label>标识符<input v-model="serviceForm.identifier" required /></label>
          <label>名称<input v-model="serviceForm.name" required /></label>
          <label>服务类型
            <select v-model="serviceForm.service_type">
              <option>command</option>
              <option>config</option>
              <option>query</option>
              <option>action</option>
            </select>
          </label>
          <label>描述<input v-model="serviceForm.description" /></label>
        </div>
        <div class="grid two compact">
          <JsonEditor v-model="serviceForm.input_schema" />
          <JsonEditor v-model="serviceForm.output_schema" />
        </div>
        <button class="primary-button" type="submit"><Cpu :size="16" /> 创建服务</button>
      </form>

      <section class="panel">
        <div class="panel-head">
          <h2>控制服务列表</h2>
          <button class="icon-button" title="刷新服务列表" @click="loadServices"><RefreshCw :size="16" /></button>
        </div>
        <div class="filters">
          <label>产品 ID<input v-model.number="serviceQuery.product_id" type="number" min="1" /></label>
          <label>页码<input v-model.number="serviceQuery.page" type="number" min="1" /></label>
          <label>每页<input v-model.number="serviceQuery.page_size" type="number" min="1" max="200" /></label>
          <button class="primary-button" type="button" @click="loadServices">筛选</button>
        </div>
        <table>
          <thead><tr><th>ID</th><th>产品</th><th>标识</th><th>名称</th><th>类型</th><th>状态</th><th>操作</th></tr></thead>
          <tbody>
            <tr v-for="item in services" :key="item.id" :class="{ selected: selectedService?.id === item.id }">
              <td>{{ item.id }}</td>
              <td>{{ item.product_id || '通用' }}</td>
              <td>{{ item.identifier }}</td>
              <td>{{ item.name }}</td>
              <td>{{ item.service_type }}</td>
              <td><StatusBadge :label="enabledStatusLabel(item.status)" :tone="statusTone(item.status, 'enabled')" /></td>
              <td>
                <div class="button-row wrap">
                  <button class="secondary-button" type="button" @click="selectService(item)"><Edit3 :size="15" /> 编辑</button>
                  <button class="primary-button" type="button" @click="run(() => api.updateControlServiceStatus(item.id, 1), loadServices)">启用</button>
                  <button class="secondary-button" type="button" @click="run(() => api.updateControlServiceStatus(item.id, 2), loadServices)">禁用</button>
                  <button class="danger-button" type="button" @click="confirmDeleteControlService(item)"><Trash2 :size="15" /> 删除</button>
                </div>
              </td>
            </tr>
            <tr v-if="!services.length"><td colspan="7" class="empty-cell">暂无服务</td></tr>
          </tbody>
        </table>
      </section>

    </template>

    <template v-if="activeTab === 'capabilities'">
      <div class="page-actions">
        <button class="primary-button" type="button" @click="showCreateCapability = !showCreateCapability"><Plus :size="16" /> 上报能力</button>
      </div>

      <form v-if="showCreateCapability" class="panel form-panel drawer-panel" @submit.prevent="run(() => api.reportNodeCapability({
        node_id: capabilityForm.node_id,
        service_identifier: capabilityForm.service_identifier,
        protocol: capabilityForm.protocol,
        endpoint: capabilityForm.endpoint || null,
        schema: parseJson(capabilityForm.schema)
      }), loadCapabilities)">
        <h2>新增或更新节点能力</h2>
        <div class="grid three compact">
          <label>节点 ID<input v-model.number="capabilityForm.node_id" type="number" min="1" required /></label>
          <label>服务标识<input v-model="capabilityForm.service_identifier" required /></label>
          <label>协议
            <select v-model="capabilityForm.protocol">
              <option>http</option>
              <option>mqtt</option>
              <option>websocket</option>
            </select>
          </label>
          <label>Endpoint<input v-model="capabilityForm.endpoint" /></label>
        </div>
        <JsonEditor v-model="capabilityForm.schema" />
        <button class="primary-button" type="submit">保存能力</button>
      </form>

      <section class="panel">
        <div class="panel-head">
          <h2>节点能力列表</h2>
          <button class="icon-button" title="刷新能力列表" @click="loadCapabilities"><RefreshCw :size="16" /></button>
        </div>
        <div class="filters">
          <label>节点 ID<input v-model.number="capabilityQuery.node_id" type="number" min="1" /></label>
          <label>页码<input v-model.number="capabilityQuery.page" type="number" min="1" /></label>
          <label>每页<input v-model.number="capabilityQuery.page_size" type="number" min="1" max="200" /></label>
          <button class="primary-button" type="button" @click="loadCapabilities">筛选</button>
        </div>
        <table>
          <thead><tr><th>ID</th><th>节点</th><th>服务</th><th>协议</th><th>Endpoint</th><th>状态</th><th>操作</th></tr></thead>
          <tbody>
            <tr v-for="item in capabilities" :key="item.id">
              <td>{{ item.id }}</td>
              <td>{{ item.node_id }}</td>
              <td>{{ item.service_identifier }}</td>
              <td>{{ item.protocol }}</td>
              <td>{{ item.endpoint || '-' }}</td>
              <td><StatusBadge :label="enabledStatusLabel(item.status)" :tone="statusTone(item.status, 'enabled')" /></td>
              <td><button class="secondary-button" type="button" @click="useCapability(item)"><Edit3 :size="15" /> 编辑</button></td>
            </tr>
            <tr v-if="!capabilities.length"><td colspan="7" class="empty-cell">暂无能力</td></tr>
          </tbody>
        </table>
      </section>
    </template>

    <template v-if="activeTab === 'commands'">
      <div class="page-actions">
        <button class="primary-button" type="button" @click="showCreateCommand = !showCreateCommand"><Plus :size="16" /> 下发指令</button>
      </div>

      <form v-if="showCreateCommand" class="panel form-panel drawer-panel" @submit.prevent="run(() => api.createControlCommand({ node_id: commandForm.node_id, service_identifier: commandForm.service_identifier, payload: parseJson(commandForm.payload) }), loadCommands)">
        <h2>创建并下发控制指令</h2>
        <div class="grid two compact">
          <label>节点 ID<input v-model.number="commandForm.node_id" type="number" min="1" required /></label>
          <label>服务标识<input v-model="commandForm.service_identifier" required /></label>
        </div>
        <JsonEditor v-model="commandForm.payload" />
        <button class="primary-button" type="submit"><Play :size="16" /> 下发</button>
      </form>

      <section class="panel">
        <div class="panel-head">
          <h2>控制指令列表</h2>
          <button class="icon-button" title="刷新指令列表" @click="loadCommands"><RefreshCw :size="16" /></button>
        </div>
        <div class="filters">
          <label>节点 ID<input v-model.number="commandQuery.node_id" type="number" min="1" /></label>
          <label>服务标识<input v-model="commandQuery.service_identifier" /></label>
          <label>状态
            <select v-model.number="commandQuery.status">
              <option :value="undefined">全部</option>
              <option :value="0">待发送</option>
              <option :value="1">已发送</option>
              <option :value="2">执行中</option>
              <option :value="3">成功</option>
              <option :value="4">失败</option>
              <option :value="5">超时</option>
            </select>
          </label>
          <label>页码<input v-model.number="commandQuery.page" type="number" min="1" /></label>
          <button class="primary-button" type="button" @click="loadCommands">筛选</button>
        </div>
        <table>
          <thead><tr><th>ID</th><th>节点</th><th>服务</th><th>状态</th><th>错误</th><th>操作</th></tr></thead>
          <tbody>
            <tr v-for="item in commands" :key="item.id">
              <td>{{ item.id }}</td>
              <td>{{ item.node_id }}</td>
              <td>{{ item.service_identifier }}</td>
              <td><StatusBadge :label="controlCommandStatusLabel(item.status)" :tone="statusTone(item.status, 'command')" /></td>
              <td>{{ item.error_message || '-' }}</td>
              <td>
                <div class="button-row wrap">
                  <button class="secondary-button" type="button" @click="run(() => api.getControlCommand(item.id))"><Search :size="15" /> 查看</button>
                  <button class="primary-button" type="button" @click="completeForm.id = item.id">回执</button>
                </div>
              </td>
            </tr>
            <tr v-if="!commands.length"><td colspan="6" class="empty-cell">暂无指令</td></tr>
          </tbody>
        </table>
      </section>

      <form class="panel form-panel" @submit.prevent="run(() => api.completeControlCommand(completeForm.id, { status: completeForm.status, result: parseJson(completeForm.result), error_message: completeForm.error_message || null }), loadCommands)">
        <h2>异步回执</h2>
        <div class="grid three compact">
          <label>指令 ID<input v-model.number="completeForm.id" type="number" min="1" required /></label>
          <label>状态
            <select v-model="completeForm.status">
              <option>success</option>
              <option>running</option>
              <option>failed</option>
              <option>timeout</option>
            </select>
          </label>
          <label>错误信息<input v-model="completeForm.error_message" /></label>
        </div>
        <JsonEditor v-model="completeForm.result" />
        <button class="primary-button" type="submit">提交回执</button>
      </form>

      <form class="panel form-panel" @submit.prevent="run(() => api.getControlCommand(commandLookup.id))">
        <h2>按 ID 查询指令详情</h2>
        <div class="button-row wrap">
          <label class="inline-field">指令 ID<input v-model.number="commandLookup.id" type="number" min="1" required /></label>
          <button class="secondary-button" type="submit"><Search :size="16" /> 查询</button>
        </div>
      </form>
    </template>

    <div v-if="serviceEditDialogOpen" class="modal-backdrop" @click.self="closeServiceEditDialog">
      <form class="modal-panel form-panel" @submit.prevent="saveControlService">
        <div class="modal-head">
          <h2>编辑控制服务 #{{ serviceEdit.id }}</h2>
          <button class="icon-button" type="button" title="关闭" @click="closeServiceEditDialog"><X :size="16" /></button>
        </div>
        <div class="grid three compact">
          <label>产品 ID<input v-model.number="serviceEdit.product_id" type="number" min="1" /></label>
          <label>名称<input v-model="serviceEdit.name" /></label>
          <label>服务类型
            <select v-model="serviceEdit.service_type">
              <option>command</option>
              <option>config</option>
              <option>query</option>
              <option>action</option>
            </select>
          </label>
          <label>描述<input v-model="serviceEdit.description" /></label>
        </div>
        <div class="grid two compact">
          <JsonEditor v-model="serviceEdit.input_schema" />
          <JsonEditor v-model="serviceEdit.output_schema" />
        </div>
        <div class="button-row">
          <button class="primary-button" type="submit"><Save :size="16" /> 保存修改</button>
          <button class="secondary-button" type="button" @click="closeServiceEditDialog">取消</button>
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

