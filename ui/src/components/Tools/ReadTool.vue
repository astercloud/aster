<template>
  <div class="read-tool">
    <!-- 头部工具栏 -->
    <div class="read-header">
      <div class="header-title">
        <Icon type="file-text" size="sm" />
        <span>文件阅读</span>
      </div>
      <div class="header-actions">
        <button
          class="action-button"
          title="重新加载"
          :disabled="!currentFile"
          @click="reloadFile"
        >
          <Icon type="refresh" size="sm" />
        </button>
        <button
          class="action-button"
          title="切换只读模式"
          :disabled="!currentFile"
          @click="toggleReadonly"
        >
          <Icon :type="isReadonly ? 'lock' : 'unlock'" size="sm" />
        </button>
        <button
          class="action-button"
          title="关闭文件"
          :disabled="!currentFile"
          @click="closeFile"
        >
          <Icon type="close" size="sm" />
        </button>
      </div>
    </div>

    <!-- 文件选择 -->
    <div v-if="!currentFile" class="file-selection">
      <div class="selection-hint">
        <Icon type="file-text" size="lg" />
        <p>选择要查看的文件</p>
        <p class="hint-text">支持文本文件、代码文件、图片、PDF等格式</p>
      </div>

      <!-- 最近文件 -->
      <div v-if="recentFiles.length > 0" class="recent-files">
        <h4>最近打开</h4>
        <div class="recent-list">
          <div
            v-for="(file, index) in recentFiles"
            :key="index"
            class="recent-item"
            @click="openFile(file)"
          >
            <div class="file-icon">
              <Icon :type="getIconForFile(file)" size="sm" />
            </div>
            <div class="file-info">
              <div class="file-name">{{ file.name }}</div>
              <div class="file-path">{{ file.path }}</div>
            </div>
          </div>
        </div>
      </div>

      <!-- 文件拖放区域 -->
      <div
        class="drop-zone"
        :class="{ 'drag-over': isDragOver }"
        @dragover.prevent="handleDragOver"
        @dragleave.prevent="handleDragLeave"
        @drop.prevent="handleDrop"
      >
        <Icon type="upload" size="lg" />
        <p>拖拽文件到此处</p>
        <p class="drop-hint">或点击选择文件</p>
        <input
          type="file"
          ref="fileInput"
          class="file-input"
          @change="handleFileSelect"
          multiple
        />
      </div>
    </div>

    <!-- 文件内容显示 -->
    <div v-if="currentFile" class="file-content">
      <!-- 文件信息栏 -->
      <div class="file-info-bar">
        <div class="file-info">
          <div class="file-icon">
            <Icon :type="getIconForFile(currentFile)" size="sm" />
          </div>
          <div class="file-details">
            <div class="file-name">{{ currentFile.name }}</div>
            <div class="file-path">{{ currentFile.path }}</div>
          </div>
        </div>
        <div class="file-stats">
          <span v-if="fileSize" class="file-size">{{ fileSize }}</span>
          <span v-if="fileModified" class="file-modified">{{ fileModified }}</span>
          <span v-if="fileType" class="file-type">{{ fileType }}</span>
        </div>
      </div>

      <!-- 图片显示 -->
      <div v-if="isImageFile" class="image-viewer">
        <img
          :src="imageData"
          :alt="currentFile.name"
          class="file-image"
          @load="handleImageLoad"
          @error="handleImageError"
        />
        <div v-if="imageInfo" class="image-info">
          <span>尺寸: {{ imageInfo.width }} × {{ imageInfo.height }}</span>
        </div>
      </div>

      <!-- PDF显示 -->
      <div v-else-if="isPdfFile" class="pdf-viewer">
        <iframe
          v-if="pdfData"
          :src="pdfData"
          class="pdf-frame"
          title="PDF Viewer"
        ></iframe>
        <div v-else class="pdf-placeholder">
          <Icon type="file-pdf" size="lg" />
          <p>PDF预览不可用</p>
          <button class="download-btn" @click="downloadFile">
            下载文件
          </button>
        </div>
      </div>

      <!-- 代码/文本显示 -->
      <div v-else-if="isTextFile" class="text-viewer">
        <div class="viewer-toolbar">
          <div class="toolbar-left">
            <select v-model="lineWrapMode" class="wrap-select">
              <option value="wrap">自动换行</option>
              <option value="nowrap">不换行</option>
            </select>
            <select v-model="fontSize" class="font-select">
              <option value="text-xs">极小</option>
              <option value="text-sm">小</option>
              <option value="text-base">正常</option>
              <option value="text-lg">大</option>
              <option value="text-xl">极大</option>
            </select>
          </div>
          <div class="toolbar-right">
            <button
              class="toolbar-btn"
              title="复制内容"
              @click="copyContent"
            >
              <Icon type="copy" size="xs" />
              复制
            </button>
            <button
              class="toolbar-btn"
              title="全选"
              @click="selectAll"
            >
              <Icon type="check-square" size="xs" />
              全选
            </button>
          </div>
        </div>

        <div class="code-container" :class="{ 'no-wrap': lineWrapMode === 'nowrap' }">
          <div class="line-numbers">
            <div
              v-for="(line, index) in contentLines"
              :key="index"
              class="line-number"
            >
              {{ index + 1 }}
            </div>
          </div>
          <div class="code-content">
            <pre
              ref="codeElement"
              :class="['code-block', fontSize, { 'readonly': isReadonly }]"
              :contenteditable="!isReadonly"
              @blur="handleContentEdit"
              @keydown="handleKeydown"
            >{{ fileContent }}</pre>
          </div>
        </div>
      </div>

      <!-- 二进制文件 -->
      <div v-else class="binary-viewer">
        <div class="binary-placeholder">
          <Icon type="file" size="lg" />
          <p>二进制文件无法直接显示</p>
          <p class="binary-hint">{{ currentFile.name }}</p>
          <button class="download-btn" @click="downloadFile">
            下载文件
          </button>
        </div>
      </div>
    </div>

    <!-- 加载状态 -->
    <div v-if="isLoading" class="loading-state">
      <Icon type="spinner" size="lg" class="animate-spin" />
      <p>正在加载文件...</p>
    </div>

    <!-- 错误状态 -->
    <div v-if="error" class="error-state">
      <Icon type="alert-circle" size="lg" />
      <p>文件加载失败</p>
      <p class="error-message">{{ error }}</p>
      <button class="retry-btn" @click="retryLoading">
        重试
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick, onMounted } from 'vue';
import Icon from '../ChatUI/Icon.vue';

