import { useState } from 'react';
import { Link } from 'react-router-dom';
import {
  FlaskConical,
  Plus,
  Trash2,
  MoreVertical,
  Play,
  CheckCircle,
  XCircle,
  Clock,
  Loader2,
  Search,
  Filter,
  FileText,
  MessageSquare,
  Layers,
  Settings,
  TrendingUp,
} from 'lucide-react';
import { Header } from '../components/Header';
import { useEvals, useDeleteEval, useRunTextEval } from '../hooks/useApi';
import { formatRelativeTime, cn } from '../lib/utils';
import type { EvalRecord, EvalStatus, EvalType } from '../api/types';

const STATUS_CONFIG: Record<EvalStatus, { label: string; color: string; icon: typeof CheckCircle }> = {
  pending: { label: '等待中', color: 'text-[var(--color-text-muted)]', icon: Clock },
  running: { label: '运行中', color: 'text-[var(--color-primary)]', icon: Loader2 },
  completed: { label: '已完成', color: 'text-[var(--color-success)]', icon: CheckCircle },
  failed: { label: '失败', color: 'text-[var(--color-error)]', icon: XCircle },
};

const TYPE_CONFIG: Record<EvalType, { label: string; icon: typeof FileText }> = {
  text: { label: '文本评估', icon: FileText },
  session: { label: '会话评估', icon: MessageSquare },
  batch: { label: '批量评估', icon: Layers },
  custom: { label: '自定义评估', icon: Settings },
};

export function Evaluations() {
  const [searchQuery, setSearchQuery] = useState('');
  const [statusFilter, setStatusFilter] = useState<EvalStatus | ''>('');
  const [typeFilter, setTypeFilter] = useState<EvalType | ''>('');
  const [showNewEvalModal, setShowNewEvalModal] = useState(false);

  const { data, isLoading, error, refetch, isFetching } = useEvals({
    status: statusFilter || undefined,
    type: typeFilter || undefined,
  });
  const deleteEval = useDeleteEval();

  const evals = data?.evals || [];

  // Filter by search
  const filteredEvals = evals.filter((evalItem) => {
    if (searchQuery) {
      const searchLower = searchQuery.toLowerCase();
      if (
        !evalItem.name.toLowerCase().includes(searchLower) &&
        !evalItem.id.toLowerCase().includes(searchLower)
      ) {
        return false;
      }
    }
    return true;
  });

  const handleDelete = async (evalItem: EvalRecord) => {
    if (confirm(`确定要删除评估 "${evalItem.name}" 吗？`)) {
      try {
        await deleteEval.mutateAsync(evalItem.id);
      } catch (e) {
        console.error('Failed to delete eval:', e);
      }
    }
  };

  return (
    <div className="flex flex-col h-full">
      <Header
        title="Evaluations"
        subtitle="评估管理"
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
                placeholder="搜索评估..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="pl-9 pr-4 py-2 bg-[var(--color-surface)] border border-[var(--color-border)] rounded-lg text-sm text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] focus:outline-none focus:border-[var(--color-primary)] w-64"
              />
            </div>

            {/* Status Filter */}
            <div className="flex items-center gap-2">
              <Filter className="w-4 h-4 text-[var(--color-text-muted)]" />
              <select
                value={statusFilter}
                onChange={(e) => setStatusFilter(e.target.value as EvalStatus | '')}
                className="px-3 py-2 bg-[var(--color-surface)] border border-[var(--color-border)] rounded-lg text-sm text-[var(--color-text-primary)] focus:outline-none focus:border-[var(--color-primary)]"
              >
                <option value="">全部状态</option>
                <option value="pending">等待中</option>
                <option value="running">运行中</option>
                <option value="completed">已完成</option>
                <option value="failed">失败</option>
              </select>
            </div>

            {/* Type Filter */}
            <select
              value={typeFilter}
              onChange={(e) => setTypeFilter(e.target.value as EvalType | '')}
              className="px-3 py-2 bg-[var(--color-surface)] border border-[var(--color-border)] rounded-lg text-sm text-[var(--color-text-primary)] focus:outline-none focus:border-[var(--color-primary)]"
            >
              <option value="">全部类型</option>
              <option value="text">文本评估</option>
              <option value="session">会话评估</option>
              <option value="batch">批量评估</option>
              <option value="custom">自定义评估</option>
            </select>

            {/* Count */}
            <span className="text-sm text-[var(--color-text-muted)]">
              {filteredEvals.length} 个评估
            </span>
          </div>

          {/* Create Button */}
          <button
            onClick={() => setShowNewEvalModal(true)}
            className="flex items-center gap-2 px-4 py-2 bg-[var(--color-primary)] text-white rounded-lg text-sm font-medium hover:bg-[var(--color-primary-hover)] transition-colors"
          >
            <Plus className="w-4 h-4" />
            新建评估
          </button>
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
          ) : filteredEvals.length === 0 ? (
            <div className="flex flex-col items-center justify-center h-64 text-[var(--color-text-muted)]">
              <FlaskConical className="w-12 h-12 mb-4" />
              <p>{evals.length === 0 ? '暂无评估' : '没有匹配的评估'}</p>
              {evals.length === 0 && (
                <button
                  onClick={() => setShowNewEvalModal(true)}
                  className="mt-4 flex items-center gap-2 px-4 py-2 bg-[var(--color-primary)] text-white rounded-lg text-sm font-medium hover:bg-[var(--color-primary-hover)] transition-colors"
                >
                  <Plus className="w-4 h-4" />
                  创建第一个评估
                </button>
              )}
            </div>
          ) : (
            <div className="space-y-3">
              {filteredEvals.map((evalItem) => (
                <EvalCard key={evalItem.id} evalItem={evalItem} onDelete={handleDelete} />
              ))}
            </div>
          )}
        </div>
      </div>

      {/* New Eval Modal */}
      {showNewEvalModal && (
        <NewEvalModal onClose={() => setShowNewEvalModal(false)} />
      )}
    </div>
  );
}

