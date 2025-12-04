<template>
  <div class="task-tool">
    <!-- 头部工具栏 -->
    <div class="task-header">
      <div class="header-title">
        <Icon type="play" size="sm" />
        <span>任务执行</span>
      </div>
      <div class="header-actions">
        <button class="action-button" title="刷新任务" @click="refreshTasks">
          <Icon type="refresh" size="sm" />
        </button>
        <button class="action-button" title="清理已完成" @click="clearCompleted">
          <Icon type="trash" size="sm" />
        </button>
      </div>
    </div>

    <!-- 任务创建区域 -->
    <div class="task-create">
      <div class="create-content">
        <input v-model="newTask.command" ref="commandInput" type="text" placeholder="输入任务命令..." class="task-input" @keydown.enter="executeTask" @keydown.esc="newTask.command = ''" />
        <div class="create-options">
          <select v-model="newTask.type" class="task-type-select">
            <option value="">任务类型</option>
            <option value="shell">Shell 命令</option>
            <option value="script">脚本执行</option>
            <option value="api">API 调用</option>
            <option value="file">文件操作</option>
          </select>
          <select v-model="newTask.priority" class="priority-select">
            <option value="">优先级</option>
            <option value="low">低</option>
            <option value="medium">中</option>
            <option value="high">高</option>
          </select>
          <button class="execute-button" :disabled="!newTask.command.trim() || isExecuting" @click="executeTask">
            <span v-if="isExecuting">执行中...</span>
            <span v-else>执行</span>
          </button>
        </div>
      </div>
    </div>

    <!-- 过滤器 -->
    <div class="task-filters">
      <button v-for="filter in filters" :key="filter.key" :class="['filter-button', { active: currentFilter === filter.key }]" @click="currentFilter = filter.key">
        <Icon :type="filter.icon as any" size="sm" />
        {{ filter.label }}
        <span class="filter-count">{{ filter.count }}</span>
      </button>
    </div>

    <!-- 任务列表 -->
    <div class="task-list">
      <div
        v-for="task in filteredTasks"
        :key="task.id"
        :class="[
          'task-item',
          {
            'task-running': task.status === 'running',
            'task-completed': task.status === 'completed',
            'task-failed': task.status === 'failed',
            'task-priority-high': task.priority === 'high',
            'task-priority-medium': task.priority === 'medium',
            'task-priority-low': task.priority === 'low',
          },
        ]"
      >
        <div class="task-main">
          <div class="task-status">
            <div v-if="task.status === 'running'" class="status-spinner"></div>
            <Icon v-else-if="task.status === 'completed'" type="check" size="sm" class="status-success" />
            <Icon v-else-if="task.status === 'failed'" type="close" size="sm" class="status-error" />
            <Icon v-else type="clock" size="sm" class="status-pending" />
          </div>

          <div class="task-content">
            <div class="task-command">{{ task.command }}</div>
            <div class="task-meta">
              <span v-if="task.type" :class="`type-badge type-${task.type}`">
                {{ getTypeText(task.type) }}
              </span>
              <span v-if="task.priority" :class="`priority-badge priority-${task.priority}`">
                {{ getPriorityText(task.priority) }}
              </span>
              <span class="created-time">
                <Icon type="clock" size="xs" />
                {{ formatDateTime(task.createdAt) }}
              </span>
              <span v-if="task.duration" class="duration">
                <Icon type="time" size="xs" />
                {{ formatDuration(task.duration) }}
              </span>
            </div>
          </div>
        </div>

        <div class="task-actions">
          <button v-if="task.status === 'completed'" class="action-btn view-btn" title="查看结果" @click="viewResult(task)">
            <Icon type="eye" size="xs" />
          </button>
          <button v-if="task.status === 'running'" class="action-btn stop-btn" title="停止任务" @click="stopTask(task.id)">
            <Icon type="stop" size="xs" />
          </button>
          <button v-if="task.status === 'failed'" class="action-btn retry-btn" title="重试" @click="retryTask(task)">
            <Icon type="refresh" size="xs" />
          </button>
          <button class="action-btn delete-btn" title="删除任务" @click="deleteTask(task.id)">
            <Icon type="trash" size="xs" />
          </button>
        </div>
      </div>

      <!-- 运行中的任务进度 -->
      <div v-for="task in runningTasks" :key="`progress-${task.id}`" class="task-progress">
        <div class="progress-info">
          <span class="progress-command">{{ task.command }}</span>
          <span class="progress-time">{{ formatDuration(Date.now() - (task.startedAt ?? Date.now())) }}</span>
        </div>
        <div class="progress-bar">
          <div class="progress-fill" :style="{ width: task.progress + '%' }"></div>
        </div>
        <div v-if="task.output" class="progress-output">{{ task.output }}</div>
      </div>

      <!-- 空状态 -->
      <div v-if="filteredTasks.length === 0 && runningTasks.length === 0" class="empty-state">
        <Icon type="inbox" size="lg" />
        <p>{{ getEmptyMessage() }}</p>
      </div>
    </div>

    <!-- 统计信息 -->
    <div class="task-stats">
      <div class="stat-item">
        <span class="stat-value">{{ totalTasks }}</span>
        <span class="stat-label">总任务</span>
      </div>
      <div class="stat-item">
        <span class="stat-value">{{ runningCount }}</span>
        <span class="stat-label">运行中</span>
      </div>
      <div class="stat-item">
        <span class="stat-value">{{ completedCount }}</span>
        <span class="stat-label">已完成</span>
      </div>
      <div class="stat-item">
        <span class="stat-value">{{ failedCount }}</span>
        <span class="stat-label">失败</span>
      </div>
    </div>

    <!-- 任务结果模态框 -->
    <div v-if="showResultModal" class="modal-overlay" @click.self="showResultModal = false">
      <div class="modal-content">
        <div class="modal-header">
          <h3>任务执行结果</h3>
          <button @click="showResultModal = false">
            <Icon type="close" size="sm" />
          </button>
        </div>
        <div class="modal-body">
          <div v-if="selectedTask" class="result-content">
            <div class="result-header">
              <span class="result-command">{{ selectedTask.command }}</span>
              <span :class="`result-status status-${selectedTask.status}`">
                {{ getStatusText(selectedTask.status) }}
              </span>
            </div>
            <div class="result-details">
              <div class="detail-item">
                <span class="detail-label">执行时间:</span>
                <span class="detail-value">{{ formatDateTime(selectedTask.createdAt) }}</span>
              </div>
              <div class="detail-item">
                <span class="detail-label">持续时间:</span>
                <span class="detail-value">{{ formatDuration(selectedTask.duration ?? 0) }}</span>
              </div>
              <div v-if="selectedTask.exitCode !== undefined" class="detail-item">
                <span class="detail-label">退出码:</span>
                <span :class="`detail-value exit-code-${selectedTask.exitCode === 0 ? 'success' : 'error'}`">
                  {{ selectedTask.exitCode }}
                </span>
              </div>
            </div>
            <div v-if="selectedTask.output" class="result-output">
              <h4>输出结果:</h4>
              <pre>{{ selectedTask.output }}</pre>
            </div>
            <div v-if="selectedTask.error" class="result-error">
              <h4>错误信息:</h4>
              <pre>{{ selectedTask.error }}</pre>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick, onMounted } from "vue";
