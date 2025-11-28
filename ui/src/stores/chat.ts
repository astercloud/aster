/**
 * 聊天状态管理
 *
 * 管理聊天消息、Agent 状态等核心功能
 */

import { defineStore } from 'pinia';
import { ref, computed } from 'vue';
import type { Message, TextMessage, Agent } from '@/types';
import { generateId } from '@/utils/format';

export const useChatStore = defineStore('chat', () => {
  // ==================
  // State
  // ==================

  // 消息列表
  const messages = ref<Message[]>([]);

  // Agent 状态
  const agent = ref<Agent>({
    id: '',
    name: 'Aster Copilot',
    description: '多模态执行、自动规划、符合企业安全的 Agent',
    status: 'idle',
    metadata: {
      model: 'aster:builder',
    },
  });

  // 是否正在输入
  const isTyping = ref(false);

  // 当前输入内容
  const currentInput = ref('');

  // 当前激活的消息 ID (用于思维过程关联)
  const activeMessageId = ref<string>('');

  // Plan Mode 状态
  const planMode = ref<{
    active: boolean;
    planContent: string;
    planId: string | null;
  }>({
    active: false,
    planContent: '',
    planId: null,
  });

  // 流式文本块累积（用于批量更新）
  const pendingTextChunks = ref<Map<string, string[]>>(new Map());

  // requestAnimationFrame ID
  let rafId: number | null = null;

  // ==================
  // Getters
  // ==================

  // 消息数量
  const messageCount = computed(() => messages.value.length);

  // 最后一条消息
  const lastMessage = computed(() => messages.value[messages.value.length - 1] || null);

  // 最后一条助手消息
  const lastAssistantMessage = computed(() => {
    const reversed = [...messages.value].reverse();
    return reversed.find(m => m.role === 'assistant') || null;
  });

  // 是否有消息
  const hasMessages = computed(() => messageCount.value > 0);

  // ==================
  // Actions
  // ==================

  /**
   * 添加消息
   */
  const addMessage = (message: Message) => {
    messages.value.push(message);
  };

  /**
   * 添加文本消息（简便方法）
   */
  const addTextMessage = (role: 'user' | 'assistant' | 'system', text: string): TextMessage => {
    const message: TextMessage = {
      id: generateId('msg'),
      type: 'text',
      role,
      content: { text },
      createdAt: Date.now(),
      status: role === 'user' ? 'sent' : undefined,
    };
    addMessage(message);
    return message;
  };

  /**
   * 创建用户消息
   */
  const createUserMessage = (text: string): TextMessage => {
    return addTextMessage('user', text);
  };

  /**
   * 创建助手消息占位符
   */
  const createAssistantPlaceholder = (): TextMessage => {
    return addTextMessage('assistant', '');
  };

  /**
   * 更新消息
   */
  const updateMessage = (messageId: string, updates: Partial<Message>) => {
    const index = messages.value.findIndex(m => m.id === messageId);
    if (index !== -1) {
      const msg = messages.value[index];
      if (msg) {
        messages.value[index] = {
          ...msg,
          ...updates,
        } as Message;
      }
    }
  };

  /**
   * 处理文本块（高频事件，使用 RAF 批量更新）
   */
  const handleTextChunk = (messageId: string, delta: string) => {
    // 累积文本块
    const chunks = pendingTextChunks.value.get(messageId) || [];
    chunks.push(delta);
    pendingTextChunks.value.set(messageId, chunks);

    // 批量更新（每帧一次）
    if (rafId === null) {
      rafId = requestAnimationFrame(() => {
        flushTextChunks();
        rafId = null;
      });
    }
  };

  /**
   * 刷新文本块（批量更新到消息）
   */
  const flushTextChunks = () => {
    for (const [messageId, chunks] of pendingTextChunks.value) {
      const msg = messages.value.find(m => m.id === messageId);
      if (msg && msg.type === 'text') {
        msg.content.text += chunks.join('');
      }
    }
    pendingTextChunks.value.clear();
  };

  /**
   * 更新最后一条助手消息的文本
   */
  const updateLastAssistantMessage = (text: string) => {
    const lastMsg = lastAssistantMessage.value;
    if (lastMsg && lastMsg.type === 'text') {
      lastMsg.content.text = text;
    }
  };

  /**
   * 追加文本到最后一条助手消息
   */
  const appendToLastAssistantMessage = (text: string) => {
    const lastMsg = lastAssistantMessage.value;
    if (lastMsg && lastMsg.type === 'text') {
      lastMsg.content.text += text;
    }
  };

  /**
   * 删除消息
   */
  const deleteMessage = (messageId: string) => {
    const index = messages.value.findIndex(m => m.id === messageId);
    if (index !== -1) {
      messages.value.splice(index, 1);
    }
  };

  /**
   * 清空所有消息
   */
  const clearMessages = () => {
    messages.value = [];
    pendingTextChunks.value.clear();
    if (rafId !== null) {
      cancelAnimationFrame(rafId);
      rafId = null;
    }
  };

  /**
   * 设置 Agent 信息
   */
  const setAgent = (agentInfo: Partial<Agent>) => {
    agent.value = {
      ...agent.value,
      ...agentInfo,
    };
  };

  /**
   * 更新 Agent 状态
   */
  const updateAgentStatus = (status: Agent['status']) => {
    agent.value.status = status;
  };

  /**
   * 设置输入状态
   */
  const setTyping = (typing: boolean) => {
    isTyping.value = typing;
  };

  /**
   * 设置当前输入
   */
  const setCurrentInput = (input: string) => {
    currentInput.value = input;
  };

  /**
   * 设置当前激活的消息 ID
   */
  const setActiveMessage = (messageId: string) => {
    activeMessageId.value = messageId;
  };

  /**
   * 进入 Plan Mode
   */
  const enterPlanMode = (planId: string, content: string) => {
    planMode.value = {
      active: true,
      planContent: content,
      planId,
    };
  };

  /**
   * 退出 Plan Mode
   */
  const exitPlanMode = () => {
    planMode.value = {
      active: false,
      planContent: '',
      planId: null,
    };
  };

  /**
   * 获取消息索引
   */
  const getMessageIndex = (messageId: string): number => {
    return messages.value.findIndex(m => m.id === messageId);
  };

  /**
   * 获取消息
   */
  const getMessage = (messageId: string): Message | undefined => {
    return messages.value.find(m => m.id === messageId);
  };

  // ==================
  // Return
  // ==================

  return {
    // State
    messages,
    agent,
    isTyping,
    currentInput,
    activeMessageId,
    planMode,

    // Getters
    messageCount,
    lastMessage,
    lastAssistantMessage,
    hasMessages,

    // Actions
    addMessage,
    addTextMessage,
    createUserMessage,
    createAssistantPlaceholder,
    updateMessage,
    handleTextChunk,
    flushTextChunks,
    updateLastAssistantMessage,
    appendToLastAssistantMessage,
    deleteMessage,
    clearMessages,
    setAgent,
    updateAgentStatus,
    setTyping,
    setCurrentInput,
    setActiveMessage,
    enterPlanMode,
    exitPlanMode,
    getMessageIndex,
    getMessage,
  };
});
