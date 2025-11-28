<template>
<div class="agent-chatui-demo">
  <div class="demo-container">
    <!-- 侧边栏 -->
    <div class="demo-sidebar">
      <div class="sidebar-header">
        <h2 class="sidebar-title">Aster Agent</h2>
        <p class="sidebar-subtitle">ChatUI + Tool Stream</p>
        <div class="ws-status" :class="{ online: wsConnected }">
          <span class="dot"></span>{{ wsConnected ? 'WS Connected' : 'WS Disconnected' }}
        </div>
      </div>
      
      <div class="agent-selector">
        <div
          v-for="agent in agents"
          :key="agent.id"
          :class="['agent-item', { active: selectedAgent?.id === agent.id }]"
          @click="selectAgent(agent)"
        >
          <div class="agent-avatar">
            <div class="avatar-placeholder">{{ agent.name[0] }}</div>
          </div>
          <div class="agent-info">
            <div class="agent-name">{{ agent.name }}</div>
            <div class="agent-desc">{{ agent.description }}</div>
          </div>
          <div :class="['agent-status', `status-${agent.status}`]"></div>
        </div>
      </div>

      <!-- Provider 选择器 -->
      <div class="provider-section">
        <ProviderSelector @change="handleProviderChange" />
      </div>

      <!-- 工作流进度 -->
      <div v-if="workflowSteps.length > 0" class="workflow-section">
        <WorkflowProgressView
          :steps="workflowSteps"
          title="工作流进度"
          :show-progress="true"
          :show-steps="true"
          :show-metadata="false"
          :allow-navigation="false"
          :max-visible-steps="5"
        />
      </div>
    </div>

    <!-- 聊天区域 -->
    <div class="demo-chat">
      <Chat
        :messages="messages"
        :placeholder="`与 ${selectedAgent?.name || 'Agent'} 对话...`"
        :disabled="isThinking"
        :quick-replies="quickReplies"
        :toolbar="toolbar"
        @send="handleSend"
        @quick-reply="handleQuickReply"
        @card-action="handleCardAction"
        @ask-user-submit="handleAskUserSubmit"
      />
    </div>
  </div>

  <!-- Plan Mode 面板 -->
  <PlanModeView
    :active="chatStore.planMode.active"
    :content="chatStore.planMode.planContent"
    :plan-id="chatStore.planMode.planId"
    @approve="handlePlanApprove"
    @reject="handlePlanReject"
    @close="handlePlanClose"
  />
</div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount } from 'vue';
import { Chat } from '@/components/ChatUI';
import { useAsterClient } from '@/composables/useAsterClient';
import { generateId } from '@/utils/format';
import { useChatStore } from '@/stores/chat';
import { useThinkingStore } from '@/stores/thinking';
import { useToolsStore } from '@/stores/tools';
import { useTodosStore } from '@/stores/todos';
import { useApprovalStore } from '@/stores/approval';
import { useWorkflowStore } from '@/stores/workflow';
import WorkflowProgressView from '@/components/Workflow/WorkflowProgressView.vue';
import ApprovalCard from '@/components/Thinking/ApprovalCard.vue';
import ProviderSelector from '@/components/Settings/ProviderSelector.vue';
import AskUserQuestionCard from '@/components/Thinking/AskUserQuestionCard.vue';
import PlanModeView from '@/components/Planning/PlanModeView.vue';

interface Agent {
  id: string;
  name: string;
  description: string;
  status: 'idle' | 'thinking' | 'busy';
}

interface Message {
  id: string;
  type: 'text' | 'thinking' | 'typing' | 'card' | 'file';
  content?: string;
  position: 'left' | 'right';
  status?: 'pending' | 'sent' | 'error';
  conversationId?: string; // 添加对话ID
  user?: {
    avatar?: string;
    name?: string;
  };
  card?: {
    title: string;
    content: string;
    actions?: Array<{ text: string; value: string }>;
  };
  // Thinking-related fields
  hasThinking?: boolean; // 是否有思考过程
}

const { ensureWebSocket, onMessage, isConnected } = useAsterClient();
const wsConnected = isConnected;

// 初始化 Pinia Stores
const chatStore = useChatStore();
const thinkingStore = useThinkingStore();
const toolsStore = useToolsStore();
const todosStore = useTodosStore();
const approvalStore = useApprovalStore();
const workflowStore = useWorkflowStore();

// 通过 computed 从 stores 获取状态
const isThinking = computed(() => thinkingStore.isThinking);
const toolRunsList = computed(() => Array.from(toolsStore.toolRuns.values()));
const workflowSteps = computed(() => workflowStore.steps);
const pendingApprovalsList = computed(() => Array.from(approvalStore.pendingApprovals.values()));
const unansweredQuestions = computed(() =>
  chatStore.messages.filter((m: any) => m.type === 'ask-user' && !m.content?.answered)
);

