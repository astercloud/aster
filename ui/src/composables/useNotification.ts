/**
 * useNotification Composable
 * 通知管理
 */

import { ref, readonly } from "vue";

export interface Notification {
  id: string;
  type: "info" | "success" | "warning" | "error";
  title?: string;
  message: string;
  duration?: number;
  closable?: boolean;
}

const notifications = ref<Notification[]>([]);
let notificationId = 0;

export function useNotification() {
  function notify(options: Omit<Notification, "id">): string {
    const id = `notification-${++notificationId}`;
    const notification: Notification = {
      id,
      type: options.type || "info",
      title: options.title,
      message: options.message,
      duration: options.duration ?? 3000,
      closable: options.closable ?? true,
    };

    notifications.value.push(notification);

    // 自动关闭
    if (notification.duration && notification.duration > 0) {
      setTimeout(() => {
        remove(id);
      }, notification.duration);
    }

    return id;
  }

  function remove(id: string) {
    const index = notifications.value.findIndex((n) => n.id === id);
    if (index > -1) {
      notifications.value.splice(index, 1);
    }
  }

  function clear() {
    notifications.value = [];
  }

  // 便捷方法
  function info(message: string, title?: string, duration?: number) {
    return notify({ type: "info", message, title, duration });
  }

  function success(message: string, title?: string, duration?: number) {
    return notify({ type: "success", message, title, duration });
  }

  function warning(message: string, title?: string, duration?: number) {
    return notify({ type: "warning", message, title, duration });
  }

  function error(message: string, title?: string, duration?: number) {
    return notify({ type: "error", message, title, duration });
  }

  return {
    notifications: readonly(notifications),
    notify,
    remove,
    clear,
    info,
    success,
    warning,
    error,
  };
}
