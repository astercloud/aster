import { useState, useEffect, useCallback, useRef } from 'react';
import type { StreamEvent, StreamEventFilters, EventStreamStats } from '../api/types';

export type ConnectionStatus = 'connecting' | 'connected' | 'disconnected' | 'error';

interface UseEventStreamOptions {
  maxEvents?: number;
  autoConnect?: boolean;
  reconnectDelay?: number;
  maxReconnectAttempts?: number;
}

interface UseEventStreamResult {
  events: StreamEvent[];
  status: ConnectionStatus;
  stats: EventStreamStats | null;
  error: string | null;
  connect: () => void;
  disconnect: () => void;
  subscribe: (filters: StreamEventFilters) => void;
  clearEvents: () => void;
}

const DEFAULT_OPTIONS: UseEventStreamOptions = {
  maxEvents: 500,
  autoConnect: true,
  reconnectDelay: 3000,
  maxReconnectAttempts: 5,
};

export function useEventStream(options: UseEventStreamOptions = {}): UseEventStreamResult {
  const opts = { ...DEFAULT_OPTIONS, ...options };

  const [events, setEvents] = useState<StreamEvent[]>([]);
  const [status, setStatus] = useState<ConnectionStatus>('disconnected');
  const [stats, setStats] = useState<EventStreamStats | null>(null);
  const [error, setError] = useState<string | null>(null);

  const wsRef = useRef<WebSocket | null>(null);
  const reconnectAttempts = useRef(0);
  const reconnectTimeout = useRef<ReturnType<typeof setTimeout> | null>(null);
  const filtersRef = useRef<StreamEventFilters>({});

  // Build WebSocket URL
  const getWsUrl = useCallback(() => {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = window.location.host;
    return `${protocol}//${host}/v1/dashboard/events/stream`;
  }, []);

  // Connect to WebSocket
  const connect = useCallback(() => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      return;
    }

    // Clean up existing connection
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }

    setStatus('connecting');
    setError(null);

    try {
      const ws = new WebSocket(getWsUrl());
      wsRef.current = ws;

      ws.onopen = () => {
        setStatus('connected');
        setError(null);
        reconnectAttempts.current = 0;

        // Always send subscription on connect (even with empty filters to subscribe to all)
        ws.send(JSON.stringify({
          action: 'subscribe',
          filters: filtersRef.current,
        }));
      };

      ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);

          switch (data.type) {
            case 'event':
              // Extract payload as the actual event data
              const eventData = data.payload as StreamEvent;
              setEvents((prev) => {
                const newEvents = [...prev, eventData];
                // Keep only the last maxEvents
                if (newEvents.length > opts.maxEvents!) {
                  return newEvents.slice(-opts.maxEvents!);
                }
                return newEvents;
              });
              break;

            case 'stats':
              if (data.stats) {
                setStats(data.stats);
              }
              break;

            case 'subscribed':
            case 'filtered':
              // Acknowledge subscription/filter updates
              break;

            case 'heartbeat':
              // Keep-alive, no action needed
              break;

            case 'error':
              setError(data.message || 'Unknown error');
              break;
          }
        } catch (e) {
          console.error('Failed to parse WebSocket message:', e);
        }
      };

      ws.onerror = () => {
        setError('WebSocket connection error');
        setStatus('error');
      };

      ws.onclose = (event) => {
        setStatus('disconnected');
        wsRef.current = null;

        // Attempt reconnection if not intentionally closed
        if (!event.wasClean && reconnectAttempts.current < opts.maxReconnectAttempts!) {
          reconnectAttempts.current++;
          reconnectTimeout.current = setTimeout(() => {
            connect();
          }, opts.reconnectDelay! * reconnectAttempts.current);
        }
      };
    } catch (e) {
      setError(`Failed to connect: ${e}`);
      setStatus('error');
    }
  }, [getWsUrl, opts.maxEvents, opts.maxReconnectAttempts, opts.reconnectDelay]);

  // Disconnect from WebSocket
  const disconnect = useCallback(() => {
    if (reconnectTimeout.current) {
      clearTimeout(reconnectTimeout.current);
      reconnectTimeout.current = null;
    }
    reconnectAttempts.current = opts.maxReconnectAttempts!; // Prevent auto-reconnect

    if (wsRef.current) {
      wsRef.current.close(1000, 'User disconnect');
      wsRef.current = null;
    }
    setStatus('disconnected');
  }, [opts.maxReconnectAttempts]);

  // Update subscription filters
  const subscribe = useCallback((filters: StreamEventFilters) => {
    filtersRef.current = filters;

    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify({
        action: 'subscribe',
        filters,
      }));
    }
  }, []);

  // Clear events
  const clearEvents = useCallback(() => {
    setEvents([]);
  }, []);

  // Auto-connect on mount
  useEffect(() => {
    if (opts.autoConnect) {
      connect();
    }

    return () => {
      if (reconnectTimeout.current) {
        clearTimeout(reconnectTimeout.current);
      }
      if (wsRef.current) {
        wsRef.current.close();
      }
    };
  }, [opts.autoConnect, connect]);

  return {
    events,
    status,
    stats,
    error,
    connect,
    disconnect,
    subscribe,
    clearEvents,
  };
}
