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
      />

      <!-- 工具流展示 -->
      <div class="tool-stream" v-if="toolRunsList.length">
        <div class="tool-stream-header">
          <h3>工具执行</h3>
          <span class="hint">实时状态 / 可取消</span>
        </div>
        <div class="tool-run" v-for="run in toolRunsList" :key="run.tool_call_id">
          <div class="tool-run-head">
            <div class="tool-name">{{ run.name }}</div>
            <div class="tool-state" :class="run.state">{{ run.state }}</div>
          </div>
          <div class="tool-progress">
            <div class="bar">
              <div class="bar-inner" :style="{ width: `${Math.round((run.progress || 0)*100)}%` }"></div>
            </div>
            <div class="meta">
              <span>{{ Math.round((run.progress || 0)*100) }}%</span>
              <span v-if="run.message">{{ run.message }}</span>
            </div>
          </div>
          <div class="tool-actions">
            <button v-if="run.cancelable && run.state === 'executing'" @click="controlTool(run.tool_call_id, 'cancel')">取消</button>
            <button v-if="run.pausable && run.state === 'executing'" @click="controlTool(run.tool_call_id, 'pause')">暂停</button>
            <button v-if="run.pausable && run.state === 'paused'" @click="controlTool(run.tool_call_id, 'resume')">继续</button>
          </div>
          <pre v-if="run.result" class="tool-result">{{ formatResult(run.result) }}</pre>
          <pre v-if="run.error" class="tool-error">Error: {{ run.error }}</pre>
        </div>
      </div>
    </div>
  </div>
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

const { client, ensureWebSocket, onMessage, isConnected } = useAsterClient();
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

const selectedAgent = ref<Agent>(agents.value[0]);
let unsubscribeFn: (() => void) | null = null;
let currentConversationId = ref<string>(''); // 跟踪当前对话回合

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
        name: agent.name,
      },
    },
  ];
};

const handleSend = async (message: { type: string; content: string }) => {
  // 为新对话生成新的对话ID
  currentConversationId.value = generateId('conversation');

  // 添加用户消息
  const userMsg: Message = {
    id: generateId('user'),
    type: 'text',
    content: message.content,
    position: 'right',
    status: 'sent',
  };
  chatStore.messages.push(userMsg);

  // 显示思考状态 - 创建 thinking 消息并关联 conversationId
  chatStore.isTyping = true;
  thinkingStore.startThinking(currentConversationId.value);
  const thinkingMsg: Message = {
    id: generateId('thinking'),
    type: 'thinking',
    position: 'left',
    conversationId: currentConversationId.value, // 关联对话ID，用于获取思考步骤
  };
  chatStore.messages.push(thinkingMsg);

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
    });
    chatStore.isTyping = false;
    thinkingStore.endThinking(currentConversationId.value);
  }
};

const handleQuickReply = (reply: { name: string; value?: string }) => {
  handleSend({
    type: 'text',
    content: reply.name,
  });
};

