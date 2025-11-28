/**
 * WebSocket 单例管理
 * 确保整个应用只有一个 WebSocket 连接
 */

import { ref } from 'vue';
import { WebSocketClient } from '@aster/client-js';

// 全局单例
let wsInstance: WebSocketClient | null = null;
const isConnected = ref(false);
const connectionUrl = ref('');

// 设置全局 WebSocket 实例（供其他 composable 使用）
export function setGlobalWebSocket(ws: WebSocketClient | null) {
  wsInstance = ws;
  isConnected.value = ws !== null;
}

export function useWebSocket() {
  const connect = async (url: string) => {
    if (wsInstance && connectionUrl.value === url) {
      console.log('♻️ Reusing existing WebSocket connection');
      return wsInstance;
    }

    if (wsInstance) {
      console.log('🔄 Disconnecting old WebSocket');
      wsInstance.disconnect();
    }

    console.log('🔌 Creating new WebSocket connection to:', url);
    
    wsInstance = new WebSocketClient({
      maxReconnectAttempts: 5,
      reconnectDelay: 1000,
      heartbeatInterval: 30000,
    });

    // 监听状态变化
    wsInstance.onStateChange((state) => {
      console.log('📡 WebSocket state changed:', state);
      isConnected.value = state === 'CONNECTED';
    });

    await wsInstance.connect(url);
    connectionUrl.value = url;
    isConnected.value = true;
    
    console.log('✅ WebSocket connected successfully');
    
    return wsInstance;
  };

  const disconnect = () => {
    if (wsInstance) {
      wsInstance.disconnect();
      wsInstance = null;
      isConnected.value = false;
      connectionUrl.value = '';
    }
  };

  const getInstance = () => wsInstance;

  return {
    connect,
    disconnect,
    getInstance,
    isConnected,
  };
}
