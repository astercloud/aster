import { useParams, useNavigate } from 'react-router-dom';
import { ArrowLeft, Clock, Hash, Cpu, AlertCircle, CheckCircle2, Loader2 } from 'lucide-react';
import { Header } from '../components/Header';
import { useTrace } from '../hooks/useApi';
import {
  formatDuration,
  formatNumber,
  formatCurrency,
  formatDate,
  cn,
} from '../lib/utils';
import type { TraceNode, TraceStatus } from '../api/types';

export function TraceDetail() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { data: trace, isLoading, refetch, isFetching } = useTrace(id || '');

  if (!id) {
    return <div>Invalid trace ID</div>;
  }

  return (
    <div className="flex flex-col h-full">
      <Header
        title={`Trace: ${id.slice(0, 8)}...`}
        subtitle={trace?.name}
        isRefreshing={isFetching}
        onRefresh={() => refetch()}
      />

      <div className="flex-1 overflow-auto p-6">
        {/* Back Button */}
        <button
          onClick={() => navigate('/traces')}
          className="flex items-center gap-2 text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] mb-4 transition-colors"
        >
          <ArrowLeft className="w-4 h-4" />
          返回列表
        </button>

        {isLoading ? (
          <div className="space-y-4">
            <div className="h-32 bg-[var(--color-surface)] rounded-lg animate-pulse-slow" />
            <div className="h-64 bg-[var(--color-surface)] rounded-lg animate-pulse-slow" />
          </div>
        ) : trace ? (
          <div className="space-y-6">
            {/* Summary Card */}
            <div className="card">
              <div className="flex items-start justify-between mb-4">
                <div className="flex items-center gap-3">
                  <StatusIcon status={trace.status} size="lg" />
                  <div>
                    <h2 className="text-xl font-semibold text-[var(--color-text-primary)]">
                      {trace.name}
                    </h2>
                    <p className="text-sm text-[var(--color-text-muted)]">
                      {trace.agent_name && `${trace.agent_name} · `}
                      {formatDate(trace.start_time)}
                    </p>
                  </div>
                </div>
                <div className="text-right">
                  <p className="text-2xl font-bold text-[var(--color-text-primary)]">
                    {formatDuration(trace.duration_ms)}
                  </p>
                  <p className="text-sm text-[var(--color-text-muted)]">
                    总耗时
                  </p>
                </div>
              </div>

              {trace.error_message && (
                <div className="p-3 bg-[var(--color-error)]/10 rounded-lg mb-4">
                  <p className="text-sm text-[var(--color-error)]">
                    {trace.error_message}
                  </p>
                </div>
              )}

              <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                <div className="p-3 bg-[var(--color-background)] rounded-lg">
                  <div className="flex items-center gap-2 text-[var(--color-text-muted)] text-sm mb-1">
                    <Hash className="w-4 h-4" />
                    Trace ID
                  </div>
                  <p className="font-mono text-sm text-[var(--color-text-primary)] truncate">
                    {trace.id}
                  </p>
                </div>
                <div className="p-3 bg-[var(--color-background)] rounded-lg">
                  <div className="flex items-center gap-2 text-[var(--color-text-muted)] text-sm mb-1">
                    <Cpu className="w-4 h-4" />
                    Spans
                  </div>
                  <p className="font-mono text-sm text-[var(--color-text-primary)]">
                    {trace.span_count}
                  </p>
                </div>
                <div className="p-3 bg-[var(--color-background)] rounded-lg">
                  <div className="flex items-center gap-2 text-[var(--color-text-muted)] text-sm mb-1">
                    <Clock className="w-4 h-4" />
                    Token 使用
                  </div>
                  <p className="font-mono text-sm text-[var(--color-text-primary)]">
                    {formatNumber(trace.token_usage.input)} / {formatNumber(trace.token_usage.output)}
                  </p>
                </div>
                <div className="p-3 bg-[var(--color-background)] rounded-lg">
                  <div className="flex items-center gap-2 text-[var(--color-text-muted)] text-sm mb-1">
                    成本
                  </div>
                  <p className="font-mono text-sm text-[var(--color-text-primary)]">
                    {formatCurrency(trace.cost.amount, trace.cost.currency)}
                  </p>
                </div>
              </div>
            </div>

            {/* Timeline */}
            {trace.root_span && (
              <div className="card">
                <h3 className="text-sm font-medium text-[var(--color-text-muted)] mb-4">
                  执行时间线
                </h3>
                <TraceTimeline
                  node={trace.root_span}
                  totalDuration={trace.duration_ms}
                />
              </div>
            )}
          </div>
        ) : (
          <div className="flex flex-col items-center justify-center h-64 text-[var(--color-text-muted)]">
            <AlertCircle className="w-12 h-12 mb-4" />
            <p>Trace 未找到</p>
          </div>
        )}
      </div>
    </div>
  );
}

interface StatusIconProps {
  status: TraceStatus;
  size?: 'sm' | 'lg';
}

function StatusIcon({ status, size = 'sm' }: StatusIconProps) {
  const sizeClass = size === 'lg' ? 'w-8 h-8' : 'w-5 h-5';

  switch (status) {
    case 'ok':
      return <CheckCircle2 className={cn(sizeClass, 'text-[var(--color-success)]')} />;
    case 'error':
      return <AlertCircle className={cn(sizeClass, 'text-[var(--color-error)]')} />;
    case 'running':
      return <Loader2 className={cn(sizeClass, 'text-[var(--color-accent)] animate-spin')} />;
    default:
      return <div className={cn(sizeClass, 'rounded-full bg-[var(--color-text-muted)]')} />;
  }
}

interface TraceTimelineProps {
  node: TraceNode;
  totalDuration: number;
  depth?: number;
}

function TraceTimeline({ node, totalDuration, depth = 0 }: TraceTimelineProps) {
  const startOffset = 0; // For root, offset is 0
  const widthPercent = Math.max((node.duration_ms / totalDuration) * 100, 1);

  const typeColors: Record<string, string> = {
    agent: 'var(--color-primary)',
    llm: 'var(--color-secondary)',
    tool: 'var(--color-accent)',
    middleware: 'var(--color-warning)',
  };

  const bgColor = typeColors[node.type] || 'var(--color-text-muted)';

  return (
    <div className="space-y-2">
      <div
        className="flex items-center gap-2"
        style={{ paddingLeft: depth * 24 }}
      >
        <div className="flex-1">
          <div className="flex items-center gap-2 mb-1">
            <StatusIcon status={node.status} size="sm" />
            <span className="text-sm font-medium text-[var(--color-text-primary)]">
              {node.name}
            </span>
            <span
              className="text-xs px-1.5 py-0.5 rounded"
              style={{ backgroundColor: `${bgColor}20`, color: bgColor }}
            >
              {node.type}
            </span>
            <span className="text-xs text-[var(--color-text-muted)]">
              {formatDuration(node.duration_ms)}
            </span>
          </div>
          <div className="h-6 bg-[var(--color-background)] rounded overflow-hidden">
            <div
              className="h-full rounded transition-all"
              style={{
                width: `${widthPercent}%`,
                marginLeft: `${startOffset}%`,
                backgroundColor: bgColor,
                opacity: node.status === 'error' ? 0.7 : 1,
              }}
            />
          </div>
        </div>
      </div>

      {node.children?.map((child, index) => (
        <TraceTimeline
          key={child.id || index}
          node={child}
          totalDuration={totalDuration}
          depth={depth + 1}
        />
      ))}
    </div>
  );
}