import Icon from "../ChatUI/Icon.vue";

interface Task {
  id: string;
  command: string;
  type: "shell" | "script" | "api" | "file" | "";
  priority: "low" | "medium" | "high" | "";
  status: "pending" | "running" | "completed" | "failed";
  output?: string;
  error?: string;
  exitCode?: number;
  progress?: number;
  createdAt: number;
  startedAt?: number;
  completedAt?: number;
  duration?: number;
}

interface Props {
  wsUrl?: string;
  sessionId?: string;
}

const props = withDefaults(defineProps<Props>(), {
  wsUrl: "ws://localhost:8080/ws",
  sessionId: "default",
});

const emit = defineEmits<{
  taskExecuted: [task: Task];
  taskCompleted: [task: Task];
  taskFailed: [task: Task];
}>();

// 响应式数据
const tasks = ref<Task[]>([]);
const runningTasks = ref<Task[]>([]);
const showResultModal = ref(false);
const selectedTask = ref<Task | null>(null);
const currentFilter = ref("all");
const isExecuting = ref(false);
const commandInput = ref<HTMLInputElement>();
const websocket = ref<WebSocket | null>(null);

const newTask = ref({
  command: "",
  type: "" as "shell" | "script" | "api" | "file" | "",
  priority: "" as "low" | "medium" | "high" | "",
});