interface FileItem {
  name: string;
  path: string;
  size: number;
  modified: number;
  content?: string;
  imageData?: string;
}

interface ImageInfo {
  width: number;
  height: number;
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
  fileSelected: [file: FileItem];
  fileEdit: [file: FileItem, content: string];
}>();

// 响应式数据
const currentFile = ref<FileItem | null>(null);
const fileContent = ref('');
const imageData = ref('');
const pdfData = ref('');
const recentFiles = ref<FileItem[]>([]);
const isLoading = ref(false);
const error = ref('');
const isReadonly = ref(true);
const isDragOver = ref(false);
const lineWrapMode = ref<'wrap' | 'nowrap'>('wrap');
const fontSize = ref<'text-xs' | 'text-sm' | 'text-base' | 'text-lg' | 'text-xl'>('text-sm');

const fileInput = ref<HTMLInputElement>();
const codeElement = ref<HTMLElement>();
const websocket = ref<WebSocket | null>(null);
const imageInfo = ref<ImageInfo | null>(null);

// 计算属性
const contentLines = computed(() => {
  return fileContent.value.split('\n');
});

const fileSize = computed(() => {
  if (!currentFile.value?.size) return '';
  const bytes = currentFile.value.size;
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
});

const fileModified = computed(() => {
  if (!currentFile.value?.modified) return '';
  const date = new Date(currentFile.value.modified);
  return date.toLocaleDateString('zh-CN');
});

const fileType = computed(() => {
  if (!currentFile.value?.name) return '';
  const extension = currentFile.value.name.split('.').pop()?.toLowerCase();
  return extension?.toUpperCase() || '';
});

const isImageFile = computed(() => {
  if (!currentFile.value?.name) return false;
  const extension = currentFile.value.name.split('.').pop()?.toLowerCase();
  return ['jpg', 'jpeg', 'png', 'gif', 'bmp', 'svg', 'webp'].includes(extension || '');
});

const isPdfFile = computed(() => {
  if (!currentFile.value?.name) return false;
  const extension = currentFile.value.name.split('.').pop()?.toLowerCase();
  return extension === 'pdf';
});

