<template>
  <div class="todo-tool">
    <!-- 头部工具栏 -->
    <div class="todo-header">
      <div class="header-title">
        <Icon type="list" size="sm" />
        <span>任务管理</span>
      </div>
      <div class="header-actions">
        <button
          class="action-button"
          title="添加任务"
          @click="showAddForm = !showAddForm"
        >
          <Icon type="plus" size="sm" />
        </button>
        <button
          class="action-button"
          title="刷新任务"
          @click="refreshTodos"
        >
          <Icon type="refresh" size="sm" />
        </button>
        <button
          class="action-button"
          title="切换视图"
          @click="toggleViewMode"
        >
          <Icon :type="viewMode === 'list' ? 'grid' : 'list'" size="sm" />
        </button>
      </div>
    </div>

    <!-- 添加任务表单 -->
    <div v-if="showAddForm" class="add-form">
      <div class="form-content">
        <input
          v-model="newTodo.content"
          ref="newTodoInput"
          type="text"
          placeholder="输入任务内容（祈使句，如：Run tests）..."
          class="todo-input"
          @keydown.enter="addTodo"
          @keydown.esc="showAddForm = false"
        />
        <input
          v-model="newTodo.activeForm"
          type="text"
          placeholder="进行时描述（可选，如：Running tests）..."
          class="todo-input"
        />
        <div class="form-actions">
          <select v-model.number="newTodo.priority" class="priority-select">
            <option :value="0">优先级</option>
            <option :value="1">低</option>
            <option :value="2">中</option>
            <option :value="3">高</option>
          </select>
          <input
            v-model="newTodo.dueDate"
            type="date"
            class="date-input"
            title="截止日期"
          />
          <button
            class="add-button"
            :disabled="!newTodo.content.trim()"
            @click="addTodo"
          >
            添加
          </button>
          <button
            class="cancel-button"
            @click="showAddForm = false; newTodo.content = ''; newTodo.activeForm = ''"
          >
            取消
          </button>
        </div>
      </div>
    </div>

    <!-- 过滤器 -->
    <div class="todo-filters">
      <button
        v-for="filter in filters"
        :key="filter.key"
        :class="['filter-button', { active: currentFilter === filter.key }]"
        @click="currentFilter = filter.key"
      >
        <Icon :type="filter.icon as any" size="sm" />
        {{ filter.label }}
        <span class="filter-count">{{ filter.count }}</span>
      </button>
    </div>

    <!-- 任务列表 -->
    <div class="todo-list">
      <div
        v-for="todo in filteredTodos"
        :key="todo.id"
        :class="['todo-item', {
          'todo-completed': todo.status === 'completed',
          'todo-in-progress': todo.status === 'in_progress',
          'todo-priority-high': todo.priority >= 3,
          'todo-priority-medium': todo.priority === 2,
          'todo-priority-low': todo.priority === 1
        }]"
      >
        <div class="todo-main">
          <div class="todo-status-indicator">
            <button
              :class="['status-btn', `status-${todo.status}`]"
              :title="getStatusText(todo.status)"
              @click="toggleStatus(todo)"
            >
              <Icon 
                :type="todo.status === 'completed' ? 'check' : todo.status === 'in_progress' ? 'play' : 'clock'" 
                size="xs" 
              />
            </button>
          </div>

          <div class="todo-content">
            <div class="todo-text">
              <span :class="{ 'completed-text': todo.status === 'completed' }">
                {{ todo.status === 'in_progress' ? todo.activeForm : todo.content }}
              </span>
            </div>

            <div class="todo-meta">
              <span :class="['status-badge', `status-${todo.status}`]">
                {{ getStatusText(todo.status) }}
              </span>
              <span v-if="todo.priority > 0" :class="`priority-badge ${getPriorityClass(todo.priority)}`">
                {{ getPriorityText(todo.priority) }}
              </span>
              <span v-if="todo.metadata?.dueDate" class="due-date" :class="{ 'overdue': isOverdue(todo.metadata.dueDate) }">
                <Icon type="calendar" size="xs" />
                {{ formatDate(todo.metadata.dueDate) }}
              </span>
              <span class="created-date">
                创建于 {{ formatDateTime(todo.createdAt) }}
              </span>
            </div>
          </div>
        </div>

        <div class="todo-actions">
          <button
            class="action-btn edit-btn"
            title="编辑任务"
            @click="editTodo(todo)"
          >
            <Icon type="edit" size="xs" />
          </button>
          <button
            class="action-btn delete-btn"
            title="删除任务"
            @click="deleteTodo(todo.id)"
          >
            <Icon type="trash" size="xs" />
          </button>
        </div>
      </div>

      <!-- 空状态 -->
      <div v-if="filteredTodos.length === 0" class="empty-state">
        <Icon type="inbox" size="lg" />
        <p>{{ getEmptyMessage() }}</p>
      </div>
    </div>

    <!-- 统计信息 -->
    <div class="todo-stats">
      <div class="stat-item">
        <span class="stat-value">{{ todos.length }}</span>
        <span class="stat-label">总计</span>
      </div>
      <div class="stat-item">
        <span class="stat-value">{{ pendingCount }}</span>
        <span class="stat-label">待处理</span>
      </div>
      <div class="stat-item">
        <span class="stat-value">{{ inProgressCount }}</span>
        <span class="stat-label">进行中</span>
      </div>
      <div class="stat-item">
        <span class="stat-value">{{ completedCount }}</span>
        <span class="stat-label">已完成</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick, onMounted } from 'vue';