// 过滤器选项
const filters = computed(() => [
  { key: "all", label: "全部", icon: "list", count: tasks.value.length },
  { key: "pending", label: "等待中", icon: "clock", count: tasks.value.filter((t) => t.status === "pending").length },
  { key: "running", label: "运行中", icon: "play", count: tasks.value.filter((t) => t.status === "running").length },
  { key: "completed", label: "已完成", icon: "check", count: tasks.value.filter((t) => t.status === "completed").length },
  { key: "failed", label: "失败", icon: "close", count: tasks.value.filter((t) => t.status === "failed").length },
]);

// 计算属性
const filteredTasks = computed(() => {
  let filtered = tasks.value;

  switch (currentFilter.value) {
    case "pending":
      filtered = filtered.filter((t) => t.status === "pending");
      break;
    case "running":
      filtered = filtered.filter((t) => t.status === "running");
      break;
    case "completed":
      filtered = filtered.filter((t) => t.status === "completed");
      break;
    case "failed":
      filtered = filtered.filter((t) => t.status === "failed");
      break;
  }

  return filtered.sort((a, b) => {
    // 优先级排序
    const priorityOrder = { high: 0, medium: 1, low: 2, "": 3 };
    if (priorityOrder[a.priority] !== priorityOrder[b.priority]) {
      return priorityOrder[a.priority] - priorityOrder[b.priority];
    }

    // 时间排序（最新的在前）
    return b.createdAt - a.createdAt;
  });
});

const totalTasks = computed(() => tasks.value.length);
const runningCount = computed(() => tasks.value.filter((t) => t.status === "running").length);
const completedCount = computed(() => tasks.value.filter((t) => t.status === "completed").length);
const failedCount = computed(() => tasks.value.filter((t) => t.status === "failed").length);

// WebSocket 连接
const connectWebSocket = () => {
  try {
    websocket.value = new WebSocket(`${props.wsUrl}?session=${props.sessionId}`);

    websocket.value.onopen = () => {
      console.log("TaskTool WebSocket connected");
      requestTasks();
    };

    websocket.value.onmessage = (event) => {
      try {
        const message = JSON.parse(event.data);
        handleWebSocketMessage(message);
      } catch (error) {
        console.error("Failed to parse WebSocket message:", error);
      }
    };

    websocket.value.onclose = () => {
      console.log("TaskTool WebSocket disconnected");
      setTimeout(connectWebSocket, 5000);
    };

    websocket.value.onerror = (error) => {
      console.error("TaskTool WebSocket error:", error);
    };
  } catch (error) {
    console.error("Failed to connect WebSocket:", error);
  }
};

