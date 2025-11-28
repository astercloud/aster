<template>
  <div class="grep-tool">
    <!-- 头部工具栏 -->
    <div class="grep-header">
      <div class="header-title">
        <Icon type="search" size="sm" />
        <span>文本搜索</span>
      </div>
      <div class="header-actions">
        <button
          class="action-button"
          title="清除结果"
          @click="clearResults"
        >
          <Icon type="trash" size="sm" />
        </button>
        <button
          class="action-button"
          title="导出结果"
          :disabled="searchResults.length === 0"
          @click="exportResults"
        >
          <Icon type="download" size="sm" />
        </button>
      </div>
    </div>

    <!-- 搜索表单 -->
    <div class="search-form">
      <div class="search-input-wrapper">
        <input
          v-model="searchQuery"
          ref="searchInput"
          type="text"
          placeholder="输入搜索文本..."
          class="search-input"
          @keydown.enter="performSearch"
          @keydown.esc="searchQuery = ''"
        />
        <div class="search-options">
          <select v-model="searchOptions.caseSensitive" class="option-select">
            <option :value="false">忽略大小写</option>
            <option :value="true">区分大小写</option>
          </select>
          <select v-model="searchOptions.useRegex" class="option-select">
            <option :value="false">普通文本</option>
            <option :value="true">正则表达式</option>
          </select>
          <select v-model="searchOptions.include" class="option-select">
            <option value="all">所有文件</option>
            <option value="code">代码文件</option>
            <option value="text">文本文件</option>
          </select>
          <input
            v-model="searchOptions.extension"
            type="text"
            placeholder="文件扩展名"
            class="extension-input"
            title="用逗号分隔多个扩展名，如: .js,.ts,.jsx"
          />
        </div>
      </div>
      <div class="search-actions">
        <button
          class="search-button"
          :disabled="!searchQuery.trim() || isSearching"
          @click="performSearch"
        >
          <Icon v-if="isSearching" type="spinner" size="sm" class="animate-spin" />
          <span v-else>搜索</span>
        </button>
      </div>
    </div>

    <!-- 搜索结果统计 -->
    <div v-if="searchResults.length > 0 || hasSearched" class="results-summary">
      <div class="summary-info">
        <span class="result-count">{{ searchResults.length }} 个结果</span>
        <span v-if="searchDuration" class="search-duration">
          耗时 {{ searchDuration }}ms
        </span>
        <span v-if="searchedFiles" class="searched-files">
          扫描 {{ searchedFiles }} 个文件
        </span>
      </div>
      <div class="summary-actions">
        <button
          v-if="searchResults.length > 0"
          class="expand-all-btn"
          @click="toggleAllExpanded"
        >
          {{ allExpanded ? '收起全部' : '展开全部' }}
        </button>
      </div>
    </div>

    <!-- 搜索结果列表 -->
    <div class="results-list" ref="resultsListRef">
      <div
        v-for="(result, index) in searchResults"
        :key="index"
        :class="['result-item', { 'result-expanded': result.expanded }]"
      >
        <div class="result-header" @click="toggleResultExpanded(index)">
          <div class="result-file">
            <Icon type="file" size="xs" />
            <span class="file-path">{{ result.file }}</span>
            <span class="line-number">{{ result.lineNumber }}</span>
          </div>
          <div class="result-actions">
            <button
              class="action-btn open-btn"
              title="打开文件"
              @click.stop="openFile(result)"
            >
              <Icon type="external-link" size="xs" />
            </button>
            <button
              class="action-btn copy-btn"
              title="复制行"
              @click.stop="copyLine(result)"
            >
              <Icon type="copy" size="xs" />
            </button>
          </div>
        </div>

        <div v-if="result.expanded" class="result-content">
          <div class="code-preview">
            <div class="preview-header">
              <span class="file-name">{{ result.file }}</span>
              <span class="line-info">第 {{ result.lineNumber }} 行</span>
            </div>
            <div class="code-lines">
              <div
                v-for="(line, lineIndex) in result.contextLines"
                :key="lineIndex"
                :class="['code-line', { 'line-highlight': lineIndex === result.contextLines.length - 2 }]"
              >
                <span class="line-number">{{ lineIndex + result.lineNumber - result.contextLines.length + 1 }}</span>
                <span class="line-content">{{ line }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 空状态 -->
      <div v-if="!isSearching && hasSearched && searchResults.length === 0" class="empty-state">
        <Icon type="search" size="lg" />
        <p>未找到匹配的结果</p>
        <p class="empty-hint">尝试调整搜索条件或文件过滤器</p>
      </div>

      <!-- 搜索中状态 -->
      <div v-if="isSearching" class="searching-state">
        <Icon type="spinner" size="lg" class="animate-spin" />
        <p>正在搜索中...</p>
      </div>
    </div>

    <!-- 搜索历史 -->
    <div class="search-history" v-if="searchHistory.length > 0">
      <div class="history-header">
        <h4>搜索历史</h4>
        <button
          class="clear-history-btn"
          @click="clearHistory"
        >
          清除历史
        </button>
      </div>
      <div class="history-list">
        <div
          v-for="(history, index) in searchHistory"
          :key="index"
          class="history-item"
          @click="loadFromHistory(history)"
        >
          <span class="history-query">{{ history.query }}</span>
          <span class="history-time">{{ formatTime(history.timestamp) }}</span>
          <button
            class="remove-history-btn"
            @click.stop="removeFromHistory(index)"
          >
            <Icon type="close" size="xs" />
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick, onMounted } from 'vue';
import Icon from '../ChatUI/Icon.vue';

