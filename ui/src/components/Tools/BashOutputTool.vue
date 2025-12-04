<template>
  <div class="bash-output-tool">
    <!-- 头部工具栏 -->
    <div class="output-header">
      <div class="header-left">
        <div class="header-title">
          <Icon type="terminal" size="sm" />
          <span>后台任务监控</span>
        </div>
        <div v-if="activeProcesses.length > 0" class="process-count">
          <span class="count-badge">{{ activeProcesses.length }}</span>
          <span class="count-text">个运行中</span>
        </div>
      </div>
      <div class="header-actions">
        <button class="action-button" title="刷新进程列表" @click="refreshProcesses">
          <Icon type="refresh" size="sm" />
        </button>
        <button class="action-button" title="设置" @click="toggleSettings">
          <Icon type="settings" size="sm" />
        </button>
        <button class="action-button" title="清理已完成" @click="clearCompleted">
          <Icon type="trash" size="sm" />
        </button>
      </div>
    </div>

    <!-- 设置面板 -->
    <div v-if="showSettings" class="settings-panel">
      <div class="settings-content">
        <h4>监控设置</h4>
        <div class="setting-group">
          <label>自动刷新间隔</label>
          <select v-model="monitorSettings.refreshInterval" class="setting-select">
            <option value="1">1秒</option>
            <option value="2">2秒</option>
            <option value="5">5秒</option>
            <option value="10">10秒</option>
          </select>
        </div>
        <div class="setting-group">
          <label>最大输出行数</label>
          <select v-model="monitorSettings.maxOutputLines" class="setting-select">
            <option value="100">100行</option>
            <option value="500">500行</option>
            <option value="1000">1000行</option>
            <option value="5000">5000行</option>
          </select>
        </div>
        <div class="setting-group">
          <label>输出缓冲区大小</label>
          <select v-model="monitorSettings.bufferSize" class="setting-select">
            <option value="1000">1KB</option>
            <option value="5000">5KB</option>
            <option value="10000">10KB</option>
            <option value="50000">50KB</option>
          </select>
        </div>
        <div class="setting-group">
          <label>
            <input v-model="monitorSettings.autoScroll" type="checkbox" class="setting-checkbox" />
            自动滚动到最新输出
          </label>
        </div>
        <div class="setting-group">
          <label>
            <input v-model="monitorSettings.showTimestamp" type="checkbox" class="setting-checkbox" />
            显示时间戳
          </label>
        </div>
        <div class="setting-group">
          <label>
            <input v-model="monitorSettings.enableNotifications" type="checkbox" class="setting-checkbox" />
            启用通知
          </label>
        </div>
        <div class="setting-actions">
          <button class="setting-btn" @click="resetSettings">重置</button>
          <button class="setting-btn primary" @click="saveSettings">保存</button>
        </div>
      </div>
    </div>

    <!-- 进程筛选器 -->
    <div class="process-filters">
      <div class="filter-group">
        <label class="filter-label">状态:</label>
        <select v-model="statusFilter" class="filter-select">
          <option value="">全部</option>
          <option value="running">运行中</option>
          <option value="completed">已完成</option>
          <option value="failed">失败</option>
          <option value="stopped">已停止</option>
        </select>
      </div>
      <div class="filter-group">
        <label class="filter-label">搜索:</label>
        <input v-model="searchQuery" type="text" placeholder="搜索进程命令..." class="filter-input" />
      </div>
    </div>

    <!-- 进程列表 -->
    <div class="processes-container">
      <div class="processes-list">
        <div v-for="process in filteredProcesses" :key="process.id" :class="['process-item', `status-${process.status}`]">
          <!-- 进程头部 -->
          <div class="process-header">
            <div class="process-info">
              <div class="process-command">
                <Icon type="terminal" size="xs" />
                <span class="command-text">{{ process.command }}</span>
              </div>
              <div class="process-meta">
                <span class="process-id">PID: {{ process.pid }}</span>
                <span class="process-user">{{ process.user }}</span>
                <span class="process-start">{{ formatTime(process.startTime) }}</span>
              </div>
            </div>
            <div class="process-status">
              <span :class="['status-badge', `status-${process.status}`]">
                <Icon :type="getStatusIcon(process.status)" size="xs" />
                {{ getStatusText(process.status) }}
              </span>
            </div>
            <div class="process-actions">
              <button v-if="process.status === 'running'" class="action-btn stop-btn" title="停止进程" @click="stopProcess(process)">
                <Icon type="square" size="xs" />
              </button>
              <button v-if="process.status === 'running'" class="action-btn kill-btn" title="强制终止进程" @click="killProcess(process)">
                <Icon type="x-circle" size="xs" />
              </button>
              <button class="action-btn clear-btn" title="清除输出" @click="clearProcessOutput(process)">
                <Icon type="trash" size="xs" />
              </button>
              <button class="action-btn focus-btn" title="关注此进程" @click="toggleProcessFocus(process)">
                <Icon :type="process.focused ? 'star' : 'star-outline'" size="xs" />
              </button>
            </div>
          </div>

          <!-- 进程统计 -->
          <div class="process-stats">
            <div class="stat-item">
              <Icon type="clock" size="xs" />
              <span class="stat-label">运行时间:</span>
              <span class="stat-value">{{ getRunningTime(process.startTime) }}</span>
            </div>
            <div class="stat-item">
              <Icon type="cpu" size="xs" />
              <span class="stat-label">CPU:</span>
              <span class="stat-value">{{ process.cpuUsage || "N/A" }}%</span>
            </div>
            <div class="stat-item">
              <Icon type="hard-drive" size="xs" />
              <span class="stat-label">内存:</span>
              <span class="stat-value">{{ process.memoryUsage || "N/A" }}</span>
            </div>
            <div class="stat-item">
              <Icon type="file-text" size="xs" />
              <span class="stat-label">输出:</span>
              <span class="stat-value">{{ process.output.length }} 行</span>
            </div>
          </div>

          <!-- 输出区域 -->
          <div v-if="process.focused || process.output.length > 0" class="process-output">
            <div class="output-header">
              <div class="output-title">
                <span>进程输出</span>
                <span class="output-count">{{ process.output.length }} 行</span>
              </div>
              <div class="output-actions">
                <button class="output-action-btn" title="滚动到底部" @click="scrollToBottom(process.id)">
                  <Icon type="arrow-down" size="xs" />
                </button>
                <button class="output-action-btn" title="全屏查看" @click="viewOutputFullscreen(process)">
                  <Icon type="maximize" size="xs" />
                </button>
                <button class="output-action-btn" title="复制输出" @click="copyProcessOutput(process)">
                  <Icon type="copy" size="xs" />
                </button>
                <button class="output-action-btn" title="保存输出" @click="saveProcessOutput(process)">
                  <Icon type="download" size="xs" />
                </button>
              </div>
            </div>
            <div :ref="`output-${process.id}`" class="output-content" @scroll="handleOutputScroll(process.id)">
              <div v-for="(line, index) in process.output" :key="index" :class="['output-line', { 'error-line': line.type === 'error' }]">
                <span v-if="monitorSettings.showTimestamp" class="timestamp">
                  {{ formatTimestamp(line.timestamp) }}
                </span>
                <span class="line-content" v-html="formatOutputLine(line.content)"></span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 空状态 -->
    <div v-if="filteredProcesses.length === 0" class="empty-state">
      <Icon type="terminal" size="lg" />
      <h3>暂无后台任务</h3>
      <p class="empty-hint">没有运行中的后台进程，所有进程都已正常完成</p>
      <button class="refresh-btn" @click="refreshProcesses">
        <Icon type="refresh" size="sm" />
        刷新进程列表
      </button>
    </div>

    <!-- 系统状态栏 -->
    <div class="status-bar">
      <div class="status-left">
        <span class="system-info">
          <Icon type="server" size="xs" />
          <span>系统负载: {{ systemStatus.load || "N/A" }}</span>
        </span>
        <span class="system-info">
          <Icon type="cpu" size="xs" />
          <span>CPU: {{ systemStatus.cpu || "N/A" }}%</span>
        </span>
        <span class="system-info">
          <Icon type="hard-drive" size="xs" />
          <span>内存: {{ systemStatus.memory || "N/A" }}</span>
        </span>
      </div>
      <div class="status-right">
        <span class="last-update"> 最后更新: {{ formatTime(lastUpdateTime) }} </span>
        <button class="status-btn" title="手动更新" @click="refreshProcesses">
          <Icon type="refresh" size="xs" />
        </button>
      </div>
    </div>

    <!-- 全屏输出模态框 -->
    <div v-if="showFullscreenOutput && selectedProcess" class="fullscreen-modal">
      <div class="fullscreen-header">
        <h3>{{ selectedProcess.command }}</h3>
        <div class="fullscreen-actions">
          <button class="fullscreen-action-btn" title="关闭全屏" @click="closeFullscreenOutput">
            <Icon type="minimize" size="sm" />
          </button>
        </div>
      </div>
      <div class="fullscreen-content">
        <div class="fullscreen-output">
          <div v-for="(line, index) in selectedProcess.output" :key="index" :class="['fullscreen-line', { 'error-line': line.type === 'error' }]">
            <span v-if="monitorSettings.showTimestamp" class="timestamp">
              {{ formatTimestamp(line.timestamp) }}
            </span>
            <span class="line-content" v-html="formatOutputLine(line.content)"></span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick } from "vue";
