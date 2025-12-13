import { RefreshCw, Bell, Clock } from 'lucide-react';
import { cn } from '../lib/utils';

interface HeaderProps {
  title: string;
  subtitle?: string;
  isRefreshing?: boolean;
  onRefresh?: () => void;
  lastUpdated?: Date;
}

export function Header({
  title,
  subtitle,
  isRefreshing,
  onRefresh,
  lastUpdated,
}: HeaderProps) {
  return (
    <header className="flex items-center justify-between h-14 px-6 bg-[var(--color-surface)] border-b border-[var(--color-border)]">
      <div>
        <h1 className="text-lg font-semibold text-[var(--color-text-primary)]">
          {title}
        </h1>
        {subtitle && (
          <p className="text-sm text-[var(--color-text-muted)]">{subtitle}</p>
        )}
      </div>

      <div className="flex items-center gap-4">
        {/* Last Updated */}
        {lastUpdated && (
          <div className="flex items-center gap-1.5 text-sm text-[var(--color-text-muted)]">
            <Clock className="w-4 h-4" />
            <span>
              更新于 {lastUpdated.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })}
            </span>
          </div>
        )}

        {/* Refresh Button */}
        {onRefresh && (
          <button
            onClick={onRefresh}
            disabled={isRefreshing}
            className={cn(
              'flex items-center gap-2 px-3 py-1.5 rounded-lg text-sm',
              'bg-[var(--color-surface-elevated)] text-[var(--color-text-secondary)]',
              'hover:bg-[var(--color-border)] transition-colors',
              'disabled:opacity-50 disabled:cursor-not-allowed'
            )}
          >
            <RefreshCw
              className={cn('w-4 h-4', isRefreshing && 'animate-spin')}
            />
            刷新
          </button>
        )}

        {/* Notifications */}
        <button
          className="relative p-2 rounded-lg hover:bg-[var(--color-surface-elevated)] text-[var(--color-text-muted)] transition-colors"
          title="Notifications"
        >
          <Bell className="w-5 h-5" />
          <span className="absolute top-1 right-1 w-2 h-2 bg-[var(--color-error)] rounded-full" />
        </button>
      </div>
    </header>
  );
}
