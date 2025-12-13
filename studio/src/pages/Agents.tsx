import { useState } from 'react';
import { Link } from 'react-router-dom';
import {
  Bot,
  Plus,
  Trash2,
  MoreVertical,
  Settings,
  Play,
  Archive,
  CheckCircle,
  XCircle,
  Clock,
  Search,
  Filter,
} from 'lucide-react';
import { Header } from '../components/Header';
import { useAgents, useDeleteAgent } from '../hooks/useApi';
import { formatRelativeTime, cn } from '../lib/utils';
import type { AgentRecord, AgentStatus } from '../api/types';

const STATUS_CONFIG: Record<AgentStatus, { label: string; color: string; icon: typeof CheckCircle }> = {
  active: { label: '运行中', color: 'text-[var(--color-success)]', icon: CheckCircle },
  disabled: { label: '已禁用', color: 'text-[var(--color-warning)]', icon: XCircle },
  archived: { label: '已归档', color: 'text-[var(--color-text-muted)]', icon: Archive },
};

export function Agents() {
  const [searchQuery, setSearchQuery] = useState('');
  const [statusFilter, setStatusFilter] = useState<AgentStatus | ''>('');

  const { data: agents, isLoading, error, refetch, isFetching } = useAgents();
  const deleteAgent = useDeleteAgent();

  // Filter agents
  const filteredAgents = (agents || []).filter((agent) => {
    // Search filter
    if (searchQuery) {
      const name = (agent.metadata?.name as string) || agent.id;
      const templateId = agent.config.template_id;
      const searchLower = searchQuery.toLowerCase();
      if (!name.toLowerCase().includes(searchLower) && !templateId.toLowerCase().includes(searchLower)) {
        return false;
      }
    }
    // Status filter
    if (statusFilter && agent.status !== statusFilter) {
      return false;
    }
    return true;
  });

  const handleDelete = async (agent: AgentRecord) => {
    if (confirm(`确定要删除 Agent "${(agent.metadata?.name as string) || agent.id}" 吗？`)) {
      try {
        await deleteAgent.mutateAsync(agent.id);
      } catch (e) {
        console.error('Failed to delete agent:', e);
      }
    }
  };

  return (
    <div className="flex flex-col h-full">
      <Header
        title="Agents"
        subtitle="Agent 配置管理"
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
                placeholder="搜索 Agent..."
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
                onChange={(e) => setStatusFilter(e.target.value as AgentStatus | '')}
                className="px-3 py-2 bg-[var(--color-surface)] border border-[var(--color-border)] rounded-lg text-sm text-[var(--color-text-primary)] focus:outline-none focus:border-[var(--color-primary)]"
              >
                <option value="">全部状态</option>
                <option value="active">运行中</option>
                <option value="disabled">已禁用</option>
                <option value="archived">已归档</option>
              </select>
            </div>

            {/* Count */}
            <span className="text-sm text-[var(--color-text-muted)]">
              {filteredAgents.length} 个 Agent
            </span>
          </div>

          {/* Create Button */}
          <Link
            to="/agents/new"
            className="flex items-center gap-2 px-4 py-2 bg-[var(--color-primary)] text-white rounded-lg text-sm font-medium hover:bg-[var(--color-primary-hover)] transition-colors"
          >
            <Plus className="w-4 h-4" />
            创建 Agent
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
          ) : filteredAgents.length === 0 ? (
            <div className="flex flex-col items-center justify-center h-64 text-[var(--color-text-muted)]">
              <Bot className="w-12 h-12 mb-4" />
              <p>{agents?.length === 0 ? '暂无 Agent' : '没有匹配的 Agent'}</p>
              {agents?.length === 0 && (
                <Link
                  to="/agents/new"
                  className="mt-4 flex items-center gap-2 px-4 py-2 bg-[var(--color-primary)] text-white rounded-lg text-sm font-medium hover:bg-[var(--color-primary-hover)] transition-colors"
                >
                  <Plus className="w-4 h-4" />
                  创建第一个 Agent
                </Link>
              )}
            </div>
          ) : (
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
              {filteredAgents.map((agent) => (
                <AgentCard key={agent.id} agent={agent} onDelete={handleDelete} />
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

interface AgentCardProps {
  agent: AgentRecord;
  onDelete: (agent: AgentRecord) => void;
}

function AgentCard({ agent, onDelete }: AgentCardProps) {
  const [showMenu, setShowMenu] = useState(false);

  const name = (agent.metadata?.name as string) || agent.id;
  const statusConfig = STATUS_CONFIG[agent.status];
  const StatusIcon = statusConfig.icon;

  return (
    <div className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-lg p-4 hover:border-[var(--color-primary)]/50 transition-colors">
      {/* Header */}
      <div className="flex items-start justify-between mb-3">
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 rounded-lg bg-[var(--color-primary)]/10 flex items-center justify-center">
            <Bot className="w-5 h-5 text-[var(--color-primary)]" />
          </div>
          <div>
            <h3 className="font-medium text-[var(--color-text-primary)] line-clamp-1">
              {name}
            </h3>
            <p className="text-xs text-[var(--color-text-muted)] font-mono">
              {agent.id.slice(0, 8)}...
            </p>
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
              <div className="absolute right-0 top-full mt-1 w-40 bg-[var(--color-surface-elevated)] border border-[var(--color-border)] rounded-lg shadow-lg z-20 py-1">
                <Link
                  to={`/agents/${agent.id}`}
                  className="flex items-center gap-2 px-3 py-2 text-sm text-[var(--color-text-secondary)] hover:bg-[var(--color-surface)] transition-colors"
                  onClick={() => setShowMenu(false)}
                >
                  <Settings className="w-4 h-4" />
                  查看详情
                </Link>
                <button
                  className="flex items-center gap-2 px-3 py-2 text-sm text-[var(--color-text-secondary)] hover:bg-[var(--color-surface)] transition-colors w-full"
                  onClick={() => {
                    setShowMenu(false);
                    // TODO: Run agent
                  }}
                >
                  <Play className="w-4 h-4" />
                  运行
                </button>
                <hr className="my-1 border-[var(--color-border)]" />
                <button
                  className="flex items-center gap-2 px-3 py-2 text-sm text-[var(--color-error)] hover:bg-[var(--color-surface)] transition-colors w-full"
                  onClick={() => {
                    setShowMenu(false);
                    onDelete(agent);
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

      {/* Status */}
      <div className="flex items-center gap-2 mb-3">
        <StatusIcon className={cn('w-4 h-4', statusConfig.color)} />
        <span className={cn('text-sm', statusConfig.color)}>{statusConfig.label}</span>
      </div>

      {/* Info */}
      <div className="space-y-2 text-sm">
        <div className="flex items-center justify-between">
          <span className="text-[var(--color-text-muted)]">模板</span>
          <span className="text-[var(--color-text-secondary)] font-mono text-xs">
            {agent.config.template_id}
          </span>
        </div>

        {agent.config.model_config?.model && (
          <div className="flex items-center justify-between">
            <span className="text-[var(--color-text-muted)]">模型</span>
            <span className="text-[var(--color-text-secondary)] text-xs">
              {agent.config.model_config.model}
            </span>
          </div>
        )}

        {agent.config.middlewares && agent.config.middlewares.length > 0 && (
          <div className="flex items-center justify-between">
            <span className="text-[var(--color-text-muted)]">中间件</span>
            <span className="text-[var(--color-text-secondary)] text-xs">
              {agent.config.middlewares.length} 个
            </span>
          </div>
        )}
      </div>

      {/* Footer */}
      <div className="mt-4 pt-3 border-t border-[var(--color-border)] flex items-center justify-between text-xs text-[var(--color-text-muted)]">
        <span className="flex items-center gap-1">
          <Clock className="w-3 h-3" />
          {formatRelativeTime(agent.created_at)}
        </span>
        <Link
          to={`/agents/${agent.id}`}
          className="text-[var(--color-primary)] hover:underline"
        >
          查看详情
        </Link>
      </div>
    </div>
  );
}