import Icon from "../ChatUI/Icon.vue";

interface OutputLine {
  content: string;
  timestamp: number;
  type: "stdout" | "stderr" | "error";
}

interface Process {
  id: string;
  pid: number;
  command: string;
  user: string;
  startTime: number;
  status: "running" | "completed" | "failed" | "stopped";
  output: OutputLine[];
  cpuUsage?: number;
  memoryUsage?: string;
  focused: boolean;
}

interface SystemStatus {
  load: string;
  cpu: number;
  memory: string;
}

interface MonitorSettings {
  refreshInterval: number;
  maxOutputLines: number;
  bufferSize: number;
  autoScroll: boolean;
  showTimestamp: boolean;
  enableNotifications: boolean;
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
  processStarted: [process: Process];
  processStopped: [process: Process];
  outputReceived: [process: Process, line: OutputLine];
}>();

// 响应式数据
const processes = ref<Process[]>([]);
const statusFilter = ref("");
const searchQuery = ref("");
const showSettings = ref(false);
const showFullscreenOutput = ref(false);
const selectedProcess = ref<Process | null>(null);
const lastUpdateTime = ref(Date.now());

// 监控设置
const monitorSettings = ref<MonitorSettings>({
  refreshInterval: 5,
  maxOutputLines: 1000,
  bufferSize: 5000,
  autoScroll: true,
  showTimestamp: true,
  enableNotifications: true,
});

