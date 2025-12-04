<template>
  <div class="glob-tool">
    <!-- 头部工具栏 -->
    <div class="glob-header">
      <div class="header-title">
        <Icon type="filter" size="sm" />
        <span>文件匹配</span>
      </div>
      <div class="header-actions">
        <button class="action-button" title="清除结果" @click="clearResults">
          <Icon type="trash" size="sm" />
        </button>
      </div>
    </div>

    <!-- 搜索表单 -->
    <div class="search-form">
      <div class="pattern-input-wrapper">
        <input v-model="pattern" ref="patternInput" type="text" placeholder="输入文件模式 (如: *.js, **/*.ts, src/**/*.vue)" class="pattern-input" @keydown.enter="performGlob" @keydown.esc="pattern = ''" />
        <div class="pattern-hints">
          <span class="hint-text">提示:</span>
          <button class="hint-btn" @click="setPattern('*.js')">*.js</button>
          <button class="hint-btn" @click="setPattern('**/*.vue')">**/*.vue</button>
          <button class="hint-btn" @click="setPattern('src/**/*.{js,ts,jsx,tsx}')">源代码</button>
          <button class="hint-btn" @click="setPattern('test/**/*.spec.js')">测试文件</button>
        </div>
      </div>
      <div class="search-actions">
        <button class="search-button" :disabled="!pattern.trim() || isSearching" @click="performGlob">
          <Icon v-if="isSearching" type="spinner" size="sm" class="animate-spin" />
          <span v-else>匹配</span>
        </button>
      </div>
    </div>

    <!-- 搜索结果统计 -->
    <div v-if="globResults.length > 0 || hasSearched" class="results-summary">
      <div class="summary-info">
        <span class="result-count">{{ globResults.length }} 个文件</span>
        <span v-if="searchDuration" class="search-duration"> 耗时 {{ searchDuration }}ms </span>
      </div>
    </div>

    <!-- 文件列表 -->
    <div class="file-list">
      <div v-for="(file, index) in globResults" :key="index" class="file-item" @click="selectFile(file)">
        <div class="file-icon">
          <Icon :type="getIconForFile(file)" size="sm" />
        </div>
        <div class="file-info">
          <div class="file-name">{{ file.name }}</div>
          <div class="file-path">{{ file.path }}</div>
          <div class="file-meta">
            <span v-if="file.size" class="file-size">{{ formatFileSize(file.size) }}</span>
            <span class="file-modified">{{ formatDate(file.modified) }}</span>
          </div>
        </div>
        <div class="file-actions">
          <button class="action-btn view-btn" title="查看文件" @click.stop="viewFile(file)">
            <Icon type="eye" size="xs" />
          </button>
          <button class="action-btn edit-btn" title="编辑文件" @click.stop="editFile(file)">
            <Icon type="edit" size="xs" />
          </button>
        </div>
      </div>

      <!-- 空状态 -->
      <div v-if="!isSearching && hasSearched && globResults.length === 0" class="empty-state">
        <Icon type="filter" size="lg" />
        <p>未找到匹配的文件</p>
        <p class="empty-hint">尝试调整文件模式</p>
      </div>

      <!-- 搜索中状态 -->
      <div v-if="isSearching" class="searching-state">
        <Icon type="spinner" size="lg" class="animate-spin" />
        <p>正在匹配文件...</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, nextTick, onMounted } from "vue";
import Icon from "../ChatUI/Icon.vue";

interface GlobFile {
  name: string;
  path: string;
  size: number;
  modified: number;
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
  fileSelected: [file: GlobFile];
  fileOpened: [file: GlobFile];
}>();

// 响应式数据
const pattern = ref("");
const globResults = ref<GlobFile[]>([]);
const isSearching = ref(false);
const hasSearched = ref(false);
const searchDuration = ref(0);

const patternInput = ref<HTMLInputElement>();
const websocket = ref<WebSocket | null>(null);

// WebSocket 连接
const connectWebSocket = () => {
  try {
    websocket.value = new WebSocket(`${props.wsUrl}?session=${props.sessionId}`);

    websocket.value.onopen = () => {
      console.log("GlobTool WebSocket connected");
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
      console.log("GlobTool WebSocket disconnected");
      setTimeout(connectWebSocket, 5000);
    };

    websocket.value.onerror = (error) => {
      console.error("GlobTool WebSocket error:", error);
    };
  } catch (error) {
    console.error("Failed to connect WebSocket:", error);
  }
};

const handleWebSocketMessage = (message: any) => {
  switch (message.type) {
    case "glob_started":
      isSearching.value = true;
      hasSearched.value = true;
      break;
    case "glob_result":
      if (message.file) {
        const file: GlobFile = message.file;
        globResults.value.push(file);
      }
      break;
    case "glob_completed":
      isSearching.value = false;
      searchDuration.value = message.duration || 0;
      break;
  }
};

const sendWebSocketMessage = (message: any) => {
  if (websocket.value && websocket.value.readyState === WebSocket.OPEN) {
    websocket.value.send(JSON.stringify(message));
  }
};

// 文件操作方法
const performGlob = () => {
  if (!pattern.value.trim()) return;

  // 清除之前的结果
  globResults.value = [];
  isSearching.value = false;
  hasSearched.value = true;

  // 执行匹配
  sendWebSocketMessage({
    type: "glob_search",
    pattern: pattern.value.trim(),
  });
};

