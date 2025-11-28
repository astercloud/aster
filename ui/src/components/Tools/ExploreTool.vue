<template>
  <div class="explore-tool">
    <!-- 头部工具栏 -->
    <div class="explore-header">
      <div class="header-title">
        <Icon type="folder" size="sm" />
        <span>文件浏览器</span>
      </div>
      <div class="header-actions">
        <button
          class="action-button"
          title="刷新"
          @click="refreshCurrentPath"
        >
          <Icon type="refresh" size="sm" />
        </button>
        <button
          class="action-button"
          title="返回上级"
          :disabled="currentPath === '/'"
          @click="goToParent"
        >
          <Icon type="arrow-up" size="sm" />
        </button>
        <button
          class="action-button"
          title="切换视图"
          @click="toggleViewMode"
        >
          <Icon :type="viewMode === 'list' ? 'grid' : 'list'" size="sm" />
        </button>
        <button
          class="action-button"
          title="显示隐藏文件"
          :class="{ 'text-blue-500': showHiddenFiles }"
          @click="toggleHiddenFiles"
        >
          <Icon type="eye" size="sm" />
        </button>
      </div>
    </div>

    <!-- 路径导航 -->
    <div class="path-navigation">
      <div class="breadcrumb">
        <button
          class="breadcrumb-item"
          @click="navigateToPath('/')"
        >
          <Icon type="home" size="xs" />
        </button>
        <template v-for="(segment, index) in pathSegments" :key="index">
          <span class="breadcrumb-separator">/</span>
          <button
            class="breadcrumb-item"
            @click="navigateToSegment(index)"
          >
            {{ segment }}
          </button>
        </template>
      </div>
      <div class="path-actions">
        <input
          v-model="newFolderName"
          type="text"
          placeholder="新建文件夹名称..."
          class="folder-input"
          @keydown.enter="createFolder"
          @keydown.esc="newFolderName = ''"
        />
        <button
          class="create-folder-btn"
          :disabled="!newFolderName.trim()"
          @click="createFolder"
        >
          <Icon type="plus" size="sm" />
          新建文件夹
        </button>
      </div>
    </div>

    <!-- 文件列表头部 -->
    <div class="list-header">
      <div class="header-sort">
        <select v-model="sortBy" class="sort-select">
          <option value="name">按名称</option>
          <option value="size">按大小</option>
          <option value="modified">按修改时间</option>
          <option value="type">按类型</option>
        </select>
        <button
          class="sort-order-btn"
          @click="toggleSortOrder"
        >
          <Icon :type="sortOrder === 'asc' ? 'arrow-up' : 'arrow-down'" size="sm" />
        </button>
      </div>
      <div class="header-info">
        <span class="item-count">{{ sortedItems.length }} 项</span>
        <span v-if="selectedItems.length > 0" class="selected-count">
          已选择 {{ selectedItems.length }} 项
        </span>
      </div>
    </div>

    <!-- 文件列表 -->
    <div class="file-list" ref="fileListRef">
      <!-- 父级目录链接 -->
      <div
        v-if="currentPath !== '/'"
        class="file-item directory-item"
        @click="goToParent"
      >
        <div class="file-icon">
          <Icon type="arrow-left" size="sm" />
        </div>
        <div class="file-info">
          <div class="file-name">..</div>
          <div class="file-meta">返回上级目录</div>
        </div>
        <div class="file-size">-</div>
        <div class="file-modified">-</div>
      </div>

      <!-- 文件和文件夹 -->
      <div
        v-for="item in sortedItems"
        :key="item.path"
        :class="[
          'file-item',
          {
            'directory-item': item.isDirectory,
            'file-selected': selectedItems.includes(item.path)
          }
        ]"
        @click="handleItemClick(item)"
        @contextmenu.prevent="showContextMenu(item, $event)"
      >
        <div class="file-checkbox" v-if="selectionMode">
          <input
            type="checkbox"
            :checked="selectedItems.includes(item.path)"
            @change.stop="toggleSelection(item.path)"
          />
        </div>

        <div class="file-icon">
          <Icon
            :type="getIconForItem(item)"
            :size="item.isDirectory ? 'sm' : 'xs'"
            :class="getIconClassForItem(item)"
          />
        </div>

        <div class="file-info">
          <div class="file-name">{{ item.name }}</div>
          <div class="file-meta">
            <span v-if="!item.isDirectory" class="file-extension">
              {{ getFileExtension(item.name) }}
            </span>
            <span v-if="item.isDirectory" class="file-count">
              {{ item.children?.length || 0 }} 项
            </span>
          </div>
        </div>

        <div class="file-size">
          {{ item.isDirectory ? '-' : formatFileSize(item.size) }}
        </div>

        <div class="file-modified">
          {{ formatDate(item.modified) }}
        </div>

        <div class="file-actions">
          <button
            v-if="!item.isDirectory"
            class="action-btn view-btn"
            title="查看文件"
            @click.stop="viewFile(item)"
          >
            <Icon type="eye" size="xs" />
          </button>
          <button
            v-if="!item.isDirectory"
            class="action-btn edit-btn"
            title="编辑文件"
            @click.stop="editFile(item)"
          >
            <Icon type="edit" size="xs" />
          </button>
          <button
            class="action-btn download-btn"
            :title="item.isDirectory ? '下载文件夹' : '下载文件'"
            @click.stop="downloadItem(item)"
          >
            <Icon type="download" size="xs" />
          </button>
          <button
            class="action-btn delete-btn"
            title="删除"
            @click.stop="deleteItem(item)"
          >
            <Icon type="trash" size="xs" />
          </button>
        </div>
      </div>

      <!-- 空状态 -->
      <div v-if="sortedItems.length === 0" class="empty-state">
        <Icon type="folder-open" size="lg" />
        <p>此目录为空</p>
      </div>
    </div>

    <!-- 状态栏 -->
    <div class="status-bar">
      <div class="status-info">
        <span class="current-path">{{ currentPath }}</span>
        <span class="total-size">总大小: {{ formatTotalSize() }}</span>
      </div>
      <div class="status-actions">
        <button
          class="selection-toggle-btn"
          :class="{ active: selectionMode }"
          @click="toggleSelectionMode"
        >
          <Icon type="check-square" size="sm" />
          选择模式
        </button>
        <button
          v-if="selectedItems.length > 0"
          class="clear-selection-btn"
          @click="clearSelection"
        >
          清除选择
        </button>
      </div>
    </div>

    <!-- 右键菜单 -->
    <div
      v-if="contextMenu.visible"
      class="context-menu"
      :style="{ left: contextMenu.x + 'px', top: contextMenu.y + 'px' }"
      @click="hideContextMenu"
    >
      <div class="context-menu-item" @click="copyItem(contextMenu.item!)">
        <Icon type="copy" size="xs" />
        复制
      </div>
      <div class="context-menu-item" @click="cutItem(contextMenu.item!)">
        <Icon type="scissors" size="xs" />
        剪切
      </div>
      <div class="context-menu-item" @click="pasteItem" :disabled="!clipboard.item">
        <Icon type="clipboard" size="xs" />
        粘贴
      </div>
      <div class="context-menu-divider"></div>
      <div
        v-if="contextMenu.item?.isDirectory"
        class="context-menu-item"
        @click="renameItem(contextMenu.item!)"
      >
        <Icon type="edit" size="xs" />
        重命名
      </div>
      <div class="context-menu-item danger" @click="deleteItem(contextMenu.item!)">
        <Icon type="trash" size="xs" />
        删除
      </div>
    </div>

    <!-- 文件预览模态框 -->
    <div v-if="showPreviewModal" class="modal-overlay" @click.self="closePreview">
      <div class="modal-content">
        <div class="modal-header">
          <h3>{{ previewFile?.name }}</h3>
          <button @click="closePreview">
            <Icon type="close" size="sm" />
          </button>
        </div>
        <div class="modal-body">
          <div v-if="previewFile" class="file-preview">
            <!-- 图片预览 -->
            <img
              v-if="isImageFile(previewFile)"
              :src="previewUrl"
              :alt="previewFile.name"
              class="preview-image"
            />

            <!-- 文本预览 -->
            <pre
              v-else-if="isTextFile(previewFile)"
              class="preview-text"
            >{{ previewContent }}</pre>

            <!-- 其他文件类型 -->
            <div v-else class="preview-info">
              <Icon :type="getIconForItem(previewFile)" size="lg" />
              <p>{{ getFileTypeText(previewFile) }}</p>
              <p>大小: {{ formatFileSize(previewFile.size) }}</p>
              <p>修改时间: {{ formatDate(previewFile.modified) }}</p>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick, onMounted, onUnmounted } from 'vue';
