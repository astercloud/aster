import { clsx, type ClassValue } from 'clsx';
import { twMerge } from 'tailwind-merge';

// Merge Tailwind classes
export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

// Format numbers
export function formatNumber(num: number, decimals: number = 0): string {
  if (num >= 1_000_000) {
    return (num / 1_000_000).toFixed(decimals) + 'M';
  }
  if (num >= 1_000) {
    return (num / 1_000).toFixed(decimals) + 'K';
  }
  return num.toFixed(decimals);
}

// Format currency
export function formatCurrency(amount: number, currency: string = 'USD'): string {
  if (currency === 'USD') {
    if (amount < 0.01) {
      return `$${(amount * 100).toFixed(2)}¢`;
    }
    return `$${amount.toFixed(4)}`;
  }
  if (currency === 'CNY') {
    return `¥${amount.toFixed(2)}`;
  }
  return `${amount.toFixed(4)} ${currency}`;
}

// Format duration in milliseconds
export function formatDuration(ms: number): string {
  if (ms < 1000) {
    return `${ms}ms`;
  }
  if (ms < 60000) {
    return `${(ms / 1000).toFixed(1)}s`;
  }
  const minutes = Math.floor(ms / 60000);
  const seconds = ((ms % 60000) / 1000).toFixed(0);
  return `${minutes}m ${seconds}s`;
}

// Format percentage
export function formatPercent(value: number, decimals: number = 1): string {
  return `${(value * 100).toFixed(decimals)}%`;
}

// Format date
export function formatDate(dateStr: string): string {
  const date = new Date(dateStr);
  return date.toLocaleDateString('zh-CN', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

// Format relative time
export function formatRelativeTime(dateStr: string): string {
  const date = new Date(dateStr);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();

  if (diffMs < 60000) {
    return '刚刚';
  }
  if (diffMs < 3600000) {
    return `${Math.floor(diffMs / 60000)} 分钟前`;
  }
  if (diffMs < 86400000) {
    return `${Math.floor(diffMs / 3600000)} 小时前`;
  }
  return `${Math.floor(diffMs / 86400000)} 天前`;
}

// Get status color class
export function getStatusColor(status: string): string {
  switch (status) {
    case 'ok':
      return 'text-[var(--color-success)]';
    case 'error':
      return 'text-[var(--color-error)]';
    case 'running':
      return 'text-[var(--color-accent)]';
    case 'warning':
      return 'text-[var(--color-warning)]';
    default:
      return 'text-[var(--color-text-secondary)]';
  }
}

// Get status background class
export function getStatusBgColor(status: string): string {
  switch (status) {
    case 'ok':
      return 'bg-[var(--color-success)]';
    case 'error':
      return 'bg-[var(--color-error)]';
    case 'running':
      return 'bg-[var(--color-accent)]';
    case 'warning':
      return 'bg-[var(--color-warning)]';
    default:
      return 'bg-[var(--color-text-muted)]';
  }
}

// Get severity color
export function getSeverityColor(severity: string): string {
  switch (severity) {
    case 'critical':
      return 'text-[var(--color-error)]';
    case 'warning':
      return 'text-[var(--color-warning)]';
    case 'info':
    default:
      return 'text-[var(--color-accent)]';
  }
}

// Truncate string
export function truncate(str: string, maxLength: number): string {
  if (str.length <= maxLength) return str;
  return str.slice(0, maxLength - 3) + '...';
}