const isTextFile = computed(() => {
  if (!currentFile.value?.name) return false;
  const extension = currentFile.value.name.split('.').pop()?.toLowerCase();
  const textExtensions = [
    'txt', 'md', 'json', 'js', 'ts', 'jsx', 'tsx', 'html', 'css', 'scss',
    'py', 'java', 'cpp', 'c', 'h', 'hpp', 'go', 'rs', 'sql', 'sh', 'bash',
    'yml', 'yaml', 'xml', 'csv', 'log', 'vue', 'svelte', 'php', 'rb',
    'swift', 'kt', 'scala', 'r', 'm', 'pl', 'lua', 'dockerfile'
  ];
  return textExtensions.includes(extension || '');
});

// WebSocket 连接
const connectWebSocket = () => {
  try {
    websocket.value = new WebSocket(`${props.wsUrl}?session=${props.sessionId}`);

    websocket.value.onopen = () => {
      console.log('ReadTool WebSocket connected');
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
      console.log('ReadTool WebSocket disconnected');
      setTimeout(connectWebSocket, 5000);
    };

    websocket.value.onerror = (error) => {
      console.error('ReadTool WebSocket error:', error);
    };
  } catch (error) {
    console.error('Failed to connect WebSocket:', error);
  }
};

const handleWebSocketMessage = (message: any) => {
  switch (message.type) {
    case 'file_content':
      if (message.file && message.content !== undefined) {
        fileContent.value = message.content;
        currentFile.value = { ...message.file, content: message.content };
        if (currentFile.value) {
          addToRecentFiles(currentFile.value);
        }
        isLoading.value = false;
        error.value = '';
      }
      break;
    case 'file_image':
      if (message.file && message.imageData) {
        imageData.value = message.imageData;
        currentFile.value = { ...message.file, imageData: message.imageData };
        if (currentFile.value) {
          addToRecentFiles(currentFile.value);
        }
        isLoading.value = false;
        error.value = '';
      }
      break;
    case 'file_error':
      error.value = message.error || '文件加载失败';
      isLoading.value = false;
      break;
  }
};

const sendWebSocketMessage = (message: any) => {
  if (websocket.value && websocket.value.readyState === WebSocket.OPEN) {
    websocket.value.send(JSON.stringify(message));
  }
};

// 文件操作方法
const openFile = (file: FileItem) => {
  currentFile.value = file;
  isLoading.value = true;
  error.value = '';

  if (isImageFile.value) {
    sendWebSocketMessage({
      type: 'read_image',
      path: file.path,
    });
  } else if (isTextFile.value) {
    sendWebSocketMessage({
      type: 'read_file',
      path: file.path,
    });
  } else {
    // 非文本文件，设置为二进制模式
    isLoading.value = false;
  }
};

const reloadFile = () => {
  if (currentFile.value) {
    openFile(currentFile.value);
  }
};

const closeFile = () => {
  currentFile.value = null;
  fileContent.value = '';
  imageData.value = '';
  pdfData.value = '';
  error.value = '';
  imageInfo.value = null;
};

const toggleReadonly = () => {
  isReadonly.value = !isReadonly.value;
};

const downloadFile = () => {
  if (currentFile.value) {
    sendWebSocketMessage({
      type: 'download_file',
      path: currentFile.value.path,
    });
  }
};

// 拖拽处理
const handleDragOver = () => {
  isDragOver.value = true;
};

const handleDragLeave = () => {
  isDragOver.value = false;
};

const handleDrop = (event: DragEvent) => {
  isDragOver.value = false;
  const files = event.dataTransfer?.files;
  if (files && files.length > 0) {
    handleFiles(files);
  }
};

const handleFileSelect = (event: Event) => {
  const target = event.target as HTMLInputElement;
  const files = target.files;
  if (files && files.length > 0) {
    handleFiles(files);
  }
};

const handleFiles = (files: FileList) => {
  Array.from(files).forEach(file => {
    const fileItem: FileItem = {
      name: file.name,
      path: file.name, // 本地文件使用文件名作为路径
      size: file.size,
      modified: Date.now(),
    };

    if (file.type.startsWith('image/')) {
      const reader = new FileReader();
      reader.onload = (e) => {
        const result = e.target?.result as string;
        imageData.value = result;
        currentFile.value = { ...fileItem, imageData: result };
        addToRecentFiles(currentFile.value);
      };
      reader.readAsDataURL(file);
    } else if (file.type.startsWith('text/') || isTextFileName(file.name)) {
      const reader = new FileReader();
      reader.onload = (e) => {
        const result = e.target?.result as string;
        fileContent.value = result;
        currentFile.value = { ...fileItem, content: result };
        addToRecentFiles(currentFile.value);
      };
      reader.readAsText(file);
    }
  });
};