import Icon from '../ChatUI/Icon.vue';

interface FileItem {
  name: string;
  path: string;
  isDirectory: boolean;
  size: number;
  modified: number;
  children?: FileItem[];
  extension?: string;
  mimeType?: string;
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
  fileOpened: [file: FileItem];
}>();

// 响应式数据
const currentPath = ref('/');
const items = ref<FileItem[]>([]);
const sortBy = ref('name');
const sortOrder = ref<'asc' | 'desc'>('asc');
const viewMode = ref<'list' | 'grid'>('list');
const showHiddenFiles = ref(false);
const selectionMode = ref(false);
const selectedItems = ref<string[]>([]);
const showPreviewModal = ref(false);
const previewFile = ref<FileItem | null>(null);
const previewContent = ref('');
const previewUrl = ref('');
const newFolderName = ref('');

// 右键菜单
const contextMenu = ref({
  visible: false,
  x: 0,
  y: 0,
  item: null as FileItem | null,
});

// 剪贴板
const clipboard = ref({
  action: 'copy' as 'copy' | 'cut',
  item: null as FileItem | null,
});

const fileListRef = ref<HTMLElement>();
const websocket = ref<WebSocket | null>(null);

// 计算属性
const pathSegments = computed(() => {
  return currentPath.value.split('/').filter(segment => segment.length > 0);
});