import Icon from '../ChatUI/Icon.vue';

// 对应后端 TodoItem 结构
interface Todo {
  id: string;
  content: string;
  status: 'pending' | 'in_progress' | 'completed';
  activeForm: string;  // 进行时形式描述
  priority: number;    // 数值优先级
  createdAt: string;
  updatedAt: string;
  completedAt?: string;
  metadata?: Record<string, any>;
}



interface Props {
  wsUrl?: string;
  sessionId?: string;
}

const props = withDefaults(defineProps<Props>(), {
  wsUrl: 'ws://localhost:8080/ws',
  sessionId: 'default',
});

const emit = defineEmits<{
  todoCreated: [todo: Todo];
  todoUpdated: [todo: Todo];
  todoDeleted: [id: string];
}>();

// 响应式数据
const todos = ref<Todo[]>([]);
const showAddForm = ref(false);
const currentFilter = ref('all');
const viewMode = ref<'list' | 'grid'>('list');
const newTodoInput = ref<HTMLInputElement>();
const websocket = ref<WebSocket | null>(null);

const newTodo = ref({
  content: '',
  activeForm: '',
  priority: 0,
  dueDate: '',
});

// 过滤器选项
const filters = computed(() => [
  { key: 'all', label: '全部', icon: 'list', count: todos.value.length },
  { key: 'pending', label: '待处理', icon: 'clock', count: todos.value.filter(t => t.status === 'pending').length },
  { key: 'in_progress', label: '进行中', icon: 'play', count: todos.value.filter(t => t.status === 'in_progress').length },
  { key: 'completed', label: '已完成', icon: 'check', count: todos.value.filter(t => t.status === 'completed').length },
  { key: 'overdue', label: '已逾期', icon: 'alert', count: overdueCount.value },
]);

// 计算属性
const filteredTodos = computed(() => {
  let filtered = todos.value;

  switch (currentFilter.value) {
    case 'pending':
      filtered = filtered.filter(t => t.status === 'pending');
      break;
    case 'in_progress':
      filtered = filtered.filter(t => t.status === 'in_progress');
      break;
    case 'completed':
      filtered = filtered.filter(t => t.status === 'completed');
      break;
    case 'overdue':
      filtered = filtered.filter(t => t.status !== 'completed' && t.metadata?.dueDate && isOverdue(t.metadata.dueDate));
      break;
  }

  return filtered.sort((a, b) => {
    // 优先级排序 (数值越大优先级越高)
    if (b.priority !== a.priority) {
      return b.priority - a.priority;
    }

    // 日期排序
    return new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime();
  });
});

const completedCount = computed(() => todos.value.filter(t => t.status === 'completed').length);
const pendingCount = computed(() => todos.value.filter(t => t.status === 'pending').length);
const inProgressCount = computed(() => todos.value.filter(t => t.status === 'in_progress').length);
const overdueCount = computed(() =>
  todos.value.filter(t => t.status !== 'completed' && t.metadata?.dueDate && isOverdue(t.metadata.dueDate)).length
);

