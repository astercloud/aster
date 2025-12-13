import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Search, Filter, ChevronRight, AlertCircle, CheckCircle2, Loader2 } from 'lucide-react';
import { Header } from '../components/Header';
import { useTraces } from '../hooks/useApi';
import {
  formatDuration,
  formatNumber,
  formatRelativeTime,
} from '../lib/utils';
import type { TraceStatus } from '../api/types';

export function Traces() {
  const navigate = useNavigate();
  const [statusFilter, setStatusFilter] = useState<string>('');
  const [searchQuery, setSearchQuery] = useState('');

  const { data, isLoading, refetch, isFetching } = useTraces({
    status: statusFilter || undefined,
    limit: 50,
  });

  const filteredTraces = data?.traces?.filter((trace) => {
    if (!searchQuery) return true;
    return (
      trace.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      trace.id.toLowerCase().includes(searchQuery.toLowerCase()) ||
      trace.agent_name?.toLowerCase().includes(searchQuery.toLowerCase())
    );
  });

  return (
    <div className="flex flex-col h-full">
      <Header
        title="Traces"
        subtitle="Agent 执行追踪"
        isRefreshing={isFetching}
        onRefresh={() => refetch()}
      />

      <div className="flex-1 overflow-auto">
        {/* Filters */}
        <div className="p-4 border-b border-[var(--color-border)] bg-[var(--color-surface)]">
          <div className="flex items-center gap-4">
            {/* Search */}
            <div className="flex-1 relative">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[var(--color-text-muted)]" />
              <input
                type="text"
                placeholder="搜索 Trace ID、名称、Agent..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="w-full pl-10 pr-4 py-2 bg-[var(--color-background)] border border-[var(--color-border)] rounded-lg text-sm text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] focus:outline-none focus:border-[var(--color-primary)]"
              />
            </div>

            {/* Status Filter */}
            <div className="flex items-center gap-2">
              <Filter className="w-4 h-4 text-[var(--color-text-muted)]" />
              <select
                value={statusFilter}
                onChange={(e) => setStatusFilter(e.target.value)}
                className="px-3 py-2 bg-[var(--color-background)] border border-[var(--color-border)] rounded-lg text-sm text-[var(--color-text-primary)] focus:outline-none focus:border-[var(--color-primary)]"
              >
                <option value="">全部状态</option>
                <option value="ok">成功</option>
                <option value="error">错误</option>
                <option value="running">运行中</option>
              </select>
            </div>
          </div>
        </div>

        {/* Trace List */}
        <div className="p-4">
          {isLoading ? (
            <div className="space-y-2">
              {[...Array(5)].map((_, i) => (
                <div
                  key={i}
                  className="h-20 bg-[var(--color-surface)] rounded-lg animate-pulse-slow"
                />
              ))}
            </div>
          ) : filteredTraces && filteredTraces.length > 0 ? (
            <div className="space-y-2">
              {filteredTraces.map((trace) => (
                <div
                  key={trace.id}
                  onClick={() => navigate(`/traces/${trace.id}`)}
                  className="card cursor-pointer hover:border-[var(--color-primary)] transition-colors"
                >
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <StatusIcon status={trace.status} />
                      <div>
                        <p className="font-medium text-[var(--color-text-primary)]">
                          {trace.name}
                        </p>
                        <p className="text-sm text-[var(--color-text-muted)]">
                          {trace.id.slice(0, 8)}...
                          {trace.agent_name && ` · ${trace.agent_name}`}
                        </p>
                      </div>
                    </div>

                    <div className="flex items-center gap-6">
                      <div className="text-right">
                        <p className="text-sm text-[var(--color-text-secondary)]">
                          {formatDuration(trace.duration_ms)}
                        </p>
                        <p className="text-xs text-[var(--color-text-muted)]">
                          {formatRelativeTime(trace.start_time)}
                        </p>
                      </div>

                      <div className="text-right">
                        <p className="text-sm font-mono text-[var(--color-text-secondary)]">
                          {formatNumber(trace.token_usage.total)} tokens
                        </p>
                        <p className="text-xs text-[var(--color-text-muted)]">
                          {trace.span_count} spans
                        </p>
                      </div>

                      <ChevronRight className="w-5 h-5 text-[var(--color-text-muted)]" />
                    </div>
                  </div>

                  {trace.error_message && (
                    <div className="mt-2 p-2 bg-[var(--color-error)]/10 rounded text-sm text-[var(--color-error)]">
                      {trace.error_message}
                    </div>
                  )}
                </div>
              ))}

              {data?.has_more && (
                <div className="text-center py-4">
                  <button className="px-4 py-2 bg-[var(--color-surface-elevated)] text-[var(--color-text-secondary)] rounded-lg hover:bg-[var(--color-border)] transition-colors">
                    加载更多
                  </button>
                </div>
              )}
            </div>
          ) : (
            <div className="flex flex-col items-center justify-center h-64 text-[var(--color-text-muted)]">
              <AlertCircle className="w-12 h-12 mb-4" />
              <p>暂无 Trace 记录</p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

function StatusIcon({ status }: { status: TraceStatus }) {
  switch (status) {
    case 'ok':
      return <CheckCircle2 className="w-5 h-5 text-[var(--color-success)]" />;
    case 'error':
      return <AlertCircle className="w-5 h-5 text-[var(--color-error)]" />;
    case 'running':
      return <Loader2 className="w-5 h-5 text-[var(--color-accent)] animate-spin" />;
    default:
      return <div className="w-5 h-5 rounded-full bg-[var(--color-text-muted)]" />;
  }
}