// 转换消息，为 thinking 类型的消息注入 thinkingSteps
const messages = computed(() => {
  return chatStore.messages.map((msg: any) => {
    if (msg.type === 'thinking') {
      // 获取该消息关联的思考步骤
      const conversationId = msg.conversationId || currentConversationId.value;
      const steps = thinkingStore.getSteps(conversationId);
      return {
        ...msg,
        thinkingSteps: steps,
        isThinkingActive: thinkingStore.isThinking && thinkingStore.currentMessageId === conversationId,
      };
    }
    return msg;
  });
});

// 模拟 Agent 列表
const agents = ref<Agent[]>([
  {
    id: '1',
    name: '写作助手',
    description: '帮助你创作优质内容',
    status: 'idle',
  },
  {
    id: '2',
    name: '代码助手',
    description: '编程问题解答专家',
    status: 'idle',
  },
  {
    id: '3',
    name: '数据分析师',
    description: '数据洞察与可视化',
    status: 'idle',
  },
]);

const selectedAgent = ref(agents.value[0] as Agent);
let unsubscribeFn: (() => void) | null = null;
let currentConversationId = ref<string>(''); // 跟踪当前对话回合
const currentProvider = ref({ provider: 'deepseek', model: 'deepseek-chat' });

const quickReplies = computed(() => [
  { name: '帮我写一篇文章', value: 'write_article' },
  { name: '分析这段代码', value: 'analyze_code' },
  { name: '生成工作流', value: 'create_workflow' },
]);

const toolbar = [
  {
    icon: 'image',
    onClick: () => console.log('上传图片'),
  },
  {
    icon: 'attach',
    onClick: () => console.log('上传文件'),
  },
  {
    icon: 'mic',
    onClick: () => console.log('语音输入'),
  },
];

const selectAgent = (agent: Agent) => {
  selectedAgent.value = agent;
  chatStore.messages = [
    {
      id: generateId('greeting'),
      type: 'text',
      content: `你好！我是${agent.name}，${agent.description}。`,
      position: 'left',
      user: {
        id: agent.id,
        name: agent.name,
      },
    } as any,
  ];
};

const handleProviderChange = (config: { provider: string; model: string }) => {
  currentProvider.value = config;
  console.log('🔄 Provider changed:', config);
};

const handleAskUserSubmit = async (payload: { requestId: string; answers: Record<string, any> }) => {
  try {
    const ws = await ensureWebSocket();
    if (!ws) {
      console.error('WebSocket not connected, cannot send answer');
      return;
    }

    // 发送答案到后端
    ws.send({
      type: 'user_answer',
      payload: {
        request_id: payload.requestId,
        answers: payload.answers,
      },
    });

    // 标记问题为已回答
    const msg = chatStore.messages.find(
      (m: any) => m.type === 'ask-user' && m.content?.request_id === payload.requestId
    );
    if (msg && msg.type === 'ask-user') {
      (msg as any).content.answered = true;
      (msg as any).content.answers = payload.answers;
    }

    console.log('✅ User answers submitted:', payload);
  } catch (error) {
    console.error('Failed to submit user answers:', error);
  }
};

const handlePlanApprove = async () => {
  try {
    const ws = await ensureWebSocket();
    if (!ws || !chatStore.planMode.planId) {
      console.error('WebSocket not connected or no plan ID');
      return;
    }

    // 发送批准决策到后端
    ws.send({
      type: 'plan_decision',
      payload: {
        plan_id: chatStore.planMode.planId,
        decision: 'approve',
      },
    });

    console.log('✅ Plan approved:', chatStore.planMode.planId);
    chatStore.exitPlanMode();
  } catch (error) {
    console.error('Failed to approve plan:', error);
  }
};

const handlePlanReject = async () => {
  try {
    const ws = await ensureWebSocket();
    if (!ws || !chatStore.planMode.planId) {
      console.error('WebSocket not connected or no plan ID');
      return;
    }

    // 发送拒绝决策到后端
    ws.send({
      type: 'plan_decision',
      payload: {
        plan_id: chatStore.planMode.planId,
        decision: 'reject',
      },
    });

    console.log('❌ Plan rejected:', chatStore.planMode.planId);
    chatStore.exitPlanMode();
  } catch (error) {
    console.error('Failed to reject plan:', error);
  }
};

const handlePlanClose = () => {
  chatStore.exitPlanMode();
};