// WebSocket 连接
const connectWebSocket = () => {
  try {
    websocket.value = new WebSocket(`${props.wsUrl}?session=${props.sessionId}`);

    websocket.value.onopen = () => {
      console.log('TodoTool WebSocket connected');
      requestTodos();
    };

    websocket.value.onmessage = (event) => {
      try {
        const message = JSON.parse(event.data);
        handleWebSocketMessage(message);
      } catch (error) {
        console.error('Failed to parse WebSocket message:', error);
      }
    };

    websocket.value.onclose = () => {
      console.log('TodoTool WebSocket disconnected');
      // 5秒后重连
      setTimeout(connectWebSocket, 5000);
    };

    websocket.value.onerror = (error) => {
      console.error('TodoTool WebSocket error:', error);
    };
  } catch (error) {
    console.error('Failed to connect WebSocket:', error);
  }
};

const handleWebSocketMessage = (message: any) => {
  const payload = message.payload || message;
  switch (message.type) {
    case 'todo_list_response':
      // 转换后端格式到前端格式
      todos.value = (payload.todos || []).map(normalizeTodo);
      break;
    case 'todo_created':
      const newTodoFromServer = normalizeTodo(payload.todo);
      const existingIndex = todos.value.findIndex(t => t.id === newTodoFromServer.id);
      if (existingIndex === -1) {
        todos.value.push(newTodoFromServer);
      }
      emit('todoCreated', newTodoFromServer);
      break;
    case 'todo_updated':
      const updatedTodo = normalizeTodo(payload.todo);
      const index = todos.value.findIndex(t => t.id === updatedTodo.id);
      if (index !== -1) {
        todos.value[index] = updatedTodo;
      }
      emit('todoUpdated', updatedTodo);
      break;
    case 'todo_deleted':
      const deletedId = payload.id;
      todos.value = todos.value.filter(t => t.id !== deletedId);
      emit('todoDeleted', deletedId);
      break;
    // 处理 agent_event 中的 todo_update 事件
    case 'agent_event':
      if (payload.type === 'todo_update' && payload.event?.todos) {
        todos.value = payload.event.todos.map(normalizeTodo);
      }
      break;
  }
};

// 标准化后端返回的 Todo 数据
const normalizeTodo = (todo: any): Todo => {
  return {
    id: todo.id || todo.ID,
    content: todo.content || todo.Content,
    status: todo.status || todo.Status || 'pending',
    activeForm: todo.activeForm || todo.ActiveForm || todo.active_form || '',
    priority: todo.priority ?? todo.Priority ?? 0,
    createdAt: todo.createdAt || todo.CreatedAt || todo.created_at || new Date().toISOString(),
    updatedAt: todo.updatedAt || todo.UpdatedAt || todo.updated_at || new Date().toISOString(),
    completedAt: todo.completedAt || todo.CompletedAt || todo.completed_at,
    metadata: todo.metadata || todo.Metadata || {},
  };
};

const sendWebSocketMessage = (message: any) => {
  if (websocket.value && websocket.value.readyState === WebSocket.OPEN) {
    websocket.value.send(JSON.stringify(message));
  }
};

// 任务操作方法
const requestTodos = () => {
  sendWebSocketMessage({ 
    type: 'todo_list_request',
    payload: {
      list_name: 'default',
    },
  });
};

const addTodo = () => {
  if (!newTodo.value.content.trim()) return;

  // 自动生成 activeForm（如果未提供）
  const activeForm = newTodo.value.activeForm.trim() || `${newTodo.value.content.trim()}中`;

  const todo = {
    content: newTodo.value.content.trim(),
    status: 'pending',
    activeForm,
    priority: newTodo.value.priority,
  };

  sendWebSocketMessage({
    type: 'todo_create',
    payload: {
      todo,
      list_name: 'default',
    },
  });

  // 重置表单
  newTodo.value = { content: '', activeForm: '', priority: 0, dueDate: '' };
  showAddForm.value = false;
};

const updateTodo = (todo: Todo) => {
  sendWebSocketMessage({
    type: 'todo_update',
    payload: {
      todo: {
        id: todo.id,
        content: todo.content,
        status: todo.status,
        activeForm: todo.activeForm,
        priority: todo.priority,
        completed: todo.status === 'completed',
      },
      list_name: 'default',
    },
  });
};

