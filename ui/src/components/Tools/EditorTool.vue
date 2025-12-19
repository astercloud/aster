<template>
  <div class="editor-tool">
    <!-- 头部工具栏 -->
    <div class="editor-header">
      <div class="header-left">
        <div class="header-title">
          <Icon type="edit" size="sm" />
          <span>代码编辑器</span>
        </div>
        <div v-if="currentFile" class="file-indicator">
          <Icon :type="getIconForFile(currentFile)" size="xs" />
          <span class="file-name">{{ currentFile.name }}</span>
          <span v-if="hasUnsavedChanges" class="unsaved-indicator">●</span>
        </div>
      </div>
      <div class="header-actions">
        <button
          v-if="currentFile"
          class="action-button"
          title="保存文件"
          :disabled="isSaving"
          @click="saveFile"
        >
          <Icon v-if="isSaving" type="spinner" size="sm" class="animate-spin" />
          <Icon v-else type="save" size="sm" />
        </button>
        <button
          v-if="currentFile"
          class="action-button"
          title="保存并关闭"
          :disabled="isSaving"
          @click="saveAndClose"
        >
          <Icon type="check" size="sm" />
        </button>
        <button
          v-if="currentFile"
          class="action-button"
          title="关闭文件"
          @click="closeFile"
        >
          <Icon type="close" size="sm" />
        </button>
        <button
          class="action-button"
          title="设置"
          @click="toggleSettings"
        >
          <Icon type="settings" size="sm" />
        </button>
      </div>
    </div>

    <!-- 设置面板 -->
    <div v-if="showSettings" class="settings-panel">
      <div class="settings-content">
        <h4>编辑器设置</h4>
        <div class="setting-group">
          <label>字体大小</label>
          <select v-model="editorSettings.fontSize" class="setting-select">
            <option value="text-xs">极小</option>
            <option value="text-sm">小</option>
            <option value="text-base">正常</option>
            <option value="text-lg">大</option>
            <option value="text-xl">极大</option>
          </select>
        </div>
        <div class="setting-group">
          <label>主题</label>
          <select v-model="editorSettings.theme" class="setting-select">
            <option value="light">浅色</option>
            <option value="dark">深色</option>
            <option value="auto">自动</option>
          </select>
        </div>
        <div class="setting-group">
          <label>
            <input
              v-model="editorSettings.wordWrap"
              type="checkbox"
              class="setting-checkbox"
            />
            自动换行
          </label>
        </div>
        <div class="setting-group">
          <label>
            <input
              v-model="editorSettings.lineNumbers"
              type="checkbox"
              class="setting-checkbox"
            />
            显示行号
          </label>
        </div>
        <div class="setting-group">
          <label>
            <input
              v-model="editorSettings.minimap"
              type="checkbox"
              class="setting-checkbox"
            />
            显示缩略图
          </label>
        </div>
        <div class="setting-actions">
          <button class="setting-btn" @click="resetSettings">重置</button>
          <button class="setting-btn primary" @click="saveSettings">保存</button>
        </div>
      </div>
    </div>

    <!-- 文件选择 -->
    <div v-if="!currentFile" class="file-selection">
      <div class="selection-hint">
        <Icon type="edit" size="lg" />
        <p>选择要编辑的文件</p>
        <p class="hint-text">支持各种代码和文本文件格式</p>
      </div>

      <!-- 创建新文件 -->
      <div class="new-file-section">
        <h4>创建新文件</h4>
        <div class="new-file-form">
          <input
            v-model="newFileName"
            type="text"
            placeholder="输入文件名 (如: example.js)"
            class="filename-input"
            @keydown.enter="createNewFile"
          />
          <select v-model="newFileTemplate" class="template-select">
            <option value="">空白文件</option>
            <option value="js">JavaScript</option>
            <option value="ts">TypeScript</option>
            <option value="vue">Vue组件</option>
            <option value="html">HTML</option>
            <option value="css">CSS</option>
            <option value="json">JSON</option>
            <option value="md">Markdown</option>
          </select>
          <button
            class="create-btn"
            :disabled="!newFileName.trim()"
            @click="createNewFile"
          >
            创建
          </button>
        </div>
      </div>

      <!-- 最近编辑的文件 -->
      <div v-if="recentFiles.length > 0" class="recent-files">
        <h4>最近编辑</h4>
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
            <div class="file-meta">
              <span class="edit-time">{{ formatEditTime(file.lastEdit) }}</span>
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
        <p>拖拽文件到此处开始编辑</p>
        <input
          type="file"
          ref="fileInput"
          class="file-input"
          @change="handleFileSelect"
          multiple
        />
      </div>
    </div>

    <!-- 编辑器主体 -->
    <div v-if="currentFile" class="editor-main">
      <!-- 工具栏 -->
      <div class="editor-toolbar">
        <div class="toolbar-left">
          <div class="toolbar-group">
            <button
              class="toolbar-btn"
              title="撤销"
              :disabled="!canUndo"
              @click="undo"
            >
              <Icon type="undo" size="xs" />
            </button>
            <button
              class="toolbar-btn"
              title="重做"
              :disabled="!canRedo"
              @click="redo"
            >
              <Icon type="redo" size="xs" />
            </button>
          </div>
          <div class="toolbar-separator"></div>
          <div class="toolbar-group">
            <button
              class="toolbar-btn"
              title="查找"
              @click="toggleFind"
            >
              <Icon type="search" size="xs" />
            </button>
            <button
              class="toolbar-btn"
              title="替换"
              @click="toggleReplace"
            >
              <Icon type="replace" size="xs" />
            </button>
            <button
              class="toolbar-btn"
              title="转到行"
              @click="goToLine"
            >
              <Icon type="hash" size="xs" />
            </button>
          </div>
          <div class="toolbar-separator"></div>
          <div class="toolbar-group">
            <button
              class="toolbar-btn"
              title="格式化代码"
              @click="formatCode"
            >
              <Icon type="align-left" size="xs" />
            </button>
            <button
              class="toolbar-btn"
              title="折叠代码"
              @click="toggleFold"
            >
              <Icon type="chevron-down" size="xs" />
            </button>
          </div>
        </div>
        <div class="toolbar-right">
          <div class="cursor-info">
            <span>行 {{ cursorPosition.line }}, 列 {{ cursorPosition.column }}</span>
          </div>
          <div class="file-info">
            <span v-if="selectionInfo">{{ selectionInfo }}</span>
            <span v-if="fileSize">{{ fileSize }}</span>
          </div>
        </div>
      </div>

      <!-- 查找替换栏 -->
      <div v-if="showFindBar" class="find-replace-bar">
        <div class="find-input-group">
          <Icon type="search" size="sm" class="input-icon" />
          <input
            v-model="findQuery"
            ref="findInput"
            type="text"
            placeholder="查找..."
            class="find-input"
            @keydown.enter="findNext"
            @keydown.shift.enter="findPrevious"
          />
          <div class="find-controls">
            <span v-if="findResults.total > 0" class="find-results">
              {{ findResults.current }} / {{ findResults.total }}
            </span>
            <button
              class="find-btn"
              title="上一个"
              :disabled="findResults.total === 0"
              @click="findPrevious"
            >
              <Icon type="chevron-up" size="xs" />
            </button>
            <button
              class="find-btn"
              title="下一个"
              :disabled="findResults.total === 0"
              @click="findNext"
            >
              <Icon type="chevron-down" size="xs" />
            </button>
            <button
              class="find-btn"
              title="区分大小写"
              :class="{ active: findOptions.caseSensitive }"
              @click="findOptions.caseSensitive = !findOptions.caseSensitive"
            >
              <Icon type="case-sensitive" size="xs" />
            </button>
            <button
              class="find-btn"
              title="正则表达式"
              :class="{ active: findOptions.useRegex }"
              @click="findOptions.useRegex = !findOptions.useRegex"
            >
              <Icon type="regex" size="xs" />
            </button>
          </div>
        </div>
        <div v-if="showReplace" class="replace-input-group">
          <Icon type="replace" size="sm" class="input-icon" />
          <input
            v-model="replaceQuery"
            type="text"
            placeholder="替换..."
            class="replace-input"
            @keydown.enter="replaceNext"
          />
          <div class="replace-controls">
            <button
              class="replace-btn"
              title="替换当前"
              :disabled="findResults.total === 0"
              @click="replaceNext"
            >
              替换
            </button>
            <button
              class="replace-btn"
              title="替换全部"
              :disabled="findResults.total === 0"
              @click="replaceAll"
            >
              全部替换
            </button>
          </div>
        </div>
      </div>

      <!-- 编辑器区域 -->
      <div class="editor-container">
        <div v-if="editorSettings.lineNumbers" class="line-numbers">
          <div
            v-for="(line, index) in contentLines"
            :key="index"
            :class="['line-number', { 'current-line': index === cursorPosition.line - 1 }]"
          >
            {{ index + 1 }}
          </div>
        </div>
        <div class="editor-content">
          <textarea
            ref="editorTextarea"
            v-model="fileContent"
            :class="['editor-textarea', editorSettings.fontSize, { 'word-wrap': editorSettings.wordWrap }]"
            :style="{
              fontFamily: editorFont,
              tabSize: editorSettings.tabSize
            }"
            spellcheck="false"
            @input="handleInput"
            @keydown="handleKeydown"
            @click="updateCursorPosition"
            @cursor="updateCursorPosition"
            @scroll="syncScroll"
          ></textarea>
          <div v-if="editorSettings.minimap" class="minimap">
            <!-- 缩略图实现 -->
            <div class="minimap-content">{{ fileContent }}</div>
          </div>
        </div>
      </div>
    </div>

    <!-- 状态栏 -->
    <div v-if="currentFile" class="status-bar">
      <div class="status-left">
        <span class="language">{{ getLanguage(currentFile.name) }}</span>
        <span v-if="encoding" class="encoding">{{ encoding }}</span>
      </div>
      <div class="status-right">
        <span v-if="hasUnsavedChanges" class="unsaved">未保存</span>
        <span v-else class="saved">已保存</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick, onMounted } from 'vue';
