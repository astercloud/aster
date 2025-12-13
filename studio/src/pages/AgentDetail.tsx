import { useState } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import {
  Bot,
  ArrowLeft,
  Save,
  Trash2,
  Settings,
  Code,
  Layers,
  Cpu,
  Box,
  CheckCircle,
  XCircle,
  Archive,
  Pencil,
  X,
} from 'lucide-react';
import { Header } from '../components/Header';
import { useAgent, useUpdateAgent, useDeleteAgent } from '../hooks/useApi';
import { formatRelativeTime, cn } from '../lib/utils';
import type { AgentStatus } from '../api/types';

const STATUS_CONFIG: Record<AgentStatus, { label: string; color: string; bgColor: string; icon: typeof CheckCircle }> = {
  active: { label: '运行中', color: 'text-[var(--color-success)]', bgColor: 'bg-[var(--color-success)]/10', icon: CheckCircle },
  disabled: { label: '已禁用', color: 'text-[var(--color-warning)]', bgColor: 'bg-[var(--color-warning)]/10', icon: XCircle },
  archived: { label: '已归档', color: 'text-[var(--color-text-muted)]', bgColor: 'bg-[var(--color-surface-elevated)]', icon: Archive },
};

export function AgentDetail() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [isEditing, setIsEditing] = useState(false);
  const [editName, setEditName] = useState('');
  const [activeTab, setActiveTab] = useState<'overview' | 'config' | 'middlewares'>('overview');

  const { data: agent, isLoading, error, refetch, isFetching } = useAgent(id || '');
  const updateAgent = useUpdateAgent();
  const deleteAgent = useDeleteAgent();

  const handleSaveName = async () => {
    if (!id || !editName.trim()) return;

    try {
      await updateAgent.mutateAsync({
        id,
        req: { name: editName.trim() },
      });
      setIsEditing(false);
    } catch (e) {
      console.error('Failed to update agent:', e);
    }
  };

  const handleDelete = async () => {
    if (!agent) return;

    const name = (agent.metadata?.name as string) || agent.id;
    if (confirm(`确定要删除 Agent "${name}" 吗？此操作不可恢复。`)) {
      try {
        await deleteAgent.mutateAsync(agent.id);
        navigate('/agents');
      } catch (e) {
        console.error('Failed to delete agent:', e);
      }
    }
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="animate-spin w-8 h-8 border-2 border-[var(--color-primary)] border-t-transparent rounded-full" />
      </div>
    );
  }

  if (error || !agent) {
    return (
      <div className="flex flex-col items-center justify-center h-full text-[var(--color-error)]">
        <XCircle className="w-12 h-12 mb-4" />
        <p>加载失败: {(error as Error)?.message || 'Agent 不存在'}</p>
        <Link
          to="/agents"
          className="mt-4 text-[var(--color-primary)] hover:underline"
        >
          返回列表
        </Link>
      </div>
    );
  }

  const name = (agent.metadata?.name as string) || agent.id;
  const statusConfig = STATUS_CONFIG[agent.status];
  const StatusIcon = statusConfig.icon;

  return (
    <div className="flex flex-col h-full">
      <Header
        title="Agent 详情"
        subtitle={name}
        isRefreshing={isFetching}
        onRefresh={() => refetch()}
      />

      <div className="flex-1 overflow-auto">
        {/* Back link */}
        <div className="px-6 py-4 border-b border-[var(--color-border)]">
          <Link
            to="/agents"
            className="inline-flex items-center gap-2 text-sm text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] transition-colors"
          >
            <ArrowLeft className="w-4 h-4" />
            返回列表
          </Link>
        </div>

        {/* Agent Header */}
        <div className="px-6 py-6 border-b border-[var(--color-border)] bg-[var(--color-surface)]">
          <div className="flex items-start justify-between">
            <div className="flex items-center gap-4">
              <div className="w-16 h-16 rounded-xl bg-[var(--color-primary)]/10 flex items-center justify-center">
                <Bot className="w-8 h-8 text-[var(--color-primary)]" />
              </div>
              <div>
                {isEditing ? (
                  <div className="flex items-center gap-2">
                    <input
                      type="text"
                      value={editName}
                      onChange={(e) => setEditName(e.target.value)}
                      className="px-3 py-1.5 bg-[var(--color-background)] border border-[var(--color-border)] rounded-lg text-lg font-semibold text-[var(--color-text-primary)] focus:outline-none focus:border-[var(--color-primary)]"
                      autoFocus
                    />
                    <button
                      onClick={handleSaveName}
                      disabled={updateAgent.isPending}
                      className="p-1.5 rounded-lg bg-[var(--color-primary)] text-white hover:bg-[var(--color-primary-hover)] transition-colors disabled:opacity-50"
                    >
                      <Save className="w-4 h-4" />
                    </button>
                    <button
                      onClick={() => setIsEditing(false)}
                      className="p-1.5 rounded-lg bg-[var(--color-surface-elevated)] text-[var(--color-text-muted)] hover:bg-[var(--color-border)] transition-colors"
                    >
                      <X className="w-4 h-4" />
                    </button>
                  </div>
                ) : (
                  <div className="flex items-center gap-2">
                    <h1 className="text-xl font-semibold text-[var(--color-text-primary)]">
                      {name}
                    </h1>
                    <button
                      onClick={() => {
                        setEditName(name);
                        setIsEditing(true);
                      }}
                      className="p-1 rounded hover:bg-[var(--color-surface-elevated)] transition-colors"
                    >
                      <Pencil className="w-4 h-4 text-[var(--color-text-muted)]" />
                    </button>
                  </div>
                )}
                <p className="text-sm text-[var(--color-text-muted)] font-mono mt-1">
                  {agent.id}
                </p>
              </div>
            </div>

            <div className="flex items-center gap-3">
              <div className={cn('flex items-center gap-2 px-3 py-1.5 rounded-lg', statusConfig.bgColor)}>
                <StatusIcon className={cn('w-4 h-4', statusConfig.color)} />
                <span className={cn('text-sm font-medium', statusConfig.color)}>
                  {statusConfig.label}
                </span>
              </div>
              <button
                onClick={handleDelete}
                disabled={deleteAgent.isPending}
                className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-[var(--color-error)]/10 text-[var(--color-error)] hover:bg-[var(--color-error)]/20 transition-colors disabled:opacity-50"
              >
                <Trash2 className="w-4 h-4" />
                删除
              </button>
            </div>
          </div>

          {/* Meta info */}
          <div className="flex items-center gap-6 mt-4 text-sm text-[var(--color-text-muted)]">
            <span>创建于 {formatRelativeTime(agent.created_at)}</span>
            <span>更新于 {formatRelativeTime(agent.updated_at)}</span>
          </div>
        </div>

        {/* Tabs */}
        <div className="border-b border-[var(--color-border)]">
          <div className="flex gap-1 px-6">
            {[
              { id: 'overview', label: '概览', icon: Settings },
              { id: 'config', label: '配置', icon: Code },
              { id: 'middlewares', label: '中间件', icon: Layers },
            ].map((tab) => (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id as typeof activeTab)}
                className={cn(
                  'flex items-center gap-2 px-4 py-3 text-sm font-medium border-b-2 transition-colors',
                  activeTab === tab.id
                    ? 'border-[var(--color-primary)] text-[var(--color-primary)]'
                    : 'border-transparent text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)]'
                )}
              >
                <tab.icon className="w-4 h-4" />
                {tab.label}
              </button>
            ))}
          </div>
        </div>

        {/* Tab Content */}
        <div className="p-6">
          {activeTab === 'overview' && <OverviewTab agent={agent} />}
          {activeTab === 'config' && <ConfigTab agent={agent} />}
          {activeTab === 'middlewares' && <MiddlewaresTab agent={agent} />}
        </div>
      </div>
    </div>
  );
}