interface SearchResult {
  file: string;
  lineNumber: number;
  content: string;
  contextLines: string[];
  expanded: boolean;
}

interface SearchHistory {
  query: string;
  options: any;
  timestamp: number;
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
  fileOpened: [file: string, line: number];
}>();

// 响应式数据
const searchQuery = ref('');
const searchResults = ref<SearchResult[]>([]);
const searchHistory = ref<SearchHistory[]>([]);
const isSearching = ref(false);
const hasSearched = ref(false);
const searchDuration = ref(0);
const searchedFiles = ref(0);
const allExpanded = ref(false);

const searchOptions = ref({
  caseSensitive: false,
  useRegex: false,
  include: 'all',
  extension: '',
});

const searchInput = ref<HTMLInputElement>();
const resultsListRef = ref<HTMLElement>();
const websocket = ref<WebSocket | null>(null);

// WebSocket 连接
const connectWebSocket = () => {
  try {
    websocket.value = new WebSocket(`${props.wsUrl}?session=${props.sessionId}`);

    websocket.value.onopen = () => {
      console.log('GrepTool WebSocket connected');
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
      console.log('GrepTool WebSocket disconnected');
      setTimeout(connectWebSocket, 5000);
    };

    websocket.value.onerror = (error) => {
      console.error('GrepTool WebSocket error:', error);
    };
  } catch (error) {
    console.error('Failed to connect WebSocket:', error);
  }
};

const handleWebSocketMessage = (message: any) => {
  switch (message.type) {
    case 'grep_started':
      isSearching.value = true;
      hasSearched.value = true;
      break;
    case 'grep_result':
      if (message.result) {
        const result: SearchResult = {
          file: message.result.file,
          lineNumber: message.result.lineNumber,
          content: message.result.content,
          contextLines: message.result.contextLines || [],
          expanded: false,
        };
        searchResults.value.push(result);
      }
      break;
    case 'grep_completed':
      isSearching.value = false;
      searchDuration.value = message.duration || 0;
      searchedFiles.value = message.searchedFiles || 0;
      break;
  }
};

const sendWebSocketMessage = (message: any) => {
  if (websocket.value && websocket.value.readyState === WebSocket.OPEN) {
    websocket.value.send(JSON.stringify(message));
  }
};

// 搜索操作方法
const performSearch = () => {
  if (!searchQuery.value.trim()) return;

  // 清除之前的结果
  searchResults.value = [];
  isSearching.value = false;
  hasSearched.value = true;

  // 添加到历史记录
  const historyEntry: SearchHistory = {
    query: searchQuery.value.trim(),
    options: { ...searchOptions.value },
    timestamp: Date.now(),
  };

  const existingIndex = searchHistory.value.findIndex(h => h.query === historyEntry.query);
  if (existingIndex !== -1) {
    searchHistory.value[existingIndex] = historyEntry;
  } else {
    searchHistory.value.unshift(historyEntry);
    if (searchHistory.value.length > 10) {
      searchHistory.value = searchHistory.value.slice(0, 10);
    }
  }

  // 执行搜索
  sendWebSocketMessage({
    type: 'grep_search',
    query: searchQuery.value.trim(),
    options: searchOptions.value,
  });
};

const clearResults = () => {
  searchResults.value = [];
  hasSearched.value = false;
  searchDuration.value = 0;
  searchedFiles.value = 0;
};