const isTextFileName = (fileName: string): boolean => {
  const extension = fileName.split('.').pop()?.toLowerCase();
  const textExtensions = [
    'txt', 'md', 'json', 'js', 'ts', 'jsx', 'tsx', 'html', 'css', 'scss',
    'py', 'java', 'cpp', 'c', 'h', 'hpp', 'go', 'rs', 'sql', 'sh', 'bash',
    'yml', 'yaml', 'xml', 'csv', 'log', 'vue', 'svelte', 'php', 'rb'
  ];
  return textExtensions.includes(extension || '');
};

// 图片处理
const handleImageLoad = (event: Event) => {
  const img = event.target as HTMLImageElement;
  imageInfo.value = {
    width: img.naturalWidth,
    height: img.naturalHeight,
  };
};

const handleImageError = () => {
  error.value = '图片加载失败';
  isLoading.value = false;
};

// 编辑处理
const handleContentEdit = () => {
  if (codeElement.value && currentFile.value && !isReadonly.value) {
    const newContent = codeElement.value.textContent || '';
    if (newContent !== fileContent.value) {
      fileContent.value = newContent;
      emit('fileEdit', { ...currentFile.value, content: newContent }, newContent);
    }
  }
};

const handleKeydown = (event: KeyboardEvent) => {
  if (isReadonly.value) {
    event.preventDefault();
    return;
  }

  // 支持Tab缩进
  if (event.key === 'Tab') {
    event.preventDefault();
    const selection = window.getSelection();
    if (selection && selection.rangeCount > 0) {
      const range = selection.getRangeAt(0);
      const tabNode = document.createTextNode('  ');
      range.insertNode(tabNode);
      range.collapse(false);
      selection.removeAllRanges();
      selection.addRange(range);
    }
  }
};

// 工具方法
const copyContent = () => {
  if (fileContent.value) {
    navigator.clipboard.writeText(fileContent.value);
  }
};

const selectAll = () => {
  if (codeElement.value) {
    const range = document.createRange();
    range.selectNodeContents(codeElement.value);
    const selection = window.getSelection();
    selection?.removeAllRanges();
    selection?.addRange(range);
  }
};

const getIconForFile = (file: FileItem) => {
  const extension = file.name.split('.').pop()?.toLowerCase();

  if (['jpg', 'jpeg', 'png', 'gif', 'bmp', 'svg', 'webp'].includes(extension || '')) {
    return 'image';
  }
  if (['pdf'].includes(extension || '')) {
    return 'file-pdf';
  }
  if (['zip', 'rar', '7z', 'tar', 'gz'].includes(extension || '')) {
    return 'archive';
  }
  if (['txt', 'md', 'json', 'js', 'ts', 'jsx', 'tsx', 'html', 'css', 'py', 'java', 'cpp', 'c', 'go', 'rs', 'sql'].includes(extension || '')) {
    return 'file-text';
  }

  return 'file';
};

const addToRecentFiles = (file: FileItem) => {
  const existingIndex = recentFiles.value.findIndex(f => f.path === file.path);
  if (existingIndex !== -1) {
    recentFiles.value[existingIndex] = file;
  } else {
    recentFiles.value.unshift(file);
    if (recentFiles.value.length > 10) {
      recentFiles.value = recentFiles.value.slice(0, 10);
    }
  }
};

const retryLoading = () => {
  if (currentFile.value) {
    reloadFile();
  }
};

// 生命周期
onMounted(() => {
  connectWebSocket();
});

// 监听文件变化
watch(currentFile, (newFile) => {
  if (newFile) {
    emit('fileSelected', newFile);
  }
});
</script>

<style scoped>
.read-tool {
  @apply flex flex-col h-full bg-surface dark:bg-surface-dark border border-border dark:border-border-dark rounded-lg;
}