const sortedItems = computed(() => {
  let filtered = items.value;

  // 过滤隐藏文件
  if (!showHiddenFiles.value) {
    filtered = filtered.filter(item => !item.name.startsWith('.'));
  }

  // 目录优先排序
  filtered.sort((a, b) => {
    if (a.isDirectory && !b.isDirectory) return -1;
    if (!a.isDirectory && b.isDirectory) return 1;
    return 0;
  });

  // 按选择字段排序
  return filtered.sort((a, b) => {
    let aValue: any = a[sortBy.value as keyof FileItem];
    let bValue: any = b[sortBy.value as keyof FileItem];

    if (typeof aValue === 'string') {
      aValue = aValue.toLowerCase();
      bValue = (bValue as string).toLowerCase();
    }

    if (sortOrder.value === 'asc') {
      return aValue > bValue ? 1 : -1;
    } else {
      return aValue < bValue ? 1 : -1;
    }
  });
});

// WebSocket 连接
const connectWebSocket = () => {
  try {
    websocket.value = new WebSocket(`${props.wsUrl}?session=${props.sessionId}`);

    websocket.value.onopen = () => {
      console.log('ExploreTool WebSocket connected');
      loadDirectory(currentPath.value);
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
      console.log('ExploreTool WebSocket disconnected');
      setTimeout(connectWebSocket, 5000);
    };

    websocket.value.onerror = (error) => {
      console.error('ExploreTool WebSocket error:', error);
    };
  } catch (error) {
    console.error('Failed to connect WebSocket:', error);
  }
};

const handleWebSocketMessage = (message: any) => {
  switch (message.type) {
    case 'directory_response':
      items.value = message.items || [];
      break;
    case 'file_content_response':
      if (previewFile.value) {
        previewContent.value = message.content || '';
      }
      break;
    case 'file_preview_response':
      if (message.url) {
        previewUrl.value = message.url;
      }
      break;
    case 'operation_completed':
      loadDirectory(currentPath.value);
      break;
  }
};