// 系统状态
const systemStatus = ref<SystemStatus>({
  load: "0.15, 0.12, 0.08",
  cpu: 25,
  memory: "8.2GB / 16GB",
});

// 自动刷新定时器
let refreshTimer: NodeJS.Timeout | null = null;

// WebSocket连接
const websocket = ref<WebSocket | null>(null);

// 计算属性
const activeProcesses = computed(() => {
  return processes.value.filter((p) => p.status === "running");
});

const filteredProcesses = computed(() => {
  let filtered = processes.value;

  // 状态筛选
  if (statusFilter.value) {
    filtered = filtered.filter((p) => p.status === statusFilter.value);
  }

  // 搜索筛选
  if (searchQuery.value.trim()) {
    const query = searchQuery.value.toLowerCase();
    filtered = filtered.filter((p) => p.command.toLowerCase().includes(query) || p.pid.toString().includes(query));
  }

  return filtered.sort((a, b) => {
    // 优先显示关注的进程
    if (a.focused && !b.focused) return -1;
    if (!a.focused && b.focused) return 1;

    // 然后按状态排序
    const statusOrder = { running: 0, completed: 1, failed: 2, stopped: 3 };
    return statusOrder[a.status] - statusOrder[b.status];
  });
});

// WebSocket连接
const connectWebSocket = () => {
  try {
    websocket.value = new WebSocket(`${props.wsUrl}?session=${props.sessionId}`);

    websocket.value.onopen = () => {
      console.log("BashOutputTool WebSocket connected");
      startAutoRefresh();
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
      console.log("BashOutputTool WebSocket disconnected");
      stopAutoRefresh();
      setTimeout(connectWebSocket, 5000);
    };

    websocket.value.onerror = (error) => {
      console.error("BashOutputTool WebSocket error:", error);
    };
  } catch (error) {
    console.error("Failed to connect WebSocket:", error);
  }
};

