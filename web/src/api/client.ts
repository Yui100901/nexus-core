import type {
  ApiResponse,
  AuditLog,
  ControlCommandData,
  ControlServiceData,
  HeartbeatResult,
  LicenseData,
  NodeCapabilityData,
  NodeData,
  NodeHeartbeat,
  OnlineSummary,
  ProductData,
  ProductVersionData,
  RegisterResult,
} from './types';

const API_BASE_KEY = 'nexus-core-api-base';
const DEFAULT_API_BASE = 'http://localhost:8080';

export function getApiBase() {
  return localStorage.getItem(API_BASE_KEY) || DEFAULT_API_BASE;
}

export function setApiBase(value: string) {
  const next = value.trim().replace(/\/+$/, '');
  localStorage.setItem(API_BASE_KEY, next || DEFAULT_API_BASE);
}

function withQuery(path: string, query?: Record<string, string | number | undefined | null>) {
  if (!query) return path;
  const params = new URLSearchParams();
  for (const [key, value] of Object.entries(query)) {
    if (value !== undefined && value !== null && value !== '') {
      params.set(key, String(value));
    }
  }
  const text = params.toString();
  return text ? `${path}?${text}` : path;
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const headers = new Headers(options.headers);
  if (options.body && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json');
  }

  const response = await fetch(`${getApiBase()}${path}`, {
    ...options,
    headers,
  });

  const text = await response.text();
  let payload: ApiResponse<T> | null = null;
  if (text) {
    try {
      payload = JSON.parse(text) as ApiResponse<T>;
    } catch {
      throw new Error(text);
    }
  }

  if (!response.ok || (payload && payload.code >= 400)) {
    throw new Error(payload?.message || response.statusText || '请求失败');
  }

  return (payload?.data ?? (payload as T)) as T;
}

function jsonBody(data: unknown): RequestInit {
  return { body: JSON.stringify(data) };
}