.read-header {
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

.file-selection {
  @apply flex-1 flex flex-col items-center justify-center p-8;
}

.selection-hint {
  @apply flex flex-col items-center text-gray-400 dark:text-gray-500 mb-8;
}

.selection-hint p {
  @apply mt-2 text-lg font-medium;
}

.hint-text {
  @apply text-sm mt-1;
}

.recent-files {
  @apply w-full max-w-md mb-8;
}

.recent-files h4 {
  @apply text-sm font-semibold text-text dark:text-text-dark mb-3;
}

.recent-list {
  @apply space-y-2;
}

.recent-item {
  @apply flex items-center gap-3 p-3 bg-gray-50 dark:bg-gray-700/50 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-600/50 cursor-pointer transition-colors;
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

.drop-zone {
  @apply w-full max-w-md p-8 border-2 border-dashed border-gray-300 dark:border-gray-600 rounded-lg text-center hover:border-blue-400 dark:hover:border-blue-500 transition-colors cursor-pointer;
}

.drop-zone.drag-over {
  @apply border-blue-500 bg-blue-50 dark:bg-blue-900/20;
}

.drop-zone p {
  @apply mt-2 text-gray-600 dark:text-gray-400;
}

.drop-hint {
  @apply text-sm mt-1;
}

.file-input {
  @apply hidden;
}

.file-content {
  @apply flex-1 flex flex-col overflow-hidden;
}

.file-info-bar {
  @apply flex items-center justify-between px-4 py-2 border-b border-gray-100 dark:border-gray-700 bg-gray-50 dark:bg-gray-700/30;
}

.file-info {
  @apply flex items-center gap-3 flex-1 min-w-0;
}

.file-details {
  @apply flex-1 min-w-0;
}

.file-stats {
  @apply flex items-center gap-3 text-xs text-gray-500 dark:text-gray-400;
}

.image-viewer {
  @apply flex-1 flex flex-col items-center justify-center p-4;
}

.file-image {
  @apply max-w-full max-h-full object-contain rounded shadow-lg;
}

.image-info {
  @apply mt-4 text-xs text-gray-500 dark:text-gray-400;
}

.pdf-viewer {
  @apply flex-1 flex flex-col;
}

.pdf-frame {
  @apply flex-1 w-full border-0;
}

.pdf-placeholder {
  @apply flex-1 flex flex-col items-center justify-center text-gray-400 dark:text-gray-500;
}

.text-viewer {
  @apply flex-1 flex flex-col overflow-hidden;
}

.viewer-toolbar {
  @apply flex items-center justify-between px-4 py-2 border-b border-gray-100 dark:border-gray-700 bg-gray-50 dark:bg-gray-700/30;
}

.toolbar-left {
  @apply flex items-center gap-2;
}

.wrap-select, .font-select {
  @apply px-2 py-1 text-sm border border-border dark:border-border-dark rounded focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-800 dark:text-white;
}

.toolbar-right {
  @apply flex gap-2;
}

.toolbar-btn {
  @apply flex items-center gap-1 px-2 py-1 text-sm text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-600 rounded transition-colors;
}

.code-container {
  @apply flex-1 flex overflow-hidden;
}

.code-container.no-wrap {
  @apply overflow-x-auto;
}

.line-numbers {
  @apply w-16 bg-gray-100 dark:bg-gray-800 border-r border-gray-200 dark:border-gray-600 flex-shrink-0;
}

.line-number {
  @apply px-2 py-1 text-right text-xs text-gray-400 dark:text-gray-500 leading-6;
}

.code-content {
  @apply flex-1 overflow-hidden;
}

.code-block {
  @apply w-full h-full px-3 py-2 font-mono text-sm bg-white dark:bg-gray-900 text-gray-800 dark:text-gray-200 leading-6 overflow-auto focus:outline-none focus:ring-2 focus:ring-blue-500;
}

.code-block.readonly {
  @apply cursor-text;
}

.binary-viewer {
  @apply flex-1 flex items-center justify-center;
}

.binary-placeholder {
  @apply flex flex-col items-center text-gray-400 dark:text-gray-500;
}

.binary-hint {
  @apply text-sm mt-1;
}

.download-btn {
  @apply mt-4 px-4 py-2 bg-blue-500 hover:bg-blue-600 text-white rounded transition-colors;
}

.loading-state {
  @apply flex flex-col items-center justify-center py-8 text-gray-400 dark:text-gray-500;
}

.error-state {
  @apply flex flex-col items-center justify-center py-8 text-red-400 dark:text-red-500;
}

.error-message {
  @apply text-sm mt-1;
}

.retry-btn {
  @apply mt-4 px-4 py-2 bg-red-500 hover:bg-red-600 text-white rounded transition-colors;
}

.animate-spin {
  @apply animate-spin;
}
</style>