import Icon from '../ChatUI/Icon.vue';

interface EditorFile {
  name: string;
  path: string;
  content: string;
  originalContent: string;
  lastEdit: number;
  size?: number;
  encoding?: string;
}

interface CursorPosition {
  line: number;
  column: number;
}

interface FindResults {
  current: number;
  total: number;
}

interface FileTemplate {
  [key: string]: string;
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
  fileSaved: [file: EditorFile];
  fileClosed: [file: EditorFile];
}>();

// 响应式数据
const currentFile = ref<EditorFile | null>(null);
const fileContent = ref('');
const recentFiles = ref<EditorFile[]>([]);
const isSaving = ref(false);
const showSettings = ref(false);
const isDragOver = ref(false);
const showFindBar = ref(false);
const showReplace = ref(false);
const newFileName = ref('');
const newFileTemplate = ref('');

// 编辑器设置
const editorSettings = ref({
  fontSize: 'text-sm',
  theme: 'dark',
  wordWrap: true,
  lineNumbers: true,
  minimap: false,
  tabSize: 2,
  fontFamily: 'Monaco, Menlo, "Ubuntu Mono", monospace',
});

// 查找替换
const findQuery = ref('');
const replaceQuery = ref('');
const findResults = ref<FindResults>({ current: 0, total: 0 });
const findOptions = ref({
  caseSensitive: false,
  useRegex: false,
});