const deleteTodo = (id: string) => {
  if (confirm('确定要删除这个任务吗？')) {
    sendWebSocketMessage({
      type: 'todo_delete',
      payload: {
        id,
        list_name: 'default',
      },
    });
  }
};

const editTodo = (todo: Todo) => {
  const newContent = prompt('编辑任务内容:', todo.content);
  if (newContent && newContent.trim() !== todo.content) {
    updateTodo({
      ...todo,
      content: newContent.trim(),
      updatedAt: new Date().toISOString(),
    });
  }
};

const refreshTodos = () => {
  requestTodos();
};

const toggleViewMode = () => {
  viewMode.value = viewMode.value === 'list' ? 'grid' : 'list';
};

// 工具方法
const getPriorityText = (priority: number) => {
  if (priority >= 3) return '高';
  if (priority >= 2) return '中';
  if (priority >= 1) return '低';
  return '';
};

const getPriorityClass = (priority: number) => {
  if (priority >= 3) return 'priority-high';
  if (priority >= 2) return 'priority-medium';
  if (priority >= 1) return 'priority-low';
  return '';
};

const getStatusText = (status: string) => {
  const map = { pending: '待处理', in_progress: '进行中', completed: '已完成' };
  return map[status as keyof typeof map] || status;
};

// 切换任务状态
const toggleStatus = (todo: Todo) => {
  let newStatus: 'pending' | 'in_progress' | 'completed';
  
  if (todo.status === 'pending') {
    // 检查是否已有进行中的任务
    const hasInProgress = todos.value.some(t => t.id !== todo.id && t.status === 'in_progress');
    if (hasInProgress) {
      alert('同时只能有一个任务处于进行中状态');
      return;
    }
    newStatus = 'in_progress';
  } else if (todo.status === 'in_progress') {
    newStatus = 'completed';
  } else {
    newStatus = 'pending';
  }

  updateTodo({
    ...todo,
    status: newStatus,
    activeForm: newStatus === 'completed' ? `已完成${todo.content}` : todo.activeForm,
  });
};

const formatDate = (dateString: string) => {
  const date = new Date(dateString);
  return date.toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' });
};

const formatDateTime = (dateString: string) => {
  const date = new Date(dateString);
  return date.toLocaleDateString('zh-CN', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  });
};

const isOverdue = (dateString: string) => {
  return new Date(dateString) < new Date(new Date().toDateString());
};

const getEmptyMessage = () => {
  const messages = {
    all: '暂无任务，点击 + 添加第一个任务',
    active: '暂无进行中的任务',
    completed: '暂无已完成的任务',
    overdue: '暂无逾期的任务',
  };
  return messages[currentFilter.value as keyof typeof messages];
};

// 生命周期
onMounted(() => {
  connectWebSocket();

  // 自动聚焦到添加输入框
  watch(showAddForm, (show) => {
    if (show) {
      nextTick(() => {
        newTodoInput.value?.focus();
      });
    }
  });
});

// 组件卸载时关闭 WebSocket连接
</script>

<style scoped>
.todo-tool {
  @apply flex flex-col h-full bg-surface dark:bg-surface-dark border border-border dark:border-border-dark rounded-lg;
  max-height: 600px;
}

.todo-header {
  @apply flex items-center justify-between px-4 py-3 border-b border-border dark:border-border-dark bg-surface dark:bg-surface-dark;
}

.header-title {
  @apply flex items-center gap-2 font-semibold text-text dark:text-text-dark;
}

.header-actions {
  @apply flex gap-1;
}

.action-button {
  @apply p-1.5 text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700 rounded transition-colors;
}

.add-form {
  @apply px-4 py-3 bg-blue-50 dark:bg-blue-900/20 border-b border-border dark:border-border-dark;
}

.form-content {
  @apply space-y-2;
}

.todo-input {
  @apply w-full px-3 py-2 border border-border dark:border-border-dark rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-800 dark:text-white;
}

.form-actions {
  @apply flex gap-2 items-center;
}

.priority-select, .date-input {
  @apply px-2 py-1 text-sm border border-border dark:border-border-dark rounded focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-800 dark:text-white;
}

.add-button {
  @apply px-3 py-1 bg-blue-500 hover:bg-blue-600 text-white text-sm rounded transition-colors disabled:opacity-50 disabled:cursor-not-allowed;
}

