/**
 * WebSocket 客户端
 * 支持自动重连、消息队列、心跳检测
 */

/**
 * WebSocket 连接状态
 */
export enum WebSocketState {
  CONNECTING = "CONNECTING",
  CONNECTED = "CONNECTED",
  DISCONNECTING = "DISCONNECTING",
  DISCONNECTED = "DISCONNECTED",
  RECONNECTING = "RECONNECTING",
  FAILED = "FAILED",
}

/**
 * WebSocket 客户端选项
 */
export interface WebSocketClientOptions {
  /** 最大重连次数 */
  maxReconnectAttempts?: number;
  /** 重连延迟（毫秒） */
  reconnectDelay?: number;
  /** 心跳间隔（毫秒） */
  heartbeatInterval?: number;
  /** 心跳超时（毫秒） */
  heartbeatTimeout?: number;
}

/**
 * 消息监听器
 */
type MessageListener = (data: any) => void;

/**
 * 状态变化监听器
 */
type StateChangeListener = (state: WebSocketState) => void;

/**
 * WebSocket 客户端
 */
export class WebSocketClient {
  private ws: WebSocket | null = null;
  private state: WebSocketState = WebSocketState.DISCONNECTED;
  private reconnectAttempts = 0;
  private messageQueue: any[] = [];
  private messageListeners: Set<MessageListener> = new Set();
  private stateChangeListeners: Set<StateChangeListener> = new Set();
  private heartbeatTimer?: NodeJS.Timeout;
  private heartbeatTimeoutTimer?: NodeJS.Timeout;
  private url: string | null = null;

  // 配置
  private readonly maxReconnectAttempts: number;
  private readonly reconnectDelay: number;
  private readonly heartbeatInterval: number;
  private readonly heartbeatTimeout: number;

  constructor(options: WebSocketClientOptions = {}) {
    this.maxReconnectAttempts = options.maxReconnectAttempts ?? 5;
    this.reconnectDelay = options.reconnectDelay ?? 1000;
    this.heartbeatInterval = options.heartbeatInterval ?? 30000; // 30秒
    this.heartbeatTimeout = options.heartbeatTimeout ?? 10000; // 10秒
  }

  /**
   * 连接到 WebSocket 服务器
   */
  async connect(url: string): Promise<void> {
    if (this.state === WebSocketState.CONNECTED || this.state === WebSocketState.CONNECTING) {
      console.warn("[WebSocket] Already connected or connecting");
      return;
    }

    this.url = url;
    this.setState(WebSocketState.CONNECTING);

    return new Promise((resolve, reject) => {
      try {
        // 在浏览器和 Node.js 中使用不同的 WebSocket
        const WebSocketImpl = typeof window !== "undefined" ? window.WebSocket : require("ws");

        this.ws = new WebSocketImpl(url);

        if (this.ws) {
          this.ws.onopen = () => {
            console.log("[WebSocket] Connected");
            this.setState(WebSocketState.CONNECTED);
            this.reconnectAttempts = 0;
            this.startHeartbeat();
            this.flushMessageQueue();
            resolve();
          };

          this.ws.onmessage = (event) => {
            this.handleMessage(event.data);
          };

          this.ws.onerror = (error) => {
            console.error("[WebSocket] Error:", error);
            reject(error);
          };

          this.ws.onclose = (event) => {
            console.log("[WebSocket] Closed:", event.code, event.reason);
            this.stopHeartbeat();

            if (this.state !== WebSocketState.DISCONNECTING) {
              this.handleReconnect();
            } else {
              this.setState(WebSocketState.DISCONNECTED);
            }
          };
        }
      } catch (error) {
        console.error("[WebSocket] Connection failed:", error);
        this.setState(WebSocketState.FAILED);
        reject(error);
      }
    });
  }

  /**
   * 断开连接
   */
  disconnect(): void {
    if (this.state === WebSocketState.DISCONNECTED || this.state === WebSocketState.DISCONNECTING) {
      return;
    }

    this.setState(WebSocketState.DISCONNECTING);
    this.stopHeartbeat();

    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }

    this.setState(WebSocketState.DISCONNECTED);
  }

  /**
   * 发送消息
   */
  send(message: any): void {
    const data = typeof message === "string" ? message : JSON.stringify(message);

    if (this.state === WebSocketState.CONNECTED && this.ws) {
      try {
        this.ws.send(data);
      } catch (error) {
        console.error("[WebSocket] Send failed:", error);
        // 发送失败，加入队列
        this.messageQueue.push(message);
      }
    } else {
      // 未连接，加入队列
      this.messageQueue.push(message);
    }
  }

  /**
   * 添加消息监听器
   */
  onMessage(listener: MessageListener): () => void {
    this.messageListeners.add(listener);
    // 返回取消监听的函数
    return () => {
      this.messageListeners.delete(listener);
    };
  }

  /**
   * 添加状态变化监听器
   */
  onStateChange(listener: StateChangeListener): () => void {
    this.stateChangeListeners.add(listener);
    // 返回取消监听的函数
    return () => {
      this.stateChangeListeners.delete(listener);
    };
  }

  /**
   * 获取当前状态
   */
  getState(): WebSocketState {
    return this.state;
  }

  /**
   * 处理接收到的消息
   */
  private handleMessage(data: string | Buffer): void {
    try {
      const message = typeof data === "string" ? JSON.parse(data) : JSON.parse(data.toString());

      // 检查是否为心跳响应
      if (message.type === "pong") {
        this.handleHeartbeatResponse();
        return;
      }

      // 通知所有监听器
      this.messageListeners.forEach((listener) => {
        try {
          listener(message);
        } catch (error) {
          console.error("[WebSocket] Message listener error:", error);
        }
      });
    } catch (error) {
      console.error("[WebSocket] Parse message failed:", error);
    }
  }

  /**
   * 刷新消息队列
   */
  private flushMessageQueue(): void {
    while (this.messageQueue.length > 0 && this.state === WebSocketState.CONNECTED) {
      const message = this.messageQueue.shift();
      this.send(message);
    }
  }

  /**
   * 处理重连
   */
  private async handleReconnect(): Promise<void> {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error("[WebSocket] Max reconnect attempts reached");
      this.setState(WebSocketState.FAILED);
      return;
    }

    this.reconnectAttempts++;
    this.setState(WebSocketState.RECONNECTING);

    // 指数退避
    const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1);
    console.log(`[WebSocket] Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts}/${this.maxReconnectAttempts})`);

    await new Promise((resolve) => setTimeout(resolve, delay));

    if (this.url) {
      try {
        await this.connect(this.url);
      } catch (error) {
        console.error("[WebSocket] Reconnect failed:", error);
        // 继续尝试重连
        this.handleReconnect();
      }
    }
  }

  /**
   * 设置状态并通知监听器
   */
  private setState(newState: WebSocketState): void {
    if (this.state !== newState) {
      this.state = newState;
      console.log(`[WebSocket] State changed: ${newState}`);

      // 通知所有监听器
      this.stateChangeListeners.forEach((listener) => {
        try {
          listener(newState);
        } catch (error) {
          console.error("[WebSocket] State change listener error:", error);
        }
      });
    }
  }

  /**
   * 启动心跳
   */
  private startHeartbeat(): void {
    this.stopHeartbeat();

    this.heartbeatTimer = setInterval(() => {
      if (this.state === WebSocketState.CONNECTED) {
        this.send({ type: "ping" });

        // 设置心跳超时
        this.heartbeatTimeoutTimer = setTimeout(() => {
          console.warn("[WebSocket] Heartbeat timeout, reconnecting...");
          this.disconnect();
          if (this.url) {
            this.connect(this.url);
          }
        }, this.heartbeatTimeout);
      }
    }, this.heartbeatInterval);
  }

  /**
   * 停止心跳
   */
  private stopHeartbeat(): void {
    if (this.heartbeatTimer) {
      clearInterval(this.heartbeatTimer);
      this.heartbeatTimer = undefined;
    }
    if (this.heartbeatTimeoutTimer) {
      clearTimeout(this.heartbeatTimeoutTimer);
      this.heartbeatTimeoutTimer = undefined;
    }
  }

  /**
   * 处理心跳响应
   */
  private handleHeartbeatResponse(): void {
    // 收到心跳响应，清除超时定时器
    if (this.heartbeatTimeoutTimer) {
      clearTimeout(this.heartbeatTimeoutTimer);
      this.heartbeatTimeoutTimer = undefined;
    }
  }
}