const handleSend = async (message: { type: string; content: string }) => {
  // 为新对话生成新的对话ID
  currentConversationId.value = generateId('conversation');

  // 添加用户消息
  const userMsg = {
    id: generateId('user'),
    type: 'text',
    content: message.content,
    position: 'right',
    status: 'sent',
  };
  chatStore.messages.push(userMsg as any);

  // 显示思考状态 - 创建 thinking 消息并关联 conversationId
  chatStore.isTyping = true;
  thinkingStore.startThinking(currentConversationId.value);
  const thinkingMsg = {
    id: generateId('thinking'),
    type: 'thinking',
    position: 'left',
    conversationId: currentConversationId.value, // 关联对话ID，用于获取思考步骤
  };
  chatStore.messages.push(thinkingMsg as any);

  try {
    const ws = await ensureWebSocket();
    if (!ws) {
      throw new Error('WebSocket not connected');
    }
    ws.send({
      type: 'chat',
      payload: {
        input: message.content,
        template_id: 'chat',
        model_config: {
          provider: currentProvider.value.provider,
          model: currentProvider.value.model,
        },
      },
    });
  } catch (error) {
    console.error('Chat error:', error);
    chatStore.messages = chatStore.messages.filter(m => !m.id.startsWith('thinking-'));
    chatStore.messages.push({
      id: generateId('error'),
      type: 'text',
      content: '抱歉，处理请求时出错了。请检查后端服务是否正常运行。',
      position: 'left',
      status: 'error',
    } as any);
    chatStore.isTyping = false;
    thinkingStore.endThinking();
  }
};

const handleQuickReply = (reply: { name: string; value?: string }) => {
  handleSend({
    type: 'text',
    content: reply.name,
  });
};

const handleCardAction = async (action: { value: string; metadata?: any }) => {
  console.log('Card action:', action);

  // 如果是 ask_user 的回答，发送到后端
  if (action.metadata?.askId) {
    try {
      const ws = await ensureWebSocket();
      if (ws) {
        ws.send({
          type: 'ask_user_response',
          payload: {
            ask_id: action.metadata.askId,
            answer: action.value,
          },
        });
      }
    } catch (err) {
      console.error('Failed to send ask_user response:', err);
    }
  }
};

/**
 * 批准审批请求
 */
const handleApprove = async (requestId: string) => {
  console.log('Approving request:', requestId);
  await approvalStore.approve(requestId);
};

/**
 * 拒绝审批请求
 */
const handleReject = async (requestId: string, reason?: string) => {
  console.log('Rejecting request:', requestId, 'reason:', reason);
  await approvalStore.reject(requestId, reason);
};

// 处理 WS 入站消息
const handleWsMessage = (msg: any) => {
  if (!msg) return;

  // 添加调试日志
  console.log('🔍 WS消息 received:', msg);

  switch (msg.type) {
    case 'text_delta': {
      const delta = msg.payload?.text || msg.payload?.delta || '';
      if (!delta) {
        console.log('⚠️ text_delta 消息没有文本内容:', msg);
        return;
      }

      console.log('✅ 处理 text_delta:', delta, '对话ID:', currentConversationId.value);

      // 不移除 thinking 消息，保留让用户可以查看思考过程
      // thinking 消息会自动折叠显示

      // 查找属于当前对话的最后一个AI回复消息
      let last: Message | undefined;
      for (let i = chatStore.messages.length - 1; i >= 0; i--) {
        const m = chatStore.messages[i] as any;
        // 查找属于当前对话的AI消息
        if (m?.position === 'left' && m?.type === 'text' &&
            m?.status !== 'system' && !m?.id?.includes('welcome') &&
            m?.conversationId === currentConversationId.value) {
          last = m;
          break;
        }
      }
      if (!last) {
        // 如果没有找到当前对话的消息，创建新的
        last = {
          id: generateId('assistant-' + currentConversationId.value),
          type: 'text',
          content: '',
          position: 'left',
          user: { name: selectedAgent.value.name },
          conversationId: currentConversationId.value,
        };
        chatStore.messages.push(last as any);
        console.log('🆕 创建新的AI消息:', last!.id);
      }

      // 更新消息内容
      const oldContent = last!.content || '';
      last!.content = oldContent + delta;
      console.log('📝 更新消息内容:', `"${oldContent}" -> "${last!.content}"`);

      // 强制触发响应式更新
      chatStore.messages = [...chatStore.messages];
      break;
    }
    case 'chat_complete': {
      chatStore.isTyping = false;
      // 不移除 thinking 消息，保留让用户可以查看思考过程
      break;
    }
    // 思考事件 - 直接路由到 handleAgentEvent
    case 'think_chunk_start':
    case 'think_chunk':
    case 'think_chunk_end': {
      handleAgentEvent(msg.type, msg.payload || {});
      break;
    }
    // 错误事件 - 直接路由到 handleAgentEvent
    case 'error':
    case 'stream_error': {
      handleAgentEvent(msg.type, msg.payload || {});
      break;
    }
    case 'agent_event': {
      const ev = msg.payload?.event;
      const evType = msg.payload?.type || ev?.type || ev?.EventType;
      if (!ev || !evType) return;
      handleAgentEvent(evType, ev);
      break;
    }
    default:
      break;
  }
};

/**
 * 完整的 Agent 事件处理函数
 * 处理所有事件类型: think_chunk, tool, approval, workflow, todo 等
 */