const sendWebSocketMessage = (message: any) => {
  if (websocket.value && websocket.value.readyState === WebSocket.OPEN) {
    websocket.value.send(JSON.stringify(message));
  }
};

// 文件操作方法
const loadDirectory = (path: string) => {
  currentPath.value = path;
  sendWebSocketMessage({
    type: 'directory_request',
    path,
  });
};

const refreshCurrentPath = () => {
  loadDirectory(currentPath.value);
};

const goToParent = () => {
  const parentPath = currentPath.value.split('/').slice(0, -1).join('/') || '/';
  loadDirectory(parentPath);
};

const navigateToPath = (path: string) => {
  loadDirectory(path);
};

const navigateToSegment = (index: number) => {
  const segments = pathSegments.value.slice(0, index + 1);
  const path = '/' + segments.join('/');
  loadDirectory(path);
};

const handleItemClick = (item: FileItem) => {
  if (item.isDirectory) {
    loadDirectory(item.path);
  } else {
    emit('fileSelected', item);
    viewFile(item);
  }
};

const viewFile = (file: FileItem) => {
  previewFile.value = file;
  showPreviewModal.value = true;

  // 请求文件内容或预览
  if (isImageFile(file)) {
    sendWebSocketMessage({
      type: 'file_preview_request',
      path: file.path,
    });
  } else if (isTextFile(file)) {
    sendWebSocketMessage({
      type: 'file_content_request',
      path: file.path,
    });
  }
};

const editFile = (file: FileItem) => {
  emit('fileOpened', file);
};

const downloadItem = (item: FileItem) => {
  sendWebSocketMessage({
    type: 'download_request',
    path: item.path,
    isDirectory: item.isDirectory,
  });
};

const deleteItem = (item: FileItem) => {
  if (confirm(`确定要${item.isDirectory ? '删除文件夹' : '删除文件'} "${item.name}" 吗？`)) {
    sendWebSocketMessage({
      type: 'delete_request',
      path: item.path,
      isDirectory: item.isDirectory,
    });
  }
};

const createFolder = () => {
  if (!newFolderName.value.trim()) return;

  const folderPath = currentPath.value + '/' + newFolderName.value.trim();

  sendWebSocketMessage({
    type: 'create_folder_request',
    path: folderPath,
  });

  newFolderName.value = '';
};

const renameItem = (item: FileItem) => {
  const newName = prompt('请输入新名称:', item.name);
  if (newName && newName.trim() && newName.trim() !== item.name) {
    sendWebSocketMessage({
      type: 'rename_request',
      oldPath: item.path,
      newName: newName.trim(),
    });
  }
};

// 选择模式相关
const toggleSelectionMode = () => {
  selectionMode.value = !selectionMode.value;
  if (!selectionMode.value) {
    selectedItems.value = [];
  }
};

const toggleSelection = (path: string) => {
  const index = selectedItems.value.indexOf(path);
  if (index > -1) {
    selectedItems.value.splice(index, 1);
  } else {
    selectedItems.value.push(path);
  }
};

const clearSelection = () => {
  selectedItems.value = [];
};

// 剪贴板操作
const copyItem = (item: FileItem) => {
  clipboard.value = { action: 'copy', item };
  hideContextMenu();
};

const cutItem = (item: FileItem) => {
  clipboard.value = { action: 'cut', item };
  hideContextMenu();
};

const pasteItem = () => {
  if (!clipboard.value.item) return;

  const targetPath = currentPath.value + '/' + clipboard.value.item.name;

  sendWebSocketMessage({
    type: clipboard.value.action === 'copy' ? 'copy_request' : 'move_request',
    sourcePath: clipboard.value.item.path,
    targetPath,
  });

  if (clipboard.value.action === 'cut') {
    clipboard.value.item = null;
  }
  hideContextMenu();
};

// UI 相关方法
const toggleViewMode = () => {
  viewMode.value = viewMode.value === 'list' ? 'grid' : 'list';
};

