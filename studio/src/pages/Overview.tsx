import {
  Activity,
  Users,
  Coins,
  AlertTriangle,
  TrendingUp,
  Clock,
  Zap,
} from 'lucide-react';
import { Header } from '../components/Header';
import { useOverview, useRecentEvents } from '../hooks/useApi';
import {
  formatNumber,
  formatCurrency,
  formatDuration,
  formatPercent,
  formatRelativeTime,
} from '../lib/utils';

export function Overview() {
  const { data: overview, isLoading, refetch, isFetching } = useOverview('24h');
  const { data: events } = useRecentEvents(10);

  return (
    <div className="flex flex-col h-full">
      <Header
        title="Overview"
        subtitle="系统概览和关键指标"
        isRefreshing={isFetching}
        onRefresh={() => refetch()}
        lastUpdated={overview ? new Date(overview.updated_at) : undefined}
      />

      <div className="flex-1 overflow-auto p-6">
        {isLoading ? (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
            {[...Array(4)].map((_, i) => (
              <div
                key={i}
                className="card h-32 animate-pulse-slow bg-[var(--color-surface)]"
              />
            ))}
          </div>
        ) : overview ? (
          <div className="space-y-6">
            {/* Stat Cards */}
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
              <StatCard
                title="活跃 Agents"
                value={overview.active_agents}
                icon={Users}
                color="primary"
              />
              <StatCard
                title="活跃会话"
                value={overview.active_sessions}
                icon={Activity}
                color="accent"
              />
              <StatCard
                title="总请求数"
                value={formatNumber(overview.total_requests)}
                icon={Zap}
                color="success"
              />
              <StatCard
                title="错误率"
                value={formatPercent(overview.error_rate)}
                icon={AlertTriangle}
                color={overview.error_rate > 0.05 ? 'error' : 'success'}
              />
            </div>

            {/* Token & Cost Summary */}
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
              <div className="card">
                <h3 className="text-sm font-medium text-[var(--color-text-muted)] mb-4">
                  Token 使用 (24h)
                </h3>
                <div className="space-y-4">
                  <div className="flex justify-between items-center">
                    <span className="text-[var(--color-text-secondary)]">
                      输入 Token
                    </span>
                    <span className="font-mono text-[var(--color-text-primary)]">
                      {formatNumber(overview.token_usage.input)}
                    </span>
                  </div>
                  <div className="flex justify-between items-center">
                    <span className="text-[var(--color-text-secondary)]">
                      输出 Token
                    </span>
                    <span className="font-mono text-[var(--color-text-primary)]">
                      {formatNumber(overview.token_usage.output)}
                    </span>
                  </div>
                  <div className="h-px bg-[var(--color-border)]" />
                  <div className="flex justify-between items-center">
                    <span className="text-[var(--color-text-primary)] font-medium">
                      总计
                    </span>
                    <span className="font-mono text-lg text-[var(--color-primary)]">
                      {formatNumber(overview.token_usage.total)}
                    </span>
                  </div>
                </div>
              </div>

              <div className="card">
                <h3 className="text-sm font-medium text-[var(--color-text-muted)] mb-4">
                  成本概览 (24h)
                </h3>
                <div className="space-y-4">
                  <div className="flex items-center gap-4">
                    <div className="p-3 rounded-lg bg-[var(--color-success)]/10">
                      <Coins className="w-6 h-6 text-[var(--color-success)]" />
                    </div>
                    <div>
                      <p className="text-2xl font-bold text-[var(--color-text-primary)]">
                        {formatCurrency(overview.cost.amount, overview.cost.currency)}
                      </p>
                      <p className="text-sm text-[var(--color-text-muted)]">
                        累计费用
                      </p>
                    </div>
                  </div>
                  <div className="flex justify-between items-center text-sm">
                    <span className="text-[var(--color-text-secondary)]">
                      平均延迟
                    </span>
                    <span className="font-mono text-[var(--color-text-primary)]">
                      {formatDuration(overview.avg_latency_ms)}
                    </span>
                  </div>
                </div>
              </div>
            </div>

            {/* Recent Events */}
            <div className="card">
              <h3 className="text-sm font-medium text-[var(--color-text-muted)] mb-4">
                最近活动
              </h3>
              {events?.events && events.events.length > 0 ? (
                <div className="space-y-2">
                  {events.events.slice(0, 5).map((event) => (
                    <div
                      key={event.cursor}
                      className="flex items-center justify-between py-2 border-b border-[var(--color-border)] last:border-0"
                    >
                      <div className="flex items-center gap-3">
                        <Clock className="w-4 h-4 text-[var(--color-text-muted)]" />
                        <span className="text-sm text-[var(--color-text-secondary)]">
                          {formatRelativeTime(event.timestamp)}
                        </span>
                      </div>
                      <span className="text-sm text-[var(--color-text-muted)] truncate max-w-xs">
                        {JSON.stringify(event.event).slice(0, 50)}...
                      </span>
                    </div>
                  ))}
                </div>
              ) : (
                <p className="text-sm text-[var(--color-text-muted)] text-center py-4">
                  暂无活动记录
                </p>
              )}
            </div>
          </div>
        ) : (
          <div className="flex items-center justify-center h-64">
            <p className="text-[var(--color-text-muted)]">加载失败，请重试</p>
          </div>
        )}
      </div>
    </div>
  );
}

interface StatCardProps {
  title: string;
  value: number | string;
  icon: React.ComponentType<{ className?: string; style?: React.CSSProperties }>;
  color: 'primary' | 'accent' | 'success' | 'warning' | 'error';
  trend?: number;
}

function StatCard({ title, value, icon: Icon, color, trend }: StatCardProps) {
  const colorMap = {
    primary: 'var(--color-primary)',
    accent: 'var(--color-accent)',
    success: 'var(--color-success)',
    warning: 'var(--color-warning)',
    error: 'var(--color-error)',
  };

  return (
    <div className="card">
      <div className="flex items-start justify-between">
        <div>
          <p className="text-sm text-[var(--color-text-muted)]">{title}</p>
          <p className="text-2xl font-bold text-[var(--color-text-primary)] mt-1">
            {value}
          </p>
        </div>
        <div
          className="p-2 rounded-lg"
          style={{ backgroundColor: `${colorMap[color]}20` }}
        >
          <Icon className="w-5 h-5" style={{ color: colorMap[color] }} />
        </div>
      </div>
      {trend !== undefined && (
        <div className="flex items-center gap-1 mt-2">
          <TrendingUp
            className={`w-4 h-4 ${
              trend >= 0 ? 'text-[var(--color-success)]' : 'text-[var(--color-error)]'
            }`}
          />
          <span
            className={`text-sm ${
              trend >= 0 ? 'text-[var(--color-success)]' : 'text-[var(--color-error)]'
            }`}
          >
            {trend >= 0 ? '+' : ''}
            {formatPercent(trend)}
          </span>
        </div>
      )}
    </div>
  );
}