const handleAgentEvent = (type: string, ev: any) => {
  const messageId = currentConversationId.value;

  // 1. 思维事件 → thinkingStore
  if (type === 'think_chunk_start') {
    thinkingStore.startThinking(messageId);
    chatStore.setActiveMessage(messageId);
    return;
  }
  if (type === 'think_chunk') {
    thinkingStore.handleThinkChunk(ev.delta || ev.content || '');
    return;
  }
  if (type === 'think_chunk_end') {
    thinkingStore.endThinking();
    
    // 如果没有思考步骤（普通模型），移除当前对话的 thinking 消息
    const steps = thinkingStore.getSteps(messageId);
    if (!steps || steps.length === 0) {
      // 只移除当前对话的 thinking 消息，不影响其他对话
      chatStore.messages = chatStore.messages.filter(
        (m: any) => !(m.type === 'thinking' && m.conversationId === messageId)
      );
    }
    return;
  }

  // 2. 工具事件 → toolsStore + thinkingStore
  if (type === 'tool:start' || type === 'tool_call_start' || (type.startsWith('tool') && type.includes('start'))) {
    const call = ev.Call || ev.call || {};
    const toolCall = {
      id: call.id || call.ID || call.tool_call_id || generateId('tool'),
      name: call.name || 'unknown',
      state: 'executing' as const,
      progress: 0,
      arguments: call.arguments || {},
      cancelable: call.cancelable ?? false,
      pausable: call.pausable ?? false,
    };

    toolsStore.handleToolStart(toolCall);

    // 同时添加到思维步骤
    thinkingStore.addStep(messageId, {
      type: 'tool_call',
      tool: {
        name: toolCall.name,
        args: toolCall.arguments,
      },
      timestamp: Date.now(),
    });
    return;
  }

  if (type === 'tool:progress' || type === 'tool_call_progress' || (type.startsWith('tool') && type.includes('progress'))) {
    const call = ev.Call || ev.call || {};
    const id = call.id || call.ID || call.tool_call_id;
    if (id) {
      toolsStore.handleToolProgress(id, ev.progress ?? call.progress ?? 0, ev.message || '');
    }
    return;
  }

  if (type === 'tool:intermediate') {
    const call = ev.Call || ev.call || {};
    const id = call.id || call.ID || call.tool_call_id;
    if (id) {
      toolsStore.handleToolIntermediate(id, ev.label || '', ev.data);
    }
    return;
  }

  if (type === 'tool:end' || type === 'tool_call_end' || (type.startsWith('tool') && type.includes('end'))) {
    const call = ev.Call || ev.call || {};
    const id = call.id || call.ID || call.tool_call_id;
    if (id) {
      const toolCall = {
        id,
        name: call.name || 'unknown',
        state: (call.error || ev.error ? 'failed' : 'completed') as 'failed' | 'completed',
        progress: 1,
        arguments: call.arguments || {},
        result: call.result || ev.result,
        error: call.error || ev.error,
      };

      toolsStore.handleToolEnd(toolCall);

      // 添加工具结果到思维步骤
      thinkingStore.addStep(messageId, {
        type: 'tool_result',
        tool: {
          name: toolCall.name,
          args: toolCall.arguments,
        },
        result: toolCall.result,
        timestamp: Date.now(),
      });
    }
    return;
  }

  // 处理旧版本工具事件 (向后兼容)
  if (type.startsWith('tool')) {
    const call = ev.Call || ev.call || {};
    const id = call.id || call.ID || call.tool_call_id;
    if (!id) return;

    const toolCall = {
      id,
      name: call.name || 'unknown',
      state: (call.state || ev.state || 'executing') as any,
      progress: ev.progress ?? call.progress ?? 0,
      arguments: call.arguments || {},
      result: call.result || ev.result,
      error: ev.error || call.error,
      cancelable: call.cancelable ?? false,
      pausable: call.pausable ?? false,
    };

    if (type.includes('start')) {
      toolsStore.handleToolStart(toolCall);
    } else if (type.includes('end') || type.includes('complete')) {
      toolsStore.handleToolEnd(toolCall);
    } else {
      toolsStore.handleToolProgress(id, toolCall.progress, ev.message || '');
    }
    return;
  }

  // 3. 审批事件 → approvalStore + thinkingStore
  if (type === 'permission_required') {
    const call = ev.call || {};
    const requestId = ev.request_id || generateId('approval');

    approvalStore.addApprovalRequest({
      id: requestId,
      messageId: messageId,
      toolName: call.name || '',
      args: call.arguments || {},
      reason: ev.reason || '',
      timestamp: Date.now(),
    });

    // 添加审批步骤到思维过程
    thinkingStore.addStep(messageId, {
      type: 'approval',
      tool: {
        name: call.name,
        args: call.arguments,
      },
      timestamp: Date.now(),
    });

    console.log('Permission required for tool:', call.name);
    return;
  }

  // 4. Todo 事件 → todosStore
  if (type === 'todo_update' || type === 'todos_updated') {
    todosStore.updateTodos(ev.todos || []);
    return;
  }

  // 5. 工作流事件 → workflowStore
  if (type === 'workflow_start' || type === 'workflow:start') {
    workflowStore.loadWorkflow({
      id: ev.workflow_id || generateId('workflow'),
      name: ev.name || ev.title || '工作流',
      title: ev.title,
      steps: ev.steps || [],
    });
    return;
  }

  if (type === 'workflow_step_start' || type === 'workflow:step_start') {
    workflowStore.updateStep(ev.step_id, { status: 'active' });
    return;
  }

  if (type === 'workflow_step_complete' || type === 'workflow:step_complete') {
    workflowStore.completeStep(ev.step_id);
    return;
  }

  if (type === 'workflow_step_update' || type === 'workflow:step_update') {
    workflowStore.updateStep(ev.step_id, {
      status: ev.status,
      metadata: ev.metadata,
    });
    return;
  }

  if (type === 'workflow_complete' || type === 'workflow:complete') {
    // 标记工作流完成
    console.log('Workflow completed');
    return;
  }

  // 6. Context Compression 事件 → 系统提示
  if (type === 'context_compression') {
    if (ev.phase === 'start') {
      // 压缩开始
      chatStore.messages.push({
        id: generateId('system'),
        type: 'system',
        content: '🗜️ 正在压缩对话历史...',
        position: 'left',
        metadata: { type: 'info' },
      } as any);
    } else if (ev.phase === 'end') {
      // 压缩完成，计算节省比率
      const ratio = ev.ratio ? Math.round((1 - ev.ratio) * 100) : 0;
      chatStore.messages.push({
        id: generateId('system'),
        type: 'system',
        content: `✅ 对话历史压缩完成，节省 ${ratio}% 空间`,
        position: 'left',
        metadata: { type: 'success' },
      } as any);
    }
    return;
  }

  // 7. 状态变更事件
  if (type === 'state_changed') {
    const state = ev.state;
    if (state === 'working' || state === 'running') {
      // agent 正在工作
    } else if (state === 'idle' || state === 'ready' || state === 'completed') {
      chatStore.isTyping = false;
    }
    return;
  }

  // 8. AskUser 事件 → 显示问题卡片
  if (type === 'ask_user') {
    console.log('📝 Ask user:', ev.questions);

    // 添加 AskUser 消息
    chatStore.messages.push({
      id: generateId('ask'),
      type: 'ask-user',
      role: 'assistant',
      createdAt: Date.now(),
      position: 'left',
      content: {
        request_id: ev.request_id || generateId('request'),
        questions: ev.questions || [],
        answered: false,
      },
    } as any);
    return;
  }

  // 9. Plan Mode 事件
  if (type === 'plan_mode_entered' || type === 'enter_plan_mode') {
    console.log('📋 Entering Plan Mode:', ev.plan_id);
    chatStore.enterPlanMode(ev.plan_id || generateId('plan'), ev.content || ev.plan_content || '');
    return;
  }

  if (type === 'plan_mode_exited' || type === 'exit_plan_mode') {
    console.log('📋 Exiting Plan Mode');
    chatStore.exitPlanMode();
    return;
  }

  // 10. Token Usage 统计
  if (type === 'token_usage' || type === 'usage') {
    const usage = {
      inputTokens: ev.input_tokens || ev.prompt_tokens || 0,
      outputTokens: ev.output_tokens || ev.completion_tokens || 0,
      totalTokens: ev.total_tokens || 0,
    };
    console.log('📊 Token usage:', usage);
    
    // 可以存储到 chatStore 或显示在 UI
    // chatStore.tokenUsage = usage;
    return;
  }

  // 11. 错误事件
  if (type === 'error' || type === 'stream_error') {
    console.error('Agent error:', ev.message || ev.code, ev.detail);
    
    // 移除当前对话的思考中消息
    chatStore.messages = chatStore.messages.filter(
      (m: any) => !(m.type === 'thinking' && m.conversationId === messageId)
    );
    
    // 结束思考状态
    thinkingStore.endThinking();
    chatStore.isTyping = false;
    
    // 解析错误类型
    const errorMessage = ev.message || ev.code || '';
    let friendlyMessage = '抱歉，处理请求时出错了。';
    let errorType = 'error';
    
    if (errorMessage.includes('server_overloaded') || errorMessage.includes('overloaded')) {
      friendlyMessage = '🔥 服务器当前负载过高，请稍后重试';
      errorType = 'overloaded';
    } else if (errorMessage.includes('rate_limit') || errorMessage.includes('too many')) {
      friendlyMessage = '⏱️ 请求过于频繁，请稍后重试';
      errorType = 'rate_limit';
    } else if (errorMessage.includes('auth_error') || errorMessage.includes('api_key')) {
      friendlyMessage = '🔑 API Key 无效或已过期';
      errorType = 'auth';
    } else if (errorMessage.includes('timeout')) {
      friendlyMessage = '⏳ 请求超时，请稍后重试';
      errorType = 'timeout';
    }
    
    // 添加错误消息
    chatStore.messages.push({
      id: generateId('error'),
      type: 'text',
      content: friendlyMessage,
      position: 'left',
      status: 'error',
      metadata: { errorType },
    } as any);
    return;
  }
};

