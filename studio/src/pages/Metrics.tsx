import { useState } from 'react';
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  LineChart,
  Line,
  PieChart,
  Pie,
  Cell,
} from 'recharts';
import { Header } from '../components/Header';
import { useTokenUsage, useCosts, usePerformance } from '../hooks/useApi';
import { formatNumber, formatCurrency, formatDuration, cn } from '../lib/utils';

const COLORS = ['#6366f1', '#8b5cf6', '#06b6d4', '#10b981', '#f59e0b', '#ef4444'];

type TimeRange = '1h' | '24h' | '7d' | '30d';

export function Metrics() {
  const [timeRange, setTimeRange] = useState<TimeRange>('24h');

  const { data: tokenUsage, isLoading: tokensLoading, refetch: refetchTokens, isFetching: tokensFetching } = useTokenUsage({ period: timeRange });
  const { data: costs, isLoading: costsLoading } = useCosts({ period: timeRange });
  const { data: performance, isLoading: perfLoading } = usePerformance(timeRange);

  const isLoading = tokensLoading || costsLoading || perfLoading;
  const isFetching = tokensFetching;

  // Prepare chart data
  const tokenTrendData = tokenUsage?.trend?.map((point) => ({
    time: new Date(point.timestamp).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }),
    input: point.input,
    output: point.output,
  })) || [];

  const costTrendData = costs?.trend?.map((point) => ({
    time: new Date(point.timestamp).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }),
    amount: point.amount,
  })) || [];

  const modelUsageData = Object.entries(tokenUsage?.by_model || {}).map(([model, usage], i) => ({
    name: model,
    value: usage.total,
    color: COLORS[i % COLORS.length],
  }));

  return (
    <div className="flex flex-col h-full">
      <Header
        title="Metrics"
        subtitle="Token 使用与成本分析"
        isRefreshing={isFetching}
        onRefresh={() => refetchTokens()}
      />

      <div className="flex-1 overflow-auto p-6">
        {/* Time Range Selector */}
        <div className="flex gap-2 mb-6">
          {(['1h', '24h', '7d', '30d'] as TimeRange[]).map((range) => (
            <button
              key={range}
              onClick={() => setTimeRange(range)}
              className={cn(
                'px-3 py-1.5 rounded-lg text-sm transition-colors',
                timeRange === range
                  ? 'bg-[var(--color-primary)] text-white'
                  : 'bg-[var(--color-surface)] text-[var(--color-text-secondary)] hover:bg-[var(--color-surface-elevated)]'
              )}
            >
              {range === '1h' ? '1 小时' : range === '24h' ? '24 小时' : range === '7d' ? '7 天' : '30 天'}
            </button>
          ))}
        </div>

        {isLoading ? (
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            {[...Array(4)].map((_, i) => (
              <div key={i} className="h-80 bg-[var(--color-surface)] rounded-lg animate-pulse-slow" />
            ))}
          </div>
        ) : (
          <div className="space-y-6">
            {/* Summary Cards */}
            <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
              <div className="card">
                <p className="text-sm text-[var(--color-text-muted)]">总 Token</p>
                <p className="text-2xl font-bold text-[var(--color-text-primary)]">
                  {formatNumber(tokenUsage?.total.total || 0)}
                </p>
              </div>
              <div className="card">
                <p className="text-sm text-[var(--color-text-muted)]">输入 Token</p>
                <p className="text-2xl font-bold text-[var(--color-primary)]">
                  {formatNumber(tokenUsage?.total.input || 0)}
                </p>
              </div>
              <div className="card">
                <p className="text-sm text-[var(--color-text-muted)]">输出 Token</p>
                <p className="text-2xl font-bold text-[var(--color-secondary)]">
                  {formatNumber(tokenUsage?.total.output || 0)}
                </p>
              </div>
              <div className="card">
                <p className="text-sm text-[var(--color-text-muted)]">总成本</p>
                <p className="text-2xl font-bold text-[var(--color-success)]">
                  {formatCurrency(costs?.total.amount || 0, costs?.total.currency || 'USD')}
                </p>
              </div>
            </div>

            {/* Charts */}
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
              {/* Token Trend */}
              <div className="card">
                <h3 className="text-sm font-medium text-[var(--color-text-muted)] mb-4">
                  Token 使用趋势
                </h3>
                <div className="h-64">
                  <ResponsiveContainer width="100%" height="100%">
                    <LineChart data={tokenTrendData}>
                      <CartesianGrid strokeDasharray="3 3" stroke="var(--color-border)" />
                      <XAxis
                        dataKey="time"
                        tick={{ fill: 'var(--color-text-muted)', fontSize: 12 }}
                        stroke="var(--color-border)"
                      />
                      <YAxis
                        tick={{ fill: 'var(--color-text-muted)', fontSize: 12 }}
                        stroke="var(--color-border)"
                        tickFormatter={(value) => formatNumber(value)}
                      />
                      <Tooltip
                        contentStyle={{
                          backgroundColor: 'var(--color-surface)',
                          border: '1px solid var(--color-border)',
                          borderRadius: '8px',
                        }}
                        labelStyle={{ color: 'var(--color-text-primary)' }}
                      />
                      <Line
                        type="monotone"
                        dataKey="input"
                        stroke="var(--color-primary)"
                        strokeWidth={2}
                        dot={false}
                        name="输入"
                      />
                      <Line
                        type="monotone"
                        dataKey="output"
                        stroke="var(--color-secondary)"
                        strokeWidth={2}
                        dot={false}
                        name="输出"
                      />
                    </LineChart>
                  </ResponsiveContainer>
                </div>
              </div>

              {/* Model Usage Distribution */}
              <div className="card">
                <h3 className="text-sm font-medium text-[var(--color-text-muted)] mb-4">
                  模型使用分布
                </h3>
                <div className="h-64 flex items-center">
                  {modelUsageData.length > 0 ? (
                    <ResponsiveContainer width="100%" height="100%">
                      <PieChart>
                        <Pie
                          data={modelUsageData}
                          cx="50%"
                          cy="50%"
                          innerRadius={60}
                          outerRadius={80}
                          paddingAngle={5}
                          dataKey="value"
                          label={({ name, percent }) => `${name} ${((percent ?? 0) * 100).toFixed(0)}%`}
                          labelLine={{ stroke: 'var(--color-text-muted)' }}
                        >
                          {modelUsageData.map((entry, index) => (
                            <Cell key={`cell-${index}`} fill={entry.color} />
                          ))}
                        </Pie>
                        <Tooltip
                          contentStyle={{
                            backgroundColor: 'var(--color-surface)',
                            border: '1px solid var(--color-border)',
                            borderRadius: '8px',
                          }}
                          formatter={(value: number) => formatNumber(value)}
                        />
                      </PieChart>
                    </ResponsiveContainer>
                  ) : (
                    <p className="w-full text-center text-[var(--color-text-muted)]">
                      暂无数据
                    </p>
                  )}
                </div>
              </div>

              {/* Cost Trend */}
              <div className="card">
                <h3 className="text-sm font-medium text-[var(--color-text-muted)] mb-4">
                  成本趋势
                </h3>
                <div className="h-64">
                  <ResponsiveContainer width="100%" height="100%">
                    <BarChart data={costTrendData}>
                      <CartesianGrid strokeDasharray="3 3" stroke="var(--color-border)" />
                      <XAxis
                        dataKey="time"
                        tick={{ fill: 'var(--color-text-muted)', fontSize: 12 }}
                        stroke="var(--color-border)"
                      />
                      <YAxis
                        tick={{ fill: 'var(--color-text-muted)', fontSize: 12 }}
                        stroke="var(--color-border)"
                        tickFormatter={(value) => `$${value.toFixed(2)}`}
                      />
                      <Tooltip
                        contentStyle={{
                          backgroundColor: 'var(--color-surface)',
                          border: '1px solid var(--color-border)',
                          borderRadius: '8px',
                        }}
                        formatter={(value: number) => [`$${value.toFixed(4)}`, '成本']}
                      />
                      <Bar dataKey="amount" fill="var(--color-success)" radius={[4, 4, 0, 0]} />
                    </BarChart>
                  </ResponsiveContainer>
                </div>
              </div>

              {/* Performance Metrics */}
              <div className="card">
                <h3 className="text-sm font-medium text-[var(--color-text-muted)] mb-4">
                  性能指标
                </h3>
                {performance ? (
                  <div className="space-y-4">
                    <div className="grid grid-cols-2 gap-4">
                      <div className="p-3 bg-[var(--color-background)] rounded-lg">
                        <p className="text-xs text-[var(--color-text-muted)]">TTFT (P95)</p>
                        <p className="text-lg font-bold text-[var(--color-text-primary)]">
                          {formatDuration(performance.ttft.p95)}
                        </p>
                      </div>
                      <div className="p-3 bg-[var(--color-background)] rounded-lg">
                        <p className="text-xs text-[var(--color-text-muted)]">TPOT (P95)</p>
                        <p className="text-lg font-bold text-[var(--color-text-primary)]">
                          {formatDuration(performance.tpot.p95)}
                        </p>
                      </div>
                    </div>
                    <div className="grid grid-cols-3 gap-4 text-center">
                      <div>
                        <p className="text-2xl font-bold text-[var(--color-text-primary)]">
                          {performance.request_count}
                        </p>
                        <p className="text-xs text-[var(--color-text-muted)]">请求数</p>
                      </div>
                      <div>
                        <p className="text-2xl font-bold text-[var(--color-error)]">
                          {performance.error_count}
                        </p>
                        <p className="text-xs text-[var(--color-text-muted)]">错误数</p>
                      </div>
                      <div>
                        <p className="text-2xl font-bold text-[var(--color-warning)]">
                          {performance.avg_loop_count.toFixed(1)}
                        </p>
                        <p className="text-xs text-[var(--color-text-muted)]">平均循环次数</p>
                      </div>
                    </div>
                  </div>
                ) : (
                  <p className="text-center text-[var(--color-text-muted)] py-8">
                    暂无性能数据
                  </p>
                )}
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