const handleWebSocketMessage = (message: any) => {
  switch (message.type) {
    case "task_list_response":
      tasks.value = message.tasks || [];
      break;
    case "task_started":
      const startedTask = message.task;
      updateTaskStatus(startedTask.id, "running");
      if (!runningTasks.value.find((t) => t.id === startedTask.id)) {
        runningTasks.value.push({ ...startedTask, startedAt: Date.now() });
      }
      break;
    case "task_progress":
      updateTaskProgress(message.task_id, message.progress, message.output);
      break;
    case "task_completed":
      const completedTask = message.task;
      updateTaskStatus(completedTask.id, "completed", completedTask.output, completedTask.exitCode);
      removeRunningTask(completedTask.id);
      emit("taskCompleted", completedTask);
      break;
    case "task_failed":
      const failedTask = message.task;
      updateTaskStatus(failedTask.id, "failed", undefined, undefined, failedTask.error);
      removeRunningTask(failedTask.id);
      emit("taskFailed", failedTask);
      break;
    case "task_deleted":
      removeTask(message.id);
      break;
  }
};

const sendWebSocketMessage = (message: any) => {
  if (websocket.value && websocket.value.readyState === WebSocket.OPEN) {
    websocket.value.send(JSON.stringify(message));
  }
};

// 任务操作方法
const requestTasks = () => {
  sendWebSocketMessage({ type: "task_list_request" });
};

const executeTask = () => {
  if (!newTask.value.command.trim()) return;

  const task: Partial<Task> = {
    command: newTask.value.command.trim(),
    type: newTask.value.type,
    priority: newTask.value.priority,
    status: "pending",
  };

  sendWebSocketMessage({
    type: "task_execute",
    task,
  });

  // 重置表单
  newTask.value = { command: "", type: "", priority: "" };
  isExecuting.value = false;

  nextTick(() => {
    commandInput.value?.focus();
  });
};

const stopTask = (taskId: string) => {
  sendWebSocketMessage({
    type: "task_stop",
    id: taskId,
  });
};

const retryTask = (task: Task) => {
  sendWebSocketMessage({
    type: "task_execute",
    task: {
      command: task.command,
      type: task.type,
      priority: task.priority,
    },
  });
};

const deleteTask = (taskId: string) => {
  if (confirm("确定要删除这个任务吗？")) {
    sendWebSocketMessage({
      type: "task_delete",
      id: taskId,
    });
  }
};

const clearCompleted = () => {
  if (confirm("确定要清理所有已完成的任务吗？")) {
    sendWebSocketMessage({
      type: "task_clear_completed",
    });
  }
};

const refreshTasks = () => {
  requestTasks();
};

const viewResult = (task: Task) => {
  selectedTask.value = task;
  showResultModal.value = true;
};

// 内部方法
const updateTaskStatus = (taskId: string, status: Task["status"], output?: string, exitCode?: number, error?: string) => {
  const task = tasks.value.find((t) => t.id === taskId);
  if (task) {
    task.status = status;
    if (output) task.output = output;
    if (exitCode !== undefined) task.exitCode = exitCode;
    if (error) task.error = error;

    if (status === "completed" || status === "failed") {
      task.completedAt = Date.now();
      task.duration = task.completedAt - task.createdAt;
    }
  }
};

const updateTaskProgress = (taskId: string, progress: number, output?: string) => {
  const task = tasks.value.find((t) => t.id === taskId);
  const runningTask = runningTasks.value.find((t) => t.id === taskId);
  if (task) {
    task.progress = progress;
    if (output) task.output = output;
  }
  if (runningTask) {
    runningTask.progress = progress;
    if (output) runningTask.output = output;
  }
};

const removeRunningTask = (taskId: string) => {
  const index = runningTasks.value.findIndex((t) => t.id === taskId);
  if (index !== -1) {
    runningTasks.value.splice(index, 1);
  }
};

const removeTask = (taskId: string) => {
  const index = tasks.value.findIndex((t) => t.id === taskId);
  if (index !== -1) {
    tasks.value.splice(index, 1);
  }
  removeRunningTask(taskId);
};

// 工具方法
const getTypeText = (type: string) => {
  const map = { shell: "Shell", script: "脚本", api: "API", file: "文件" };
  return map[type as keyof typeof map] || type;
};

