<template>
  <div class="bash-tool">
    <!-- 头部工具栏 -->
    <div class="bash-header">
      <div class="header-left">
        <div class="header-title">
          <Icon type="terminal" size="sm" />
          <span>命令行终端</span>
        </div>
        <div v-if="currentDirectory" class="current-directory">
          <Icon type="folder" size="xs" />
          <span>{{ currentDirectory }}</span>
        </div>
      </div>
      <div class="header-actions">
        <button class="action-button" title="新建终端" @click="createNewTab">
          <Icon type="plus" size="sm" />
        </button>
        <button class="action-button" title="设置" @click="toggleSettings">
          <Icon type="settings" size="sm" />
        </button>
      </div>
    </div>

    <!-- 设置面板 -->
    <div v-if="showSettings" class="settings-panel">
      <div class="settings-content">
        <h4>终端设置</h4>
        <div class="setting-group">
          <label>字体大小</label>
          <select v-model="terminalSettings.fontSize" class="setting-select">
            <option value="text-xs">极小</option>
            <option value="text-sm">小</option>
            <option value="text-base">正常</option>
            <option value="text-lg">大</option>
            <option value="text-xl">极大</option>
          </select>
        </div>
        <div class="setting-group">
          <label>主题</label>
          <select v-model="terminalSettings.theme" class="setting-select">
            <option value="dark">深色</option>
            <option value="light">浅色</option>
            <option value="solarized">Solarized</option>
            <option value="monokai">Monokai</option>
          </select>
        </div>
        <div class="setting-group">
          <label>光标样式</label>
          <select v-model="terminalSettings.cursorStyle" class="setting-select">
            <option value="block">块状</option>
            <option value="underline">下划线</option>
            <option value="bar">竖线</option>
          </select>
        </div>
        <div class="setting-group">
          <label>
            <input v-model="terminalSettings.bell" type="checkbox" class="setting-checkbox" />
            响铃声
          </label>
        </div>
        <div class="setting-group">
          <label>
            <input v-model="terminalSettings.autoScroll" type="checkbox" class="setting-checkbox" />
            自动滚动
          </label>
        </div>
        <div class="setting-actions">
          <button class="setting-btn" @click="resetSettings">重置</button>
          <button class="setting-btn primary" @click="saveSettings">保存</button>
        </div>
      </div>
    </div>

    <!-- 标签页 -->
    <div class="tabs-container">
      <div class="tabs">
        <div v-for="(tab, index) in tabs" :key="tab.id" :class="['tab', { active: activeTabId === tab.id, running: tab.isRunning }]" @click="switchTab(tab.id)">
          <span class="tab-title">{{ tab.title }}</span>
          <button class="tab-close" title="关闭标签" @click.stop="closeTab(tab.id)">
            <Icon type="close" size="xs" />
          </button>
        </div>
      </div>
    </div>

    <!-- 终端内容 -->
    <div class="terminal-container">
      <div v-for="tab in tabs" :key="tab.id" v-show="activeTabId === tab.id" class="terminal-content">
        <!-- 输出区域 -->
        <div ref="outputContainer" class="output-container" :class="[terminalSettings.fontSize, `theme-${terminalSettings.theme}`]" @scroll="handleScroll">
          <div v-for="(line, index) in tab.output" :key="index" :class="['output-line', line.type]" v-html="formatOutputLine(line)"></div>

          <!-- 当前命令行 -->
          <div v-if="!tab.isRunning" class="input-line">
            <span class="prompt">{{ tab.prompt }}</span>
            <input
              ref="commandInput"
              v-model="tab.currentCommand"
              type="text"
              class="command-input"
              :class="[terminalSettings.fontSize, `cursor-${terminalSettings.cursorStyle}`]"
              spellcheck="false"
              @keydown="handleCommandKeydown"
              @keyup="handleCommandKeyup"
              @input="handleCommandInput"
              @click="moveCursorToEnd"
            />
            <span class="cursor-indicator"></span>
          </div>

          <!-- 运行中指示器 -->
          <div v-else class="running-indicator">
            <span class="prompt">{{ tab.prompt }}</span>
            <span class="current-command">{{ tab.currentCommand }}</span>
            <Icon type="spinner" size="sm" class="animate-spin" />
          </div>
        </div>

        <!-- 快捷命令栏 -->
        <div class="quick-commands">
          <div class="quick-commands-title">快捷命令:</div>
          <div class="quick-commands-list">
            <button v-for="cmd in quickCommands" :key="cmd.command" class="quick-cmd-btn" :title="cmd.description" @click="executeQuickCommand(cmd.command)">
              {{ cmd.command }}
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- 状态栏 -->
    <div class="status-bar">
      <div class="status-left">
        <span v-if="currentTab" class="process-info">
          <Icon type="terminal" size="xs" />
          进程: {{ currentTab.processId || "N/A" }}
        </span>
        <span v-if="currentTab?.isRunning" class="running-status">
          <Icon type="spinner" size="xs" class="animate-spin" />
          运行中
        </span>
      </div>
      <div class="status-right">
        <span class="command-count">{{ commandHistory.length }} 条历史</span>
        <span v-if="currentTab" class="tab-info">{{ getTabInfo(currentTab) }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, nextTick, onMounted, onUnmounted } from "vue";
