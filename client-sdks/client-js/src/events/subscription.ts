/**
 * 事件订阅系统
 * 支持异步迭代、背压控制、事件过滤
 */

import { WebSocketClient, WebSocketState } from "../transport/websocket";
import { StreamEvent, EventEnvelope, Channel, EventFilter } from "./types";

/**
 * 事件订阅
 * 实现 AsyncIterable 接口，支持 for await...of
 */
export class EventSubscription {
  private ws: WebSocketClient;
  private channels: Channel[];
  private filter?: EventFilter;
  private eventQueue: EventEnvelope[] = [];
  private isActive = true;
  private resolvers: Array<(value: IteratorResult<EventEnvelope>) => void> = [];
  private unsubscribeWs?: () => void;

  constructor(ws: WebSocketClient, channels: Channel[], filter?: EventFilter) {
    this.ws = ws;
    this.channels = channels;
    this.filter = filter;

    // 监听 WebSocket 消息
    this.unsubscribeWs = this.ws.onMessage((envelope: EventEnvelope) => {
      this.handleEvent(envelope);
    });

    // 发送订阅请求
    this.sendSubscribeRequest();
  }

  /**
   * 异步迭代器实现
   * 支持 for await (const event of subscription)
   */
  async *[Symbol.asyncIterator](): AsyncIterator<EventEnvelope> {
    while (this.isActive) {
      // 如果队列中有事件，立即返回
      if (this.eventQueue.length > 0) {
        const event = this.eventQueue.shift()!;
        yield event;
        continue;
      }

      // 等待新事件
      const result = await new Promise<IteratorResult<EventEnvelope>>((resolve) => {
        if (!this.isActive) {
          resolve({ done: true, value: undefined });
          return;
        }

        this.resolvers.push(resolve);
      });

      if (result.done) {
        break;
      }

      yield result.value;
    }
  }

  /**
   * 取消订阅
   */
  unsubscribe(): void {
    if (!this.isActive) {
      return;
    }

    this.isActive = false;

    // 取消 WebSocket 监听
    if (this.unsubscribeWs) {
      this.unsubscribeWs();
    }

    // 发送取消订阅请求
    this.sendUnsubscribeRequest();

    // 清空队列
    this.eventQueue = [];

    // 通知所有等待的迭代器
    this.resolvers.forEach((resolve) => {
      resolve({ done: true, value: undefined });
    });
    this.resolvers = [];
  }

  /**
   * 处理接收到的事件
   */
  private handleEvent(envelope: EventEnvelope): void {
    // 检查订阅是否活跃
    if (!this.isActive) {
      return;
    }

    // 检查事件通道是否匹配
    if (!this.channels.includes(envelope.event.channel)) {
      return;
    }

    // 应用过滤器
    if (this.filter) {
      // Agent ID 过滤
      if (this.filter.agentId && envelope.agentId !== this.filter.agentId) {
        return;
      }

      // 事件类型过滤
      if (this.filter.eventTypes && !this.filter.eventTypes.includes(envelope.event.type)) {
        return;
      }
    }

    // 如果有等待的迭代器，立即返回
    if (this.resolvers.length > 0) {
      const resolve = this.resolvers.shift()!;
      resolve({ done: false, value: envelope });
    } else {
      // 否则加入队列（背压控制）
      this.eventQueue.push(envelope);

      // 防止队列过大
      if (this.eventQueue.length > 1000) {
        console.warn("[EventSubscription] Event queue overflow, dropping oldest events");
        this.eventQueue = this.eventQueue.slice(-500); // 保留最新的 500 个
      }
    }
  }

  /**
   * 发送订阅请求
   */
  private sendSubscribeRequest(): void {
    this.ws.send({
      type: "subscribe",
      channels: this.channels,
      filter: this.filter,
    });
  }

  /**
   * 发送取消订阅请求
   */
  private sendUnsubscribeRequest(): void {
    this.ws.send({
      type: "unsubscribe",
      channels: this.channels,
    });
  }
}

/**
 * 订阅管理器
 * 管理多个订阅
 */
export class SubscriptionManager {
  private ws: WebSocketClient;
  private subscriptions = new Map<string, EventSubscription>();
  private subscriptionCounter = 0;

  constructor(ws: WebSocketClient) {
    this.ws = ws;
  }

  /**
   * 创建订阅
   */
  subscribe(channels: Channel[], filter?: EventFilter): EventSubscription {
    const subscription = new EventSubscription(this.ws, channels, filter);
    const subscriptionId = `sub_${++this.subscriptionCounter}`;

    this.subscriptions.set(subscriptionId, subscription);

    return subscription;
  }

  /**
   * 取消特定订阅
   */
  unsubscribe(subscriptionId: string): void {
    const subscription = this.subscriptions.get(subscriptionId);
    if (subscription) {
      subscription.unsubscribe();
      this.subscriptions.delete(subscriptionId);
    }
  }

  /**
   * 取消所有订阅
   */
  unsubscribeAll(): void {
    this.subscriptions.forEach((subscription) => {
      subscription.unsubscribe();
    });
    this.subscriptions.clear();
  }

  /**
   * 获取活跃订阅数
   */
  getActiveCount(): number {
    return this.subscriptions.size;
  }
}