const getPriorityText = (priority: string) => {
  const map = { high: "高", medium: "中", low: "低" };
  return map[priority as keyof typeof map] || priority;
};

const getStatusText = (status: string) => {
  const map = { pending: "等待中", running: "运行中", completed: "已完成", failed: "失败" };
  return map[status as keyof typeof map] || status;
};

const formatDateTime = (timestamp: number) => {
  const date = new Date(timestamp);
  return date.toLocaleString("zh-CN");
};

const formatDuration = (duration: number) => {
  if (duration < 1000) return `${duration}ms`;
  if (duration < 60000) return `${(duration / 1000).toFixed(1)}s`;
  return `${(duration / 60000).toFixed(1)}m`;
};

const getEmptyMessage = () => {
  const messages = {
    all: "暂无任务，输入命令开始执行任务",
    pending: "暂无等待中的任务",
    running: "暂无运行中的任务",
    completed: "暂无已完成的任务",
    failed: "暂无失败的任务",
  };
  return messages[currentFilter.value as keyof typeof messages];
};

// 生命周期
onMounted(() => {
  connectWebSocket();

  // 自动聚焦到命令输入框
  watch(newTask, (value) => {
    if (value.command) {
      nextTick(() => {
        commandInput.value?.focus();
      });
    }
  });
});
</script>

<style scoped>
.task-tool {
  @apply flex flex-col h-full bg-surface dark:bg-surface-dark border border-border dark:border-border-dark rounded-lg;
  max-height: 600px;
}

