<template>
  <div class="write-tool">
    <!-- 头部工具栏 -->
    <div class="write-header">
      <div class="header-title">
        <Icon type="file-plus" size="sm" />
        <span>文件写入工具</span>
      </div>
      <div class="header-actions">
        <button
          class="action-button"
          title="新建文件"
          @click="createNewFile"
        >
          <Icon type="plus" size="sm" />
        </button>
        <button
          class="action-button"
          title="清空内容"
          :disabled="!currentFile || !fileContent"
          @click="clearContent"
        >
          <Icon type="trash" size="sm" />
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
        <h4>写入设置</h4>
        <div class="setting-group">
          <label>文件编码</label>
          <select v-model="writeSettings.encoding" class="setting-select">
            <option value="utf-8">UTF-8</option>
            <option value="utf-16">UTF-16</option>
            <option value="gbk">GBK</option>
            <option value="ascii">ASCII</option>
          </select>
        </div>
        <div class="setting-group">
          <label>换行符</label>
          <select v-model="writeSettings.lineEnding" class="setting-select">
            <option value="\n">LF (Unix/Linux)</option>
            <option value="\r\n">CRLF (Windows)</option>
            <option value="\r">CR (Mac)</option>
          </select>
        </div>
        <div class="setting-group">
          <label>写入模式</label>
          <select v-model="writeSettings.mode" class="setting-select">
            <option value="write">覆盖写入</option>
            <option value="append">追加写入</option>
          </select>
        </div>
        <div class="setting-group">
          <label>
            <input
              v-model="writeSettings.backup"
              type="checkbox"
              class="setting-checkbox"
            />
            写入前备份
          </label>
        </div>
        <div class="setting-group">
          <label>
            <input
              v-model="writeSettings.autoSave"
              type="checkbox"
              class="setting-checkbox"
            />
            自动保存
          </label>
        </div>
        <div class="setting-group">
          <label>自动保存间隔 (秒)</label>
          <input
            v-model.number="writeSettings.autoSaveInterval"
            type="number"
            min="1"
            max="300"
            class="setting-input"
            :disabled="!writeSettings.autoSave"
          />
        </div>
        <div class="setting-actions">
          <button class="setting-btn" @click="resetSettings">重置</button>
          <button class="setting-btn primary" @click="saveSettings">保存</button>
        </div>
      </div>
    </div>

    <!-- 文件选择区域 -->
    <div v-if="!currentFile" class="file-selection">
      <div class="selection-hint">
        <Icon type="file-plus" size="lg" />
        <p>选择或创建要写入的文件</p>
        <p class="hint-text">支持拖拽文件或直接创建新文件</p>
      </div>

      <!-- 路径输入 -->
      <div class="path-input-section">
        <h4>文件路径</h4>
        <div class="path-input-group">
          <Icon type="folder" size="sm" class="input-icon" />
          <input
            v-model="filePath"
            type="text"
            placeholder="输入文件路径 (如: /path/to/file.txt)"
            class="path-input"
            @keydown.enter="createFileAtPath"
          />
          <button
            class="browse-btn"
            title="浏览文件"
            @click="browseFiles"
          >
            <Icon type="folder-open" size="sm" />
          </button>
        </div>
      </div>

      <!-- 文件模板 -->
      <div class="template-section">
        <h4>文件模板</h4>
        <div class="template-grid">
          <div
            v-for="template in fileTemplates"
            :key="template.name"
            class="template-card"
            @click="selectTemplate(template)"
          >
            <div class="template-icon">
              <Icon :type="(template.icon as any)" size="sm" />
            </div>
            <div class="template-info">
              <div class="template-name">{{ template.name }}</div>
              <div class="template-description">{{ template.description }}</div>
            </div>
          </div>
        </div>
      </div>

      <!-- 最近文件 -->
      <div v-if="recentFiles.length > 0" class="recent-files">
        <h4>最近写入</h4>
        <div class="recent-list">
          <div
            v-for="(file, index) in recentFiles"
            :key="index"
            class="recent-item"
            @click="openRecentFile(file)"
          >
            <div class="file-icon">
              <Icon :type="(getIconForFile(file) as any)" size="sm" />
            </div>
            <div class="file-info">
              <div class="file-name">{{ file.name }}</div>
              <div class="file-path">{{ file.path }}</div>
              <div class="file-meta">
                <span class="write-time">{{ formatWriteTime(file.lastWrite) }}</span>
                <span class="file-size">{{ formatFileSize(file.size) }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 拖拽区域 -->
      <div
        class="drop-zone"
        :class="{ 'drag-over': isDragOver }"
        @dragover.prevent="handleDragOver"
        @dragleave.prevent="handleDragLeave"
        @drop.prevent="handleDrop"
      >
        <Icon type="upload" size="lg" />
        <p>拖拽文件到此处</p>
        <p class="drop-hint">或点击下方按钮选择文件</p>
        <input
          type="file"
          ref="fileInput"
          class="file-input"
          @change="handleFileSelect"
          multiple
        />
      </div>
    </div>

    <!-- 编辑器区域 -->
    <div v-if="currentFile" class="editor-area">
      <!-- 文件信息栏 -->
      <div class="file-info-bar">
        <div class="file-info">
          <div class="file-icon">
            <Icon :type="(getIconForFile(currentFile) as any)" size="sm" />
          </div>
          <div class="file-details">
            <div class="file-name">{{ currentFile.name }}</div>
            <div class="file-path">{{ currentFile.path }}</div>
          </div>
          <div class="file-status">
            <span v-if="hasUnsavedChanges" class="unsaved-indicator">● 未保存</span>
            <span v-else class="saved-indicator">✓ 已保存</span>
          </div>
        </div>
        <div class="file-actions">
          <button
            class="action-btn preview-btn"
            title="预览"
            @click="togglePreview"
          >
            <Icon type="eye" size="xs" />
            预览
          </button>
          <button
            class="action-btn save-btn"
            title="保存文件"
            :disabled="isSaving || !hasUnsavedChanges"
            @click="saveFile"
          >
            <Icon v-if="isSaving" type="spinner" size="xs" class="animate-spin" />
            <Icon v-else type="save" size="xs" />
            保存
          </button>
          <button
            class="action-btn close-btn"
            title="关闭文件"
            @click="closeFile"
          >
            <Icon type="close" size="xs" />
          </button>
        </div>
      </div>

      <!-- 编辑器工具栏 -->
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
              title="查找替换"
              @click="toggleFindReplace"
            >
              <Icon type="search" size="xs" />
            </button>
            <button
              class="toolbar-btn"
              title="格式化"
              @click="formatContent"
            >
              <Icon type="align-left" size="xs" />
            </button>
          </div>
          <div class="toolbar-separator"></div>
          <div class="toolbar-group">
            <button
              class="toolbar-btn"
              title="插入时间戳"
              @click="insertTimestamp"
            >
              <Icon type="clock" size="xs" />
            </button>
            <button
              class="toolbar-btn"
              title="插入路径"
              @click="insertFilePath"
            >
              <Icon type="folder" size="xs" />
            </button>
          </div>
        </div>
        <div class="toolbar-right">
          <div class="cursor-info">
            <span>行 {{ cursorPosition.line }}, 列 {{ cursorPosition.column }}</span>
          </div>
          <div class="file-stats">
            <span>{{ formatFileSize(getContentSize()) }}</span>
            <span>{{ getContentLines() }} 行</span>
          </div>
        </div>
      </div>

      <!-- 查找替换栏 -->
      <div v-if="showFindReplace" class="find-replace-bar">
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
            <span v-if="findResults.current > 0" class="find-results">
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
              title="全词匹配"
              :class="{ active: findOptions.wholeWord }"
              @click="findOptions.wholeWord = !findOptions.wholeWord"
            >
              <Icon type="whole-word" size="xs" />
            </button>
          </div>
        </div>
        <div class="replace-input-group">
          <Icon type="replace" size="sm" class="input-icon" />
          <input
            v-model="replaceQuery"
            type="text"
            placeholder="替换为..."
            class="replace-input"
            @keydown.enter="replaceNext"
          />
          <div class="replace-controls">
            <button
              class="replace-btn"
              title="替换"
              :disabled="findResults.total === 0"
              @click="replaceNext"
            >
              替换
            </button>
            <button
              class="replace-btn"
              title="全部替换"
              :disabled="findResults.total === 0"
              @click="replaceAll"
            >
              全部替换
            </button>
          </div>
        </div>
      </div>

      <!-- 编辑器主体 -->
      <div class="editor-container">
        <div v-if="writeSettings.showLineNumbers" class="line-numbers">
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
            :class="['editor-textarea', writeSettings.fontSize, { 'word-wrap': writeSettings.wordWrap }]"
            :style="{
              fontFamily: writeSettings.fontFamily,
              tabSize: writeSettings.tabSize
            }"
            spellcheck="false"
            @input="handleInput"
            @keydown="handleKeydown"
            @click="updateCursorPosition"
            @cursor="updateCursorPosition"
            @scroll="syncScroll"
          ></textarea>

          <!-- 预览面板 -->
          <div v-if="showPreview" class="preview-panel">
            <div class="preview-header">
              <h4>预览</h4>
              <button
                class="preview-close"
                @click="showPreview = false"
              >
                <Icon type="close" size="xs" />
              </button>
            </div>
            <div class="preview-content" v-html="formatPreview()"></div>
          </div>
        </div>
      </div>
    </div>

    <!-- 状态栏 -->
    <div v-if="currentFile" class="status-bar">
      <div class="status-left">
        <span class="encoding">{{ writeSettings.encoding.toUpperCase() }}</span>
        <span class="mode">{{ writeSettings.mode === 'write' ? '覆盖' : '追加' }}</span>
        <span v-if="writeSettings.autoSave" class="auto-save">自动保存</span>
      </div>
      <div class="status-right">
        <span v-if="isSaving" class="saving-indicator">
          <Icon type="spinner" size="xs" class="animate-spin" />
          保存中...
        </span>
        <span v-else-if="lastSaveTime" class="last-save">
          最后保存: {{ formatSaveTime(lastSaveTime) }}
        </span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick, onMounted, onUnmounted } from 'vue';