// 光标和选择
const cursorPosition = ref<CursorPosition>({ line: 1, column: 1 });
const canUndo = ref(false);
const canRedo = ref(false);

// 编辑历史
const editHistory = ref<string[]>([]);
const historyIndex = ref(-1);

const fileInput = ref<HTMLInputElement>();
const findInput = ref<HTMLInputElement>();
const editorTextarea = ref<HTMLTextAreaElement>();
const websocket = ref<WebSocket | null>(null);

// 文件模板
const fileTemplates: FileTemplate = {
  js: `// JavaScript File
function main() {
  // Your code here
}

main();
`,
  ts: `// TypeScript File
function main(): void {
  // Your code here
}

main();
`,
  vue: `<template>
  <div class="component">
    <!-- Your template here -->
  </div>
</template>

<script setup lang="ts">
// Your script here
</script>

<style scoped>
.component {
  /* Your styles here */
}
</style>
`,
  html: `<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Document</title>
</head>
<body>
  <!-- Your content here -->
</body>
</html>
`,
  css: `/* CSS File */
/* Your styles here */
`,
  json: `{
  "name": "project",
  "version": "1.0.0",
  "description": ""
}
`,
  md: `# Title

## Section

Your content here.
`,
};

// 计算属性
const contentLines = computed(() => {
  return fileContent.value.split('\n');
});