const clearResults = () => {
  globResults.value = [];
  hasSearched.value = false;
  searchDuration.value = 0;
};

const setPattern = (newPattern: string) => {
  pattern.value = newPattern;
};

const selectFile = (file: GlobFile) => {
  emit("fileSelected", file);
};

const viewFile = (file: GlobFile) => {
  emit("fileOpened", file);
};

const editFile = (file: GlobFile) => {
  emit("fileOpened", file);
};

// 工具方法
const getIconForFile = (file: GlobFile) => {
  const extension = file.name.split(".").pop()?.toLowerCase();

  if (["jpg", "jpeg", "png", "gif", "bmp", "svg", "webp"].includes(extension || "")) {
    return "image";
  }
  if (["pdf"].includes(extension || "")) {
    return "file-pdf";
  }
  if (["zip", "rar", "7z", "tar", "gz"].includes(extension || "")) {
    return "archive";
  }
  if (["txt", "md", "json", "js", "ts", "jsx", "tsx", "html", "css", "py", "java", "cpp", "c", "go", "rs", "sql"].includes(extension || "")) {
    return "file-text";
  }

  return "file";
};

const formatFileSize = (bytes: number) => {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + " " + sizes[i];
};

const formatDate = (timestamp: number) => {
  const date = new Date(timestamp);
  return date.toLocaleDateString("zh-CN");
};

// 生命周期
onMounted(() => {
  connectWebSocket();

  // 自动聚焦模式输入框
  watch(pattern, (value) => {
    if (value) {
      nextTick(() => {
        patternInput.value?.focus();
      });
    }
  });
});
</script>

<style scoped>
.glob-tool {
  @apply flex flex-col h-full bg-surface dark:bg-surface-dark border border-border dark:border-border-dark rounded-lg;
}

.glob-header {
  @apply flex items-center justify-between px-4 py-3 border-b border-border dark:border-border-dark bg-surface dark:bg-surface-dark;
}

.header-title {
  @apply flex items-center gap-2 font-semibold text-text dark:text-text-dark;
}

.header-actions {
  @apply flex gap-1;
}

.action-button {
  @apply p-1.5 text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700 rounded transition-colors disabled:opacity-50 disabled:cursor-not-allowed;
}

.search-form {
  @apply px-4 py-3 border-b border-border dark:border-border-dark space-y-2;
}

.pattern-input-wrapper {
  @apply space-y-2;
}

.pattern-input {
  @apply w-full px-3 py-2 border border-border dark:border-border-dark rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-800 dark:text-white;
}

.pattern-hints {
  @apply flex items-center gap-2 flex-wrap;
}

.hint-text {
  @apply text-sm text-gray-500 dark:text-gray-400;
}

.hint-btn {
  @apply px-2 py-1 text-xs bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded hover:bg-gray-200 dark:hover:bg-gray-600 transition-colors;
}

.search-actions {
  @apply flex justify-end;
}

.search-button {
  @apply flex items-center gap-2 px-4 py-2 bg-blue-500 hover:bg-blue-600 text-white rounded transition-colors disabled:opacity-50 disabled:cursor-not-allowed;
}

.results-summary {
  @apply px-4 py-2 border-b border-border dark:border-border-dark bg-gray-50 dark:bg-gray-700/50;
}

.summary-info {
  @apply flex items-center gap-3 text-sm text-gray-600 dark:text-gray-300;
}

.result-count {
  @apply font-medium text-text dark:text-text-dark;
}

.file-list {
  @apply flex-1 overflow-y-auto;
}

.file-item {
  @apply flex items-center gap-3 p-3 border-b border-gray-100 dark:border-gray-700 last:border-b-0 hover:bg-gray-50 dark:hover:bg-gray-700/50 cursor-pointer;
}

.file-icon {
  @apply flex-shrink-0 w-8 h-8 flex items-center justify-center;
}

.file-info {
  @apply flex-1 min-w-0;
}

.file-name {
  @apply font-medium text-text dark:text-text-dark truncate;
}

.file-path {
  @apply text-xs text-gray-500 dark:text-gray-400 truncate;
}

.file-meta {
  @apply flex items-center gap-3 text-xs text-gray-500 dark:text-gray-400;
}

.file-actions {
  @apply flex gap-1 opacity-0 hover:opacity-100 transition-opacity;
}

.file-item:hover .file-actions {
  @apply opacity-100;
}

.action-btn {
  @apply p-1 text-gray-400 dark:text-gray-500 hover:text-gray-600 dark:hover:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-600 rounded transition-colors;
}

.view-btn:hover {
  @apply text-blue-500 hover:bg-blue-50 dark:hover:bg-blue-900/20;
}

.edit-btn:hover {
  @apply text-green-500 hover:bg-green-50 dark:hover:bg-green-900/20;
}

.empty-state {
  @apply flex flex-col items-center justify-center py-8 text-gray-400 dark:text-gray-500;
}

.empty-hint {
  @apply text-xs mt-1;
}

.searching-state {
  @apply flex flex-col items-center justify-center py-8 text-gray-400 dark:text-gray-500;
}

.animate-spin {
  @apply animate-spin;
}
</style>