import Icon from "../ChatUI/Icon.vue";

interface OutputLine {
  type: "input" | "output" | "error" | "success" | "warning";
  content: string;
  timestamp: number;
}

interface Tab {
  id: string;
  title: string;
  prompt: string;
  currentCommand: string;
  output: OutputLine[];
  isRunning: boolean;
  processId?: string;
  workingDirectory: string;
  history: string[];
  historyIndex: number;
}

interface QuickCommand {
  command: string;
  description: string;
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
  commandExecuted: [command: string, result: OutputLine[]];
  terminalStateChanged: [state: any];
}>();

// 响应式数据
const tabs = ref<Tab[]>([]);
const activeTabId = ref("");
const commandHistory = ref<string[]>([]);
const historyIndex = ref(-1);
const showSettings = ref(false);
const currentDirectory = ref("");

// 终端设置
const terminalSettings = ref({
  fontSize: "text-sm",
  theme: "dark",
  cursorStyle: "block",
  bell: false,
  autoScroll: true,
  maxHistory: 1000,
  autoSaveHistory: true,
});

// 快捷命令
const quickCommands = ref<QuickCommand[]>([
  { command: "ls -la", description: "详细列表" },
  { command: "pwd", description: "当前目录" },
  { command: "cd ..", description: "上级目录" },
  { command: "git status", description: "Git状态" },
  { command: "npm run dev", description: "启动开发服务器" },
  { command: "ps aux", description: "进程列表" },
  { command: "top", description: "系统监控" },
  { command: "clear", description: "清屏" },
]);

const outputContainer = ref<HTMLElement[]>([]);
const commandInput = ref<HTMLInputElement[]>([]);
const websocket = ref<WebSocket | null>(null);

// 计算属性
const currentTab = computed(() => {
  return tabs.value.find((tab) => tab.id === activeTabId.value);
});

// WebSocket 连接
const connectWebSocket = () => {
  try {
    websocket.value = new WebSocket(`${props.wsUrl}?session=${props.sessionId}`);

    websocket.value.onopen = () => {
      console.log("BashTool WebSocket connected");
      initializeTerminal();
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
      console.log("BashTool WebSocket disconnected");
      setTimeout(connectWebSocket, 5000);
    };

    websocket.value.onerror = (error) => {
      console.error("BashTool WebSocket error:", error);
    };
  } catch (error) {
    console.error("Failed to connect WebSocket:", error);
  }
};

const handleWebSocketMessage = (message: any) => {
  switch (message.type) {
    case "bash_output":
      if (message.tabId && message.output) {
        handleBashOutput(message.tabId, message.output);
      }
      break;
    case "bash_complete":
      if (message.tabId) {
        handleBashComplete(message.tabId, message.exitCode, message.processId);
      }
      break;
    case "directory_changed":
      if (message.directory) {
        currentDirectory.value = message.directory;
        updateTabPrompt(message.directory);
      }
      break;
  }
};

const sendWebSocketMessage = (message: any) => {
  if (websocket.value && websocket.value.readyState === WebSocket.OPEN) {
    websocket.value.send(JSON.stringify(message));
  }
};

// 终端初始化
const initializeTerminal = () => {
  // 创建第一个标签页
  createNewTab();

  // 加载历史记录
  loadCommandHistory();

  // 获取当前目录
  sendWebSocketMessage({
    type: "get_current_directory",
  });
};

