<script setup lang="ts">
import { reactive, ref } from 'vue';
import { Cpu, Play, RefreshCw, Search, Trash2 } from 'lucide-vue-next';
import { api } from '../api/client';
import type { ControlCommandData, ControlServiceData, NodeCapabilityData } from '../api/types';
import JsonEditor from '../components/JsonEditor.vue';
import ResultPanel from '../components/ResultPanel.vue';
import StatusBadge from '../components/StatusBadge.vue';
import { controlCommandStatusLabel, enabledStatusLabel, statusTone } from '../utils/status';

const activeTab = ref<'services' | 'capabilities' | 'commands'>('services');
const error = ref('');
const result = ref<unknown>({});
const services = ref<ControlServiceData[]>([]);
const capabilities = ref<NodeCapabilityData[]>([]);
const command = ref<ControlCommandData | null>(null);

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
const serviceAction = reactive({ id: 1, status: 1 });

const capabilityQuery = reactive({ node_id: undefined as number | undefined, page: 1, page_size: 20 });
const capabilityForm = reactive({
  node_id: 1,
  service_identifier: 'restart_process',
  protocol: 'http',
  endpoint: 'http://127.0.0.1:19090/control/restart',
  schema: '{\n  "fields": {\n    "proc": { "source": "process_name", "type": "string", "required": true },\n    "delay": { "source": "delay_seconds", "type": "integer", "default": 0 }\n  }\n}',
});

const commandForm = reactive({
  node_id: 1,
  service_identifier: 'restart_process',
  payload: '{\n  "process_name": "worker",\n  "delay_seconds": "3"\n}',
});
const commandQuery = reactive({ id: 1 });
const completeForm = reactive({ id: 1, status: 'success', result: '{\n  "ok": true\n}', error_message: '' });

async function run(action: () => Promise<unknown>) {
  error.value = '';
  try {
    result.value = await action();
  } catch (err) {
    error.value = err instanceof Error ? err.message : '操作失败';
  }
}

function parseJson(text: string) {
  return JSON.parse(text || '{}');
}

async function loadServices() {
  await run(async () => {
    services.value = await api.listControlServices(serviceQuery);
    return services.value;
  });
}

async function loadCapabilities() {
  await run(async () => {
    capabilities.value = await api.listNodeCapabilities(capabilityQuery);
    return capabilities.value;
  });
}
</script>

