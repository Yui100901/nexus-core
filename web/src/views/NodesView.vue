<script setup lang="ts">
import { reactive, ref } from 'vue';
import { Ban, Link2, Search, ShieldCheck, Trash2 } from 'lucide-vue-next';
import { api } from '../api/client';
import type { NodeData } from '../api/types';
import JsonEditor from '../components/JsonEditor.vue';
import ResultPanel from '../components/ResultPanel.vue';
import StatusBadge from '../components/StatusBadge.vue';
import { nodeStatusLabel, prettyJson, statusTone } from '../utils/status';

const result = ref<unknown>({});
const error = ref('');
const current = ref<NodeData | null>(null);
const createForm = reactive({ device_code: 'demo-node-001', metadata: '{\n  "os": "windows"\n}' });
const queryForm = reactive({ id: 1, device_code: '' });
const updateForm = reactive({ id: 1, device_code: '', metadata: '{}' });
const bindingForm = reactive({ node_id: 1, license_id: 1 });
const banForm = reactive({ node_id: 1, reason: '' });

async function run(action: () => Promise<unknown>) {
  error.value = '';
  try {
    result.value = await action();
  } catch (err) {
    error.value = err instanceof Error ? err.message : '操作失败';
  }
}

function setCurrent(data: NodeData) {
  current.value = data;
  updateForm.id = data.id;
  updateForm.device_code = data.device_code;
  updateForm.metadata = prettyJson(data.metadata || '{}');
  bindingForm.node_id = data.id;
  banForm.node_id = data.id;
}
</script>

<template>
  <section class="page">
    <div class="page-head">
      <div>
        <h1>节点管理</h1>
        <p>节点创建、绑定、封禁、解封和元数据维护。</p>
      </div>
    </div>

    <p v-if="error" class="alert bad">{{ error }}</p>

    <div class="grid three">
      <form class="panel form-panel" @submit.prevent="run(() => api.createNode({ device_code: createForm.device_code, metadata: createForm.metadata }))">
        <h2>创建节点</h2>
        <label>设备码<input v-model="createForm.device_code" required /></label>
        <JsonEditor v-model="createForm.metadata" :rows="6" />
        <button class="primary-button" type="submit"><Link2 :size="16" /> 创建</button>
      </form>

      <form class="panel form-panel">
        <h2>查询节点</h2>
        <label>节点 ID<input v-model.number="queryForm.id" type="number" min="1" /></label>
        <button class="primary-button" type="button" @click="run(async () => { const data = await api.getNode(queryForm.id); setCurrent(data); return data; })"><Search :size="16" /> 按 ID 查询</button>
        <label>设备码<input v-model="queryForm.device_code" /></label>
        <button class="secondary-button" type="button" @click="run(async () => { const data = await api.getNodeByDeviceCode(queryForm.device_code); setCurrent(data); return data; })"><Search :size="16" /> 按设备码查询</button>
        <div v-if="current" class="detail-list">
          <span>ID</span><strong>{{ current.id }}</strong>
          <span>状态</span><StatusBadge :label="nodeStatusLabel(current.status)" :tone="statusTone(current.status, 'node')" />
        </div>
      </form>

      <form class="panel form-panel" @submit.prevent="run(() => api.updateNode(updateForm.id, { device_code: updateForm.device_code || null, metadata: updateForm.metadata }))">
        <h2>编辑节点</h2>
        <label>节点 ID<input v-model.number="updateForm.id" type="number" min="1" required /></label>
        <label>设备码<input v-model="updateForm.device_code" /></label>
        <JsonEditor v-model="updateForm.metadata" :rows="6" />
        <div class="button-row">
          <button class="primary-button" type="submit">保存</button>
          <button class="danger-button" type="button" @click="run(() => api.deleteNode(updateForm.id))"><Trash2 :size="16" /> 删除</button>
        </div>
      </form>
    </div>

    <div class="grid two">
      <form class="panel form-panel">
        <h2>绑定关系</h2>
        <label>节点 ID<input v-model.number="bindingForm.node_id" type="number" min="1" /></label>
        <label>License ID<input v-model.number="bindingForm.license_id" type="number" min="1" /></label>
        <div class="button-row wrap">
          <button class="primary-button" type="button" @click="run(() => api.bindNode(bindingForm.node_id, bindingForm.license_id))"><Link2 :size="16" /> 绑定</button>
          <button class="danger-button" type="button" @click="run(() => api.unbindNode(bindingForm.node_id, bindingForm.license_id))">解绑</button>
          <button class="danger-button" type="button" @click="run(() => api.cleanUnboundNodes())">清理无绑定节点</button>
        </div>
      </form>

      <form class="panel form-panel">
        <h2>封禁状态</h2>
        <label>节点 ID<input v-model.number="banForm.node_id" type="number" min="1" /></label>
        <label>封禁原因<input v-model="banForm.reason" /></label>
        <div class="button-row wrap">
          <button class="danger-button" type="button" @click="run(() => api.banNode(banForm.node_id, banForm.reason || null))"><Ban :size="16" /> 封禁</button>
          <button class="primary-button" type="button" @click="run(() => api.unbanNode(banForm.node_id))"><ShieldCheck :size="16" /> 解封</button>
        </div>
      </form>
    </div>

    <ResultPanel :value="result" />
  </section>
</template>