const exportResults = () => {
  if (searchResults.value.length === 0) return;

  const csvContent = [
    '文件路径,行号,匹配内容',
    ...searchResults.value.map(result =>
      `"${result.file}","${result.lineNumber}","${result.content.replace(/"/g, '""')}"`
    )
  ].join('\n');

  const blob = new Blob([csvContent], { type: 'text/csv' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = `grep_results_${Date.now()}.csv`;
  a.click();
  URL.revokeObjectURL(url);
};

// 结果操作方法
const toggleResultExpanded = (index: number) => {
  const result = searchResults.value[index];
  if (result) {
    result.expanded = !result.expanded;
  }
};

const toggleAllExpanded = () => {
  const newExpanded = !allExpanded.value;
  searchResults.value.forEach(result => {
    result.expanded = newExpanded;
  });
  allExpanded.value = newExpanded;
};

const openFile = (result: SearchResult) => {
  emit('fileOpened', result.file, result.lineNumber);
};

const copyLine = (result: SearchResult) => {
  navigator.clipboard.writeText(result.content);
};

// 历史记录方法
const loadFromHistory = (history: SearchHistory) => {
  searchQuery.value = history.query;
  searchOptions.value = { ...history.options };
  performSearch();
};

const removeFromHistory = (index: number) => {
  searchHistory.value.splice(index, 1);
};

const clearHistory = () => {
  searchHistory.value = [];
};

// 工具方法
const formatTime = (timestamp: number) => {
  const date = new Date(timestamp);
  return date.toLocaleString('zh-CN');
};

// 生命周期
onMounted(() => {
  connectWebSocket();

  // 自动聚焦搜索输入框
  watch(searchQuery, (value) => {
    if (value) {
      nextTick(() => {
        searchInput.value?.focus();
      });
    }
  });
});
</script>

<style scoped>
.grep-tool {
  @apply flex flex-col h-full bg-surface dark:bg-surface-dark border border-border dark:border-border-dark rounded-lg;
}

.grep-header {
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

.search-input-wrapper {
  @apply space-y-2;
}

.search-input {
  @apply w-full px-3 py-2 border border-border dark:border-border-dark rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-800 dark:text-white;
}

.search-options {
  @apply flex flex-wrap gap-2;
}

.option-select, .extension-input {
  @apply px-2 py-1 text-sm border border-border dark:border-border-dark rounded focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-800 dark:text-white;
}

.extension-input {
  @apply min-w-0 flex-1;
}

.search-actions {
  @apply flex justify-end;
}

.search-button {
  @apply flex items-center gap-2 px-4 py-2 bg-blue-500 hover:bg-blue-600 text-white rounded transition-colors disabled:opacity-50 disabled:cursor-not-allowed;
}

.results-summary {
  @apply flex items-center justify-between px-4 py-2 border-b border-border dark:border-border-dark bg-gray-50 dark:bg-gray-700/50;
}

.summary-info {
  @apply flex items-center gap-3 text-sm text-gray-600 dark:text-gray-300;
}

.result-count {
  @apply font-medium text-text dark:text-text-dark;
}

.summary-actions {
  @apply flex gap-2;
}

.expand-all-btn {
  @apply px-2 py-1 text-sm text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300 border border-blue-300 dark:border-blue-600 rounded transition-colors;
}

.results-list {
  @apply flex-1 overflow-y-auto;
}

.result-item {
  @apply border-b border-gray-100 dark:border-gray-700 last:border-b-0;
}

.result-header {
  @apply flex items-center justify-between p-3 hover:bg-gray-50 dark:hover:bg-gray-700/50 cursor-pointer;
}

.result-file {
  @apply flex items-center gap-2 flex-1 min-w-0;
}

.file-path {
  @apply text-sm font-medium text-blue-600 dark:text-blue-400 truncate;
}

.line-number {
  @apply text-xs px-2 py-1 bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-300 rounded;
}

.result-actions {
  @apply flex gap-1 opacity-0 hover:opacity-100 transition-opacity;
}

.result-item:hover .result-actions {
  @apply opacity-100;
}

.action-btn {
  @apply p-1 text-gray-400 dark:text-gray-500 hover:text-gray-600 dark:hover:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-600 rounded transition-colors;
}

.result-content {
  @apply bg-gray-50 dark:bg-gray-700/30 border-t border-gray-200 dark:border-gray-600;
}

.code-preview {
  @apply font-mono text-sm;
}

.preview-header {
  @apply flex items-center justify-between px-3 py-2 bg-gray-100 dark:bg-gray-600 text-xs text-gray-600 dark:text-gray-300 border-b border-gray-200 dark:border-gray-500;
}

.file-name {
  @apply font-medium;
}

.line-info {
  @apply text-gray-500 dark:text-gray-400;
}

.code-lines {
  @apply overflow-x-auto;
}

.code-line {
  @apply flex border-b border-gray-100 dark:border-gray-700;
}

.code-line:last-child {
  @apply border-b-0;
}

.line-number {
  @apply w-16 px-2 py-2 text-right text-gray-400 dark:text-gray-500 bg-gray-100 dark:bg-gray-700 border-r border-gray-200 dark:border-gray-600;
}

.line-content {
  @apply px-3 py-2 flex-1;
}

.line-highlight {
  @apply bg-yellow-100 dark:bg-yellow-900/30;
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

.search-history {
  @apply border-t border-border dark:border-border-dark bg-gray-50 dark:bg-gray-700/30;
}

.history-header {
  @apply flex items-center justify-between px-4 py-2 border-b border-gray-200 dark:border-gray-600;
}

.history-header h4 {
  @apply text-sm font-semibold text-text dark:text-text-dark;
}

.clear-history-btn {
  @apply text-xs text-red-500 dark:text-red-400 hover:text-red-600 dark:hover:text-red-300;
}

.history-list {
  @apply max-h-32 overflow-y-auto;
}

.history-item {
  @apply flex items-center justify-between px-4 py-2 hover:bg-gray-100 dark:hover:bg-gray-600 cursor-pointer border-b border-gray-200 dark:border-gray-600 last:border-b-0;
}

.history-query {
  @apply text-sm text-gray-700 dark:text-gray-200 truncate flex-1;
}

.history-time {
  @apply text-xs text-gray-500 dark:text-gray-400;
}

.remove-history-btn {
  @apply p-1 text-gray-400 hover:text-red-500 transition-colors;
}

.animate-spin {
  @apply animate-spin;
}
</style>