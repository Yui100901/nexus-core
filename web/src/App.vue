<script setup lang="ts">
import { ref } from 'vue';
import { RouterLink, RouterView } from 'vue-router';
import {
  Activity,
  Boxes,
  ClipboardList,
  Cpu,
  Gauge,
  KeyRound,
  Network,
  PlugZap,
  RefreshCw,
  Save,
} from 'lucide-vue-next';
import { getApiBase, setApiBase, api } from './api/client';
import ToastHost from './components/ToastHost.vue';
import { notifyError, notifySuccess } from './composables/useToast';

const apiBase = ref(getApiBase());
const health = ref<'idle' | 'ok' | 'bad'>('idle');
const healthMessage = ref('未检测');

const navItems = [
  { to: '/', label: '仪表盘', icon: Gauge },
  { to: '/products', label: '产品', icon: Boxes },
  { to: '/licenses', label: 'License', icon: KeyRound },
  { to: '/nodes', label: '节点', icon: Network },
  { to: '/control', label: '控制', icon: Cpu },
  { to: '/access-lab', label: '接入调试', icon: PlugZap },
  { to: '/audit', label: '审计', icon: ClipboardList },
];

function saveApiBase() {
  setApiBase(apiBase.value);
  apiBase.value = getApiBase();
  notifySuccess('API 地址已保存');
}

async function checkHealth() {
  try {
    const data = await api.health();
    health.value = data.status === 'ok' ? 'ok' : 'bad';
    healthMessage.value = data.status;
  } catch (err) {
    health.value = 'bad';
    healthMessage.value = err instanceof Error ? err.message : '连接失败';
    notifyError(healthMessage.value);
  }
}

checkHealth();
</script>

<template>
  <div class="app-shell">
    <aside class="sidebar">
      <div class="brand">
        <Activity :size="24" />
        <div>
          <strong>Nexus Core</strong>
          <span>License 管理台</span>
        </div>
      </div>

      <nav class="nav-list">
        <RouterLink v-for="item in navItems" :key="item.to" :to="item.to" class="nav-item">
          <component :is="item.icon" :size="18" />
          <span>{{ item.label }}</span>
        </RouterLink>
      </nav>
    </aside>

    <main class="main-pane">
      <header class="topbar">
        <div class="api-config">
          <label for="apiBase">API</label>
          <input id="apiBase" v-model="apiBase" type="text" />
          <button class="icon-button" type="button" title="保存 API 地址" @click="saveApiBase">
            <Save :size="16" />
          </button>
          <button class="icon-button" type="button" title="检测服务健康状态" @click="checkHealth">
            <RefreshCw :size="16" />
          </button>
        </div>
        <div class="health-pill" :class="health">
          <span></span>
          {{ healthMessage }}
        </div>
      </header>

      <RouterView />
    </main>
    <ToastHost />
  </div>
</template>
