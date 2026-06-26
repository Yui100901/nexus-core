export interface ApiResponse<T> {
  code: number;
  message: string;
  data?: T;
}

export interface ProductData {
  id: number;
  name: string;
  description?: string | null;
}

export interface ProductVersionData {
  id: number;
  product_id: number;
  version_code: string;
  release_date?: string | null;
}

export interface LicenseData {
  id: number;
  product_id: number;
  license_key: string;
  validity_hours: number;
  status: number;
  remark?: string | null;
}

export interface NodeData {
  id: number;
  device_code: string;
  status: number;
  metadata?: string | null;
}

export interface OnlineSummary {
  total_online: number;
  nodes: Array<{
    product_id: number;
    device_code: string;
    license_key: string;
  }>;
  by_product: Array<{ key: string; count: number }>;
  by_license: Array<{ key: string; count: number }>;
}

export interface NodeHeartbeat {
  id: number;
  device_code: string;
  status: number;
  last_seen_at?: string | null;
  online_at?: string | null;
  offline_at?: string | null;
}

export interface AuditLog {
  id: number;
  resource_type: string;
  resource_id: number;
  action: string;
  operator: string;
  data: unknown;
  created_at?: string;
}

export interface ControlServiceData {
  id: number;
  product_id?: number | null;
  identifier: string;
  name: string;
  description?: string | null;
  service_type: string;
  input_schema: unknown;
  output_schema: unknown;
  status: number;
}

export interface NodeCapabilityData {
  id: number;
  node_id: number;
  service_identifier: string;
  schema: unknown;
  protocol: string;
  endpoint?: string | null;
  status: number;
}

export interface ControlCommandData {
  id: number;
  node_id: number;
  service_identifier: string;
  payload: unknown;
  converted_payload: unknown;
  status: number;
  result: unknown;
  error_message?: string | null;
}

export interface RegisterResult {
  node_id: number;
  license_id: number;
  product_id: number;
  license_key: string;
  license_status: number;
  feature_mask: string;
  max_nodes: number;
  current_node_count: number;
  max_concurrent: number;
  heartbeat_interval: number;
  binding_established: boolean;
}

export interface HeartbeatResult {
  online?: boolean;
  pending_control?: {
    count: number;
    command_ids: number[];
  };
  [key: string]: unknown;
}