const controlTool = async (toolCallId: string, action: 'cancel' | 'pause' | 'resume') => {
  try {
    const ws = await ensureWebSocket();
    if (!ws) return;
    ws.send({
      type: 'tool:control',
      payload: {
        tool_call_id: toolCallId,
        action,
      },
    });
  } catch (err) {
    console.error('control tool failed', err);
  }
};

const formatResult = (res: any) => {
  try {
    return typeof res === 'string' ? res : JSON.stringify(res, null, 2);
  } catch {
    return String(res);
  }
};

const formatIntermediateValue = (value: any) => {
  try {
    if (typeof value === 'string') return value;
    if (typeof value === 'number' || typeof value === 'boolean') return String(value);
    return JSON.stringify(value);
  } catch {
    return String(value);
  }
};

onMounted(async () => {
  // 初始化时选中第一个agent并显示欢迎消息
  if (selectedAgent.value) {
    selectAgent(selectedAgent.value);
  }

  await ensureWebSocket();
  if (unsubscribeFn) unsubscribeFn();
  unsubscribeFn = onMessage(handleWsMessage);

  // 开发环境: 添加测试工具到浏览器控制台
  if (import.meta.env.DEV) {
    (window as any).testUI = {
      /**
       * 测试思考过程显示
       * 模拟: 思考 → 工具调用 → 工具结果
       */
      thinking: () => {
        const msgId = currentConversationId.value || generateId('test');
        chatStore.setActiveMessage(msgId);

        // 创建测试消息
        chatStore.messages.push({
          id: msgId,
          type: 'text',
          content: '正在分析问题...',
          position: 'left',
          conversationId: msgId,
        } as any);

        // 1. 开始思考
        thinkingStore.startThinking(msgId);
        console.log('✅ 启动思考过程');

        // 2. 添加推理步骤
        setTimeout(() => {
          thinkingStore.handleThinkChunk('分析当前情况...\n');
          thinkingStore.handleThinkChunk('考虑可能的解决方案...\n');
          console.log('✅ 添加推理内容');
        }, 500);

        // 3. 添加工具调用步骤
        setTimeout(() => {
          thinkingStore.addStep(msgId, {
            type: 'tool_call',
            tool: { name: 'bash', args: { command: 'ls -la' } },
            timestamp: Date.now(),
          });
          console.log('✅ 添加工具调用步骤');
        }, 1500);

        // 4. 添加工具结果步骤
        setTimeout(() => {
          thinkingStore.addStep(msgId, {
            type: 'tool_result',
            result: 'file1.txt\nfile2.js\npackage.json',
            timestamp: Date.now(),
          });
          console.log('✅ 添加工具结果步骤');
        }, 2500);

        // 5. 结束思考
        setTimeout(() => {
          thinkingStore.endThinking();
          console.log('✅ 结束思考,ThinkingBlock 应该可以折叠了');
        }, 3500);

        console.log('🧪 测试思考过程已启动,将在 3.5 秒内完成');
      },

      /**
       * 测试审批卡片显示
       * 显示需要用户审批的操作
       */
      approval: () => {
        const msgId = currentConversationId.value || generateId('test');
        chatStore.setActiveMessage(msgId);

        // 添加审批请求
        const approvalId = generateId('approval');
        approvalStore.addApprovalRequest({
          id: approvalId,
          messageId: msgId,
          toolName: 'file_delete',
          args: { path: '/important/config.json' },
          reason: '该操作将删除系统配置文件,可能影响应用正常运行。请确认是否继续?',
          timestamp: Date.now(),
        });

        // 添加审批步骤到思考过程
        thinkingStore.startThinking(msgId);
        thinkingStore.addStep(msgId, {
          type: 'approval',
          tool: { name: 'file_delete', args: { path: '/important/config.json' } },
          timestamp: Date.now(),
        });

        console.log('🧪 审批卡片已显示');
        console.log('💡 提示: ThinkingBlock 应该自动展开并高亮');
        console.log('💡 批准后可以调用: testUI.approveRequest("' + approvalId + '")');
      },

      /**
       * 批准测试审批请求
       */
      approveRequest: (requestId: string) => {
        approvalStore.approve(requestId);
        console.log('✅ 已批准请求:', requestId);
      },

      /**
       * 拒绝测试审批请求
       */
      rejectRequest: (requestId: string, reason?: string) => {
        approvalStore.reject(requestId, reason);
        console.log('❌ 已拒绝请求:', requestId);
      },

      /**
       * 测试工作流进度显示
       * 模拟多步骤任务执行
       */
      workflow: () => {
        // 加载工作流
        workflowStore.loadWorkflow({
          id: 'test-wf-' + Date.now(),
          name: '测试工作流: 构建项目',
          title: '测试工作流: 构建项目',
          steps: [
            {
              id: 'step1',
              title: '准备环境',
              description: '安装依赖包',
            },
            {
              id: 'step2',
              title: '运行测试',
              description: '执行单元测试和集成测试',
            },
            {
              id: 'step3',
              title: '构建项目',
              description: '编译 TypeScript 并打包',
            },
            {
              id: 'step4',
              title: '部署上线',
              description: '上传到生产环境',
            },
          ],
        });

        console.log('✅ 工作流已加载,左侧边栏应该显示进度');

        // 模拟步骤进行
        setTimeout(() => {
          workflowStore.completeStep('step2');
          workflowStore.updateStep('step3', { status: 'active' });
          console.log('✅ 步骤 2 完成,步骤 3 开始');
        }, 2000);

        setTimeout(() => {
          workflowStore.completeStep('step3');
          workflowStore.updateStep('step4', { status: 'active' });
          console.log('✅ 步骤 3 完成,步骤 4 开始');
        }, 4000);

        setTimeout(() => {
          workflowStore.completeStep('step4');
          console.log('✅ 工作流全部完成!');
        }, 6000);

        console.log('🧪 工作流测试已启动,将在 6 秒内完成');
      },

      /**
       * 测试工具执行进度
       * 显示工具执行和进度条
       */
      tool: () => {
        const msgId = currentConversationId.value || generateId('test');
        // 确保有活动的 thinking 消息
        if (!chatStore.messages.find((m: any) => m.type === 'thinking' && m.conversationId === msgId)) {
          chatStore.messages.push({
            id: generateId('thinking'),
            type: 'thinking',
            position: 'left',
            conversationId: msgId,
          } as any);
          thinkingStore.startThinking(msgId);
        }

        const toolCall = {
          id: generateId('tool'),
          name: 'web_search',
          state: 'executing' as const,
          progress: 0,
          arguments: { query: 'latest AI news 2025' },
          cancelable: true,
          pausable: false,
        };

        // 1. 开始执行工具
        toolsStore.handleToolStart(toolCall);
        thinkingStore.addStep(msgId, {
          type: 'tool_call',
          tool: { name: toolCall.name, args: toolCall.arguments },
          timestamp: Date.now(),
        });
        console.log('✅ 工具开始执行');

        // 模拟进度更新
        let progress = 0;
        const interval = setInterval(() => {
          progress += 0.15;
          if (progress >= 1) {
            clearInterval(interval);
            // 工具完成
            const result = {
              articles: [
                { title: 'GPT-5 发布在即', url: 'https://example.com/1' },
                { title: 'Claude 4 性能提升 50%', url: 'https://example.com/2' },
              ],
            };
            
            toolsStore.handleToolEnd({
              ...toolCall,
              state: 'completed',
              progress: 1,
              result,
            });
            
            thinkingStore.addStep(msgId, {
              type: 'tool_result',
              tool: { name: toolCall.name, args: toolCall.arguments },
              result,
              timestamp: Date.now(),
            });
            
            console.log('✅ 工具执行完成,显示结果');
          } else {
            // 更新进度
            const messages = ['正在连接...', '检索中...', '处理数据...'];
            const msg = messages[Math.floor(progress * messages.length)];
            toolsStore.handleToolProgress(toolCall.id, progress, msg);
          }
        }, 400);

        console.log('🧪 工具执行测试已启动, 将在 ThinkingBlock 中显示');
      },

      /**
       * 清除所有测试数据
       */
      clear: () => {
        thinkingStore.clearAllSteps();
        workflowStore.clearWorkflow();
        toolsStore.clearAllTools();
        approvalStore.clearAll();
        console.log('🧹 已清除所有测试数据');
      },

      /**
       * 测试 AskUser 问题卡片
       * 模拟: Agent 向用户提问
       */
      askUser: () => {
        const askId = generateId('ask-id');
        chatStore.messages.push({
          id: generateId('ask'),
          type: 'card',
          position: 'left',
          card: {
            title: '请选择操作',
            content: '您想要如何处理这个文件？',
            actions: [
              { text: '编辑', value: 'edit' },
              { text: '删除', value: 'delete' },
              { text: '跳过', value: 'skip' },
            ],
          },
          metadata: {
            askId: askId,
            questionType: 'single_choice',
          },
        } as any);
        console.log('✅ AskUser 问题卡片已创建');
        console.log(`💡 问题 ID: ${askId}`);
      },

      /**
       * 测试 Token 使用统计
       * 模拟: 显示 Token 消耗信息
       */
      tokenUsage: () => {
        const usage = {
          inputTokens: 1234,
          outputTokens: 567,
          totalTokens: 1801,
        };
        console.log('📊 Token Usage 统计:');
        console.log(`   输入 Token: ${usage.inputTokens}`);
        console.log(`   输出 Token: ${usage.outputTokens}`);
        console.log(`   总计 Token: ${usage.totalTokens}`);
        console.log('💡 实际使用时会从后端 token_usage 事件接收数据');
      },

      /**
       * 显示帮助信息
       */
      help: () => {
        console.log(`
🧪 测试工具使用说明
==================

1. testUI.thinking()          - 测试思考过程 (ThinkingBlock)
   显示: 推理 → 工具调用 → 工具结果

2. testUI.approval()          - 测试审批卡片 (ApprovalCard)
   显示: 需要批准的危险操作

3. testUI.approveRequest(id)  - 批准审批请求
   参数: approval request ID

4. testUI.rejectRequest(id, reason?) - 拒绝审批请求
   参数: approval request ID, 可选原因

5. testUI.workflow()          - 测试工作流进度 (WorkflowProgressView)
   显示: 多步骤任务执行进度

6. testUI.tool()              - 测试工具执行 (工具流)
   显示: 工具执行过程和进度条

7. testUI.askUser()           - 测试问题卡片 (AskUser)
   显示: Agent 向用户提问

8. testUI.tokenUsage()        - 测试 Token 统计
   显示: Token 消耗信息

9. testUI.clear()             - 清除所有测试数据
   重置: 所有 stores 到初始状态

10. testUI.help()             - 显示此帮助信息

💡 提示:
- 可以多次调用测试函数观察效果
- 使用 testUI.clear() 清理后重新测试
- 打开 Vue DevTools 查看 Pinia stores 状态变化
        `);
      },
    };

    console.log('🧪 测试工具已加载!');
    console.log('💡 输入 testUI.help() 查看使用说明');
  }
});