const handleWebSocketMessage = (message: any) => {
  switch (message.type) {
    case "process_list":
      if (message.processes) {
        updateProcesses(message.processes);
      }
      break;
    case "process_output":
      if (message.processId && message.line) {
        handleProcessOutput(message.processId, message.line);
      }
      break;
    case "process_status":
      if (message.process) {
        updateProcessStatus(message.process);
      }
      break;
    case "process_started":
      if (message.process) {
        addProcess(message.process);
      }
      break;
    case "process_stopped":
      if (message.processId) {
        removeProcess(message.processId);
      }
      break;
    case "system_status":
      if (message.status) {
        systemStatus.value = message.status;
      }
      break;
  }
};

const sendWebSocketMessage = (message: any) => {
  if (websocket.value && websocket.value.readyState === WebSocket.OPEN) {
    websocket.value.send(JSON.stringify(message));
  }
};

// 进程管理
const updateProcesses = (processList: Process[]) => {
  processList.forEach((processData) => {
    const existingProcess = processes.value.find((p) => p.id === processData.id);
    if (existingProcess) {
      Object.assign(existingProcess, processData);
    } else {
      processes.value.push(processData);
    }
  });
  lastUpdateTime.value = Date.now();
};

const handleProcessOutput = (processId: string, line: OutputLine) => {
  const process = processes.value.find((p) => p.id === processId);
  if (process) {
    // 限制输出行数
    if (process.output.length >= monitorSettings.value.maxOutputLines) {
      process.output = process.output.slice(-monitorSettings.value.maxOutputLines + 100);
    }

    process.output.push(line);

    // 自动滚动到底部
    if (monitorSettings.value.autoScroll && process.focused) {
      nextTick(() => {
        scrollToBottom(processId);
      });
    }

    emit("outputReceived", process, line);

    // 错误通知
    if (monitorSettings.value.enableNotifications && line.type === "error") {
      showNotification(`进程 ${process.command} 产生错误输出`, "error");
    }
  }
};

const updateProcessStatus = (processUpdate: Partial<Process>) => {
  const process = processes.value.find((p) => p.id === processUpdate.id);
  if (process) {
    const oldStatus = process.status;
    Object.assign(process, processUpdate);

    // 状态变化通知
    if (monitorSettings.value.enableNotifications && oldStatus !== processUpdate.status) {
      if (processUpdate.status === "completed") {
        showNotification(`进程 ${process.command} 已完成`, "success");
      } else if (processUpdate.status === "failed") {
        showNotification(`进程 ${process.command} 执行失败`, "error");
      }
    }

    emit("processStopped", process);
  }
};

const addProcess = (newProcess: Process) => {
  processes.value.push(newProcess);
  lastUpdateTime.value = Date.now();

  if (monitorSettings.value.enableNotifications) {
    showNotification(`新进程启动: ${newProcess.command}`, "info");
  }

  emit("processStarted", newProcess);
};

const removeProcess = (processId: string) => {
  const index = processes.value.findIndex((p) => p.id === processId);
  if (index !== -1) {
    processes.value.splice(index, 1);
    lastUpdateTime.value = Date.now();
  }
};

// 进程操作
const stopProcess = async (process: Process) => {
  if (!confirm(`确定要停止进程 "${process.command}" 吗？`)) return;

  sendWebSocketMessage({
    type: "stop_process",
    processId: process.id,
  });
};

const killProcess = async (process: Process) => {
  if (!confirm(`确定要强制终止进程 "${process.command}" 吗？`)) return;

  sendWebSocketMessage({
    type: "kill_process",
    processId: process.id,
  });
};

const clearProcessOutput = (process: Process) => {
  process.output = [];
};

const toggleProcessFocus = (process: Process) => {
  process.focused = !process.focused;

  // 保存关注状态
  saveFocusedProcesses();
};