const toggleSortOrder = () => {
  sortOrder.value = sortOrder.value === 'asc' ? 'desc' : 'asc';
};

const toggleHiddenFiles = () => {
  showHiddenFiles.value = !showHiddenFiles.value;
};

const closePreview = () => {
  showPreviewModal.value = false;
  previewFile.value = null;
  previewContent.value = '';
  previewUrl.value = '';
};

// 右键菜单
const showContextMenu = (item: FileItem, event: MouseEvent) => {
  contextMenu.value = {
    visible: true,
    x: event.clientX,
    y: event.clientY,
    item,
  };

  // 监听点击事件来隐藏菜单
  nextTick(() => {
    document.addEventListener('click', hideContextMenu, { once: true });
  });
};

const hideContextMenu = () => {
  contextMenu.value.visible = false;
};

// 工具方法
const getIconForItem = (item: FileItem) => {
  if (item.isDirectory) return 'folder';
  if (isImageFile(item)) return 'image';
  if (isTextFile(item)) return 'file-text';
  if (item.extension === 'pdf') return 'file-pdf';
  if (['zip', 'rar', '7z', 'tar', 'gz'].includes(item.extension || '')) return 'archive';
  return 'file';
};

const getIconClassForItem = (item: FileItem) => {
  if (item.isDirectory) return 'text-blue-500';
  if (isImageFile(item)) return 'text-green-500';
  if (isTextFile(item)) return 'text-gray-500';
  return 'text-gray-400';
};

const getFileExtension = (filename: string) => {
  const parts = filename.split('.');
  return parts.length > 1 ? (parts[parts.length - 1] ?? '').toLowerCase() : '';
};

const isImageFile = (item: FileItem) => {
  const imageExtensions = ['jpg', 'jpeg', 'png', 'gif', 'bmp', 'svg', 'webp'];
  return imageExtensions.includes(item.extension || '');
};

const isTextFile = (item: FileItem) => {
  const textExtensions = ['txt', 'md', 'json', 'js', 'ts', 'html', 'css', 'py', 'java', 'cpp', 'c', 'go', 'rs', 'sql', 'xml', 'yaml', 'yml'];
  return textExtensions.includes(item.extension || '');
};

const getFileTypeText = (item: FileItem) => {
  if (item.isDirectory) return '文件夹';
  if (isImageFile(item)) return '图片文件';
  if (isTextFile(item)) return '文本文件';
  return getFileExtension(item.name).toUpperCase() + ' 文件';
};

const formatFileSize = (bytes: number) => {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
};

const formatTotalSize = () => {
  const totalSize = items.value
    .filter(item => !item.isDirectory)
    .reduce((sum, item) => sum + item.size, 0);
  return formatFileSize(totalSize);
};

const formatDate = (timestamp: number) => {
  const date = new Date(timestamp);
  return date.toLocaleDateString('zh-CN', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  });
};

// 生命周期
onMounted(() => {
  connectWebSocket();
});

onUnmounted(() => {
  document.removeEventListener('click', hideContextMenu);
});
</script>

<style scoped>
.explore-tool {
  @apply flex flex-col h-full bg-surface dark:bg-surface-dark border border-border dark:border-border-dark rounded-lg;
}