.cancel-button {
  @apply px-3 py-1 text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 text-sm rounded transition-colors;
}

.todo-filters {
  @apply flex gap-1 px-4 py-2 border-b border-border dark:border-border-dark bg-surface dark:bg-surface-dark;
}

.filter-button {
  @apply flex items-center gap-2 px-3 py-1.5 text-sm text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white hover:bg-gray-100 dark:hover:bg-gray-700 rounded transition-colors;
}

.filter-button.active {
  @apply bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300;
}

.filter-count {
  @apply text-xs bg-gray-200 dark:bg-gray-600 px-1.5 py-0.5 rounded-full;
}

.todo-list {
  @apply flex-1 overflow-y-auto px-4 py-2 space-y-1;
}

.todo-item {
  @apply flex items-center gap-3 p-3 rounded-lg border border-border dark:border-border-dark bg-surface dark:bg-surface-dark hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors;
}

.todo-completed {
  @apply opacity-60;
}

.todo-completed .todo-text {
  @apply line-through;
}

.todo-in-progress {
  @apply bg-blue-50 dark:bg-blue-900/20 border-blue-200 dark:border-blue-800;
}

.todo-status-indicator {
  @apply flex-shrink-0;
}

.status-btn {
  @apply w-6 h-6 rounded-full flex items-center justify-center transition-colors;
}

.status-btn.status-pending {
  @apply bg-gray-200 dark:bg-gray-600 text-gray-500 dark:text-gray-400 hover:bg-gray-300 dark:hover:bg-gray-500;
}

.status-btn.status-in_progress {
  @apply bg-blue-500 text-white hover:bg-blue-600;
}

.status-btn.status-completed {
  @apply bg-green-500 text-white hover:bg-green-600;
}

.status-badge {
  @apply text-xs px-1.5 py-0.5 rounded font-medium;
}

.status-badge.status-pending {
  @apply bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-300;
}

.status-badge.status-in_progress {
  @apply bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300;
}

.status-badge.status-completed {
  @apply bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-300;
}

.todo-priority-high {
  @apply border-l-4 border-l-red-500;
}

.todo-priority-medium {
  @apply border-l-4 border-l-yellow-500;
}

.todo-priority-low {
  @apply border-l-4 border-l-green-500;
}

.todo-main {
  @apply flex items-center gap-3 flex-1 min-w-0;
}

.todo-checkbox {
  @apply flex-shrink-0;
}

.todo-content {
  @apply flex-1 min-w-0;
}

.todo-text {
  @apply text-sm text-text dark:text-text-dark break-words;
}

.completed-text {
  @apply text-gray-500 dark:text-gray-400;
}

.todo-meta {
  @apply flex items-center gap-2 mt-1;
}

.priority-badge {
  @apply text-xs px-1.5 py-0.5 rounded font-medium;
}

.priority-high {
  @apply bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-300;
}

.priority-medium {
  @apply bg-yellow-100 dark:bg-yellow-900/30 text-yellow-700 dark:text-yellow-300;
}

.priority-low {
  @apply bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-300;
}

.due-date {
  @apply flex items-center gap-1 text-xs text-gray-500 dark:text-gray-400;
}

.due-date.overdue {
  @apply text-red-500 dark:text-red-400;
}

.created-date {
  @apply text-xs text-gray-400 dark:text-gray-500;
}

.todo-actions {
  @apply flex gap-1 opacity-0 hover:opacity-100 transition-opacity;
}

.todo-item:hover .todo-actions {
  @apply opacity-100;
}

.action-btn {
  @apply p-1 text-gray-400 dark:text-gray-500 hover:text-gray-600 dark:hover:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-600 rounded transition-colors;
}

.delete-btn:hover {
  @apply text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20;
}

.empty-state {
  @apply flex flex-col items-center justify-center py-8 text-gray-400 dark:text-gray-500;
}

.todo-stats {
  @apply flex items-center justify-around px-4 py-3 border-t border-border dark:border-border-dark bg-surface dark:bg-surface-dark;
}

.stat-item {
  @apply text-center;
}

.stat-value {
  @apply block text-lg font-semibold text-text dark:text-text-dark;
}

.stat-label {
  @apply block text-xs text-gray-500 dark:text-gray-400;
}
</style>