const copyProcessOutput = async (process: Process) => {
  try {
    const output = process.output.map((line) => line.content).join("\n");
    await navigator.clipboard.writeText(output);
    showNotification("输出已复制到剪贴板", "success");
  } catch (error) {
    console.error("Failed to copy output:", error);
  }
};

const saveProcessOutput = (process: Process) => {
  const output = process.output.map((line) => line.content).join("\n");
  const blob = new Blob([output], { type: "text/plain" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = `process-${process.pid}-${Date.now()}.log`;
  a.click();
  URL.revokeObjectURL(url);
};

const viewOutputFullscreen = (process: Process) => {
  selectedProcess.value = process;
  showFullscreenOutput.value = true;
};

const closeFullscreenOutput = () => {
  showFullscreenOutput.value = false;
  selectedProcess.value = null;
};

// 刷新和管理
const refreshProcesses = () => {
  sendWebSocketMessage({
    type: "get_process_list",
  });
};

const clearCompleted = () => {
  if (!confirm("确定要清除所有已完成的进程吗？")) return;

  processes.value = processes.value.filter((p) => p.status === "running");
  saveProcesses();
};

// 自动刷新
const startAutoRefresh = () => {
  stopAutoRefresh();
  refreshTimer = setInterval(() => {
    refreshProcesses();
    updateSystemStatus();
  }, monitorSettings.value.refreshInterval * 1000);
};

const stopAutoRefresh = () => {
  if (refreshTimer) {
    clearInterval(refreshTimer);
    refreshTimer = null;
  }
};

// 系统状态更新
const updateSystemStatus = () => {
  // 模拟系统状态变化
  const cpu = 15 + Math.random() * 30;
  const memoryPercent = 40 + Math.random() * 30;
  const totalMemory = 16;
  const usedMemory = ((totalMemory * memoryPercent) / 100).toFixed(1);

  systemStatus.value = {
    load: `${(Math.random() * 0.3).toFixed(2)}, ${(Math.random() * 0.3).toFixed(2)}, ${(Math.random() * 0.3).toFixed(2)}`,
    cpu: Math.round(cpu),
    memory: `${usedMemory}GB / ${totalMemory}GB`,
  };
};

// UI交互
const scrollToBottom = (processId: string) => {
  const outputElement = document.querySelector(`[ref="output-${processId}"]`) as HTMLElement;
  if (outputElement) {
    outputElement.scrollTop = outputElement.scrollHeight;
  }
};

const handleOutputScroll = (processId: string) => {
  const outputElement = document.querySelector(`[ref="output-${processId}"]`) as HTMLElement;
  if (outputElement) {
    const isAtBottom = outputElement.scrollHeight - outputElement.scrollTop <= outputElement.clientHeight + 10;
    if (isAtBottom) {
      outputElement.dataset.autoScroll = "true";
    } else {
      outputElement.dataset.autoScroll = "false";
    }
  }
};

// 通知系统
const showNotification = (message: string, type: "info" | "success" | "error" = "info") => {
  // 简单的通知实现
  console.log(`[${type.toUpperCase()}] ${message}`);

  // 这里可以集成更复杂的通知系统
  if ("Notification" in window && Notification.permission === "granted") {
    new Notification(message, {
      body: `后台任务监控 - ${type}`,
      icon: "/favicon.ico",
    });
  }
};

// 设置管理
const toggleSettings = () => {
  showSettings.value = !showSettings.value;
};

const saveSettings = () => {
  localStorage.setItem("bash-output-settings", JSON.stringify(monitorSettings.value));
  showSettings.value = false;

  // 重启自动刷新
  stopAutoRefresh();
  if (activeProcesses.value.length > 0) {
    startAutoRefresh();
  }
};

const resetSettings = () => {
  monitorSettings.value = {
    refreshInterval: 5,
    maxOutputLines: 1000,
    bufferSize: 5000,
    autoScroll: true,
    showTimestamp: true,
    enableNotifications: true,
  };

  // 重启自动刷新
  stopAutoRefresh();
  if (activeProcesses.value.length > 0) {
    startAutoRefresh();
  }
};

// 本地存储
const saveProcesses = () => {
  try {
    localStorage.setItem("bash-output-processes", JSON.stringify(processes.value));
  } catch (error) {
    console.warn("Failed to save processes:", error);
  }
};

const loadProcesses = () => {
  try {
    const saved = localStorage.getItem("bash-output-processes");
    if (saved) {
      processes.value = JSON.parse(saved);
    }
  } catch (error) {
    console.warn("Failed to load processes:", error);
  }
};

const saveFocusedProcesses = () => {
  try {
    const focusedIds = processes.value.filter((p) => p.focused).map((p) => p.id);
    localStorage.setItem("bash-output-focused-processes", JSON.stringify(focusedIds));
  } catch (error) {
    console.warn("Failed to save focused processes:", error);
  }
};

const loadFocusedProcesses = () => {
  try {
    const saved = localStorage.getItem("bash-output-focused-processes");
    if (saved) {
      const focusedIds = JSON.parse(saved);
      processes.value.forEach((process) => {
        process.focused = focusedIds.includes(process.id);
      });
    }
  } catch (error) {
    console.warn("Failed to load focused processes:", error);
  }
};

// 工具方法
const getStatusIcon = (status: string): any => {
  const icons: Record<string, string> = {
    running: "play",
    completed: "check",
    failed: "x-circle",
    stopped: "square",
  };
  return icons[status] || "help-circle";
};

const getStatusText = (status: string) => {
  const texts: Record<string, string> = {
    running: "运行中",
    completed: "已完成",
    failed: "失败",
    stopped: "已停止",
  };
  return texts[status] || "未知";
};

const formatTime = (timestamp: number) => {
  const date = new Date(timestamp);
  const now = new Date();
  const diff = now.getTime() - date.getTime();

  if (diff < 60000) {
    const seconds = Math.floor(diff / 1000);
    return `${seconds}秒前`;
  } else if (diff < 3600000) {
    const minutes = Math.floor(diff / 60000);
    return `${minutes}分钟前`;
  } else if (diff < 86400000) {
    const hours = Math.floor(diff / 3600000);
    return `${hours}小时前`;
  } else {
    return date.toLocaleDateString("zh-CN");
  }
};

const formatTimestamp = (timestamp: number) => {
  const date = new Date(timestamp);
  return date.toLocaleTimeString("zh-CN");
};

const getRunningTime = (startTime: number) => {
  const now = Date.now();
  const diff = now - startTime;

  const hours = Math.floor(diff / 3600000);
  const minutes = Math.floor((diff % 3600000) / 60000);
  const seconds = Math.floor((diff % 60000) / 1000);

  const parts = [];
  if (hours > 0) parts.push(`${hours}小时`);
  if (minutes > 0) parts.push(`${minutes}分钟`);
  if (seconds > 0) parts.push(`${seconds}秒`);

  return parts.length > 0 ? parts.join("") : "0秒";
};

const formatOutputLine = (content: string): string => {
  // 简单的ANSI颜色解析
  return content
    .replace(/\x1b\[31m/g, '<span class="ansi-red">')
    .replace(/\x1b\[32m/g, '<span class="ansi-green">')
    .replace(/\x1b\[33m/g, '<span class="ansi-yellow">')
    .replace(/\x1b\[34m/g, '<span class="ansi-blue">')
    .replace(/\x1b\[35m/g, '<span class="ansi-magenta">')
    .replace(/\x1b\[36m/g, '<span class="ansi-cyan">')
    .replace(/\x1b\[37m/g, '<span class="ansi-white">')
    .replace(/\x1b\[0m/g, "</span>");
};

// 通知权限请求
const requestNotificationPermission = () => {
  if ("Notification" in window && Notification.permission === "default") {
    Notification.requestPermission();
  }
};

// 生命周期
onMounted(() => {
  connectWebSocket();
  loadProcesses();
  loadFocusedProcesses();
  requestNotificationPermission();
  updateSystemStatus();

  // 初始刷新
  refreshProcesses();
});

onUnmounted(() => {
  stopAutoRefresh();
  saveProcesses();
});
</script>

<style scoped>
.bash-output-tool {
  @apply flex flex-col h-full bg-surface dark:bg-surface-dark border border-border dark:border-border-dark rounded-lg;
}

.output-header {
  @apply flex items-center justify-between px-4 py-3 border-b border-border dark:border-border-dark bg-surface dark:bg-surface-dark;
}

.header-left {
  @apply flex items-center gap-3 flex-1 min-w-0;
}

.header-title {
  @apply flex items-center gap-2 font-semibold text-text dark:text-text-dark;
}

.process-count {
  @apply flex items-center gap-1;
}

.count-badge {
  @apply px-2 py-1 text-xs font-semibold text-white bg-blue-500 rounded-full;
}

.count-text {
  @apply text-sm text-gray-600 dark:text-gray-400;
}

.header-actions {
  @apply flex gap-1;
}

.action-button {
  @apply p-1.5 text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700 rounded transition-colors;
}

.settings-panel {
  @apply border-b border-border dark:border-border-dark bg-gray-50 dark:bg-gray-700/30;
}

.settings-content {
  @apply p-4 space-y-4;
}

.settings-content h4 {
  @apply text-sm font-semibold text-text dark:text-text-dark mb-3;
}

.setting-group {
  @apply flex items-center justify-between;
}

.setting-group label {
  @apply text-sm text-gray-700 dark:text-gray-300;
}

.setting-select {
  @apply ml-2 px-2 py-1 text-sm border border-border dark:border-border-dark rounded focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-800 dark:text-white;
}

.setting-actions {
  @apply flex gap-2 justify-end mt-4;
}

.setting-btn {
  @apply px-3 py-1 text-sm border border-border dark:border-border-dark rounded transition-colors;
}

.setting-btn.primary {
  @apply bg-blue-500 hover:bg-blue-600 text-white border-blue-500;
}

.process-filters {
  @apply flex items-center gap-4 p-4 border-b border-gray-200 dark:border-gray-600 bg-gray-50 dark:bg-gray-700;
}

.filter-group {
  @apply flex items-center gap-2;
}

.filter-label {
  @apply text-sm font-medium text-gray-700 dark:text-gray-300;
}

.filter-select,
.filter-input {
  @apply px-3 py-1 text-sm border border-gray-200 dark:border-gray-600 rounded focus:outline-none focus:ring-1 focus:ring-blue-500 dark:bg-gray-700 dark:text-white;
}

.filter-input {
  @apply min-w-48;
}

.processes-container {
  @apply flex-1 overflow-y-auto;
}

.processes-list {
  @apply p-4 space-y-4;
}

.process-item {
  @apply bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-600 rounded-lg overflow-hidden;
}

.process-item.status-running {
  @apply border-l-4 border-l-green-500;
}

.process-item.status-completed {
  @apply border-l-4 border-l-blue-500;
}

.process-item.status-failed {
  @apply border-l-4 border-l-red-500;
}

.process-item.status-stopped {
  @apply border-l-4 border-l-gray-500;
}

.process-header {
  @apply flex items-center justify-between p-4 border-b border-gray-100 dark:border-gray-700;
}

.process-info {
  @apply flex-1 min-w-0;
}

.process-command {
  @apply flex items-center gap-2 mb-1;
}

.command-text {
  @apply font-mono text-sm text-gray-800 dark:text-gray-200 truncate;
}

.process-meta {
  @apply flex items-center gap-3 text-xs text-gray-500 dark:text-gray-400;
}

.process-status {
  @apply flex-shrink-0;
}

.status-badge {
  @apply flex items-center gap-1 px-2 py-1 text-xs font-medium rounded-full;
}

.status-badge.status-running {
  @apply bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400;
}

.status-badge.status-completed {
  @apply bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400;
}

.status-badge.status-failed {
  @apply bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400;
}

.status-badge.status-stopped {
  @apply bg-gray-100 text-gray-700 dark:bg-gray-700 dark:text-gray-400;
}

.process-actions {
  @apply flex gap-1;
}

.action-btn {
  @apply p-1 text-gray-400 dark:text-gray-500 hover:text-gray-600 dark:hover:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-600 rounded transition-colors;
}

.action-btn.stop-btn:hover {
  @apply text-yellow-500 hover:bg-yellow-50 dark:hover:bg-yellow-900/20;
}

.action-btn.kill-btn:hover {
  @apply text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20;
}

.action-btn.clear-btn:hover {
  @apply text-gray-500 hover:bg-gray-50 dark:hover:bg-gray-600;
}

.action-btn.focus-btn:hover {
  @apply text-blue-500 hover:bg-blue-50 dark:hover:bg-blue-900/20;
}

.process-stats {
  @apply flex items-center gap-4 p-4 bg-gray-50 dark:bg-gray-700 text-xs;
}

.stat-item {
  @apply flex items-center gap-1;
}

.stat-label {
  @apply text-gray-500 dark:text-gray-400;
}

.stat-value {
  @apply text-gray-700 dark:text-gray-300 font-medium;
}

.process-output {
  @apply border-t border-gray-100 dark:border-gray-700;
}

.output-header {
  @apply flex items-center justify-between p-3 bg-gray-50 dark:bg-gray-700 border-b border-gray-100 dark:border-gray-600;
}

.output-title {
  @apply flex items-center gap-2 text-sm font-medium text-gray-700 dark:text-gray-300;
}

.output-count {
  @apply text-xs text-gray-500 dark:text-gray-400;
}

.output-actions {
  @apply flex gap-1;
}

.output-action-btn {
  @apply p-1.5 text-gray-400 dark:text-gray-500 hover:text-gray-600 dark:hover:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-600 rounded transition-colors;
}

.output-content {
  @apply h-64 overflow-y-auto p-3 font-mono text-xs bg-gray-900 text-gray-100;
}

.output-line {
  @apply leading-relaxed break-all;
}

.output-line.error-line {
  @apply text-red-400;
}

.timestamp {
  @apply text-gray-500 dark:text-gray-400 mr-3;
}

.line-content {
  @apply flex-1;
}

.empty-state {
  @apply flex-1 flex flex-col items-center justify-center p-8 text-gray-400 dark:text-gray-500;
}

.empty-state h3 {
  @apply text-lg font-medium mb-2;
}

.empty-hint {
  @apply text-sm text-center mb-6;
}

.refresh-btn {
  @apply flex items-center gap-2 px-4 py-2 bg-blue-500 hover:bg-blue-600 text-white rounded transition-colors;
}

.status-bar {
  @apply flex items-center justify-between px-4 py-2 border-t border-border dark:border-border-dark bg-gray-50 dark:bg-gray-700 text-xs text-gray-600 dark:text-gray-400;
}

.status-left,
.status-right {
  @apply flex items-center gap-3;
}

.system-info {
  @apply flex items-center gap-1;
}

.last-update {
  @apply text-gray-500 dark:text-gray-400;
}

.status-btn {
  @apply p-1 text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-600 rounded transition-colors;
}

/* 全屏模态框 */
.fullscreen-modal {
  @apply fixed inset-0 bg-black bg-opacity-75 flex flex-col z-50;
}

.fullscreen-header {
  @apply flex items-center justify-between p-4 bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-600;
}

.fullscreen-header h3 {
  @apply text-lg font-semibold text-gray-800 dark:text-gray-200;
}

.fullscreen-actions {
  @apply flex gap-2;
}

.fullscreen-action-btn {
  @apply p-2 text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-600 rounded transition-colors;
}

.fullscreen-content {
  @apply flex-1 overflow-hidden bg-gray-900 text-gray-100;
}

.fullscreen-output {
  @apply h-full overflow-y-auto p-4 font-mono text-xs;
}

.fullscreen-line {
  @apply leading-relaxed break-all;
}

.fullscreen-line.error-line {
  @apply text-red-400;
}

/* ANSI颜色样式 */
:deep(.ansi-red) {
  @apply text-red-500;
}
:deep(.ansi-green) {
  @apply text-green-500;
}
:deep(.ansi-yellow) {
  @apply text-yellow-500;
}
:deep(.ansi-blue) {
  @apply text-blue-500;
}
:deep(.ansi-magenta) {
  @apply text-magenta-500;
}
:deep(.ansi-cyan) {
  @apply text-cyan-500;
}
:deep(.ansi-white) {
  @apply text-gray-100;
}
</style>