const hasUnsavedChanges = computed(() => {
  return currentFile.value && currentFile.value.content !== currentFile.value.originalContent;
});

const fileSize = computed(() => {
  if (!currentFile.value?.size) return '';
  const bytes = new Blob([fileContent.value]).size;
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
});

const selectionInfo = computed(() => {
  const textarea = editorTextarea.value;
  if (!textarea) return '';

  const start = textarea.selectionStart;
  const end = textarea.selectionEnd;

  if (start === end) return '';

  const selectedText = textarea.value.substring(start, end);
  const lines = selectedText.split('\n');

  if (lines.length === 1) {
    return `已选择 ${end - start} 个字符`;
  } else {
    return `已选择 ${lines.length} 行`;
  }
});

const editorFont = computed(() => {
  return editorSettings.value.fontFamily;
});

const encoding = computed(() => {
  return currentFile.value?.encoding || 'UTF-8';
});

// WebSocket 连接
const connectWebSocket = () => {
  try {
    websocket.value = new WebSocket(`${props.wsUrl}?session=${props.sessionId}`);

    websocket.value.onopen = () => {
      console.log('EditorTool WebSocket connected');
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
      console.log('EditorTool WebSocket disconnected');
      setTimeout(connectWebSocket, 5000);
    };

    websocket.value.onerror = (error) => {
      console.error('EditorTool WebSocket error:', error);
    };
  } catch (error) {
    console.error('Failed to connect WebSocket:', error);
  }
};

const handleWebSocketMessage = (message: any) => {
  switch (message.type) {
    case 'file_saved':
      if (message.file && currentFile.value) {
        currentFile.value.originalContent = currentFile.value.content;
        isSaving.value = false;
        emit('fileSaved', currentFile.value);
      }
      break;
    case 'save_error':
      error.value = message.error || '文件保存失败';
      isSaving.value = false;
      break;
  }
};

const sendWebSocketMessage = (message: any) => {
  if (websocket.value && websocket.value.readyState === WebSocket.OPEN) {
    websocket.value.send(JSON.stringify(message));
  }
};

// 文件操作方法
const openFile = (file: EditorFile) => {
  currentFile.value = { ...file, originalContent: file.content };
  fileContent.value = file.content;
  addToRecentFiles(currentFile.value);
  resetFind();
};

const createNewFile = () => {
  if (!newFileName.value.trim()) return;

  const template = fileTemplates[newFileTemplate.value] || '';
  const newFile: EditorFile = {
    name: newFileName.value.trim(),
    path: newFileName.value.trim(),
    content: template,
    originalContent: template,
    lastEdit: Date.now(),
    encoding: 'UTF-8',
  };

  openFile(newFile);
  newFileName.value = '';
  newFileTemplate.value = '';
};

const saveFile = async () => {
  if (!currentFile.value || isSaving.value) return;

  isSaving.value = true;
  currentFile.value.content = fileContent.value;
  currentFile.value.lastEdit = Date.now();

  sendWebSocketMessage({
    type: 'save_file',
    file: currentFile.value,
  });
};

const saveAndClose = async () => {
  if (currentFile.value) {
    await saveFile();
    closeFile();
  }
};

const closeFile = () => {
  if (currentFile.value) {
    emit('fileClosed', currentFile.value);
    currentFile.value = null;
    fileContent.value = '';
    resetFind();
  }
};

// 查找替换方法
const toggleFind = () => {
  showFindBar.value = !showFindBar.value;
  showReplace.value = false;
  if (showFindBar.value) {
    nextTick(() => {
      findInput.value?.focus();
    });
  }
};

const toggleReplace = () => {
  showReplace.value = !showReplace.value;
  showFindBar.value = true;
  if (showReplace.value) {
    nextTick(() => {
      findInput.value?.focus();
    });
  }
};