function OverviewTab({ agent }: { agent: NonNullable<ReturnType<typeof useAgent>['data']> }) {
  const config = agent.config;

  return (
    <div className="grid gap-6 md:grid-cols-2">
      {/* Template Info */}
      <div className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-lg p-4">
        <div className="flex items-center gap-2 mb-4">
          <Box className="w-5 h-5 text-[var(--color-primary)]" />
          <h3 className="font-medium text-[var(--color-text-primary)]">模板</h3>
        </div>
        <dl className="space-y-3">
          <div>
            <dt className="text-xs text-[var(--color-text-muted)] uppercase tracking-wider">模板 ID</dt>
            <dd className="mt-1 text-sm text-[var(--color-text-primary)] font-mono">
              {config.template_id}
            </dd>
          </div>
          {config.template_version && (
            <div>
              <dt className="text-xs text-[var(--color-text-muted)] uppercase tracking-wider">版本</dt>
              <dd className="mt-1 text-sm text-[var(--color-text-primary)]">
                {config.template_version}
              </dd>
            </div>
          )}
        </dl>
      </div>

      {/* Model Info */}
      <div className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-lg p-4">
        <div className="flex items-center gap-2 mb-4">
          <Cpu className="w-5 h-5 text-[var(--color-primary)]" />
          <h3 className="font-medium text-[var(--color-text-primary)]">模型配置</h3>
        </div>
        {config.model_config ? (
          <dl className="space-y-3">
            {config.model_config.provider && (
              <div>
                <dt className="text-xs text-[var(--color-text-muted)] uppercase tracking-wider">提供商</dt>
                <dd className="mt-1 text-sm text-[var(--color-text-primary)]">
                  {config.model_config.provider}
                </dd>
              </div>
            )}
            {config.model_config.model && (
              <div>
                <dt className="text-xs text-[var(--color-text-muted)] uppercase tracking-wider">模型</dt>
                <dd className="mt-1 text-sm text-[var(--color-text-primary)]">
                  {config.model_config.model}
                </dd>
              </div>
            )}
            {config.model_config.temperature !== undefined && (
              <div>
                <dt className="text-xs text-[var(--color-text-muted)] uppercase tracking-wider">Temperature</dt>
                <dd className="mt-1 text-sm text-[var(--color-text-primary)]">
                  {config.model_config.temperature}
                </dd>
              </div>
            )}
            {config.model_config.max_tokens && (
              <div>
                <dt className="text-xs text-[var(--color-text-muted)] uppercase tracking-wider">Max Tokens</dt>
                <dd className="mt-1 text-sm text-[var(--color-text-primary)]">
                  {config.model_config.max_tokens.toLocaleString()}
                </dd>
              </div>
            )}
          </dl>
        ) : (
          <p className="text-sm text-[var(--color-text-muted)]">使用模板默认配置</p>
        )}
      </div>

      {/* Sandbox Info */}
      {config.sandbox && (
        <div className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-lg p-4">
          <div className="flex items-center gap-2 mb-4">
            <Box className="w-5 h-5 text-[var(--color-primary)]" />
            <h3 className="font-medium text-[var(--color-text-primary)]">沙箱配置</h3>
          </div>
          <dl className="space-y-3">
            {config.sandbox.type && (
              <div>
                <dt className="text-xs text-[var(--color-text-muted)] uppercase tracking-wider">类型</dt>
                <dd className="mt-1 text-sm text-[var(--color-text-primary)]">
                  {config.sandbox.type}
                </dd>
              </div>
            )}
            {config.sandbox.working_dir && (
              <div>
                <dt className="text-xs text-[var(--color-text-muted)] uppercase tracking-wider">工作目录</dt>
                <dd className="mt-1 text-sm text-[var(--color-text-primary)] font-mono">
                  {config.sandbox.working_dir}
                </dd>
              </div>
            )}
          </dl>
        </div>
      )}

      {/* Metadata */}
      {agent.metadata && Object.keys(agent.metadata).length > 0 && (
        <div className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-lg p-4">
          <div className="flex items-center gap-2 mb-4">
            <Settings className="w-5 h-5 text-[var(--color-primary)]" />
            <h3 className="font-medium text-[var(--color-text-primary)]">元数据</h3>
          </div>
          <dl className="space-y-3">
            {Object.entries(agent.metadata).map(([key, value]) => (
              <div key={key}>
                <dt className="text-xs text-[var(--color-text-muted)] uppercase tracking-wider">{key}</dt>
                <dd className="mt-1 text-sm text-[var(--color-text-primary)]">
                  {typeof value === 'object' ? JSON.stringify(value) : String(value)}
                </dd>
              </div>
            ))}
          </dl>
        </div>
      )}
    </div>
  );
}