interface EvalCardProps {
  evalItem: EvalRecord;
  onDelete: (evalItem: EvalRecord) => void;
}

function EvalCard({ evalItem, onDelete }: EvalCardProps) {
  const [showMenu, setShowMenu] = useState(false);

  const statusConfig = STATUS_CONFIG[evalItem.status];
  const typeConfig = TYPE_CONFIG[evalItem.type];
  const StatusIcon = statusConfig.icon;
  const TypeIcon = typeConfig.icon;

  const scoreColor = evalItem.score !== undefined
    ? evalItem.score >= 0.8 ? 'text-[var(--color-success)]'
      : evalItem.score >= 0.5 ? 'text-[var(--color-warning)]'
      : 'text-[var(--color-error)]'
    : '';

  return (
    <div className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-lg p-4 hover:border-[var(--color-primary)]/50 transition-colors">
      <div className="flex items-start justify-between">
        {/* Left */}
        <div className="flex items-start gap-4 flex-1 min-w-0">
          {/* Icon */}
          <div className="w-10 h-10 rounded-lg bg-[var(--color-primary)]/10 flex items-center justify-center flex-shrink-0">
            <TypeIcon className="w-5 h-5 text-[var(--color-primary)]" />
          </div>

          {/* Info */}
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2">
              <h3 className="font-medium text-[var(--color-text-primary)] truncate">
                {evalItem.name}
              </h3>
              <span className="px-2 py-0.5 text-xs rounded-full bg-[var(--color-surface-elevated)] text-[var(--color-text-muted)]">
                {typeConfig.label}
              </span>
            </div>
            <p className="text-xs text-[var(--color-text-muted)] font-mono mt-1">
              {evalItem.id.slice(0, 12)}...
            </p>
          </div>
        </div>

        {/* Right */}
        <div className="flex items-center gap-4">
          {/* Score */}
          {evalItem.score !== undefined && (
            <div className="text-right">
              <div className={cn('text-lg font-semibold', scoreColor)}>
                {(evalItem.score * 100).toFixed(1)}%
              </div>
              <div className="text-xs text-[var(--color-text-muted)]">评分</div>
            </div>
          )}

          {/* Status */}
          <div className="flex items-center gap-2 min-w-[80px]">
            <StatusIcon className={cn(
              'w-4 h-4',
              statusConfig.color,
              evalItem.status === 'running' && 'animate-spin'
            )} />
            <span className={cn('text-sm', statusConfig.color)}>
              {statusConfig.label}
            </span>
          </div>

          {/* Duration */}
          {evalItem.duration_ms !== undefined && (
            <div className="text-right min-w-[60px]">
              <div className="text-sm text-[var(--color-text-secondary)]">
                {evalItem.duration_ms}ms
              </div>
              <div className="text-xs text-[var(--color-text-muted)]">耗时</div>
            </div>
          )}

          {/* Time */}
          <div className="text-right min-w-[80px]">
            <div className="text-sm text-[var(--color-text-secondary)]">
              {formatRelativeTime(evalItem.started_at)}
            </div>
          </div>

          {/* Menu */}
          <div className="relative">
            <button
              onClick={() => setShowMenu(!showMenu)}
              className="p-1 rounded hover:bg-[var(--color-surface-elevated)] transition-colors"
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
                    to={`/evaluations/${evalItem.id}`}
                    className="flex items-center gap-2 px-3 py-2 text-sm text-[var(--color-text-secondary)] hover:bg-[var(--color-surface)] transition-colors"
                    onClick={() => setShowMenu(false)}
                  >
                    <TrendingUp className="w-4 h-4" />
                    查看详情
                  </Link>
                  <hr className="my-1 border-[var(--color-border)]" />
                  <button
                    className="flex items-center gap-2 px-3 py-2 text-sm text-[var(--color-error)] hover:bg-[var(--color-surface)] transition-colors w-full"
                    onClick={() => {
                      setShowMenu(false);
                      onDelete(evalItem);
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

      {/* Metrics */}
      {evalItem.metrics && (
        <div className="mt-3 pt-3 border-t border-[var(--color-border)] flex items-center gap-6 text-sm">
          {evalItem.metrics.accuracy !== undefined && (
            <div className="flex items-center gap-1">
              <span className="text-[var(--color-text-muted)]">准确率:</span>
              <span className="text-[var(--color-text-secondary)]">
                {(evalItem.metrics.accuracy * 100).toFixed(1)}%
              </span>
            </div>
          )}
          {evalItem.metrics.latency_ms !== undefined && (
            <div className="flex items-center gap-1">
              <span className="text-[var(--color-text-muted)]">延迟:</span>
              <span className="text-[var(--color-text-secondary)]">
                {evalItem.metrics.latency_ms}ms
              </span>
            </div>
          )}
          {evalItem.metrics.token_count !== undefined && (
            <div className="flex items-center gap-1">
              <span className="text-[var(--color-text-muted)]">Token:</span>
              <span className="text-[var(--color-text-secondary)]">
                {evalItem.metrics.token_count.toLocaleString()}
              </span>
            </div>
          )}
          {evalItem.metrics.cost !== undefined && (
            <div className="flex items-center gap-1">
              <span className="text-[var(--color-text-muted)]">成本:</span>
              <span className="text-[var(--color-text-secondary)]">
                ${evalItem.metrics.cost.toFixed(4)}
              </span>
            </div>
          )}
        </div>
      )}

      {/* Error */}
      {evalItem.error && (
        <div className="mt-3 pt-3 border-t border-[var(--color-border)]">
          <div className="text-sm text-[var(--color-error)] bg-[var(--color-error)]/10 px-3 py-2 rounded">
            {evalItem.error}
          </div>
        </div>
      )}
    </div>
  );
}

interface NewEvalModalProps {
  onClose: () => void;
}

function NewEvalModal({ onClose }: NewEvalModalProps) {
  const [evalType, setEvalType] = useState<EvalType>('text');
  const [name, setName] = useState('');
  const [prompt, setPrompt] = useState('');
  const [expected, setExpected] = useState('');
  const [sessionId, setSessionId] = useState('');

  const runTextEval = useRunTextEval();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (evalType === 'text') {
      try {
        await runTextEval.mutateAsync({
          name: name || undefined,
          prompt,
          expected: expected || undefined,
        });
        onClose();
      } catch (error) {
        console.error('Failed to run eval:', error);
      }
    }
    // TODO: Handle other eval types
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/50" onClick={onClose} />
      <div className="relative bg-[var(--color-surface)] border border-[var(--color-border)] rounded-lg shadow-xl w-full max-w-lg mx-4">
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b border-[var(--color-border)]">
          <h2 className="text-lg font-semibold text-[var(--color-text-primary)]">
            新建评估
          </h2>
          <button
            onClick={onClose}
            className="p-1 rounded hover:bg-[var(--color-surface-elevated)] transition-colors"
          >
            <XCircle className="w-5 h-5 text-[var(--color-text-muted)]" />
          </button>
        </div>

        {/* Body */}
        <form onSubmit={handleSubmit} className="p-4 space-y-4">
          {/* Type Selection */}
          <div>
            <label className="block text-sm font-medium text-[var(--color-text-secondary)] mb-2">
              评估类型
            </label>
            <div className="grid grid-cols-2 gap-2">
              {(Object.entries(TYPE_CONFIG) as [EvalType, typeof TYPE_CONFIG.text][]).map(([type, config]) => {
                const Icon = config.icon;
                return (
                  <button
                    key={type}
                    type="button"
                    onClick={() => setEvalType(type)}
                    className={cn(
                      'flex items-center gap-2 p-3 rounded-lg border transition-colors',
                      evalType === type
                        ? 'border-[var(--color-primary)] bg-[var(--color-primary)]/10 text-[var(--color-primary)]'
                        : 'border-[var(--color-border)] text-[var(--color-text-secondary)] hover:bg-[var(--color-surface-elevated)]'
                    )}
                  >
                    <Icon className="w-4 h-4" />
                    <span className="text-sm">{config.label}</span>
                  </button>
                );
              })}
            </div>
          </div>

          {/* Name */}
          <div>
            <label className="block text-sm font-medium text-[var(--color-text-secondary)] mb-2">
              名称 (可选)
            </label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="评估名称"
              className="w-full px-3 py-2 bg-[var(--color-surface)] border border-[var(--color-border)] rounded-lg text-sm text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] focus:outline-none focus:border-[var(--color-primary)]"
            />
          </div>

          {/* Type-specific fields */}
          {evalType === 'text' && (
            <>
              <div>
                <label className="block text-sm font-medium text-[var(--color-text-secondary)] mb-2">
                  Prompt *
                </label>
                <textarea
                  value={prompt}
                  onChange={(e) => setPrompt(e.target.value)}
                  placeholder="输入要评估的 prompt"
                  required
                  rows={3}
                  className="w-full px-3 py-2 bg-[var(--color-surface)] border border-[var(--color-border)] rounded-lg text-sm text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] focus:outline-none focus:border-[var(--color-primary)] resize-none"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-[var(--color-text-secondary)] mb-2">
                  预期输出 (可选)
                </label>
                <textarea
                  value={expected}
                  onChange={(e) => setExpected(e.target.value)}
                  placeholder="预期的输出结果"
                  rows={2}
                  className="w-full px-3 py-2 bg-[var(--color-surface)] border border-[var(--color-border)] rounded-lg text-sm text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] focus:outline-none focus:border-[var(--color-primary)] resize-none"
                />
              </div>
            </>
          )}

          {evalType === 'session' && (
            <div>
              <label className="block text-sm font-medium text-[var(--color-text-secondary)] mb-2">
                Session ID *
              </label>
              <input
                type="text"
                value={sessionId}
                onChange={(e) => setSessionId(e.target.value)}
                placeholder="要评估的会话 ID"
                required
                className="w-full px-3 py-2 bg-[var(--color-surface)] border border-[var(--color-border)] rounded-lg text-sm text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] focus:outline-none focus:border-[var(--color-primary)]"
              />
            </div>
          )}

          {(evalType === 'batch' || evalType === 'custom') && (
            <div className="text-center py-8 text-[var(--color-text-muted)]">
              <Settings className="w-8 h-8 mx-auto mb-2" />
              <p className="text-sm">该评估类型暂未实现</p>
            </div>
          )}

          {/* Actions */}
          <div className="flex items-center justify-end gap-3 pt-4 border-t border-[var(--color-border)]">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 text-sm text-[var(--color-text-secondary)] hover:bg-[var(--color-surface-elevated)] rounded-lg transition-colors"
            >
              取消
            </button>
            <button
              type="submit"
              disabled={runTextEval.isPending || (evalType === 'text' && !prompt)}
              className="flex items-center gap-2 px-4 py-2 bg-[var(--color-primary)] text-white rounded-lg text-sm font-medium hover:bg-[var(--color-primary-hover)] transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {runTextEval.isPending ? (
                <Loader2 className="w-4 h-4 animate-spin" />
              ) : (
                <Play className="w-4 h-4" />
              )}
              运行评估
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