const findNext = () => {
  if (!findQuery.value.trim() || !editorTextarea.value) return;

  const textarea = editorTextarea.value;
  const query = findOptions.value.useRegex
    ? new RegExp(findQuery.value, findOptions.value.caseSensitive ? 'g' : 'gi')
    : findQuery.value;

  let start = textarea.selectionEnd;
  if (findOptions.value.useRegex) {
    const regex = new RegExp(findQuery.value, findOptions.value.caseSensitive ? 'g' : 'gi');
    regex.lastIndex = start;
    const match = regex.exec(textarea.value);
    if (match) {
      textarea.setSelectionRange(match.index, match.index + match[0].length);
      updateFindResults(regex);
    }
  } else {
    const text = findOptions.value.caseSensitive
      ? textarea.value
      : textarea.value.toLowerCase();
    const searchQuery = findOptions.value.caseSensitive
      ? query
      : query.toString().toLowerCase();

    const index = text.indexOf(searchQuery.toString(), start);
    if (index !== -1) {
      textarea.setSelectionRange(index, index + searchQuery.toString().length);
      updateFindResults();
    }
  }
};

const findPrevious = () => {
  if (!findQuery.value.trim() || !editorTextarea.value) return;

  const textarea = editorTextarea.value;
  const text = findOptions.value.caseSensitive
    ? textarea.value
    : textarea.value.toLowerCase();
  const searchQuery = findOptions.value.caseSensitive
    ? findQuery.value
    : findQuery.value.toLowerCase();

  let start = textarea.selectionStart - 1;
  const index = text.lastIndexOf(searchQuery, start);

  if (index !== -1) {
    textarea.setSelectionRange(index, index + searchQuery.length);
    updateFindResults();
  }
};

const replaceNext = () => {
  if (!editorTextarea.value || !findQuery.value.trim()) return;

  const textarea = editorTextarea.value;
  const selectedText = textarea.value.substring(textarea.selectionStart, textarea.selectionEnd);

  if (selectedText && (findOptions.value.caseSensitive ? selectedText === findQuery.value : selectedText.toLowerCase() === findQuery.value.toLowerCase())) {
    const newValue = findOptions.value.useRegex ? selectedText.replace(new RegExp(findQuery.value, findOptions.value.caseSensitive ? 'g' : 'gi'), replaceQuery.value) : replaceQuery.value;
    fileContent.value = textarea.value.substring(0, textarea.selectionStart) + newValue + textarea.value.substring(textarea.selectionEnd);

    nextTick(() => {
      findNext();
    });
  } else {
    findNext();
  }
};

const replaceAll = () => {
  if (!findQuery.value.trim()) return;

  const regex = findOptions.value.useRegex
    ? new RegExp(findQuery.value, findOptions.value.caseSensitive ? 'g' : 'gi')
    : new RegExp(findQuery.value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), findOptions.value.caseSensitive ? 'g' : 'gi');

  fileContent.value = fileContent.value.replace(regex, replaceQuery.value);
  updateFindResults();
};

const goToLine = () => {
  const lineNumber = prompt('转到行号:');
  if (lineNumber && !isNaN(Number(lineNumber))) {
    const line = Number(lineNumber) - 1;
    const lines = contentLines.value;

    if (line >= 0 && line < lines.length) {
      let offset = 0;
      for (let i = 0; i < line; i++) {
        offset += lines[i].length + 1; // +1 for newline
      }

      if (editorTextarea.value) {
        editorTextarea.value.focus();
        editorTextarea.value.setSelectionRange(offset, offset);
      }
    }
  }
};

const formatCode = () => {
  // 简单的代码格式化
  if (!currentFile.value) return;

  const extension = currentFile.value.name.split('.').pop()?.toLowerCase();

  if (['js', 'ts', 'jsx', 'tsx', 'json'].includes(extension || '')) {
    try {
      // 简单的JSON格式化
      if (extension === 'json') {
        const parsed = JSON.parse(fileContent.value);
        fileContent.value = JSON.stringify(parsed, null, 2);
      }
    } catch (error) {
      console.warn('Code formatting failed:', error);
    }
  }
};

const toggleFold = () => {
  // 代码折叠功能（简化实现）
  console.log('Toggle fold functionality');
};

// 编辑器事件处理
const handleInput = (event: Event) => {
  const textarea = event.target as HTMLTextAreaElement;
  updateCursorPosition();

  // 更新编辑历史
  if (historyIndex.value < editHistory.value.length - 1) {
    editHistory.value = editHistory.value.slice(0, historyIndex.value + 1);
  }
  editHistory.value.push(textarea.value);
  historyIndex.value++;

  canUndo.value = historyIndex.value > 0;
  canRedo.value = false;

  // 查找高亮
  if (showFindBar.value && findQuery.value.trim()) {
    updateFindResults();
  }
};

