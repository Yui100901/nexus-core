<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { RefreshCw } from 'lucide-vue-next';
import { api } from '../api/client';
import type { NodeHeartbeat, OnlineSummary } from '../api/types';
import StatusBadge from '../components/StatusBadge.vue';
import ResultPanel from '../components/ResultPanel.vue';
import { formatDate, nodeStatusLabel, statusTone } from '../utils/status';

const loading = ref(false);
const error = ref('');
const online = ref<OnlineSummary | null>(null);
const heartbeats = ref<NodeHeartbeat[]>([]);

async function load() {
  loading.value = true;
  error.value = '';
  try {
    const [onlineData, heartbeatData] = await Promise.all([
      api.onlineSummary(),
      api.nodeHeartbeats({ page: 1, page_size: 20 }),
    ]);
    online.value = onlineData;
    heartbeats.value = heartbeatData;
  } catch (err) {
    error.value = err instanceof Error ? err.message : '加载失败';
  } finally {
    loading.value = false;
  }
}

onMounted(load);
</script>

<template>
  <section class="page">
    <div class="page-head">
      <div>
        <h1>仪表盘</h1>
        <p>在线状态、最近心跳和运行期统计。</p>
      </div>
      <button class="primary-button" type="button" @click="load">
        <RefreshCw :size="16" />
        刷新
      </button>
    </div>

    <p v-if="error" class="alert bad">{{ error }}</p>

    <div class="metric-grid">
      <article class="metric">
        <span>在线节点</span>
        <strong>{{ online?.total_online ?? 0 }}</strong>
      </article>
      <article class="metric">
        <span>产品在线分组</span>
        <strong>{{ online?.by_product.length ?? 0 }}</strong>
      </article>
      <article class="metric">
        <span>License 在线分组</span>
        <strong>{{ online?.by_license.length ?? 0 }}</strong>
      </article>
      <article class="metric">
        <span>最近心跳记录</span>
        <strong>{{ heartbeats.length }}</strong>
      </article>
    </div>

    <div class="grid two">
      <section class="panel">
        <h2>在线节点</h2>
        <table>
          <thead>
            <tr>
              <th>产品 ID</th>
              <th>设备码</th>
              <th>License Key</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="node in online?.nodes || []" :key="`${node.product_id}-${node.device_code}`">
              <td>{{ node.product_id }}</td>
              <td>{{ node.device_code }}</td>
              <td class="mono">{{ node.license_key }}</td>
            </tr>
            <tr v-if="!online?.nodes.length">
              <td colspan="3" class="empty-cell">暂无在线节点</td>
            </tr>
          </tbody>
        </table>
      </section>

      <section class="panel">
        <h2>最近心跳</h2>
        <table>
          <thead>
            <tr>
              <th>ID</th>
              <th>设备码</th>
              <th>状态</th>
              <th>最近心跳</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in heartbeats" :key="item.id">
              <td>{{ item.id }}</td>
              <td>{{ item.device_code }}</td>
              <td>
                <StatusBadge :label="nodeStatusLabel(item.status)" :tone="statusTone(item.status, 'node')" />
              </td>
              <td>{{ formatDate(item.last_seen_at) }}</td>
            </tr>
            <tr v-if="!heartbeats.length">
              <td colspan="4" class="empty-cell">{{ loading ? '加载中' : '暂无数据' }}</td>
            </tr>
          </tbody>
        </table>
      </section>
    </div>

    <ResultPanel title="在线摘要原始数据" :value="online || {}" />
  </section>
</template>