const handleCardAction = (action: { value: string }) => {
  console.log('Card action:', action);
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

      // 第一次收到文本时，移除thinking消息
      if (chatStore.messages.some(m => m.type === 'thinking')) {
        chatStore.messages = chatStore.messages.filter(m => m.type !== 'thinking');
        console.log('🗑️ 移除思考状态消息');
      }

      // 查找属于当前对话的最后一个AI回复消息
      let last: Message | undefined;
      for (let i = chatStore.messages.length - 1; i >= 0; i--) {
        const m = chatStore.messages[i];
        // 查找属于当前对话的AI消息
        if (m.position === 'left' && m.type === 'text' &&
            m.status !== 'system' && !m.id.includes('welcome') &&
            m.conversationId === currentConversationId.value) {
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
        chatStore.messages.push(last);
        console.log('🆕 创建新的AI消息:', last.id);
      }

      // 更新消息内容
      const oldContent = last.content || '';
      last.content = oldContent + delta;
      console.log('📝 更新消息内容:', `"${oldContent}" -> "${last.content}"`);

      // 强制触发响应式更新
      chatStore.messages = [...chatStore.messages];
      break;
    }
    case 'chat_complete': {
      chatStore.isTyping = false;
      chatStore.messages = chatStore.messages.filter(m => !m.id.startsWith('thinking-'));
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
    thinkingStore.endThinking(messageId);
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

  if (type === 'tool:end' || type === 'tool_call_end' || (type.startsWith('tool') && type.includes('end'))) {
    const call = ev.Call || ev.call || {};
    const id = call.id || call.ID || call.tool_call_id;
    if (id) {
      const toolCall = {
        id,
        name: call.name || 'unknown',
        state: (call.error || ev.error ? 'failed' : 'completed') as const,
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
      title: ev.title || '工作流',
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

  // 6. 状态变更事件
  if (type === 'state_changed') {
    const state = ev.state;
    if (state === 'working' || state === 'running') {
      // agent 正在工作
    } else if (state === 'idle' || state === 'ready' || state === 'completed') {
      chatStore.isTyping = false;
    }
    return;
  }

  // 7. 错误事件
  if (type === 'error') {
    console.error('Agent error:', ev.message, ev.detail);
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

onMounted(async () => {
  // 初始化时选中第一个agent并显示欢迎消息
  selectAgent(selectedAgent.value);

  await ensureWebSocket();
  if (unsubscribeFn) unsubscribeFn();
  unsubscribeFn = onMessage(handleWsMessage);

  // 开发环境下暴露测试函数到 window
  if (import.meta.env.DEV) {
    const w = window as any;
    
    // 测试思考过程
    w.testThinking = () => {
      const msgId = generateId('conversation');
      currentConversationId.value = msgId;
      
      // 1. 添加 thinking 消息到消息列表
      const thinkingMsg = {
        id: generateId('thinking'),
        type: 'thinking',
        position: 'left',
        conversationId: msgId,
      };
      chatStore.messages.push(thinkingMsg as any);
      
      // 2. 启动思考过程
      thinkingStore.startThinking(msgId);
      thinkingStore.handleThinkChunk('正在分析问题...');
      
      // 3. 添加工具调用步骤
      setTimeout(() => {
        thinkingStore.addStep(msgId, {
          type: 'tool_call',
          tool: { name: 'bash', args: { command: 'ls -la' } },
          timestamp: Date.now(),
        });
      }, 1000);
      
      // 4. 添加工具结果步骤
      setTimeout(() => {
        thinkingStore.addStep(msgId, {
          type: 'tool_result',
          tool: { name: 'bash', args: { command: 'ls -la' } },
          result: 'total 48\ndrwxr-xr-x  12 user  staff   384 Nov 27 10:00 .',
          timestamp: Date.now(),
        });
      }, 2000);
      
      console.log('✅ testThinking() 已触发，检查消息流中的 ThinkingBlock');
    };

    // 测试审批卡片
    w.testApproval = () => {
      const msgId = currentConversationId.value || generateId('test');
      currentConversationId.value = msgId;
      approvalStore.addApprovalRequest({
        id: generateId('approval'),
        messageId: msgId,
        toolName: 'file_delete',
        args: { path: '/important/config.json' },
        reason: '需要删除重要配置文件',
        timestamp: Date.now(),
      });
      thinkingStore.addStep(msgId, {
        type: 'approval',
        tool: { name: 'file_delete', args: { path: '/important/config.json' } },
        timestamp: Date.now(),
      });
      console.log('✅ testApproval() 已触发，检查 ApprovalCard 是否显示');
    };

    // 测试工作流
    w.testWorkflow = () => {
      workflowStore.loadWorkflow({
        id: generateId('workflow'),
        title: '测试工作流',
        steps: [
          { id: 'step1', title: '准备环境', status: 'completed' },
          { id: 'step2', title: '执行任务', status: 'active' },
          { id: 'step3', title: '验证结果', status: 'pending' },
        ],
      });
      console.log('✅ testWorkflow() 已触发，检查侧边栏 WorkflowProgressView 是否显示');
    };

    // 测试工具执行
    w.testTool = () => {
      const toolId = generateId('tool');
      toolsStore.handleToolStart({
        id: toolId,
        name: 'web_search',
        state: 'executing',
        progress: 0,
        arguments: { query: 'Aster Agent Framework' },
        cancelable: true,
        pausable: false,
      });
      let progress = 0;
      const interval = setInterval(() => {
        progress += 0.2;
        if (progress >= 1) {
          clearInterval(interval);
          toolsStore.handleToolEnd({
            id: toolId,
            name: 'web_search',
            state: 'completed',
            progress: 1,
            arguments: { query: 'Aster Agent Framework' },
            result: { results: ['Result 1', 'Result 2', 'Result 3'] },
          });
        } else {
          toolsStore.handleToolProgress(toolId, progress, `搜索中... ${Math.round(progress * 100)}%`);
        }
      }, 500);
      console.log('✅ testTool() 已触发，检查工具执行区域');
    };

    console.log('🧪 开发测试函数已加载:');
    console.log('  - testThinking()  测试思考过程');
    console.log('  - testApproval()  测试审批卡片');
    console.log('  - testWorkflow()  测试工作流');
    console.log('  - testTool()      测试工具执行');
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

/* 工具流展示样式保持不变 */
.tool-stream {
  @apply p-4 border-t border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-900;
}

.tool-stream-header {
  @apply flex items-center justify-between mb-4;
}

.tool-stream-header h3 {
  @apply text-lg font-semibold text-gray-900 dark:text-white;
}

.tool-stream-header .hint {
  @apply text-xs text-gray-500 dark:text-gray-400;
}

.tool-run {
  @apply bg-white dark:bg-gray-800 rounded-lg p-4 mb-3 border border-gray-200 dark:border-gray-700;
}

.tool-run-head {
  @apply flex items-center justify-between mb-3;
}

.tool-name {
  @apply font-mono text-sm font-semibold text-gray-900 dark:text-white;
}

.tool-state {
  @apply text-xs px-2 py-1 rounded-full;
}

.tool-state.executing {
  @apply bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400;
}

.tool-state.completed {
  @apply bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400;
}

.tool-state.failed {
  @apply bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400;
}

.tool-state.paused {
  @apply bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400;
}

.tool-progress {
  @apply mb-3;
}

.tool-progress .bar {
  @apply w-full h-2 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden mb-2;
}

.tool-progress .bar-inner {
  @apply h-full bg-blue-500 transition-all duration-300;
}

.tool-progress .meta {
  @apply flex items-center justify-between text-xs text-gray-600 dark:text-gray-400;
}

.tool-actions {
  @apply flex gap-2 mb-3;
}

.tool-actions button {
  @apply px-3 py-1 text-sm rounded-md bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-700 dark:text-gray-300 transition-colors;
}

.tool-result,
.tool-error {
  @apply text-xs font-mono p-3 rounded-md overflow-x-auto;
}

.tool-result {
  @apply bg-gray-100 dark:bg-gray-900 text-gray-800 dark:text-gray-200;
}

.tool-error {
  @apply bg-red-50 dark:bg-red-900/20 text-red-700 dark:text-red-400;
}
</style>