onBeforeUnmount(() => {
  if (unsubscribeFn) unsubscribeFn();
});
</script>

<style scoped>
.agent-chatui-demo {
  @apply min-h-screen bg-gray-50 dark:bg-gray-900;
}

.demo-container {
  @apply h-screen flex;
}

.demo-sidebar {
  @apply w-80 bg-white dark:bg-gray-800 border-r border-gray-200 dark:border-gray-700 flex flex-col;
}

.sidebar-header {
  @apply p-6 border-b border-gray-200 dark:border-gray-700;
}

.sidebar-title {
  @apply text-2xl font-bold text-gray-900 dark:text-white;
}

.sidebar-subtitle {
  @apply text-sm text-gray-500 dark:text-gray-400 mt-1;
}

.agent-selector {
  @apply overflow-y-auto p-4 space-y-2;
}

.provider-section {
  @apply p-4 border-t border-gray-200 dark:border-gray-700;
}

.workflow-section {
  @apply p-4 border-t border-gray-200 dark:border-gray-700;
}

.agent-item {
  @apply flex items-center gap-3 p-3 rounded-lg cursor-pointer transition-colors hover:bg-gray-50 dark:hover:bg-gray-700;
}

.agent-item.active {
  @apply bg-blue-50 dark:bg-blue-900/30 border border-blue-200 dark:border-blue-800;
}