import Icon from '../ChatUI/Icon.vue';

interface WriteFile {
  name: string;
  path: string;
  content: string;
  originalContent: string;
  lastWrite: number;
  size: number;
  encoding: string;
}

interface FileTemplate {
  name: string;
  description: string;
  icon: string;
  content: string;
  extension: string;
}

interface CursorPosition {
  line: number;
  column: number;
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
  fileWritten: [file: WriteFile];
  fileClosed: [file: WriteFile];
}>();

// 响应式数据
const currentFile = ref<WriteFile | null>(null);
const fileContent = ref('');
const filePath = ref('');
const recentFiles = ref<WriteFile[]>([]);
const isSaving = ref(false);
const showSettings = ref(false);
const showPreview = ref(false);
const showFindReplace = ref(false);
const isDragOver = ref(false);

// 查找替换
const findQuery = ref('');
const replaceQuery = ref('');
const findResults = ref({ current: 0, total: 0 });
const findOptions = ref({
  caseSensitive: false,
  wholeWord: false,
});

// 光标和编辑状态
const cursorPosition = ref<CursorPosition>({ line: 1, column: 1 });
const canUndo = ref(false);
const canRedo = ref(false);

// 编辑历史
const editHistory = ref<string[]>([]);
const historyIndex = ref(-1);

