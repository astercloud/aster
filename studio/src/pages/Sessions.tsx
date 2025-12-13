import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Search,
  Filter,
  ChevronRight,
  MessageSquare,
  CheckCircle2,
  Clock,
  XCircle,
} from 'lucide-react';
import { Header } from '../components/Header';
import { useSessions } from '../hooks/useApi';
import {
  formatNumber,
  formatRelativeTime,
} from '../lib/utils';
import type { SessionStatus } from '../api/types';

export function Sessions() {
  const navigate = useNavigate();
  const [statusFilter, setStatusFilter] = useState<string>('');
  const [searchQuery, setSearchQuery] = useState('');

  const { data, isLoading, refetch, isFetching } = useSessions({
    status: statusFilter || undefined,
    limit: 50,
  });

  const filteredSessions = data?.sessions?.filter((session) => {
    if (!searchQuery) return true;
    return (
      session.id.toLowerCase().includes(searchQuery.toLowerCase()) ||
      session.agent_id?.toLowerCase().includes(searchQuery.toLowerCase()) ||
      session.agent_name?.toLowerCase().includes(searchQuery.toLowerCase())
    );
  });

  return (
    <div className="flex flex-col h-full">
      <Header
        title="Sessions"
        subtitle="会话历史记录"
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
                placeholder="搜索 Session ID、Agent..."
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
                <option value="active">活跃</option>
                <option value="completed">已完成</option>
                <option value="suspended">暂停</option>
              </select>
            </div>
          </div>
        </div>

        {/* Session List */}
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
          ) : filteredSessions && filteredSessions.length > 0 ? (
            <div className="space-y-2">
              {filteredSessions.map((session) => (
                <div
                  key={session.id}
                  onClick={() => navigate(`/sessions/${session.id}`)}
                  className="card cursor-pointer hover:border-[var(--color-primary)] transition-colors"
                >
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <StatusIcon status={session.status} />
                      <div>
                        <p className="font-medium text-[var(--color-text-primary)]">
                          {session.agent_name || session.agent_id || 'Unknown Agent'}
                        </p>
                        <p className="text-sm text-[var(--color-text-muted)]">
                          {session.id.slice(0, 8)}...
                        </p>
                      </div>
                    </div>

                    <div className="flex items-center gap-6">
                      <div className="text-right">
                        <div className="flex items-center gap-1 text-sm text-[var(--color-text-secondary)]">
                          <MessageSquare className="w-4 h-4" />
                          {session.message_count} 条消息
                        </div>
                        <p className="text-xs text-[var(--color-text-muted)]">
                          {formatRelativeTime(session.updated_at)}
                        </p>
                      </div>

                      <div className="text-right">
                        <p className="text-sm font-mono text-[var(--color-text-secondary)]">
                          {formatNumber(session.token_usage.total)} tokens
                        </p>
                        <p className="text-xs text-[var(--color-text-muted)]">
                          {formatNumber(session.token_usage.input)} / {formatNumber(session.token_usage.output)}
                        </p>
                      </div>

                      <ChevronRight className="w-5 h-5 text-[var(--color-text-muted)]" />
                    </div>
                  </div>
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
              <MessageSquare className="w-12 h-12 mb-4" />
              <p>暂无会话记录</p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

function StatusIcon({ status }: { status: SessionStatus }) {
  switch (status) {
    case 'active':
      return <Clock className="w-5 h-5 text-[var(--color-accent)]" />;
    case 'completed':
      return <CheckCircle2 className="w-5 h-5 text-[var(--color-success)]" />;
    case 'suspended':
      return <XCircle className="w-5 h-5 text-[var(--color-warning)]" />;
    default:
      return <div className="w-5 h-5 rounded-full bg-[var(--color-text-muted)]" />;
  }
}
