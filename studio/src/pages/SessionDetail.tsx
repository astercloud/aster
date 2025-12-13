import { useParams, useNavigate } from 'react-router-dom';
import {
  ArrowLeft,
  MessageSquare,
  User,
  Bot,
  Clock,
  Hash,
} from 'lucide-react';
import { Header } from '../components/Header';
import { useSession } from '../hooks/useApi';
import {
  formatNumber,
  formatDate,
  formatRelativeTime,
  cn,
} from '../lib/utils';
import type { SessionMessage } from '../api/types';

export function SessionDetail() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { data: session, isLoading, refetch, isFetching } = useSession(id || '');

  if (!id) {
    return <div>Invalid session ID</div>;
  }

  return (
    <div className="flex flex-col h-full">
      <Header
        title={`Session: ${id.slice(0, 8)}...`}
        subtitle={session?.agent_name || session?.agent_id}
        isRefreshing={isFetching}
        onRefresh={() => refetch()}
      />

      <div className="flex-1 overflow-auto p-6">
        {/* Back Button */}
        <button
          onClick={() => navigate('/sessions')}
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
        ) : session ? (
          <div className="space-y-6">
            {/* Summary Card */}
            <div className="card">
              <div className="flex items-start justify-between mb-4">
                <div className="flex items-center gap-3">
                  <div className={cn(
                    'w-10 h-10 rounded-lg flex items-center justify-center',
                    session.status === 'active' && 'bg-[var(--color-accent)]/20',
                    session.status === 'completed' && 'bg-[var(--color-success)]/20',
                    session.status === 'suspended' && 'bg-[var(--color-warning)]/20'
                  )}>
                    <MessageSquare className={cn(
                      'w-5 h-5',
                      session.status === 'active' && 'text-[var(--color-accent)]',
                      session.status === 'completed' && 'text-[var(--color-success)]',
                      session.status === 'suspended' && 'text-[var(--color-warning)]'
                    )} />
                  </div>
                  <div>
                    <h2 className="text-xl font-semibold text-[var(--color-text-primary)]">
                      {session.agent_name || session.agent_id || 'Unknown Agent'}
                    </h2>
                    <p className="text-sm text-[var(--color-text-muted)]">
                      {formatDate(session.created_at)}
                    </p>
                  </div>
                </div>
                <div className={cn(
                  'px-3 py-1 rounded-full text-sm',
                  session.status === 'active' && 'bg-[var(--color-accent)]/20 text-[var(--color-accent)]',
                  session.status === 'completed' && 'bg-[var(--color-success)]/20 text-[var(--color-success)]',
                  session.status === 'suspended' && 'bg-[var(--color-warning)]/20 text-[var(--color-warning)]'
                )}>
                  {session.status === 'active' ? '活跃' : session.status === 'completed' ? '已完成' : '暂停'}
                </div>
              </div>

              <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                <div className="p-3 bg-[var(--color-background)] rounded-lg">
                  <div className="flex items-center gap-2 text-[var(--color-text-muted)] text-sm mb-1">
                    <Hash className="w-4 h-4" />
                    Session ID
                  </div>
                  <p className="font-mono text-sm text-[var(--color-text-primary)] truncate">
                    {session.id}
                  </p>
                </div>
                <div className="p-3 bg-[var(--color-background)] rounded-lg">
                  <div className="flex items-center gap-2 text-[var(--color-text-muted)] text-sm mb-1">
                    <MessageSquare className="w-4 h-4" />
                    消息数
                  </div>
                  <p className="font-mono text-sm text-[var(--color-text-primary)]">
                    {session.message_count}
                  </p>
                </div>
                <div className="p-3 bg-[var(--color-background)] rounded-lg">
                  <div className="flex items-center gap-2 text-[var(--color-text-muted)] text-sm mb-1">
                    <Clock className="w-4 h-4" />
                    Token 使用
                  </div>
                  <p className="font-mono text-sm text-[var(--color-text-primary)]">
                    {formatNumber(session.token_usage.input)} / {formatNumber(session.token_usage.output)}
                  </p>
                </div>
                <div className="p-3 bg-[var(--color-background)] rounded-lg">
                  <div className="flex items-center gap-2 text-[var(--color-text-muted)] text-sm mb-1">
                    最后更新
                  </div>
                  <p className="text-sm text-[var(--color-text-primary)]">
                    {formatRelativeTime(session.updated_at)}
                  </p>
                </div>
              </div>
            </div>

            {/* Messages Timeline */}
            <div className="card">
              <h3 className="text-sm font-medium text-[var(--color-text-muted)] mb-4">
                对话记录
              </h3>
              {session.messages && session.messages.length > 0 ? (
                <div className="space-y-4">
                  {session.messages.map((message, index) => (
                    <MessageBubble key={index} message={message} />
                  ))}
                </div>
              ) : (
                <p className="text-center text-[var(--color-text-muted)] py-8">
                  暂无消息记录
                </p>
              )}
            </div>
          </div>
        ) : (
          <div className="flex flex-col items-center justify-center h-64 text-[var(--color-text-muted)]">
            <MessageSquare className="w-12 h-12 mb-4" />
            <p>Session 未找到</p>
          </div>
        )}
      </div>
    </div>
  );
}

function MessageBubble({ message }: { message: SessionMessage }) {
  const isUser = message.role === 'user';

  return (
    <div className={cn(
      'flex gap-3',
      isUser && 'flex-row-reverse'
    )}>
      <div className={cn(
        'w-8 h-8 rounded-full flex items-center justify-center flex-shrink-0',
        isUser ? 'bg-[var(--color-primary)]/20' : 'bg-[var(--color-accent)]/20'
      )}>
        {isUser ? (
          <User className="w-4 h-4 text-[var(--color-primary)]" />
        ) : (
          <Bot className="w-4 h-4 text-[var(--color-accent)]" />
        )}
      </div>
      <div className={cn(
        'flex-1 max-w-[80%]',
        isUser && 'text-right'
      )}>
        <div className={cn(
          'inline-block p-3 rounded-lg text-sm',
          isUser
            ? 'bg-[var(--color-primary)] text-white rounded-tr-none'
            : 'bg-[var(--color-surface-elevated)] text-[var(--color-text-primary)] rounded-tl-none'
        )}>
          <p className="whitespace-pre-wrap break-words">
            {message.content || '(空消息)'}
          </p>
        </div>
        <p className={cn(
          'text-xs text-[var(--color-text-muted)] mt-1',
          isUser && 'text-right'
        )}>
          {formatRelativeTime(message.timestamp)}
        </p>
      </div>
    </div>
  );
}