.explore-header {
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

.path-navigation {
  @apply flex items-center justify-between px-4 py-2 border-b border-border dark:border-border-dark bg-gray-50 dark:bg-gray-700/50;
}

.breadcrumb {
  @apply flex items-center gap-1;
}

.breadcrumb-item {
  @apply flex items-center gap-1 px-2 py-1 text-sm text-blue-600 dark:text-blue-400 hover:bg-blue-50 dark:hover:bg-blue-900/20 rounded transition-colors;
}

.breadcrumb-separator {
  @apply text-gray-400 dark:text-gray-500;
}

.path-actions {
  @apply flex items-center gap-2;
}

.folder-input {
  @apply px-2 py-1 text-sm border border-border dark:border-border-dark rounded focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-800 dark:text-white;
}

.create-folder-btn {
  @apply flex items-center gap-1 px-2 py-1 text-sm bg-green-500 hover:bg-green-600 text-white rounded transition-colors disabled:opacity-50 disabled:cursor-not-allowed;
}

.list-header {
  @apply flex items-center justify-between px-4 py-2 border-b border-border dark:border-border-dark bg-surface dark:bg-surface-dark;
}

.header-sort {
  @apply flex items-center gap-2;
}

.sort-select {
  @apply px-2 py-1 text-sm border border-border dark:border-border-dark rounded focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-800 dark:text-white;
}

.sort-order-btn {
  @apply p-1 text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 rounded transition-colors;
}

.header-info {
  @apply flex items-center gap-3 text-sm text-gray-500 dark:text-gray-400;
}

.selected-count {
  @apply text-blue-500 dark:text-blue-400;
}

.file-list {
  @apply flex-1 overflow-y-auto;
}

.file-item {
  @apply flex items-center gap-3 px-4 py-2 hover:bg-gray-50 dark:hover:bg-gray-700/50 cursor-pointer border-b border-gray-100 dark:border-gray-700;
}

.file-item:last-child {
  @apply border-b-0;
}

.directory-item {
  @apply font-medium;
}

.file-selected {
  @apply bg-blue-50 dark:bg-blue-900/20;
}

.file-checkbox {
  @apply flex-shrink-0;
}

.file-icon {
  @apply flex-shrink-0 w-8 h-8 flex items-center justify-center;
}

.file-info {
  @apply flex-1 min-w-0;
}

.file-name {
  @apply truncate font-medium text-text dark:text-text-dark;
}

.file-meta {
  @apply text-xs text-gray-500 dark:text-gray-400;
}

.file-size {
  @apply text-sm text-gray-600 dark:text-gray-300 w-20 text-right;
}

.file-modified {
  @apply text-sm text-gray-600 dark:text-gray-300 w-32 text-right;
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

.delete-btn:hover {
  @apply text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20;
}

.empty-state {
  @apply flex flex-col items-center justify-center py-8 text-gray-400 dark:text-gray-500;
}

.status-bar {
  @apply flex items-center justify-between px-4 py-2 border-t border-border dark:border-border-dark bg-surface dark:bg-surface-dark;
}

.status-info {
  @apply flex items-center gap-3 text-xs text-gray-500 dark:text-gray-400;
}

.status-actions {
  @apply flex items-center gap-2;
}

.selection-toggle-btn {
  @apply flex items-center gap-1 px-2 py-1 text-sm border border-border dark:border-border-dark rounded transition-colors hover:bg-gray-100 dark:hover:bg-gray-700;
}

.selection-toggle-btn.active {
  @apply bg-blue-500 text-white border-blue-500;
}

.clear-selection-btn {
  @apply text-sm text-blue-500 dark:text-blue-400 hover:text-blue-600 dark:hover:text-blue-300;
}

/* 右键菜单 */
.context-menu {
  @apply fixed bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-600 rounded shadow-lg py-1 z-50 min-w-32;
}

.context-menu-item {
  @apply flex items-center gap-2 px-3 py-2 text-sm text-gray-700 dark:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700 cursor-pointer;
}

.context-menu-item.danger {
  @apply text-red-600 dark:text-red-400;
}

.context-menu-item:disabled {
  @apply text-gray-400 dark:text-gray-500 cursor-not-allowed;
}

.context-menu-divider {
  @apply h-px bg-gray-200 dark:bg-gray-600 my-1;
}

/* 模态框 */
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

.file-preview {
  @apply h-full;
}

.preview-image {
  @apply max-w-full h-auto object-contain;
}

.preview-text {
  @apply bg-gray-100 dark:bg-gray-700 p-4 rounded text-sm font-mono overflow-x-auto whitespace-pre-wrap;
  max-height: 60vh;
}

.preview-info {
  @apply flex flex-col items-center justify-center h-full text-gray-500 dark:text-gray-400;
}
</style>