export const api = {
  health: () => request<{ status: string }>('/health'),

  onlineSummary: () => request<OnlineSummary>('/monitor/online'),
  nodeHeartbeats: (query: { page?: number; page_size?: number }) =>
    request<NodeHeartbeat[]>(withQuery('/monitor/nodes/heartbeats', query)),
  auditLogs: (query: { resource_type?: string; resource_id?: number; action?: string; page?: number; page_size?: number }) =>
    request<AuditLog[]>(withQuery('/audit-logs', query)),

  createProduct: (data: { name: string; description?: string | null }) =>
    request<ProductData>('/products', { method: 'POST', ...jsonBody(data) }),
  listProducts: (query: { name?: string; status?: number; page?: number; page_size?: number }) =>
    request<ProductData[]>(withQuery('/products', query)),
  getProduct: (id: number) => request<ProductData>(`/products/${id}`),
  updateProduct: (id: number, data: { name?: string | null; description?: string | null }) =>
    request<ProductData>(`/products/${id}`, { method: 'PATCH', ...jsonBody(data) }),
  deleteProduct: (id: number) => request<void>(`/products/${id}`, { method: 'DELETE' }),
  createProductVersion: (data: {
    product_id: number;
    version_code: string;
    description?: string | null;
    release_date?: string | null;
    release_method: number;
  }) => request<ProductVersionData>('/products/versions', { method: 'POST', ...jsonBody(data) }),
  releaseVersion: (data: { product_id: number; version_id: number; release_date?: string | null }) =>
    request<number>('/products/versions/release', { method: 'POST', ...jsonBody(data) }),
  deprecateVersion: (data: { product_id: number; version_id: number }) =>
    request<void>('/products/versions/deprecate', { method: 'POST', ...jsonBody(data) }),
  setMinVersion: (data: { product_id: number; version_id: number }) =>
    request<void>('/products/min-supported-version', { method: 'POST', ...jsonBody(data) }),

  createLicense: (data: { product_id: number; validity_hours: number; max_nodes: number; max_concurrent: number; remark?: string | null }) =>
    request<LicenseData>('/licenses', { method: 'POST', ...jsonBody(data) }),
  batchCreateLicenses: (data: { product_id: number; validity_hours: number; max_nodes: number; max_concurrent: number; count: number; remark?: string | null }) =>
    request<LicenseData[]>('/licenses/batch', { method: 'POST', ...jsonBody(data) }),
  listLicenses: (query: { product_id?: number; status?: number; license_key?: string; page?: number; page_size?: number }) =>
    request<LicenseData[]>(withQuery('/licenses', query)),
  getLicense: (id: number) => request<LicenseData>(`/licenses/${id}`),
  getLicenseByKey: (key: string) => request<LicenseData>(`/license-keys/${encodeURIComponent(key)}`),
  updateLicense: (id: number, data: { max_nodes: number; max_concurrent: number; feature_mask: string; remark?: string | null }) =>
    request<string>(`/licenses/${id}`, { method: 'PATCH', ...jsonBody(data) }),
  renewLicense: (id: number, extra_hours: number) =>
    request<string>(`/licenses/${id}/renew`, { method: 'POST', ...jsonBody({ extra_hours }) }),
  revokeLicense: (id: number) => request<void>(`/licenses/${id}/revoke`, { method: 'POST' }),
  restoreLicense: (id: number) => request<LicenseData>(`/licenses/${id}/restore`, { method: 'POST' }),
  deleteLicense: (id: number) => request<string>(`/licenses/${id}`, { method: 'DELETE' }),
  cleanLicenseBindings: (id: number) => request<string>(`/licenses/${id}/bindings`, { method: 'DELETE' }),
  cleanInvalidLicenses: () => request<void>('/license-cleanups/invalid', { method: 'DELETE' }),

  createNode: (data: { device_code: string; metadata?: string | null }) =>
    request<NodeData>('/nodes', { method: 'POST', ...jsonBody(data) }),
  listNodes: (query: { device_code?: string; status?: number; page?: number; page_size?: number }) =>
    request<NodeData[]>(withQuery('/nodes', query)),
  getNode: (id: number) => request<NodeData>(`/nodes/${id}`),
  getNodeByDeviceCode: (deviceCode: string) => request<NodeData>(`/node-devices/${encodeURIComponent(deviceCode)}`),
  updateNode: (id: number, data: { device_code?: string | null; metadata?: string | null }) =>
    request<NodeData>(`/nodes/${id}`, { method: 'PATCH', ...jsonBody(data) }),
  deleteNode: (id: number) => request<void>(`/nodes/${id}`, { method: 'DELETE' }),
  banNode: (id: number, reason?: string | null) =>
    request<void>(`/nodes/${id}/ban`, { method: 'POST', ...jsonBody({ reason }) }),
  unbanNode: (id: number) => request<void>(`/nodes/${id}/unban`, { method: 'POST' }),
  bindNode: (node_id: number, license_id: number) =>
    request<string>('/node-bindings', { method: 'POST', ...jsonBody({ node_id, license_id }) }),
  unbindNode: (node_id: number, license_id: number) =>
    request<void>('/node-bindings', { method: 'DELETE', ...jsonBody({ node_id, license_id }) }),
  cleanUnboundNodes: () => request<void>('/node-cleanups/unbound', { method: 'DELETE' }),

  listControlServices: (query: { product_id?: number; page?: number; page_size?: number }) =>
    request<ControlServiceData[]>(withQuery('/control-services', query)),
  createControlService: (data: {
    product_id?: number | null;
    identifier: string;
    name: string;
    description?: string | null;
    service_type: string;
    input_schema: unknown;
    output_schema: unknown;
  }) => request<ControlServiceData>('/control-services', { method: 'POST', ...jsonBody(data) }),
  getControlService: (id: number) => request<ControlServiceData>(`/control-services/${id}`),
  updateControlService: (id: number, data: Partial<Omit<ControlServiceData, 'id' | 'identifier' | 'status'>>) =>
    request<ControlServiceData>(`/control-services/${id}`, { method: 'PATCH', ...jsonBody(data) }),
  updateControlServiceStatus: (id: number, status: number) =>
    request<ControlServiceData>(`/control-services/${id}/status`, { method: 'POST', ...jsonBody({ status }) }),
  deleteControlService: (id: number) => request<void>(`/control-services/${id}`, { method: 'DELETE' }),

  listNodeCapabilities: (query: { node_id?: number; page?: number; page_size?: number }) =>
    request<NodeCapabilityData[]>(withQuery('/node-capabilities', query)),
  reportNodeCapability: (data: { node_id: number; service_identifier: string; schema: unknown; protocol: string; endpoint?: string | null }) =>
    request<NodeCapabilityData>('/node-capabilities', { method: 'POST', ...jsonBody(data) }),

  createControlCommand: (data: { node_id: number; service_identifier: string; payload: unknown }) =>
    request<ControlCommandData>('/control-commands', { method: 'POST', ...jsonBody(data) }),
  listControlCommands: (query: { node_id?: number; service_identifier?: string; status?: number; page?: number; page_size?: number }) =>
    request<ControlCommandData[]>(withQuery('/control-commands', query)),
  getControlCommand: (id: number) => request<ControlCommandData>(`/control-commands/${id}`),
  completeControlCommand: (id: number, data: { status: string; result?: unknown; error_message?: string | null }) =>
    request<ControlCommandData>(`/control-commands/${id}/complete`, { method: 'POST', ...jsonBody(data) }),

  registerAccess: (data: { device_code: string; license_key: string; product_id: number; version_code: string }) =>
    request<RegisterResult>('/access/register', { method: 'POST', ...jsonBody(data) }),
  heartbeat: (data: { device_code: string; license_key: string; product_id: number; version_code: string }) =>
    request<HeartbeatResult>('/access/heartbeat', { method: 'POST', ...jsonBody(data) }),
};