// 自动保存
const autoSaveTimer = ref<NodeJS.Timeout | null>(null);

// 错误和保存时间
const error = ref<string | null>(null);
const lastSaveTime = ref<number | null>(null);

// 写入设置
const writeSettings = ref({
  fontSize: 'text-sm',
  fontFamily: 'Monaco, Menlo, "Ubuntu Mono", monospace',
  tabSize: 2,
  wordWrap: true,
  showLineNumbers: true,
  encoding: 'utf-8',
  lineEnding: '\n',
  mode: 'write',
  backup: false,
  autoSave: false,
  autoSaveInterval: 30,
  // 查找替换选项
  regex: false,
  caseSensitive: false,
});

const fileInput = ref<HTMLInputElement>();
const findInput = ref<HTMLInputElement>();
const editorTextarea = ref<HTMLTextAreaElement>();
const websocket = ref<WebSocket | null>(null);

// 文件模板
const fileTemplates: FileTemplate[] = [
  {
    name: '文本文件',
    description: '纯文本文档',
    icon: 'file-text',
    content: '',
    extension: '.txt',
  },
  {
    name: 'Markdown',
    description: 'Markdown文档',
    icon: 'file-text',
    content: `# 文档标题

## 章节

### 子章节

内容...`,
    extension: '.md',
  },
  {
    name: 'JSON',
    description: 'JSON配置文件',
    icon: 'file-code',
    content: `{\n  "name": "",\n  "version": "1.0.0",\n  "description": ""\n}`,
    extension: '.json',
  },
  {
    name: 'HTML',
    description: 'HTML网页文件',
    icon: 'file-code',
    content: `<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>页面标题</title>
</head>
<body>
  <h1>Hello World</h1>
</body>
</html>`,
    extension: '.html',
  },
  {
    name: 'CSS',
    description: 'CSS样式文件',
    icon: 'file-code',
    content: `/* CSS 样式 */
body {
  font-family: Arial, sans-serif;
  margin: 0;
  padding: 20px;
}

.container {
  max-width: 1200px;
  margin: 0 auto;
}`,
    extension: '.css',
  },
  {
    name: 'JavaScript',
    description: 'JavaScript脚本文件',
    icon: 'file-code',
    content: `// JavaScript 脚本
function main() {
  console.log('Hello World!');

  // 你的代码
}

// 执行主函数
main();`,
    extension: '.js',
  },
  {
    name: 'YAML',
    description: 'YAML配置文件',
    icon: 'file-code',
    content: `# YAML 配置
app:
  name: ""
  version: ""
  debug: false

database:
  host: localhost
  port: 5432
  name: ""`,
    extension: '.yml',
  },
  {
    name: '环境变量',
    description: '.env环境变量文件',
    icon: 'file-code',
    content: `# 环境变量配置
NODE_ENV=development
PORT=3000
API_URL=http://localhost:8080`,
    extension: '.env',
  },
];

