<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue';
import { RefreshCw } from 'lucide-vue-next';
import { api } from '../api/client';
import type { AuditLog } from '../api/types';
import ResultPanel from '../components/ResultPanel.vue';

const error = ref('');
const logs = ref<AuditLog[]>([]);
const query = reactive({
  resource_type: '',
  resource_id: undefined as number | undefined,
  action: '',
  page: 1,
  page_size: 30,
});

async function load() {
  error.value = '';
  try {
    logs.value = await api.auditLogs(query);
  } catch (err) {
    error.value = err instanceof Error ? err.message : '加载失败';
  }
}

onMounted(load);
</script>

<template>
  <section class="page">
    <div class="page-head">
      <div>
        <h1>审计日志</h1>
        <p>产品、License、节点和控制链路的关键操作记录。</p>
      </div>
      <button class="primary-button" type="button" @click="load"><RefreshCw :size="16" /> 查询</button>
    </div>

    <p v-if="error" class="alert bad">{{ error }}</p>

    <section class="panel">
      <div class="filters">
        <label>资源类型
          <select v-model="query.resource_type">
            <option value="">全部</option>
            <option>product</option>
            <option>license</option>
            <option>node</option>
            <option>control_command</option>
          </select>
        </label>
        <label>资源 ID<input v-model.number="query.resource_id" type="number" min="1" /></label>
        <label>动作<input v-model="query.action" placeholder="create / update / ban" /></label>
        <label>页码<input v-model.number="query.page" type="number" min="1" /></label>
        <label>每页<input v-model.number="query.page_size" type="number" min="1" max="200" /></label>
      </div>

      <table>
        <thead>
          <tr>
            <th>ID</th>
            <th>资源</th>
            <th>动作</th>
            <th>操作者</th>
            <th>数据</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="log in logs" :key="log.id">
            <td>{{ log.id }}</td>
            <td>{{ log.resource_type }} #{{ log.resource_id }}</td>
            <td>{{ log.action }}</td>
            <td>{{ log.operator }}</td>
            <td><code>{{ log.data }}</code></td>
          </tr>
          <tr v-if="!logs.length">
            <td colspan="5" class="empty-cell">暂无数据</td>
          </tr>
        </tbody>
      </table>
    </section>

    <ResultPanel title="审计日志原始数据" :value="logs" />
  </section>
</template>