const handleKeydown = (event: KeyboardEvent) => {
  const textarea = event.target as HTMLTextAreaElement;

  // 快捷键
  if (event.ctrlKey || event.metaKey) {
    switch (event.key) {
      case 's':
        event.preventDefault();
        saveFile();
        break;
      case 'f':
        event.preventDefault();
        toggleFind();
        break;
      case 'h':
        event.preventDefault();
        toggleReplace();
        break;
      case 'z':
        if (event.shiftKey) {
          event.preventDefault();
          redo();
        } else {
          event.preventDefault();
          undo();
        }
        break;
      case 'y':
        event.preventDefault();
        redo();
        break;
    }
  }

  // Tab支持
  if (event.key === 'Tab') {
    event.preventDefault();
    const start = textarea.selectionStart;
    const end = textarea.selectionEnd;
    const value = textarea.value;

    if (event.shiftKey) {
      // Shift+Tab: 减少缩进
      const lineStart = value.lastIndexOf('\n', start - 1) + 1;
      const lineEnd = value.indexOf('\n', end);
      const line = value.substring(lineStart, lineEnd === -1 ? value.length : lineEnd);

      if (line.startsWith('  ')) {
        const newLine = line.substring(2);
        fileContent.value = value.substring(0, lineStart) + newLine + value.substring(lineEnd === -1 ? value.length : lineEnd);
        nextTick(() => {
          textarea.selectionStart = start - 2;
          textarea.selectionEnd = end - 2;
        });
      }
    } else {
      // Tab: 增加缩进
      const tabSize = editorSettings.value.tabSize;
      const tabString = ' '.repeat(tabSize);
      fileContent.value = value.substring(0, start) + tabString + value.substring(end);
      nextTick(() => {
        textarea.selectionStart = textarea.selectionEnd = start + tabSize;
      });
    }
  }
};

const updateCursorPosition = () => {
  if (!editorTextarea.value) return;

  const textarea = editorTextarea.value;
  const text = textarea.value;
  const position = textarea.selectionStart;

  let line = 1;
  let column = 1;

  for (let i = 0; i < position; i++) {
    if (text[i] === '\n') {
      line++;
      column = 1;
    } else {
      column++;
    }
  }

  cursorPosition.value = { line, column };
};

const syncScroll = (event: Event) => {
  const textarea = event.target as HTMLTextAreaElement;
  // 同步滚动条到缩略图
  const minimap = textarea.parentElement?.querySelector('.minimap-content');
  if (minimap) {
    const scrollPercentage = textarea.scrollTop / (textarea.scrollHeight - textarea.clientHeight);
    minimap.scrollTop = scrollPercentage * (minimap.scrollHeight - minimap.clientHeight);
  }
};

const undo = () => {
  if (historyIndex.value > 0) {
    historyIndex.value--;
    fileContent.value = editHistory.value[historyIndex.value];
    canUndo.value = historyIndex.value > 0;
    canRedo.value = true;
  }
};

const redo = () => {
  if (historyIndex.value < editHistory.value.length - 1) {
    historyIndex.value++;
    fileContent.value = editHistory.value[historyIndex.value];
    canUndo.value = true;
    canRedo.value = historyIndex.value < editHistory.value.length - 1;
  }
};

// 设置相关方法
const toggleSettings = () => {
  showSettings.value = !showSettings.value;
};

const saveSettings = () => {
  localStorage.setItem('editor-settings', JSON.stringify(editorSettings.value));
  showSettings.value = false;
};

const resetSettings = () => {
  editorSettings.value = {
    fontSize: 'text-sm',
    theme: 'dark',
    wordWrap: true,
    lineNumbers: true,
    minimap: false,
    tabSize: 2,
    fontFamily: 'Monaco, Menlo, "Ubuntu Mono", monospace',
  };
};

// 查找结果更新
const updateFindResults = (regex?: RegExp) => {
  if (!findQuery.value.trim() || !editorTextarea.value) {
    findResults.value = { current: 0, total: 0 };
    return;
  }

  const textarea = editorTextarea.value;
  const text = textarea.value;

  if (regex) {
    const matches = [...text.matchAll(regex)];
    findResults.value = { current: matches.length, total: matches.length };
  } else {
    const query = findOptions.value.caseSensitive ? findQuery.value : findQuery.value.toLowerCase();
    const searchIn = findOptions.value.caseSensitive ? text : text.toLowerCase();
    const matches = [...searchIn.matchAll(new RegExp(query.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'g'))];
    findResults.value = { current: matches.length, total: matches.length };
  }
};