// 计算属性
const contentLines = computed(() => {
  return fileContent.value.split('\n');
});

const hasUnsavedChanges = computed(() => {
  return currentFile.value && currentFile.value.content !== currentFile.value.originalContent;
});

// WebSocket 连接
const connectWebSocket = () => {
  try {
    websocket.value = new WebSocket(`${props.wsUrl}?session=${props.sessionId}`);

    websocket.value.onopen = () => {
      console.log('WriteTool WebSocket connected');
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
      console.log('WriteTool WebSocket disconnected');
      setTimeout(connectWebSocket, 5000);
    };

    websocket.value.onerror = (error) => {
      console.error('WriteTool WebSocket error:', error);
    };
  } catch (error) {
    console.error('Failed to connect WebSocket:', error);
  }
};

const handleWebSocketMessage = (message: any) => {
  switch (message.type) {
    case 'file_written':
      if (message.file && currentFile.value) {
        currentFile.value.originalContent = currentFile.value.content;
        currentFile.value.lastWrite = Date.now();
        currentFile.value.size = new Blob([currentFile.value.content]).size;
        isSaving.value = false;
        addToRecentFiles(currentFile.value);
        emit('fileWritten', currentFile.value);
      }
      break;
    case 'write_error':
      error.value = message.error || '文件写入失败';
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
const createNewFile = () => {
  const newFile: WriteFile = {
    name: 'untitled.txt',
    path: '/untitled.txt',
    content: '',
    originalContent: '',
    lastWrite: Date.now(),
    size: 0,
    encoding: writeSettings.value.encoding,
  };
  openFile(newFile);
};

const createFileAtPath = () => {
  if (!filePath.value.trim()) return;

  const path = filePath.value.trim();
  const name = path.split('/').pop() || 'untitled.txt';

  const newFile: WriteFile = {
    name,
    path,
    content: '',
    originalContent: '',
    lastWrite: Date.now(),
    size: 0,
    encoding: writeSettings.value.encoding,
  };

  openFile(newFile);
  filePath.value = '';
};

const openFile = (file: WriteFile) => {
  currentFile.value = file;
  fileContent.value = file.content;
  resetFindReplace();
  setupAutoSave();

  // 重置编辑历史
  editHistory.value = [file.content];
  historyIndex.value = 0;
  canUndo.value = false;
  canRedo.value = false;

  nextTick(() => {
    editorTextarea.value?.focus();
  });
};

const openRecentFile = (file: WriteFile) => {
  openFile(file);
};

const selectTemplate = (template: FileTemplate) => {
  const timestamp = new Date().toISOString().slice(0, 19).replace(/[:-]/g, '');
  const fileName = `template_${timestamp}${template.extension}`;

  const newFile: WriteFile = {
    name: fileName,
    path: `/${fileName}`,
    content: template.content,
    originalContent: template.content,
    lastWrite: Date.now(),
    size: new Blob([template.content]).size,
    encoding: writeSettings.value.encoding,
  };

  openFile(newFile);
};

const saveFile = async () => {
  if (!currentFile.value || isSaving.value) return;

  isSaving.value = true;
  currentFile.value.content = fileContent.value;
  currentFile.value.lastWrite = Date.now();

  sendWebSocketMessage({
    type: 'write_file',
    file: {
      ...currentFile.value,
      content: fileContent.value,
      settings: writeSettings.value,
    },
  });
};

const closeFile = () => {
  if (currentFile.value && hasUnsavedChanges.value) {
    if (!confirm('文件有未保存的更改，确定要关闭吗？')) {
      return;
    }
  }

  if (currentFile.value) {
    emit('fileClosed', currentFile.value);
    currentFile.value = null;
    fileContent.value = '';
    resetFindReplace();
    clearAutoSave();
  }
};

const clearContent = () => {
  if (currentFile.value && confirm('确定要清空文件内容吗？')) {
    fileContent.value = '';
    handleInput({ target: { value: '' } } as any);
  }
};

// 查找替换方法
const toggleFindReplace = () => {
  showFindReplace.value = !showFindReplace.value;
  if (showFindReplace.value) {
    nextTick(() => {
      findInput.value?.focus();
    });
  }
};

const findNext = () => {
  if (!findQuery.value.trim() || !editorTextarea.value) return;

  const textarea = editorTextarea.value;
  const query = createSearchQuery();

  const start = textarea.selectionEnd;
  const text = textarea.value;

  let match;
  if (writeSettings.value.regex) {
    const regex = new RegExp(findQuery.value, writeSettings.value.caseSensitive ? 'g' : 'gi');
    regex.lastIndex = start;
    match = regex.exec(text);
  } else {
    const searchQuery = writeSettings.value.caseSensitive ? query : query.toLowerCase();
    const searchText = writeSettings.value.caseSensitive ? text : text.toLowerCase();
    const index = searchText.indexOf(searchQuery, start);
    if (index !== -1) {
      match = { index: index, 0: searchQuery };
    }
  }

  if (match) {
    textarea.focus();
    textarea.setSelectionRange(match.index, match.index + match[0].length);
    updateFindResults();
  }
};

const findPrevious = () => {
  if (!findQuery.value.trim() || !editorTextarea.value) return;

  const textarea = editorTextarea.value;
  const query = createSearchQuery();

  const start = textarea.selectionStart - 1;
  const text = textarea.value;

  let match;
  if (writeSettings.value.regex) {
    const regex = new RegExp(findQuery.value, writeSettings.value.caseSensitive ? 'g' : 'gi');
    const matches = [...text.matchAll(regex)];
    const currentIndex = matches.findIndex(m => m.index >= textarea.selectionStart);
    if (currentIndex > 0) {
      match = matches[currentIndex - 1];
    } else if (matches.length > 0) {
      match = matches[matches.length - 1];
    }
  } else {
    const searchQuery = writeSettings.value.caseSensitive ? query : query.toLowerCase();
    const searchText = writeSettings.value.caseSensitive ? text : text.toLowerCase();
    const index = searchText.lastIndexOf(searchQuery, start);
    if (index !== -1) {
      match = { index: index, 0: searchQuery };
    }
  }

  if (match) {
    textarea.focus();
    textarea.setSelectionRange(match.index, match.index + match[0].length);
    updateFindResults();
  }
};

const replaceNext = () => {
  if (!editorTextarea.value || !findQuery.value.trim()) return;

  const textarea = editorTextarea.value;
  const selectedText = textarea.value.substring(textarea.selectionStart, textarea.selectionEnd);

  if (selectedText && isMatch(selectedText)) {
    const newValue = replaceQuery.value;
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

  const regex = createReplaceRegex();
  fileContent.value = fileContent.value.replace(regex, replaceQuery.value);
  updateFindResults();
};

const createSearchQuery = () => {
  let query = findQuery.value;
  if (findOptions.value.wholeWord) {
    query = `\\b${query}\\b`;
  }
  return query;
};

const createReplaceRegex = () => {
  const flags = writeSettings.value.caseSensitive ? 'g' : 'gi';
  let pattern = findQuery.value;
  if (findOptions.value.wholeWord) {
    pattern = `\\b${pattern}\\b`;
  }
  return new RegExp(pattern.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), flags);
};

const isMatch = (text: string): boolean => {
  if (findOptions.value.caseSensitive) {
    return findOptions.value.wholeWord
      ? new RegExp(`\\b${findQuery.value}\\b`).test(text)
      : text.includes(findQuery.value);
  } else {
    return findOptions.value.wholeWord
      ? new RegExp(`\\b${findQuery.value}\\b`, 'i').test(text)
      : text.toLowerCase().includes(findQuery.value.toLowerCase());
  }
};

const updateFindResults = () => {
  if (!findQuery.value.trim() || !editorTextarea.value) {
    findResults.value = { current: 0, total: 0 };
    return;
  }

  const regex = createReplaceRegex();
  const matches = [...editorTextarea.value.value.matchAll(regex)];
  findResults.value = { current: matches.length, total: matches.length };
};

const resetFindReplace = () => {
  findQuery.value = '';
  replaceQuery.value = '';
  findResults.value = { current: 0, total: 0 };
  showFindReplace.value = false;
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

  // 更新查找结果
  if (showFindReplace.value && findQuery.value.trim()) {
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
        toggleFindReplace();
        break;
      case 'h':
        event.preventDefault();
        toggleFindReplace();
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
      const tabSize = writeSettings.value.tabSize;
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
  // 可以在这里同步滚动
};

const undo = () => {
  if (historyIndex.value > 0) {
    historyIndex.value--;
    fileContent.value = editHistory.value[historyIndex.value] ?? '';
    canUndo.value = historyIndex.value > 0;
    canRedo.value = true;
  }
};

const redo = () => {
  if (historyIndex.value < editHistory.value.length - 1) {
    historyIndex.value++;
    fileContent.value = editHistory.value[historyIndex.value] ?? '';
    canUndo.value = true;
    canRedo.value = historyIndex.value < editHistory.value.length - 1;
  }
};

// 工具方法
const formatContent = () => {
  if (!currentFile.value) return;

  const extension = currentFile.value.name.split('.').pop()?.toLowerCase();

  if (extension === 'json') {
    try {
      const parsed = JSON.parse(fileContent.value);
      fileContent.value = JSON.stringify(parsed, null, 2);
    } catch (error) {
      console.warn('JSON formatting failed:', error);
    }
  }
};

const insertTimestamp = () => {
  const timestamp = new Date().toISOString();
  insertTextAtCursor(timestamp);
};

const insertFilePath = () => {
  if (currentFile.value) {
    insertTextAtCursor(currentFile.value.path);
  }
};

const insertTextAtCursor = (text: string) => {
  if (!editorTextarea.value) return;

  const textarea = editorTextarea.value;
  const start = textarea.selectionStart;
  const end = textarea.selectionEnd;
  const value = textarea.value;

  fileContent.value = value.substring(0, start) + text + value.substring(end);

  nextTick(() => {
    textarea.focus();
    textarea.setSelectionRange(start + text.length, start + text.length);
  });
};

const togglePreview = () => {
  showPreview.value = !showPreview.value;
};

const formatPreview = () => {
  if (!currentFile.value) return '';

  const extension = currentFile.value.name.split('.').pop()?.toLowerCase();

  if (extension === 'md') {
    // 简单的Markdown预览
    return fileContent.value
      .replace(/^# (.*$)/gim, '<h1>$1</h1>')
      .replace(/^## (.*$)/gim, '<h2>$1</h2>')
      .replace(/^### (.*$)/gim, '<h3>$1</h3>')
      .replace(/\*\*(.*)\*\*/gim, '<strong>$1</strong>')
      .replace(/\*(.*)\*/gim, '<em>$1</em>')
      .replace(/`([^`]*)`/gim, '<code>$1</code>')
      .replace(/\n/gim, '<br>');
  }

  return `<pre>${fileContent.value}</pre>`;
};

// 自动保存
const setupAutoSave = () => {
  clearAutoSave();

  if (writeSettings.value.autoSave && writeSettings.value.autoSaveInterval > 0) {
    autoSaveTimer.value = setInterval(() => {
      if (hasUnsavedChanges.value) {
        saveFile();
      }
    }, writeSettings.value.autoSaveInterval * 1000);
  }
};

const clearAutoSave = () => {
  if (autoSaveTimer.value) {
    clearInterval(autoSaveTimer.value);
    autoSaveTimer.value = null;
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
    const reader = new FileReader();
    reader.onload = (e) => {
      const result = e.target?.result as string;
      const newFile: WriteFile = {
        name: file.name,
        path: `/${file.name}`,
        content: result,
        originalContent: result,
        lastWrite: Date.now(),
        size: file.size,
        encoding: writeSettings.value.encoding,
      };
      openFile(newFile);
    };
    reader.readAsText(file, writeSettings.value.encoding as any);
  });
};

const browseFiles = () => {
  // 在实际应用中，这里可以打开文件浏览器
  fileInput.value?.click();
};

// 设置管理
const toggleSettings = () => {
  showSettings.value = !showSettings.value;
};

const saveSettings = () => {
  localStorage.setItem('write-settings', JSON.stringify(writeSettings.value));
  showSettings.value = false;
  setupAutoSave();
};

const resetSettings = () => {
  writeSettings.value = {
    fontSize: 'text-sm',
    fontFamily: 'Monaco, Menlo, "Ubuntu Mono", monospace',
    tabSize: 2,
    wordWrap: true,
    showLineNumbers: true,
    encoding: 'utf-8',
    lineEnding: '\n',
    mode: 'write',
    backup: false,
    autoSave: false,
    autoSaveInterval: 30,
    regex: false,
    caseSensitive: false,
  };
  setupAutoSave();
};

// 工具方法
const getIconForFile = (file: WriteFile) => {
  const extension = file.name.split('.').pop()?.toLowerCase();
  const iconMap: { [key: string]: string } = {
    txt: 'file-text',
    md: 'file-text',
    json: 'file-code',
    html: 'file-code',
    css: 'file-code',
    js: 'file-code',
    ts: 'file-code',
    yml: 'file-code',
    yaml: 'file-code',
    env: 'file-code',
    xml: 'file-code',
    csv: 'file-spreadsheet',
    log: 'file-text',
  };
  return iconMap[extension || ''] || 'file';
};

const formatFileSize = (bytes: number) => {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
};

const formatWriteTime = (timestamp: number) => {
  const date = new Date(timestamp);
  return date.toLocaleString('zh-CN');
};

const formatSaveTime = (timestamp: number) => {
  const date = new Date(timestamp);
  const now = new Date();
  const diff = now.getTime() - date.getTime();
  const minutes = Math.floor(diff / (1000 * 60));

  if (minutes < 1) {
    return '刚刚';
  } else if (minutes < 60) {
    return `${minutes}分钟前`;
  } else {
    return date.toLocaleTimeString('zh-CN');
  }
};

const getContentSize = () => {
  return new Blob([fileContent.value]).size;
};

const getContentLines = () => {
  return fileContent.value.split('\n').length;
};

const addToRecentFiles = (file: WriteFile) => {
  const existingIndex = recentFiles.value.findIndex(f => f.path === file.path);
  if (existingIndex !== -1) {
    recentFiles.value[existingIndex] = file;
  } else {
    recentFiles.value.unshift(file);
    if (recentFiles.value.length > 10) {
      recentFiles.value = recentFiles.value.slice(0, 10);
    }
  }

  // 保存到本地存储
  try {
    localStorage.setItem('write-recent-files', JSON.stringify(recentFiles.value));
  } catch (error) {
    console.warn('Failed to save recent files:', error);
  }
};

// 生命周期
onMounted(() => {
  connectWebSocket();

  // 加载设置
  try {
    const saved = localStorage.getItem('write-settings');
    if (saved) {
      writeSettings.value = { ...writeSettings.value, ...JSON.parse(saved) };
    }
  } catch (error) {
    console.warn('Failed to load write settings:', error);
  }

  // 加载最近文件
  try {
    const saved = localStorage.getItem('write-recent-files');
    if (saved) {
      recentFiles.value = JSON.parse(saved);
    }
  } catch (error) {
    console.warn('Failed to load recent files:', error);
  }

  // 初始化编辑历史
  editHistory.value = [''];
  historyIndex.value = 0;
});

onUnmounted(() => {
  clearAutoSave();
});

// 监听设置变化
watch(writeSettings, () => {
  setupAutoSave();
}, { deep: true });

// 监听自动保存设置变化
watch([() => writeSettings.value.autoSave, () => writeSettings.value.autoSaveInterval], () => {
  setupAutoSave();
});
</script>

<style scoped>
.write-tool {
  @apply flex flex-col h-full bg-surface dark:bg-surface-dark border border-border dark:border-border-dark rounded-lg;
}

.write-header {
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

.setting-select, .setting-input {
  @apply ml-2;
}

.setting-select {
  @apply px-2 py-1 text-sm border border-border dark:border-border-dark rounded focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-800 dark:text-white;
}

.setting-input {
  @apply w-20 px-2 py-1 text-sm border border-border dark:border-border-dark rounded focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-800 dark:text-white;
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
  @apply flex-1 flex flex-col items-center justify-center p-8 overflow-y-auto;
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

.path-input-section {
  @apply w-full max-w-2xl mb-8;
}

.path-input-section h4 {
  @apply text-sm font-semibold text-text dark:text-text-dark mb-3;
}

.path-input-group {
  @apply flex items-center gap-2;
}

.input-icon {
  @apply text-gray-400 dark:text-gray-500;
}

.path-input {
  @apply flex-1 px-3 py-2 border border-border dark:border-border-dark rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-800 dark:text-white;
}

.browse-btn {
  @apply p-2 text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700 rounded transition-colors;
}

.template-section {
  @apply w-full max-w-4xl mb-8;
}

.template-section h4 {
  @apply text-sm font-semibold text-text dark:text-text-dark mb-3;
}

.template-grid {
  @apply grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-3;
}

.template-card {
  @apply flex items-center gap-3 p-3 bg-white dark:bg-gray-700 border border-gray-200 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-600 cursor-pointer transition-colors;
}

.template-icon {
  @apply flex-shrink-0 w-8 h-8 flex items-center justify-center bg-blue-50 dark:bg-blue-900/20 rounded text-blue-500 dark:text-blue-400;
}

.template-info {
  @apply flex-1 min-w-0;
}

.template-name {
  @apply text-sm font-medium text-text dark:text-text-dark;
}

.template-description {
  @apply text-xs text-gray-500 dark:text-gray-400 truncate;
}

.recent-files {
  @apply w-full max-w-2xl mb-8;
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

.file-meta {
  @apply flex items-center gap-3 text-xs text-gray-500 dark:text-gray-400;
}

.write-time {
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

.drop-hint {
  @apply text-sm mt-1;
}

.file-input {
  @apply hidden;
}

.editor-area {
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

.file-status {
  @apply flex-shrink-0;
}

.unsaved-indicator {
  @apply text-orange-500 text-sm;
}

.saved-indicator {
  @apply text-green-500 text-sm;
}

.file-actions {
  @apply flex gap-1;
}

.action-btn {
  @apply flex items-center gap-1 px-2 py-1 text-sm text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-600 rounded transition-colors disabled:opacity-50 disabled:cursor-not-allowed;
}

.preview-btn:hover {
  @apply text-blue-500 hover:bg-blue-50 dark:hover:bg-blue-900/20;
}

.save-btn:hover {
  @apply text-green-500 hover:bg-green-50 dark:hover:bg-green-900/20;
}

.close-btn:hover {
  @apply text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20;
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

.file-stats {
  @apply flex items-center gap-3;
}

.find-replace-bar {
  @apply border-b border-gray-200 dark:border-gray-600 bg-white dark:bg-gray-800;
}

.find-input-group, .replace-input-group {
  @apply flex items-center gap-2 px-4 py-2;
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
  @apply flex-1 w-full h-full px-3 py-2 bg-white dark:bg-gray-900 text-gray-800 dark:text-gray-200 leading-6 border-0 resize-none focus:outline-none font-mono;
}

.editor-textarea.word-wrap {
  @apply whitespace-pre-wrap;
}

.editor-textarea:not(.word-wrap) {
  @apply whitespace-pre overflow-x-auto;
}

.preview-panel {
  @apply absolute top-0 right-0 w-1/2 h-full bg-white dark:bg-gray-800 border-l border-gray-200 dark:border-gray-600 shadow-lg;
}

.preview-header {
  @apply flex items-center justify-between px-4 py-2 border-b border-gray-200 dark:border-gray-600 bg-gray-50 dark:bg-gray-700;
}

.preview-header h4 {
  @apply text-sm font-semibold text-text dark:text-text-dark;
}

.preview-close {
  @apply p-1 text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-600 rounded transition-colors;
}

.preview-content {
  @apply p-4 overflow-y-auto h-full text-sm;
}

.status-bar {
  @apply flex items-center justify-between px-4 py-1 border-t border-border dark:border-border-dark bg-gray-100 dark:bg-gray-800 text-xs text-gray-600 dark:text-gray-400;
}

.status-left, .status-right {
  @apply flex items-center gap-3;
}

.encoding, .mode {
  @apply px-2 py-0.5 bg-gray-200 dark:bg-gray-700 rounded;
}

.auto-save {
  @apply text-green-500;
}

.saving-indicator {
  @apply text-blue-500 flex items-center gap-1;
}

.animate-spin {
  @apply animate-spin;
}
</style>