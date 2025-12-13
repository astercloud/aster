import { useState } from 'react';
import { Link } from 'react-router-dom';
import {
  Trophy,
  Plus,
  Trash2,
  MoreVertical,
  Play,
  CheckCircle,
  XCircle,
  Clock,
  Loader2,
  Search,
  TrendingUp,
  BarChart3,
} from 'lucide-react';
import { Header } from '../components/Header';
import { useBenchmarks, useDeleteBenchmark, useRunBenchmark } from '../hooks/useApi';
import { formatRelativeTime, cn } from '../lib/utils';
import type { BenchmarkRecord } from '../api/types';

export function Benchmarks() {
  const [searchQuery, setSearchQuery] = useState('');

  const { data, isLoading, error, refetch, isFetching } = useBenchmarks();
  const deleteBenchmark = useDeleteBenchmark();
  const runBenchmark = useRunBenchmark();

  const benchmarks = data?.benchmarks || [];

  // Filter by search
  const filteredBenchmarks = benchmarks.filter((benchmark) => {
    if (searchQuery) {
      const searchLower = searchQuery.toLowerCase();
      if (
        !benchmark.name.toLowerCase().includes(searchLower) &&
        !benchmark.id.toLowerCase().includes(searchLower)
      ) {
        return false;
      }
    }
    return true;
  });

  const handleDelete = async (benchmark: BenchmarkRecord) => {
    if (confirm(`确定要删除 Benchmark "${benchmark.name}" 吗？`)) {
      try {
        await deleteBenchmark.mutateAsync(benchmark.id);
      } catch (e) {
        console.error('Failed to delete benchmark:', e);
      }
    }
  };

  const handleRun = async (benchmark: BenchmarkRecord) => {
    try {
      await runBenchmark.mutateAsync({ id: benchmark.id });
    } catch (e) {
      console.error('Failed to run benchmark:', e);
    }
  };

  return (
    <div className="flex flex-col h-full">
      <Header
        title="Benchmarks"
        subtitle="基准测试管理"
        isRefreshing={isFetching}
        onRefresh={() => refetch()}
      />

      <div className="flex-1 overflow-auto">
        {/* Toolbar */}
        <div className="flex items-center justify-between p-4 border-b border-[var(--color-border)]">
          <div className="flex items-center gap-4">
            {/* Search */}
            <div className="relative">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[var(--color-text-muted)]" />
              <input
                type="text"
                placeholder="搜索 Benchmark..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="pl-9 pr-4 py-2 bg-[var(--color-surface)] border border-[var(--color-border)] rounded-lg text-sm text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] focus:outline-none focus:border-[var(--color-primary)] w-64"
              />
            </div>

            {/* Count */}
            <span className="text-sm text-[var(--color-text-muted)]">
              {filteredBenchmarks.length} 个 Benchmark
            </span>
          </div>

          {/* Create Button */}
          <Link
            to="/benchmarks/new"
            className="flex items-center gap-2 px-4 py-2 bg-[var(--color-primary)] text-white rounded-lg text-sm font-medium hover:bg-[var(--color-primary-hover)] transition-colors"
          >
            <Plus className="w-4 h-4" />
            新建 Benchmark
          </Link>
        </div>

        {/* Content */}
        <div className="p-4">
          {isLoading ? (
            <div className="flex items-center justify-center h-64">
              <div className="animate-spin w-8 h-8 border-2 border-[var(--color-primary)] border-t-transparent rounded-full" />
            </div>
          ) : error ? (
            <div className="flex flex-col items-center justify-center h-64 text-[var(--color-error)]">
              <XCircle className="w-12 h-12 mb-4" />
              <p>加载失败: {(error as Error).message}</p>
            </div>
          ) : filteredBenchmarks.length === 0 ? (
            <div className="flex flex-col items-center justify-center h-64 text-[var(--color-text-muted)]">
              <Trophy className="w-12 h-12 mb-4" />
              <p>{benchmarks.length === 0 ? '暂无 Benchmark' : '没有匹配的 Benchmark'}</p>
              {benchmarks.length === 0 && (
                <Link
                  to="/benchmarks/new"
                  className="mt-4 flex items-center gap-2 px-4 py-2 bg-[var(--color-primary)] text-white rounded-lg text-sm font-medium hover:bg-[var(--color-primary-hover)] transition-colors"
                >
                  <Plus className="w-4 h-4" />
                  创建第一个 Benchmark
                </Link>
              )}
            </div>
          ) : (
            <div className="space-y-4">
              {filteredBenchmarks.map((benchmark) => (
                <BenchmarkCard
                  key={benchmark.id}
                  benchmark={benchmark}
                  onDelete={handleDelete}
                  onRun={handleRun}
                  isRunning={runBenchmark.isPending}
                />
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

interface BenchmarkCardProps {
  benchmark: BenchmarkRecord;
  onDelete: (benchmark: BenchmarkRecord) => void;
  onRun: (benchmark: BenchmarkRecord) => void;
  isRunning: boolean;
}

function BenchmarkCard({ benchmark, onDelete, onRun, isRunning }: BenchmarkCardProps) {
  const [showMenu, setShowMenu] = useState(false);

  const summary = benchmark.summary;
  const hasResults = summary && summary.total_runs > 0;

  const successRate = hasResults
    ? (summary.successful_runs / summary.total_runs) * 100
    : 0;

  const successRateColor = successRate >= 80
    ? 'text-[var(--color-success)]'
    : successRate >= 50
      ? 'text-[var(--color-warning)]'
      : 'text-[var(--color-error)]';

  return (
    <div className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-lg p-4 hover:border-[var(--color-primary)]/50 transition-colors">
      <div className="flex items-start justify-between">
        {/* Left */}
        <div className="flex items-start gap-4 flex-1 min-w-0">
          {/* Icon */}
          <div className="w-12 h-12 rounded-lg bg-gradient-to-br from-[var(--color-primary)] to-[var(--color-secondary)] flex items-center justify-center flex-shrink-0">
            <Trophy className="w-6 h-6 text-white" />
          </div>

          {/* Info */}
          <div className="flex-1 min-w-0">
            <h3 className="font-semibold text-[var(--color-text-primary)] truncate text-lg">
              {benchmark.name}
            </h3>
            <p className="text-xs text-[var(--color-text-muted)] font-mono mt-1">
              ID: {benchmark.id.slice(0, 16)}...
            </p>
            <p className="text-sm text-[var(--color-text-muted)] mt-1 flex items-center gap-1">
              <Clock className="w-3 h-3" />
              创建于 {formatRelativeTime(benchmark.created_at)}
            </p>
          </div>
        </div>

        {/* Right - Actions */}
        <div className="flex items-center gap-2">
          <button
            onClick={() => onRun(benchmark)}
            disabled={isRunning}
            className="flex items-center gap-2 px-3 py-1.5 bg-[var(--color-success)]/10 text-[var(--color-success)] rounded-lg text-sm font-medium hover:bg-[var(--color-success)]/20 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {isRunning ? (
              <Loader2 className="w-4 h-4 animate-spin" />
            ) : (
              <Play className="w-4 h-4" />
            )}
            运行
          </button>

          <div className="relative">
            <button
              onClick={() => setShowMenu(!showMenu)}
              className="p-2 rounded hover:bg-[var(--color-surface-elevated)] transition-colors"
            >
              <MoreVertical className="w-4 h-4 text-[var(--color-text-muted)]" />
            </button>

            {showMenu && (
              <>
                <div
                  className="fixed inset-0 z-10"
                  onClick={() => setShowMenu(false)}
                />
                <div className="absolute right-0 top-full mt-1 w-36 bg-[var(--color-surface-elevated)] border border-[var(--color-border)] rounded-lg shadow-lg z-20 py-1">
                  <Link
                    to={`/benchmarks/${benchmark.id}`}
                    className="flex items-center gap-2 px-3 py-2 text-sm text-[var(--color-text-secondary)] hover:bg-[var(--color-surface)] transition-colors"
                    onClick={() => setShowMenu(false)}
                  >
                    <BarChart3 className="w-4 h-4" />
                    查看详情
                  </Link>
                  <hr className="my-1 border-[var(--color-border)]" />
                  <button
                    className="flex items-center gap-2 px-3 py-2 text-sm text-[var(--color-error)] hover:bg-[var(--color-surface)] transition-colors w-full"
                    onClick={() => {
                      setShowMenu(false);
                      onDelete(benchmark);
                    }}
                  >
                    <Trash2 className="w-4 h-4" />
                    删除
                  </button>
                </div>
              </>
            )}
          </div>
        </div>
      </div>

      {/* Summary Stats */}
      {hasResults && (
        <div className="mt-4 pt-4 border-t border-[var(--color-border)]">
          <div className="grid grid-cols-2 md:grid-cols-5 gap-4">
            {/* Total Runs */}
            <div className="text-center">
              <div className="text-2xl font-semibold text-[var(--color-text-primary)]">
                {summary.total_runs}
              </div>
              <div className="text-xs text-[var(--color-text-muted)]">总运行次数</div>
            </div>

            {/* Success Rate */}
            <div className="text-center">
              <div className={cn('text-2xl font-semibold', successRateColor)}>
                {successRate.toFixed(1)}%
              </div>
              <div className="text-xs text-[var(--color-text-muted)]">成功率</div>
            </div>

            {/* Avg Score */}
            {summary.avg_score !== undefined && (
              <div className="text-center">
                <div className="text-2xl font-semibold text-[var(--color-primary)]">
                  {(summary.avg_score * 100).toFixed(1)}%
                </div>
                <div className="text-xs text-[var(--color-text-muted)]">平均评分</div>
              </div>
            )}

            {/* Avg Latency */}
            {summary.avg_latency_ms !== undefined && (
              <div className="text-center">
                <div className="text-2xl font-semibold text-[var(--color-text-secondary)]">
                  {summary.avg_latency_ms.toFixed(0)}ms
                </div>
                <div className="text-xs text-[var(--color-text-muted)]">平均延迟</div>
              </div>
            )}

            {/* Total Cost */}
            {summary.total_cost !== undefined && (
              <div className="text-center">
                <div className="text-2xl font-semibold text-[var(--color-warning)]">
                  ${summary.total_cost.toFixed(2)}
                </div>
                <div className="text-xs text-[var(--color-text-muted)]">总成本</div>
              </div>
            )}
          </div>

          {/* Progress Bar */}
          <div className="mt-4">
            <div className="flex items-center justify-between text-xs text-[var(--color-text-muted)] mb-1">
              <span>成功: {summary.successful_runs}</span>
              <span>失败: {summary.failed_runs}</span>
            </div>
            <div className="h-2 bg-[var(--color-surface-elevated)] rounded-full overflow-hidden">
              <div
                className="h-full bg-[var(--color-success)] transition-all duration-300"
                style={{ width: `${successRate}%` }}
              />
            </div>
          </div>
        </div>
      )}

      {/* No Results */}
      {!hasResults && (
        <div className="mt-4 pt-4 border-t border-[var(--color-border)] text-center">
          <div className="flex items-center justify-center gap-2 text-[var(--color-text-muted)]">
            <TrendingUp className="w-4 h-4" />
            <span className="text-sm">尚未运行，点击"运行"开始测试</span>
          </div>
        </div>
      )}

      {/* Recent Runs */}
      {benchmark.runs && benchmark.runs.length > 0 && (
        <div className="mt-4 pt-4 border-t border-[var(--color-border)]">
          <h4 className="text-sm font-medium text-[var(--color-text-secondary)] mb-2">
            最近运行
          </h4>
          <div className="space-y-2">
            {benchmark.runs.slice(0, 3).map((run) => (
              <div
                key={run.id}
                className="flex items-center justify-between text-sm p-2 bg-[var(--color-surface-elevated)] rounded"
              >
                <div className="flex items-center gap-2">
                  {run.error ? (
                    <XCircle className="w-4 h-4 text-[var(--color-error)]" />
                  ) : (
                    <CheckCircle className="w-4 h-4 text-[var(--color-success)]" />
                  )}
                  <span className="text-[var(--color-text-muted)] font-mono text-xs">
                    {run.id.slice(0, 8)}
                  </span>
                </div>
                <div className="flex items-center gap-4">
                  {run.score !== undefined && (
                    <span className="text-[var(--color-text-secondary)]">
                      {(run.score * 100).toFixed(1)}%
                    </span>
                  )}
                  {run.duration_ms !== undefined && (
                    <span className="text-[var(--color-text-muted)]">
                      {run.duration_ms}ms
                    </span>
                  )}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
