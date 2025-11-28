/**
 * Aster Client Composable
 * 封装 @aster/client-js SDK 供 Vue3 使用
 */

import { ref, onUnmounted } from 'vue';
import { aster, WebSocketClient, SubscriptionManager } from '@aster/client-js';
import { setGlobalWebSocket } from './useWebSocket';

export interface AsterClientConfig {
  baseUrl?: string;
  apiKey?: string;
  wsUrl?: string;
}

export function useAsterClient(config: AsterClientConfig = {}) {
  const baseUrl = config.baseUrl || import.meta.env.VITE_API_URL || 'http://localhost:8080';
  const apiKey = config.apiKey || import.meta.env.VITE_API_KEY || '';
  const wsUrlEnv = config.wsUrl || import.meta.env.VITE_WS_URL;
  
  // 构建请求头
  const buildHeaders = (additionalHeaders: Record<string, string> = {}) => {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      ...additionalHeaders,
    };
    
    // 如果有 API Key，添加到请求头
    if (apiKey) {
      headers['X-API-Key'] = apiKey;
    }
    
    return headers;
  };
  
  // 创建 Aster Client 并添加 agent 管理方法
  const client = {
    agents: {
      async list() {
        const response = await fetch(`${baseUrl}/v1/agents`, {
          headers: buildHeaders(),
        });
        return response.json();
      },
      async get(id: string) {
        const response = await fetch(`${baseUrl}/v1/agents/${id}`, {
          headers: buildHeaders(),
        });
        return response.json();
      },
      async create(data: any) {
        const response = await fetch(`${baseUrl}/v1/agents`, {
          method: 'POST',
          headers: buildHeaders(),
          body: JSON.stringify(data),
        });
        return response.json();
      },
      async update(id: string, data: any) {
        const response = await fetch(`${baseUrl}/v1/agents/${id}`, {
          method: 'PUT',
          headers: buildHeaders(),
          body: JSON.stringify(data),
        });
        return response.json();
      },
      async delete(id: string) {
        const response = await fetch(`${baseUrl}/v1/agents/${id}`, {
          method: 'DELETE',
          headers: buildHeaders(),
        });
        return response.status === 204 ? { success: true } : response.json();
      },
      async chat(id: string, message: string) {
        const response = await fetch(`${baseUrl}/v1/agents/${id}/send`, {
          method: 'POST',
          headers: buildHeaders(),
          body: JSON.stringify({ message }),
        });
        return response.json();
      },
      // 直接聊天（无需预先创建 Agent）
      async chatDirect(message: string, templateId: string = 'chat') {
        const response = await fetch(`${baseUrl}/v1/agents/chat`, {
          method: 'POST',
          headers: buildHeaders(),
          body: JSON.stringify({
            template_id: templateId,
            input: message,
          }),
        });
        return response.json();
      },
    },
    workflows: {
      async list() {
        const response = await fetch(`${baseUrl}/v1/workflows`, {
          headers: buildHeaders(),
        });
        return response.json();
      },
      async get(id: string) {
        const response = await fetch(`${baseUrl}/v1/workflows/${id}`, {
          headers: buildHeaders(),
        });
        return response.json();
      },
      async create(data: any) {
        const response = await fetch(`${baseUrl}/v1/workflows`, {
          method: 'POST',
          headers: buildHeaders(),
          body: JSON.stringify(data),
        });
        return response.json();
      },
      async update(id: string, data: any) {
        const response = await fetch(`${baseUrl}/v1/workflows/${id}`, {
          method: 'PATCH',
          headers: buildHeaders(),
          body: JSON.stringify(data),
        });
        return response.json();
      },
      async delete(id: string) {
        const response = await fetch(`${baseUrl}/v1/workflows/${id}`, {
          method: 'DELETE',
          headers: buildHeaders(),
        });
        return response.status === 204 ? { success: true } : response.json();
      },
      async execute(id: string) {
        const response = await fetch(`${baseUrl}/v1/workflows/${id}/execute`, {
          method: 'POST',
          headers: buildHeaders(),
        });
        return response.json();
      },
    },
    rooms: {
      async list() {
        const response = await fetch(`${baseUrl}/v1/rooms`, {
          headers: buildHeaders(),
        });
        return response.json();
      },
      async get(id: string) {
        const response = await fetch(`${baseUrl}/v1/rooms/${id}`, {
          headers: buildHeaders(),
        });
        return response.json();
      },
      async create(data: any) {
        const response = await fetch(`${baseUrl}/v1/rooms`, {
          method: 'POST',
          headers: buildHeaders(),
          body: JSON.stringify(data),
        });
        return response.json();
      },
      async delete(id: string) {
        const response = await fetch(`${baseUrl}/v1/rooms/${id}`, {
          method: 'DELETE',
          headers: buildHeaders(),
        });
        return response.status === 204 ? { success: true } : response.json();
      },
      async join(id: string, data: any) {
        const response = await fetch(`${baseUrl}/v1/rooms/${id}/join`, {
          method: 'POST',
          headers: buildHeaders(),
          body: JSON.stringify(data),
        });
        return response.json();
      },
      async leave(id: string) {
        const response = await fetch(`${baseUrl}/v1/rooms/${id}/leave`, {
          method: 'POST',
          headers: buildHeaders(),
        });
        return response.json();
      },
    },
  };

  // WebSocket 状态
  const ws = ref<WebSocketClient | null>(null);
  const isConnected = ref(false);
  const subscriptionManager = ref<SubscriptionManager | null>(null);

  // 初始化 WebSocket
  const initWebSocket = async () => {
    try {
      // 构建 WebSocket URL
      const wsUrl = wsUrlEnv || baseUrl.replace(/^http/, 'ws') + '/v1/ws';
      
      ws.value = new WebSocketClient({
        maxReconnectAttempts: 5,
        reconnectDelay: 1000,
        heartbeatInterval: 30000,
      });

      await ws.value.connect(wsUrl);
      isConnected.value = true;
      
      // 设置全局 WebSocket 实例，供 approvalStore 等使用
      setGlobalWebSocket(ws.value as any);
      
      subscriptionManager.value = new SubscriptionManager(ws.value as any);
      
      console.log('✅ Aster WebSocket connected to', wsUrl);
      console.log('✅ ws.value:', ws.value);
      console.log('✅ isConnected.value:', isConnected.value);
    } catch (error) {
      console.error('❌ Failed to connect WebSocket:', error);
      isConnected.value = false;
    }
  };

  const ensureWebSocket = async () => {
    if (!ws.value || isConnected.value === false) {
      await initWebSocket();
    }
    return ws.value;
  };

  const onMessage = (handler: (msg: any) => void) => {
    if (!ws.value) {
      throw new Error('WebSocket not initialized');
    }
    return ws.value.onMessage(handler);
  };

  // 订阅事件
  const subscribe = (
    channels: ('progress' | 'control' | 'monitor')[],
    filter?: { agentId?: string; eventTypes?: string[] }
  ) => {
    if (!subscriptionManager.value) {
      throw new Error('WebSocket not initialized');
    }
    return subscriptionManager.value.subscribe(channels, filter);
  };

  // 断开连接
  const disconnect = () => {
    if (ws.value) {
      ws.value.disconnect();
      ws.value = null;
      isConnected.value = false;
    }
  };

  // 生命周期
  onUnmounted(() => {
    disconnect();
  });

  return {
    client,
    ws,
    isConnected,
    subscribe,
    onMessage,
    ensureWebSocket,
    disconnect,
    reconnect: initWebSocket,
  };
}