.task-header {
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

.task-create {
  @apply px-4 py-3 bg-blue-50 dark:bg-blue-900/20 border-b border-border dark:border-border-dark;
}

.create-content {
  @apply space-y-2;
}

.task-input {
  @apply w-full px-3 py-2 border border-border dark:border-border-dark rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-800 dark:text-white;
}

.create-options {
  @apply flex gap-2 items-center;
}

.task-type-select,
.priority-select {
  @apply px-2 py-1 text-sm border border-border dark:border-border-dark rounded focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-800 dark:text-white;
}

.execute-button {
  @apply px-3 py-1 bg-blue-500 hover:bg-blue-600 text-white text-sm rounded transition-colors disabled:opacity-50 disabled:cursor-not-allowed;
}

.task-filters {
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

.task-list {
  @apply flex-1 overflow-y-auto px-4 py-2 space-y-1;
}

.task-item {
  @apply flex items-center gap-3 p-3 rounded-lg border border-border dark:border-border-dark bg-surface dark:bg-surface-dark hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors;
}

.task-running {
  @apply border-l-4 border-l-blue-500 bg-blue-50 dark:bg-blue-900/20;
}

.task-completed {
  @apply opacity-75;
}

.task-failed {
  @apply border-l-4 border-l-red-500 bg-red-50 dark:bg-red-900/20;
}

.task-priority-high {
  @apply border-l-4 border-l-red-500;
}

.task-priority-medium {
  @apply border-l-4 border-l-yellow-500;
}

.task-priority-low {
  @apply border-l-4 border-l-green-500;
}

.task-main {
  @apply flex items-center gap-3 flex-1 min-w-0;
}

.task-status {
  @apply flex-shrink-0;
}

.status-spinner {
  @apply w-4 h-4 border-2 border-blue-500 border-t-transparent rounded-full animate-spin;
}

.status-success {
  @apply text-green-500;
}

.status-error {
  @apply text-red-500;
}

.status-pending {
  @apply text-gray-400;
}

.task-content {
  @apply flex-1 min-w-0;
}

.task-command {
  @apply text-sm font-medium text-text dark:text-text-dark truncate;
}

.task-meta {
  @apply flex items-center gap-2 mt-1;
}

.type-badge {
  @apply text-xs px-1.5 py-0.5 rounded font-medium;
}

.type-shell {
  @apply bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300;
}

.type-script {
  @apply bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-300;
}

.type-api {
  @apply bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300;
}

.type-file {
  @apply bg-purple-100 dark:bg-purple-900/30 text-purple-700 dark:text-purple-300;
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

.created-time,
.duration {
  @apply flex items-center gap-1 text-xs text-gray-500 dark:text-gray-400;
}

.task-actions {
  @apply flex gap-1 opacity-0 hover:opacity-100 transition-opacity;
}

.task-item:hover .task-actions {
  @apply opacity-100;
}

.action-btn {
  @apply p-1 text-gray-400 dark:text-gray-500 hover:text-gray-600 dark:hover:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-600 rounded transition-colors;
}

.stop-btn:hover {
  @apply text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20;
}

.retry-btn:hover {
  @apply text-blue-500 hover:bg-blue-50 dark:hover:bg-blue-900/20;
}

.view-btn:hover {
  @apply text-green-500 hover:bg-green-50 dark:hover:bg-green-900/20;
}

.delete-btn:hover {
  @apply text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20;
}

.task-progress {
  @apply p-3 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-700 rounded-lg mb-2;
}

.progress-info {
  @apply flex justify-between items-center mb-2;
}

.progress-command {
  @apply text-sm font-medium text-blue-700 dark:text-blue-300;
}

.progress-time {
  @apply text-xs text-blue-600 dark:text-blue-400;
}

.progress-bar {
  @apply w-full bg-blue-200 dark:bg-blue-700 rounded-full h-2 mb-2;
}

.progress-fill {
  @apply bg-blue-500 h-2 rounded-full transition-all duration-300;
}

.progress-output {
  @apply text-xs text-gray-600 dark:text-gray-300 bg-gray-100 dark:bg-gray-700 p-2 rounded font-mono;
  max-height: 60px;
  overflow-y: auto;
}

.empty-state {
  @apply flex flex-col items-center justify-center py-8 text-gray-400 dark:text-gray-500;
}

.task-stats {
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

/* 模态框样式 */
.modal-overlay {
  @apply fixed inset-0 bg-black/50 flex items-center justify-center z-50;
}

.modal-content {
  @apply bg-white dark:bg-gray-800 rounded-lg shadow-xl w-full max-w-4xl max-h-[80vh] flex flex-col;
}

.modal-header {
  @apply flex items-center justify-between p-4 border-b border-gray-200 dark:border-gray-700;
}

.modal-header h3 {
  @apply text-lg font-semibold text-gray-900 dark:text-white;
}

.modal-body {
  @apply flex-1 overflow-y-auto p-4;
}

.result-content {
  @apply space-y-4;
}

.result-header {
  @apply flex items-center justify-between pb-2 border-b border-gray-200 dark:border-gray-700;
}

.result-command {
  @apply font-mono text-sm text-gray-900 dark:text-white;
}

.result-status {
  @apply px-2 py-1 rounded text-xs font-medium;
}

.status-completed {
  @apply bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-300;
}

.status-failed {
  @apply bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-300;
}

.result-details {
  @apply grid grid-cols-2 gap-2;
}

.detail-item {
  @apply flex justify-between;
}

.detail-label {
  @apply text-sm text-gray-600 dark:text-gray-400;
}

.detail-value {
  @apply text-sm text-gray-900 dark:text-white;
}

.exit-code-success {
  @apply text-green-600 dark:text-green-400;
}

.exit-code-error {
  @apply text-red-600 dark:text-red-400;
}

.result-output,
.result-error {
  @apply space-y-2;
}

.result-output h4,
.result-error h4 {
  @apply text-sm font-semibold text-gray-900 dark:text-white;
}

.result-output pre,
.result-error pre {
  @apply bg-gray-100 dark:bg-gray-700 p-3 rounded text-xs overflow-x-auto font-mono text-gray-800 dark:text-gray-200;
}

.result-error pre {
  @apply text-red-800 dark:text-red-200;
}
</style>