.agent-avatar {
  @apply w-10 h-10 rounded-full overflow-hidden flex-shrink-0;
}

.avatar-placeholder {
  @apply w-full h-full bg-gradient-to-br from-blue-400 to-blue-600 flex items-center justify-center text-white font-bold text-lg;
}

.agent-info {
  @apply flex-1 min-w-0;
}

.agent-name {
  @apply text-sm font-semibold text-gray-900 dark:text-white truncate;
}

.agent-desc {
  @apply text-xs text-gray-500 dark:text-gray-400 truncate;
}

.agent-status {
  @apply w-2 h-2 rounded-full flex-shrink-0;
}

.status-idle {
  @apply bg-green-500;
}

.status-thinking {
  @apply bg-blue-500 animate-pulse;
}

.status-busy {
  @apply bg-amber-500 animate-pulse;
}

.demo-chat {
  @apply flex-1 flex flex-col;
}

/* 思考过程展示 */
.thinking-stream {
  @apply p-4 border-t border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800;
}

.thinking-container {
  @apply mb-4 last:mb-0;
}

/* WebSocket 状态指示器 */
.ws-status {
  @apply mt-4 flex items-center gap-2 text-sm;
}

.ws-status.online {
  @apply text-green-600 dark:text-green-400;
}

.ws-status .dot {
  @apply w-2 h-2 rounded-full bg-gray-400;
}

.ws-status.online .dot {
  @apply bg-green-500 animate-pulse;
}


</style>