// 标签页管理
const createNewTab = () => {
  const tabId = `tab-${Date.now()}`;
  const newTab: Tab = {
    id: tabId,
    title: `终端 ${tabs.value.length + 1}`,
    prompt: "$ ",
    currentCommand: "",
    output: [],
    isRunning: false,
    workingDirectory: currentDirectory.value || "~",
    history: [],
    historyIndex: -1,
  };

  tabs.value.push(newTab);
  activeTabId.value = tabId;

  nextTick(() => {
    focusCommandInput();
  });
};

const switchTab = (tabId: string) => {
  activeTabId.value = tabId;
  nextTick(() => {
    focusCommandInput();
  });
};

const closeTab = (tabId: string) => {
  const tabIndex = tabs.value.findIndex((tab) => tab.id === tabId);
  if (tabIndex === -1) return;

  const tab = tabs.value[tabIndex];
  if (!tab) return;

  // 如果进程正在运行，先终止
  if (tab.isRunning && tab.processId) {
    sendWebSocketMessage({
      type: "kill_process",
      processId: tab.processId,
    });
  }

  // 移除标签页
  tabs.value.splice(tabIndex, 1);

  // 如果关闭的是当前标签页，切换到其他标签页
  if (activeTabId.value === tabId) {
    if (tabs.value.length > 0) {
      const newTab = tabs.value[Math.max(0, tabIndex - 1)];
      activeTabId.value = newTab?.id ?? "";
    } else {
      createNewTab();
    }
  }
};

// 命令执行
const executeCommand = (command: string, tabId: string) => {
  const tab = tabs.value.find((t) => t.id === tabId);
  if (!tab || tab.isRunning || !command.trim()) return;

  // 添加到输出历史
  tab.output.push({
    type: "input",
    content: `${tab.prompt}${command}`,
    timestamp: Date.now(),
  });

  // 添加到历史记录
  addToHistory(command);
  tab.history.push(command);
  tab.historyIndex = tab.history.length;

  // 设置运行状态
  tab.isRunning = true;
  tab.currentCommand = command;

  // 发送到服务器执行
  sendWebSocketMessage({
    type: "execute_command",
    command: command,
    tabId: tabId,
    workingDirectory: tab.workingDirectory,
  });

  // 滚动到底部
  nextTick(() => {
    scrollToBottom();
  });

  emit("commandExecuted", command, tab.output);
};

const executeQuickCommand = (command: string) => {
  if (currentTab.value) {
    currentTab.value.currentCommand = command;
    executeCommand(command, currentTab.value.id);
  }
};

// 命令输入处理
const handleCommandKeydown = (event: KeyboardEvent) => {
  const tab = currentTab.value;
  if (!tab || tab.isRunning) return;

  const input = event.target as HTMLInputElement;

  switch (event.key) {
    case "Enter":
      event.preventDefault();
      const command = input.value.trim();
      if (command) {
        executeCommand(command, tab.id);
        input.value = "";
      }
      break;

    case "ArrowUp":
      event.preventDefault();
      navigateHistory(-1);
      break;

    case "ArrowDown":
      event.preventDefault();
      navigateHistory(1);
      break;

    case "Tab":
      event.preventDefault();
      // 简单的自动补全逻辑
      handleAutocomplete(input);
      break;

    case "c":
      // 处理 Ctrl+C
      if (event.ctrlKey || event.metaKey) {
        event.preventDefault();
        if (tab.isRunning && tab.processId) {
          // 发送中断信号
          sendWebSocketMessage({
            type: "interrupt_process",
            processId: tab.processId,
            tabId: tab.id,
          });
        } else {
          // 清空当前输入
          input.value = "";
        }
      }
      break;

    case "l":
      if (event.ctrlKey || event.metaKey) {
        event.preventDefault();
        // 清屏
        tab.output = [];
      }
      break;
  }
};

const handleCommandKeyup = (event: KeyboardEvent) => {
  // 可以在这里处理其他键盘事件
};

const handleCommandInput = (event: Event) => {
  const input = event.target as HTMLInputElement;
  const tab = currentTab.value;
  if (tab) {
    tab.currentCommand = input.value;
  }
};

