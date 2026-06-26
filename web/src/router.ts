import { createRouter, createWebHistory } from 'vue-router';
import DashboardView from './views/DashboardView.vue';
import ProductsView from './views/ProductsView.vue';
import LicensesView from './views/LicensesView.vue';
import NodesView from './views/NodesView.vue';
import ControlView from './views/ControlView.vue';
import AccessLabView from './views/AccessLabView.vue';
import AuditView from './views/AuditView.vue';

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'dashboard', component: DashboardView },
    { path: '/products', name: 'products', component: ProductsView },
    { path: '/licenses', name: 'licenses', component: LicensesView },
    { path: '/nodes', name: 'nodes', component: NodesView },
    { path: '/control', name: 'control', component: ControlView },
    { path: '/access-lab', name: 'access-lab', component: AccessLabView },
    { path: '/audit', name: 'audit', component: AuditView },
  ],
});

export default router;
