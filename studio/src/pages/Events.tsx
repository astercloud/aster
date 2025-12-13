import { useState, useEffect, useRef, useCallback } from 'react';
import {
  Zap,
  Filter,
  Pause,
  Play,
  Trash2,
  Wifi,
  WifiOff,
  RefreshCw,
  ChevronDown,
  ChevronRight,
  AlertCircle,
  CheckCircle2,
  Clock,
  Activity,
} from 'lucide-react';
import { Header } from '../components/Header';
import { useEventStream } from '../hooks/useEventStream';
import type { ConnectionStatus } from '../hooks/useEventStream';
import { formatRelativeTime, cn } from '../lib/utils';
import type { StreamEvent, StreamEventFilters, EventChannel } from '../api/types';

// All available event types for filtering
const EVENT_TYPES = [
  'token_usage',
  'tool_executed',
  'step_complete',
  'state_changed',
  'error',
  'text_chunk',
  'text_chunk_start',
  'text_chunk_end',
  'think_chunk',
  'tool:start',
  'tool:end',
  'tool:progress',
  'done',
  'permission_required',
  'permission_decided',
] as const;

const CHANNELS: EventChannel[] = ['progress', 'control', 'monitor'];

export function Events() {
  const [isPaused, setIsPaused] = useState(false);
  const [filter, setFilter] = useState('');
  const [showFilters, setShowFilters] = useState(false);
  const [selectedChannels, setSelectedChannels] = useState<EventChannel[]>([]);
  const [selectedTypes, setSelectedTypes] = useState<string[]>([]);
  const listRef = useRef<HTMLDivElement>(null);
  const pausedEventsRef = useRef<StreamEvent[]>([]);

  const {
    events: streamEvents,
    status,
    stats,
    error,
    connect,
    disconnect,
    subscribe,
    clearEvents,
  } = useEventStream({ maxEvents: 500 });

  // Track events including paused state
  const [displayEvents, setDisplayEvents] = useState<StreamEvent[]>([]);

  // Update display events when stream events change
  useEffect(() => {
    if (!isPaused) {
      setDisplayEvents(streamEvents);
      pausedEventsRef.current = [];
    } else {
      // Track new events while paused
      const newEvents = streamEvents.slice(displayEvents.length);
      pausedEventsRef.current = [...pausedEventsRef.current, ...newEvents];
    }
  }, [streamEvents, isPaused, displayEvents.length]);

  // Resume: merge paused events
  const handleResume = useCallback(() => {
    setDisplayEvents((prev) => [...prev, ...pausedEventsRef.current]);
    pausedEventsRef.current = [];
    setIsPaused(false);
  }, []);

  // Apply filters when they change
  useEffect(() => {
    const filters: StreamEventFilters = {};
    if (selectedChannels.length > 0) {
      filters.channels = selectedChannels;
    }
    if (selectedTypes.length > 0) {
      filters.event_types = selectedTypes;
    }
    subscribe(filters);
  }, [selectedChannels, selectedTypes, subscribe]);

  // Auto-scroll to bottom
  useEffect(() => {
    if (listRef.current && !isPaused) {
      listRef.current.scrollTop = listRef.current.scrollHeight;
    }
  }, [displayEvents, isPaused]);

  // Filter events by search text
  const filteredEvents = displayEvents.filter((event) => {
    if (!filter) return true;
    const eventStr = JSON.stringify(event).toLowerCase();
    return eventStr.includes(filter.toLowerCase());
  });

  const handleClear = () => {
    clearEvents();
    setDisplayEvents([]);
    pausedEventsRef.current = [];
  };

  const toggleChannel = (channel: EventChannel) => {
    setSelectedChannels((prev) =>
      prev.includes(channel) ? prev.filter((c) => c !== channel) : [...prev, channel]
    );
  };

  const toggleEventType = (type: string) => {
    setSelectedTypes((prev) =>
      prev.includes(type) ? prev.filter((t) => t !== type) : [...prev, type]
    );
  };

  return (
    <div className="flex flex-col h-full">
      <Header title="Events" subtitle="实时事件流" />

      <div className="flex-1 flex flex-col overflow-hidden">
        {/* Connection Status Bar */}
        <ConnectionStatusBar
          status={status}
          error={error}
          stats={stats}
          onConnect={connect}
          onDisconnect={disconnect}
        />

        {/* Toolbar */}
        <div className="flex items-center justify-between p-4 border-b border-[var(--color-border)] bg-[var(--color-surface)]">
          <div className="flex items-center gap-4">
            {/* Filter Toggle */}
            <button
              onClick={() => setShowFilters(!showFilters)}
              className={cn(
                'flex items-center gap-2 px-3 py-1.5 rounded-lg text-sm transition-colors',
                showFilters
                  ? 'bg-[var(--color-primary)]/10 text-[var(--color-primary)]'
                  : 'bg-[var(--color-surface-elevated)] text-[var(--color-text-secondary)] hover:bg-[var(--color-border)]'
              )}
            >
              <Filter className="w-4 h-4" />
              筛选器
              {(selectedChannels.length > 0 || selectedTypes.length > 0) && (
                <span className="px-1.5 py-0.5 bg-[var(--color-primary)] text-white text-xs rounded-full">
                  {selectedChannels.length + selectedTypes.length}
                </span>
              )}
            </button>

            {/* Search */}
            <input
              type="text"
              placeholder="搜索事件..."
              value={filter}
              onChange={(e) => setFilter(e.target.value)}
              className="px-3 py-1.5 bg-[var(--color-background)] border border-[var(--color-border)] rounded-lg text-sm text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] focus:outline-none focus:border-[var(--color-primary)] w-64"
            />

            {/* Event count */}
            <span className="text-sm text-[var(--color-text-muted)]">
              {filteredEvents.length} 条事件
              {isPaused && pausedEventsRef.current.length > 0 && (
                <span className="text-[var(--color-warning)]">
                  {' '}
                  (+{pausedEventsRef.current.length} 新)
                </span>
              )}
            </span>
          </div>

          <div className="flex items-center gap-2">
            {/* Pause/Resume */}
            <button
              onClick={() => (isPaused ? handleResume() : setIsPaused(true))}
              className={cn(
                'flex items-center gap-2 px-3 py-1.5 rounded-lg text-sm transition-colors',
                isPaused
                  ? 'bg-[var(--color-warning)]/10 text-[var(--color-warning)]'
                  : 'bg-[var(--color-surface-elevated)] text-[var(--color-text-secondary)] hover:bg-[var(--color-border)]'
              )}
            >
              {isPaused ? (
                <>
                  <Play className="w-4 h-4" />
                  继续
                </>
              ) : (
                <>
                  <Pause className="w-4 h-4" />
                  暂停
                </>
              )}
            </button>

            {/* Clear */}
            <button
              onClick={handleClear}
              className="flex items-center gap-2 px-3 py-1.5 rounded-lg text-sm bg-[var(--color-surface-elevated)] text-[var(--color-text-secondary)] hover:bg-[var(--color-border)] transition-colors"
            >
              <Trash2 className="w-4 h-4" />
              清除
            </button>
          </div>
        </div>

        {/* Filter Panel */}
        {showFilters && (
          <div className="p-4 border-b border-[var(--color-border)] bg-[var(--color-surface)]">
            <div className="space-y-4">
              {/* Channel Filter */}
              <div>
                <label className="text-xs font-medium text-[var(--color-text-muted)] uppercase tracking-wider mb-2 block">
                  通道
                </label>
                <div className="flex flex-wrap gap-2">
                  {CHANNELS.map((channel) => (
                    <button
                      key={channel}
                      onClick={() => toggleChannel(channel)}
                      className={cn(
                        'px-2.5 py-1 rounded text-xs font-medium transition-colors',
                        selectedChannels.includes(channel)
                          ? 'bg-[var(--color-primary)] text-white'
                          : 'bg-[var(--color-surface-elevated)] text-[var(--color-text-secondary)] hover:bg-[var(--color-border)]'
                      )}
                    >
                      {channel}
                    </button>
                  ))}
                </div>
              </div>

              {/* Event Type Filter */}
              <div>
                <label className="text-xs font-medium text-[var(--color-text-muted)] uppercase tracking-wider mb-2 block">
                  事件类型
                </label>
                <div className="flex flex-wrap gap-2">
                  {EVENT_TYPES.map((type) => (
                    <button
                      key={type}
                      onClick={() => toggleEventType(type)}
                      className={cn(
                        'px-2.5 py-1 rounded text-xs font-medium transition-colors',
                        selectedTypes.includes(type)
                          ? 'bg-[var(--color-primary)] text-white'
                          : 'bg-[var(--color-surface-elevated)] text-[var(--color-text-secondary)] hover:bg-[var(--color-border)]'
                      )}
                    >
                      {type}
                    </button>
                  ))}
                </div>
              </div>

              {/* Clear Filters */}
              {(selectedChannels.length > 0 || selectedTypes.length > 0) && (
                <button
                  onClick={() => {
                    setSelectedChannels([]);
                    setSelectedTypes([]);
                  }}
                  className="text-xs text-[var(--color-primary)] hover:underline"
                >
                  清除所有筛选器
                </button>
              )}
            </div>
          </div>
        )}

        {/* Event List */}
        <div ref={listRef} className="flex-1 overflow-auto p-4 font-mono text-sm">
          {filteredEvents.length > 0 ? (
            <div className="space-y-1">
              {filteredEvents.map((event, index) => (
                <EventRow key={`${event.timestamp}-${index}`} event={event} />
              ))}
            </div>
          ) : (
            <div className="flex flex-col items-center justify-center h-full text-[var(--color-text-muted)]">
              <Zap className="w-12 h-12 mb-4" />
              <p>{status === 'connected' ? '等待事件...' : '连接中...'}</p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

// Connection Status Bar Component
interface ConnectionStatusBarProps {
  status: ConnectionStatus;
  error: string | null;
  stats: { connected_at: string; subscribed_agents: number; events_received: number } | null;
  onConnect: () => void;
  onDisconnect: () => void;
}

function ConnectionStatusBar({
  status,
  error,
  stats,
  onConnect,
  onDisconnect,
}: ConnectionStatusBarProps) {
  const statusConfig = {
    connected: {
      icon: Wifi,
      color: 'text-[var(--color-success)]',
      bg: 'bg-[var(--color-success)]/10',
      text: '已连接',
    },
    connecting: {
      icon: RefreshCw,
      color: 'text-[var(--color-warning)]',
      bg: 'bg-[var(--color-warning)]/10',
      text: '连接中...',
    },
    disconnected: {
      icon: WifiOff,
      color: 'text-[var(--color-text-muted)]',
      bg: 'bg-[var(--color-surface-elevated)]',
      text: '已断开',
    },
    error: {
      icon: AlertCircle,
      color: 'text-[var(--color-error)]',
      bg: 'bg-[var(--color-error)]/10',
      text: '连接错误',
    },
  };

  const config = statusConfig[status];
  const Icon = config.icon;

  return (
    <div
      className={cn(
        'flex items-center justify-between px-4 py-2 border-b border-[var(--color-border)]',
        config.bg
      )}
    >
      <div className="flex items-center gap-4">
        <div className={cn('flex items-center gap-2', config.color)}>
          <Icon className={cn('w-4 h-4', status === 'connecting' && 'animate-spin')} />
          <span className="text-sm font-medium">{config.text}</span>
        </div>

        {error && (
          <span className="text-sm text-[var(--color-error)]">{error}</span>
        )}

        {status === 'connected' && stats && (
          <div className="flex items-center gap-4 text-xs text-[var(--color-text-muted)]">
            <span className="flex items-center gap-1">
              <Activity className="w-3 h-3" />
              {stats.subscribed_agents} 个 Agent
            </span>
            <span className="flex items-center gap-1">
              <CheckCircle2 className="w-3 h-3" />
              {stats.events_received} 条已接收
            </span>
            <span className="flex items-center gap-1">
              <Clock className="w-3 h-3" />
              {formatRelativeTime(stats.connected_at)}
            </span>
          </div>
        )}
      </div>

      <div>
        {status === 'connected' ? (
          <button
            onClick={onDisconnect}
            className="px-3 py-1 text-xs rounded bg-[var(--color-surface)] text-[var(--color-text-secondary)] hover:bg-[var(--color-border)] transition-colors"
          >
            断开连接
          </button>
        ) : status !== 'connecting' ? (
          <button
            onClick={onConnect}
            className="px-3 py-1 text-xs rounded bg-[var(--color-primary)] text-white hover:bg-[var(--color-primary-hover)] transition-colors"
          >
            重新连接
          </button>
        ) : null}
      </div>
    </div>
  );
}

// Event Row Component
interface EventRowProps {
  event: StreamEvent;
}

function EventRow({ event }: EventRowProps) {
  const [expanded, setExpanded] = useState(false);

  const eventType = event.event_type || event.type || 'unknown';
  const typeColor = getEventTypeColor(eventType);
  const channelColor = getChannelColor(event.channel);

  return (
    <div
      className={cn(
        'p-2 rounded hover:bg-[var(--color-surface)] cursor-pointer transition-colors',
        expanded && 'bg-[var(--color-surface)]'
      )}
      onClick={() => setExpanded(!expanded)}
    >
      <div className="flex items-center gap-3">
        {/* Expand icon */}
        <span className="text-[var(--color-text-muted)] w-4">
          {expanded ? (
            <ChevronDown className="w-4 h-4" />
          ) : (
            <ChevronRight className="w-4 h-4" />
          )}
        </span>

        {/* Timestamp */}
        <span className="text-[var(--color-text-muted)] text-xs w-20 flex-shrink-0">
          {formatRelativeTime(event.timestamp)}
        </span>

        {/* Channel badge */}
        {event.channel && (
          <span
            className="px-1.5 py-0.5 rounded text-xs flex-shrink-0"
            style={{ backgroundColor: `${channelColor}15`, color: channelColor }}
          >
            {event.channel}
          </span>
        )}

        {/* Event type badge */}
        <span
          className="px-1.5 py-0.5 rounded text-xs flex-shrink-0"
          style={{ backgroundColor: `${typeColor}20`, color: typeColor }}
        >
          {eventType}
        </span>

        {/* Agent ID */}
        {event.agent_id && (
          <span className="text-xs text-[var(--color-text-muted)] flex-shrink-0">
            @{event.agent_id.slice(0, 8)}
          </span>
        )}

        {/* Summary */}
        <span className="text-[var(--color-text-secondary)] truncate">
          {getSummary(event)}
        </span>
      </div>

      {expanded && (
        <div className="mt-2 ml-6 p-2 bg-[var(--color-background)] rounded overflow-auto max-h-64">
          <pre className="text-xs text-[var(--color-text-secondary)] whitespace-pre-wrap">
            {JSON.stringify(event, null, 2)}
          </pre>
        </div>
      )}
    </div>
  );
}

function getEventTypeColor(type: string): string {
  const colors: Record<string, string> = {
    token_usage: '#3b82f6',
    tool_executed: '#8b5cf6',
    'tool:start': '#8b5cf6',
    'tool:end': '#8b5cf6',
    'tool:progress': '#8b5cf6',
    step_complete: '#22c55e',
    state_changed: '#06b6d4',
    error: '#ef4444',
    text_chunk: '#64748b',
    text_chunk_start: '#64748b',
    text_chunk_end: '#64748b',
    think_chunk: '#f59e0b',
    done: '#22c55e',
    permission_required: '#f59e0b',
    permission_decided: '#f59e0b',
    heartbeat: '#94a3b8',
  };
  return colors[type] || '#94a3b8';
}

function getChannelColor(channel?: string): string {
  const colors: Record<string, string> = {
    progress: '#3b82f6',
    control: '#f59e0b',
    monitor: '#22c55e',
  };
  return colors[channel || ''] || '#94a3b8';
}

function getSummary(event: StreamEvent): string {
  const data = event.data;
  if (!data) return '';

  // Token usage
  if (data.input_tokens !== undefined) {
    return `输入: ${data.input_tokens}, 输出: ${data.output_tokens}`;
  }

  // Tool events
  if (data.tool_name) {
    if (data.error) {
      return `${data.tool_name}: 失败 - ${data.error}`;
    }
    return `${data.tool_name}`;
  }

  // Text events
  if (data.delta !== undefined) {
    return String(data.delta).slice(0, 60);
  }

  // Error events
  if (data.message) {
    return String(data.message).slice(0, 80);
  }

  // State changes
  if (data.state) {
    return `状态: ${data.state}`;
  }

  // Step complete
  if (data.step !== undefined && data.duration_ms !== undefined) {
    return `步骤 ${data.step} 完成 (${data.duration_ms}ms)`;
  }

  // Done events
  if (data.reason) {
    return `原因: ${data.reason}`;
  }

  return '';
}
