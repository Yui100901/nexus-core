export function licenseStatusLabel(status: number) {
  return ({ 0: '未激活', 1: '已激活', 2: '已过期', 3: '已吊销' } as Record<number, string>)[status] || `未知 ${status}`;
}

export function nodeStatusLabel(status: number) {
  return ({ 0: '正常', 1: '离线', 2: '封禁' } as Record<number, string>)[status] || `未知 ${status}`;
}

export function controlCommandStatusLabel(status: number) {
  return ({ 0: '待发送', 1: '已发送', 2: '执行中', 3: '成功', 4: '失败', 5: '超时' } as Record<number, string>)[status] || `未知 ${status}`;
}

export function enabledStatusLabel(status: number) {
  return ({ 1: '启用', 2: '禁用' } as Record<number, string>)[status] || `未知 ${status}`;
}

export function statusTone(status: number, type: 'license' | 'node' | 'command' | 'enabled') {
  if (type === 'license') {
    return status === 1 ? 'good' : status === 0 ? 'idle' : status === 2 ? 'warn' : 'bad';
  }
  if (type === 'node') {
    return status === 0 ? 'good' : status === 1 ? 'idle' : 'bad';
  }
  if (type === 'command') {
    return status === 3 ? 'good' : status === 4 || status === 5 ? 'bad' : status === 2 ? 'warn' : 'idle';
  }
  return status === 1 ? 'good' : 'idle';
}

export function formatDate(value?: string | null) {
  if (!value) return '-';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString();
}

export function prettyJson(value: unknown) {
  if (value === undefined || value === null || value === '') return '{}';
  if (typeof value === 'string') {
    try {
      return JSON.stringify(JSON.parse(value), null, 2);
    } catch {
      return value;
    }
  }
  return JSON.stringify(value, null, 2);
}
