import { reactive } from 'vue';

export type ToastTone = 'success' | 'error' | 'warning' | 'info';

export interface ToastItem {
  id: number;
  tone: ToastTone;
  title: string;
  message?: string;
}

const toasts = reactive<ToastItem[]>([]);
let nextID = 1;

export function useToasts() {
  return toasts;
}

export function dismissToast(id: number) {
  const index = toasts.findIndex((item) => item.id === id);
  if (index >= 0) {
    toasts.splice(index, 1);
  }
}

export function notify(toast: Omit<ToastItem, 'id'>, duration = 2800) {
  const item = { ...toast, id: nextID++ };
  toasts.push(item);
  window.setTimeout(() => dismissToast(item.id), duration);
  return item.id;
}

export function notifySuccess(message = '操作成功') {
  return notify({ tone: 'success', title: '成功', message });
}

export function notifyError(message = '操作失败') {
  return notify({ tone: 'error', title: '失败', message }, 4200);
}

export function notifyInfo(message: string) {
  return notify({ tone: 'info', title: '提示', message });
}

export function errorMessage(err: unknown, fallback = '操作失败') {
  return err instanceof Error ? err.message : fallback;
}