const navigateHistory = (direction: number) => {
  const tab = currentTab.value;
  if (!tab || tab.history.length === 0) return;

  const newIndex = tab.historyIndex + direction;

  if (newIndex >= 0 && newIndex < tab.history.length) {
    tab.historyIndex = newIndex;
    const input = commandInput.value[tabs.value.findIndex((t) => t.id === tab.id)];
    if (input) {
      input.value = tab.history[newIndex] ?? "";
      moveCursorToEnd();
    }
  } else if (newIndex === tab.history.length) {
    // 回到空输入
    tab.historyIndex = tab.history.length;
    const input = commandInput.value[tabs.value.findIndex((t) => t.id === tab.id)];
    if (input) {
      input.value = "";
    }
  }
};

const handleAutocomplete = (input: HTMLInputElement) => {
  // 简化的自动补全逻辑
  const command = input.value;
  if (!command) return;

  // 常用命令补全
  const commonCommands = ["ls", "cd", "pwd", "mkdir", "rm", "cp", "mv", "cat", "less", "more", "grep", "find", "ps", "kill", "top", "df", "du", "chmod", "chown", "git", "npm", "yarn", "node", "python", "python3", "pip", "pip3"];

  const matches = commonCommands.filter((cmd) => cmd.startsWith(command));

  if (matches.length === 1) {
    input.value = matches[0] + " ";
  } else if (matches.length > 1) {
    // 显示所有匹配的命令
    const tab = currentTab.value;
    if (tab) {
      tab.output.push({
        type: "output",
        content: matches.join("  "),
        timestamp: Date.now(),
      });
      scrollToBottom();
    }
  }
};

// 输出处理
const handleBashOutput = (tabId: string, output: string) => {
  const tab = tabs.value.find((t) => t.id === tabId);
  if (!tab) return;

  // 解析ANSI颜色代码
  const formattedOutput = parseAnsiColors(output);

  tab.output.push({
    type: "output",
    content: formattedOutput,
    timestamp: Date.now(),
  });

  if (terminalSettings.value.autoScroll) {
    nextTick(() => {
      scrollToBottom();
    });
  }
};

const handleBashComplete = (tabId: string, exitCode: number, processId?: string) => {
  const tab = tabs.value.find((t) => t.id === tabId);
  if (!tab) return;

  tab.isRunning = false;
  tab.currentCommand = "";

  if (processId) {
    tab.processId = undefined;
  }

  // 显示执行结果
  if (exitCode === 0) {
    tab.output.push({
      type: "success",
      content: `✓ 命令执行完成 (退出码: ${exitCode})`,
      timestamp: Date.now(),
    });
  } else {
    tab.output.push({
      type: "error",
      content: `✗ 命令执行失败 (退出码: ${exitCode})`,
      timestamp: Date.now(),
    });
  }

  nextTick(() => {
    scrollToBottom();
    focusCommandInput();
  });
};

const updateTabPrompt = (directory: string) => {
  tabs.value.forEach((tab) => {
    tab.workingDirectory = directory;
    tab.prompt = `[${directory}]$ `;
  });
};