function ConfigTab({ agent }: { agent: NonNullable<ReturnType<typeof useAgent>['data']> }) {
  return (
    <div className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-lg p-4">
      <h3 className="font-medium text-[var(--color-text-primary)] mb-4">完整配置 (JSON)</h3>
      <pre className="bg-[var(--color-background)] p-4 rounded-lg overflow-auto max-h-[600px] text-sm font-mono text-[var(--color-text-secondary)]">
        {JSON.stringify(agent.config, null, 2)}
      </pre>
    </div>
  );
}

function MiddlewaresTab({ agent }: { agent: NonNullable<ReturnType<typeof useAgent>['data']> }) {
  const middlewares = agent.config.middlewares || [];
  const middlewareConfig = agent.config.middleware_config || {};

  if (middlewares.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-[var(--color-text-muted)]">
        <Layers className="w-12 h-12 mb-4" />
        <p>未配置中间件</p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {middlewares.map((middleware, index) => {
        const config = middlewareConfig[middleware];
        return (
          <div
            key={middleware}
            className="bg-[var(--color-surface)] border border-[var(--color-border)] rounded-lg p-4"
          >
            <div className="flex items-center gap-3 mb-3">
              <div className="w-8 h-8 rounded-lg bg-[var(--color-primary)]/10 flex items-center justify-center text-sm font-medium text-[var(--color-primary)]">
                {index + 1}
              </div>
              <h3 className="font-medium text-[var(--color-text-primary)]">{middleware}</h3>
            </div>
            {config ? (
              <pre className="bg-[var(--color-background)] p-3 rounded-lg text-xs font-mono text-[var(--color-text-secondary)] overflow-auto">
                {JSON.stringify(config, null, 2)}
              </pre>
            ) : (
              <p className="text-sm text-[var(--color-text-muted)]">默认配置</p>
            )}
          </div>
        );
      })}
    </div>
  );
}