<template>
  <section class="page">
    <div class="page-head">
      <div>
        <h1>节点控制</h1>
        <p>控制服务定义、节点能力、指令下发和异步回执。</p>
      </div>
      <div class="segmented">
        <button :class="{ active: activeTab === 'services' }" @click="activeTab = 'services'">服务</button>
        <button :class="{ active: activeTab === 'capabilities' }" @click="activeTab = 'capabilities'">能力</button>
        <button :class="{ active: activeTab === 'commands' }" @click="activeTab = 'commands'">指令</button>
      </div>
    </div>

    <p v-if="error" class="alert bad">{{ error }}</p>

    <template v-if="activeTab === 'services'">
      <div class="grid two">
        <form class="panel form-panel" @submit.prevent="run(() => api.createControlService({
          product_id: serviceForm.product_id || null,
          identifier: serviceForm.identifier,
          name: serviceForm.name,
          description: serviceForm.description || null,
          service_type: serviceForm.service_type,
          input_schema: parseJson(serviceForm.input_schema),
          output_schema: parseJson(serviceForm.output_schema)
        }))">
          <h2>创建控制服务</h2>
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
          <JsonEditor v-model="serviceForm.input_schema" />
          <JsonEditor v-model="serviceForm.output_schema" />
          <button class="primary-button" type="submit"><Cpu :size="16" /> 创建服务</button>
        </form>

        <section class="panel">
          <div class="panel-head">
            <h2>服务列表</h2>
            <button class="icon-button" title="刷新服务列表" @click="loadServices"><RefreshCw :size="16" /></button>
          </div>
          <div class="filters">
            <label>产品 ID<input v-model.number="serviceQuery.product_id" type="number" min="1" /></label>
            <label>页码<input v-model.number="serviceQuery.page" type="number" min="1" /></label>
          </div>
          <table>
            <thead><tr><th>ID</th><th>标识</th><th>类型</th><th>状态</th></tr></thead>
            <tbody>
              <tr v-for="item in services" :key="item.id">
                <td>{{ item.id }}</td>
                <td>{{ item.identifier }}</td>
                <td>{{ item.service_type }}</td>
                <td><StatusBadge :label="enabledStatusLabel(item.status)" :tone="statusTone(item.status, 'enabled')" /></td>
              </tr>
              <tr v-if="!services.length"><td colspan="4" class="empty-cell">暂无数据</td></tr>
            </tbody>
          </table>
          <div class="button-row wrap">
            <label class="inline-field">服务 ID<input v-model.number="serviceAction.id" type="number" min="1" /></label>
            <button class="secondary-button" type="button" @click="run(() => api.getControlService(serviceAction.id))"><Search :size="16" /> 查询</button>
            <button class="primary-button" type="button" @click="run(() => api.updateControlServiceStatus(serviceAction.id, serviceAction.status))">更新状态</button>
            <select v-model.number="serviceAction.status">
              <option :value="1">启用</option>
              <option :value="2">禁用</option>
            </select>
            <button class="danger-button" type="button" @click="run(() => api.deleteControlService(serviceAction.id))"><Trash2 :size="16" /> 删除</button>
          </div>
        </section>
      </div>
    </template>

    <template v-if="activeTab === 'capabilities'">
      <div class="grid two">
        <form class="panel form-panel" @submit.prevent="run(() => api.reportNodeCapability({
          node_id: capabilityForm.node_id,
          service_identifier: capabilityForm.service_identifier,
          protocol: capabilityForm.protocol,
          endpoint: capabilityForm.endpoint || null,
          schema: parseJson(capabilityForm.schema)
        }))">
          <h2>上报节点能力</h2>
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
          <JsonEditor v-model="capabilityForm.schema" />
          <button class="primary-button" type="submit">保存能力</button>
        </form>

        <section class="panel">
          <div class="panel-head">
            <h2>能力列表</h2>
            <button class="icon-button" title="刷新能力列表" @click="loadCapabilities"><RefreshCw :size="16" /></button>
          </div>
          <div class="filters">
            <label>节点 ID<input v-model.number="capabilityQuery.node_id" type="number" min="1" /></label>
            <label>页码<input v-model.number="capabilityQuery.page" type="number" min="1" /></label>
          </div>
          <table>
            <thead><tr><th>ID</th><th>节点</th><th>服务</th><th>协议</th><th>状态</th></tr></thead>
            <tbody>
              <tr v-for="item in capabilities" :key="item.id">
                <td>{{ item.id }}</td>
                <td>{{ item.node_id }}</td>
                <td>{{ item.service_identifier }}</td>
                <td>{{ item.protocol }}</td>
                <td><StatusBadge :label="enabledStatusLabel(item.status)" :tone="statusTone(item.status, 'enabled')" /></td>
              </tr>
              <tr v-if="!capabilities.length"><td colspan="5" class="empty-cell">暂无数据</td></tr>
            </tbody>
          </table>
        </section>
      </div>
    </template>

    <template v-if="activeTab === 'commands'">
      <div class="grid three">
        <form class="panel form-panel" @submit.prevent="run(async () => { command = await api.createControlCommand({ node_id: commandForm.node_id, service_identifier: commandForm.service_identifier, payload: parseJson(commandForm.payload) }); return command; })">
          <h2>下发指令</h2>
          <label>节点 ID<input v-model.number="commandForm.node_id" type="number" min="1" required /></label>
          <label>服务标识<input v-model="commandForm.service_identifier" required /></label>
          <JsonEditor v-model="commandForm.payload" />
          <button class="primary-button" type="submit"><Play :size="16" /> 下发</button>
        </form>

        <form class="panel form-panel" @submit.prevent="run(async () => { command = await api.getControlCommand(commandQuery.id); return command; })">
          <h2>查询指令</h2>
          <label>指令 ID<input v-model.number="commandQuery.id" type="number" min="1" required /></label>
          <button class="primary-button" type="submit"><Search :size="16" /> 查询</button>
          <div v-if="command" class="detail-list">
            <span>状态</span>
            <StatusBadge :label="controlCommandStatusLabel(command.status)" :tone="statusTone(command.status, 'command')" />
            <span>错误</span><strong>{{ command.error_message || '-' }}</strong>
          </div>
        </form>

        <form class="panel form-panel" @submit.prevent="run(() => api.completeControlCommand(completeForm.id, { status: completeForm.status, result: parseJson(completeForm.result), error_message: completeForm.error_message || null }))">
          <h2>异步回执</h2>
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
          <JsonEditor v-model="completeForm.result" />
          <button class="primary-button" type="submit">提交回执</button>
        </form>
      </div>
    </template>

    <ResultPanel :value="result" />
  </section>
</template>