// 工具方法
const parseAnsiColors = (text: string): string => {
  // 简化的ANSI颜色解析
  return text
    .replace(/\x1b\[31m/g, '<span class="ansi-red">')
    .replace(/\x1b\[32m/g, '<span class="ansi-green">')
    .replace(/\x1b\[33m/g, '<span class="ansi-yellow">')
    .replace(/\x1b\[34m/g, '<span class="ansi-blue">')
    .replace(/\x1b\[35m/g, '<span class="ansi-magenta">')
    .replace(/\x1b\[36m/g, '<span class="ansi-cyan">')
    .replace(/\x1b\[37m/g, '<span class="ansi-white">')
    .replace(/\x1b\[0m/g, "</span>")
    .replace(/\x1b\[1m/g, '<span class="ansi-bold">')
    .replace(/\x1b\[22m/g, "</span>");
};

const formatOutputLine = (line: OutputLine): string => {
  return line.content;
};

const focusCommandInput = () => {
  const activeTabIndex = tabs.value.findIndex((tab) => tab.id === activeTabId.value);
  if (activeTabIndex !== -1 && commandInput.value[activeTabIndex]) {
    commandInput.value[activeTabIndex].focus();
    moveCursorToEnd();
  }
};

const moveCursorToEnd = () => {
  const activeTabIndex = tabs.value.findIndex((tab) => tab.id === activeTabId.value);
  if (activeTabIndex !== -1 && commandInput.value[activeTabIndex]) {
    const input = commandInput.value[activeTabIndex];
    input.setSelectionRange(input.value.length, input.value.length);
  }
};

const scrollToBottom = () => {
  const activeTabIndex = tabs.value.findIndex((tab) => tab.id === activeTabId.value);
  if (activeTabIndex !== -1 && outputContainer.value[activeTabIndex]) {
    const container = outputContainer.value[activeTabIndex];
    container.scrollTop = container.scrollHeight;
  }
};

const handleScroll = (event: Event) => {
  // 可以在这里处理滚动事件
};

const addToHistory = (command: string) => {
  // 避免重复记录相同的命令
  if (commandHistory.value[commandHistory.value.length - 1] !== command) {
    commandHistory.value.push(command);

    // 限制历史记录数量
    if (commandHistory.value.length > terminalSettings.value.maxHistory) {
      commandHistory.value = commandHistory.value.slice(-terminalSettings.value.maxHistory);
    }

    if (terminalSettings.value.autoSaveHistory) {
      saveCommandHistory();
    }
  }
};

const loadCommandHistory = () => {
  try {
    const saved = localStorage.getItem("bash-command-history");
    if (saved) {
      commandHistory.value = JSON.parse(saved);
    }
  } catch (error) {
    console.warn("Failed to load command history:", error);
  }
};

const saveCommandHistory = () => {
  try {
    localStorage.setItem("bash-command-history", JSON.stringify(commandHistory.value));
  } catch (error) {
    console.warn("Failed to save command history:", error);
  }
};

const getTabInfo = (tab: Tab): string => {
  if (tab.isRunning) {
    return "运行中";
  }
  return `${tab.output.length} 行输出`;
};

// 设置管理
const toggleSettings = () => {
  showSettings.value = !showSettings.value;
};

const saveSettings = () => {
  localStorage.setItem("terminal-settings", JSON.stringify(terminalSettings.value));
  showSettings.value = false;
};

const resetSettings = () => {
  terminalSettings.value = {
    fontSize: "text-sm",
    theme: "dark",
    cursorStyle: "block",
    bell: false,
    autoScroll: true,
    maxHistory: 1000,
    autoSaveHistory: true,
  };
};

// 生命周期
onMounted(() => {
  connectWebSocket();

  // 加载设置
  try {
    const saved = localStorage.getItem("terminal-settings");
    if (saved) {
      terminalSettings.value = { ...terminalSettings.value, ...JSON.parse(saved) };
    }
  } catch (error) {
    console.warn("Failed to load terminal settings:", error);
  }

  // 全局键盘事件监听
  document.addEventListener("keydown", handleGlobalKeydown);
});

onUnmounted(() => {
  document.removeEventListener("keydown", handleGlobalKeydown);
});

const handleGlobalKeydown = (event: KeyboardEvent) => {
  // 快捷键：Ctrl+T 新建标签页
  if ((event.ctrlKey || event.metaKey) && event.key === "t") {
    event.preventDefault();
    createNewTab();
  }

  // 快捷键：Ctrl+W 关闭当前标签页
  if ((event.ctrlKey || event.metaKey) && event.key === "w") {
    event.preventDefault();
    if (currentTab.value) {
      closeTab(currentTab.value.id);
    }
  }

  // 快捷键：Ctrl+Tab 切换标签页
  if (event.ctrlKey && event.key === "Tab") {
    event.preventDefault();
    const currentIndex = tabs.value.findIndex((tab) => tab.id === activeTabId.value);
    const nextIndex = (currentIndex + 1) % tabs.value.length;
    const nextTab = tabs.value[nextIndex];
    if (nextTab) {
      switchTab(nextTab.id);
    }
  }
};
</script>

<style scoped>
.bash-tool {
  @apply flex flex-col h-full bg-surface dark:bg-surface-dark border border-border dark:border-border-dark rounded-lg;
}

.bash-header {
  @apply flex items-center justify-between px-4 py-3 border-b border-border dark:border-border-dark bg-surface dark:bg-surface-dark;
}

.header-left {
  @apply flex items-center gap-3 flex-1 min-w-0;
}

.header-title {
  @apply flex items-center gap-2 font-semibold text-text dark:text-text-dark;
}

.current-directory {
  @apply flex items-center gap-2 px-3 py-1 bg-gray-100 dark:bg-gray-700 rounded-full text-sm text-gray-600 dark:text-gray-400;
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

.setting-select,
.setting-checkbox {
  @apply ml-2;
}

.setting-select {
  @apply px-2 py-1 text-sm border border-border dark:border-border-dark rounded focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-800 dark:text-white;
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

.tabs-container {
  @apply border-b border-gray-200 dark:border-gray-600;
}

.tabs {
  @apply flex overflow-x-auto;
}

.tab {
  @apply flex items-center gap-2 px-4 py-2 bg-gray-100 dark:bg-gray-700 border-r border-gray-200 dark:border-gray-600 cursor-pointer hover:bg-gray-200 dark:hover:bg-gray-600 transition-colors;
}

.tab.active {
  @apply bg-white dark:bg-gray-800 text-blue-600 dark:text-blue-400;
}

.tab.running {
  @apply text-green-600 dark:text-green-400;
}

.tab-title {
  @apply text-sm font-medium truncate;
}

.tab-close {
  @apply p-0.5 text-gray-400 hover:text-red-500 transition-colors;
}

.terminal-container {
  @apply flex-1 relative overflow-hidden;
}

.terminal-content {
  @apply flex flex-col h-full;
}

.output-container {
  @apply flex-1 overflow-y-auto p-4 font-mono leading-relaxed;
}

/* 主题样式 */
.theme-dark {
  @apply bg-gray-900 text-gray-100;
}

.theme-light {
  @apply bg-white text-gray-900;
}

.theme-solarized {
  @apply bg-[#002b36] text-[#839496];
}

.theme-monokai {
  @apply bg-[#272822] text-[#f8f8f2];
}

.output-line {
  @apply whitespace-pre-wrap break-all;
}

.output-line.input {
  @apply text-blue-400 dark:text-blue-300;
}

.output-line.error {
  @apply text-red-400 dark:text-red-300;
}

.output-line.success {
  @apply text-green-400 dark:text-green-300;
}

.output-line.warning {
  @apply text-yellow-400 dark:text-yellow-300;
}

/* ANSI 颜色样式 */
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
  @apply text-gray-300;
}
:deep(.ansi-bold) {
  @apply font-bold;
}

.input-line {
  @apply flex items-center;
}

.prompt {
  @apply text-green-400 dark:text-green-300 mr-2;
}

.command-input {
  @apply flex-1 bg-transparent border-0 outline-none;
}

.cursor-block {
  @apply relative;
}

.cursor-block::after {
  @apply absolute w-2 h-4 bg-white;
  content: "";
  animation: blink 1s infinite;
}

.cursor-underline {
  @apply border-b-2 border-white;
}

.cursor-bar {
  @apply border-l-2 border-white;
}

.running-indicator {
  @apply flex items-center;
}

.current-command {
  @apply mr-2;
}

.quick-commands {
  @apply border-t border-gray-200 dark:border-gray-600 bg-gray-50 dark:bg-gray-700 p-2;
}

.quick-commands-title {
  @apply text-xs font-medium text-gray-600 dark:text-gray-400 mb-1;
}

.quick-commands-list {
  @apply flex flex-wrap gap-1;
}

.quick-cmd-btn {
  @apply px-2 py-1 text-xs bg-white dark:bg-gray-600 border border-gray-300 dark:border-gray-500 rounded hover:bg-gray-100 dark:hover:bg-gray-500 transition-colors;
}

.status-bar {
  @apply flex items-center justify-between px-4 py-1 border-t border-border dark:border-border-dark bg-gray-100 dark:bg-gray-800 text-xs text-gray-600 dark:text-gray-400;
}

.status-left,
.status-right {
  @apply flex items-center gap-3;
}

.running-status {
  @apply text-green-500 flex items-center gap-1;
}

.animate-spin {
  @apply animate-spin;
}

@keyframes blink {
  0%,
  50% {
    opacity: 1;
  }
  51%,
  100% {
    opacity: 0;
  }
}
</style>