const resetFind = () => {
  findQuery.value = '';
  replaceQuery.value = '';
  findResults.value = { current: 0, total: 0 };
  showFindBar.value = false;
  showReplace.value = false;
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
    if (file.type.startsWith('text/') || isTextFileName(file.name)) {
      const reader = new FileReader();
      reader.onload = (e) => {
        const result = e.target?.result as string;
        const newFile: EditorFile = {
          name: file.name,
          path: file.name,
          content: result,
          originalContent: result,
          lastEdit: Date.now(),
          size: file.size,
          encoding: 'UTF-8',
        };
        openFile(newFile);
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

// 工具方法
const getIconForFile = (file: EditorFile) => {
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

const getLanguage = (fileName: string) => {
  const extension = fileName.split('.').pop()?.toLowerCase();
  const languages: { [key: string]: string } = {
    js: 'JavaScript',
    ts: 'TypeScript',
    jsx: 'React',
    tsx: 'React TS',
    vue: 'Vue',
    html: 'HTML',
    css: 'CSS',
    scss: 'SCSS',
    py: 'Python',
    java: 'Java',
    cpp: 'C++',
    c: 'C',
    go: 'Go',
    rs: 'Rust',
    sql: 'SQL',
    json: 'JSON',
    md: 'Markdown',
    yml: 'YAML',
    yaml: 'YAML',
    xml: 'XML',
    sh: 'Shell',
    bash: 'Bash',
    php: 'PHP',
    rb: 'Ruby',
  };

  return languages[extension || ''] || 'Plain Text';
};

const formatEditTime = (timestamp: number) => {
  const date = new Date(timestamp);
  const now = new Date();
  const diff = now.getTime() - date.getTime();
  const hours = Math.floor(diff / (1000 * 60 * 60));

  if (hours < 1) {
    const minutes = Math.floor(diff / (1000 * 60));
    return `${minutes}分钟前`;
  } else if (hours < 24) {
    return `${hours}小时前`;
  } else {
    return date.toLocaleDateString('zh-CN');
  }
};

const addToRecentFiles = (file: EditorFile) => {
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

// 生命周期
onMounted(() => {
  connectWebSocket();

  // 加载设置
  const savedSettings = localStorage.getItem('editor-settings');
  if (savedSettings) {
    try {
      editorSettings.value = { ...editorSettings.value, ...JSON.parse(savedSettings) };
    } catch (error) {
      console.warn('Failed to load editor settings:', error);
    }
  }

  // 初始化编辑历史
  editHistory.value = [''];
  historyIndex.value = 0;
});

// 监听内容变化，更新查找结果
watch([findQuery, findOptions], () => {
  if (showFindBar.value) {
    updateFindResults();
  }
});
</script>

<style scoped>
.editor-tool {
  @apply flex flex-col h-full bg-surface dark:bg-surface-dark border border-border dark:border-border-dark rounded-lg;
}

.editor-header {
  @apply flex items-center justify-between px-4 py-3 border-b border-border dark:border-border-dark bg-surface dark:bg-surface-dark;
}

.header-left {
  @apply flex items-center gap-3 flex-1 min-w-0;
}

.header-title {
  @apply flex items-center gap-2 font-semibold text-text dark:text-text-dark;
}

.file-indicator {
  @apply flex items-center gap-2 px-3 py-1 bg-blue-50 dark:bg-blue-900/20 rounded-full;
}

.file-indicator .file-name {
  @apply text-sm font-medium text-blue-600 dark:text-blue-400;
}

.unsaved-indicator {
  @apply text-blue-500 text-xs;
}

.header-actions {
  @apply flex gap-1;
}

.action-button {
  @apply p-1.5 text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700 rounded transition-colors disabled:opacity-50 disabled:cursor-not-allowed;
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

.setting-select, .setting-checkbox {
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

.new-file-section {
  @apply w-full max-w-md mb-8;
}

.new-file-section h4 {
  @apply text-sm font-semibold text-text dark:text-text-dark mb-3;
}

.new-file-form {
  @apply flex gap-2;
}

.filename-input {
  @apply flex-1 px-3 py-2 border border-border dark:border-border-dark rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-800 dark:text-white;
}

.template-select {
  @apply px-3 py-2 border border-border dark:border-border-dark rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-800 dark:text-white;
}

.create-btn {
  @apply px-4 py-2 bg-blue-500 hover:bg-blue-600 text-white rounded-md transition-colors disabled:opacity-50 disabled:cursor-not-allowed;
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

.file-meta {
  @apply flex-shrink-0 text-xs text-gray-500 dark:text-gray-400;
}

.edit-time {
  @apply whitespace-nowrap;
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

.file-input {
  @apply hidden;
}

.editor-main {
  @apply flex-1 flex flex-col overflow-hidden;
}

.editor-toolbar {
  @apply flex items-center justify-between px-4 py-2 border-b border-gray-100 dark:border-gray-700 bg-gray-50 dark:bg-gray-700/30;
}

.toolbar-left {
  @apply flex items-center gap-2;
}

.toolbar-group {
  @apply flex gap-1;
}

.toolbar-separator {
  @apply w-px h-5 bg-gray-300 dark:bg-gray-600 mx-1;
}

.toolbar-btn {
  @apply p-1.5 text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-600 rounded transition-colors disabled:opacity-50 disabled:cursor-not-allowed;
}

.toolbar-right {
  @apply flex items-center gap-4 text-xs text-gray-500 dark:text-gray-400;
}

.cursor-info {
  @apply font-mono;
}

.find-replace-bar {
  @apply border-b border-gray-200 dark:border-gray-600 bg-white dark:bg-gray-800;
}

.find-input-group, .replace-input-group {
  @apply flex items-center gap-2 px-4 py-2;
}

.input-icon {
  @apply text-gray-400 dark:text-gray-500;
}

.find-input, .replace-input {
  @apply flex-1 px-3 py-1.5 border border-border dark:border-border-dark rounded focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-700 dark:text-white;
}

.find-controls, .replace-controls {
  @apply flex items-center gap-1;
}

.find-btn, .replace-btn {
  @apply p-1 text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-600 rounded transition-colors disabled:opacity-50 disabled:cursor-not-allowed;
}

.find-btn.active {
  @apply text-blue-500 bg-blue-50 dark:bg-blue-900/20;
}

.find-results {
  @apply px-2 text-xs text-gray-500 dark:text-gray-400;
}

.editor-container {
  @apply flex-1 flex overflow-hidden;
}

.line-numbers {
  @apply w-16 bg-gray-100 dark:bg-gray-800 border-r border-gray-200 dark:border-gray-600 flex-shrink-0;
}

.line-number {
  @apply px-2 py-1 text-right text-xs text-gray-400 dark:text-gray-500 leading-6;
}

.line-number.current-line {
  @apply bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400;
}

.editor-content {
  @apply flex-1 flex relative overflow-hidden;
}

.editor-textarea {
  @apply flex-1 w-full h-full px-3 py-2 bg-white dark:bg-gray-900 text-gray-800 dark:text-gray-200 leading-6 border-0 resize-none focus:outline-none;
}

.editor-textarea.word-wrap {
  @apply whitespace-pre-wrap;
}

.editor-textarea:not(.word-wrap) {
  @apply whitespace-pre overflow-x-auto;
}

.minimap {
  @apply absolute top-0 right-0 w-20 h-full bg-gray-50 dark:bg-gray-800 opacity-50 pointer-events-none overflow-hidden;
}

.minimap-content {
  @apply text-xs font-mono leading-tight whitespace-pre-wrap break-all p-2;
}

.status-bar {
  @apply flex items-center justify-between px-4 py-1 border-t border-border dark:border-border-dark bg-gray-100 dark:bg-gray-800 text-xs text-gray-600 dark:text-gray-400;
}

.status-left, .status-right {
  @apply flex items-center gap-3;
}

.language, .encoding {
  @apply px-2 py-0.5 bg-gray-200 dark:bg-gray-700 rounded;
}

.unsaved {
  @apply text-orange-500 font-medium;
}

.saved {
  @apply text-green-500 font-medium;
}

.animate-spin {
  @apply animate-spin;
}
</style>
