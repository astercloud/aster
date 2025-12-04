<template>
  <div class="edit-tool">
    <!-- 头部工具栏 -->
    <div class="edit-header">
      <div class="header-left">
        <div class="header-title">
          <Icon type="edit" size="sm" />
          <span>文件编辑器</span>
        </div>
        <div v-if="currentFile" class="file-indicator">
          <Icon :type="getIconForFile(currentFile) as any" size="xs" />
          <span class="file-name">{{ currentFile.name }}</span>
          <span v-if="hasUnsavedChanges" class="unsaved-indicator">●</span>
        </div>
      </div>
      <div class="header-actions">
        <button v-if="currentFile" class="action-button" title="保存文件" :disabled="isSaving" @click="saveFile">
          <Icon v-if="isSaving" type="spinner" size="sm" class="animate-spin" />
          <Icon v-else type="save" size="sm" />
        </button>
        <button v-if="currentFile" class="action-button" title="保存并关闭" :disabled="isSaving" @click="saveAndClose">
          <Icon type="check" size="sm" />
        </button>
        <button v-if="currentFile" class="action-button" title="关闭文件" @click="closeFile">
          <Icon type="close" size="sm" />
        </button>
        <button class="action-button" title="设置" @click="toggleSettings">
          <Icon type="settings" size="sm" />
        </button>
      </div>
    </div>

    <!-- 设置面板 -->
    <div v-if="showSettings" class="settings-panel">
      <div class="settings-content">
        <h4>编辑器设置</h4>
        <div class="setting-group">
          <label>编辑器主题</label>
          <select v-model="editorSettings.theme" class="setting-select">
            <option value="vs-code">VS Code</option>
            <option value="github">GitHub</option>
            <option value="monokai">Monokai</option>
            <option value="solarized">Solarized</option>
            <option value="dracula">Dracula</option>
          </select>
        </div>
        <div class="setting-group">
          <label>字体大小</label>
          <select v-model="editorSettings.fontSize" class="setting-select">
            <option value="12">12px</option>
            <option value="14">14px</option>
            <option value="16">16px</option>
            <option value="18">18px</option>
            <option value="20">20px</option>
          </select>
        </div>
        <div class="setting-group">
          <label>字体</label>
          <select v-model="editorSettings.fontFamily" class="setting-select">
            <option value="Consolas, Monaco, 'Courier New', monospace">Consolas</option>
            <option value="'Fira Code', monospace">Fira Code</option>
            <option value="'Source Code Pro', monospace">Source Code Pro</option>
            <option value="'JetBrains Mono', monospace">JetBrains Mono</option>
          </select>
        </div>
        <div class="setting-group">
          <label>Tab大小</label>
          <input v-model.number="editorSettings.tabSize" type="number" min="2" max="8" class="setting-input" />
        </div>
        <div class="setting-group">
          <label>
            <input v-model="editorSettings.wordWrap" type="checkbox" class="setting-checkbox" />
            自动换行
          </label>
        </div>
        <div class="setting-group">
          <label>
            <input v-model="editorSettings.lineNumbers" type="checkbox" class="setting-checkbox" />
            显示行号
          </label>
        </div>
        <div class="setting-group">
          <label>
            <input v-model="editorSettings.minimap" type="checkbox" class="setting-checkbox" />
            显示缩略图
          </label>
        </div>
        <div class="setting-group">
          <label>
            <input v-model="editorSettings.autoSave" type="checkbox" class="setting-checkbox" />
            自动保存
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

      <!-- 快速访问 -->
      <div class="quick-access">
        <h4>快速访问</h4>
        <div class="quick-actions">
          <button class="quick-action-btn" @click="browseFiles">
            <Icon type="folder-open" size="sm" />
            浏览文件
          </button>
          <button class="quick-action-btn" @click="createNewFile">
            <Icon type="file-plus" size="sm" />
            新建文件
          </button>
          <button class="quick-action-btn" @click="openFromUrl">
            <Icon type="link" size="sm" />
            从URL打开
          </button>
        </div>
      </div>

      <!-- 最近编辑的文件 -->
      <div v-if="recentFiles.length > 0" class="recent-files">
        <h4>最近编辑</h4>
        <div class="recent-list">
          <div v-for="(file, index) in recentFiles" :key="index" class="recent-item" @click="openFile(file)">
            <div class="file-icon">
              <Icon :type="getIconForFile(file) as any" size="sm" />
            </div>
            <div class="file-info">
              <div class="file-name">{{ file.name }}</div>
              <div class="file-path">{{ file.path }}</div>
              <div class="file-meta">
                <span class="edit-time">{{ formatEditTime(file.lastEdit) }}</span>
                <span class="language">{{ getLanguage(file.name) }}</span>
              </div>
            </div>
            <div class="file-actions">
              <button class="file-action-btn" title="从磁盘重新加载" @click.stop="reloadFromFile(file)">
                <Icon type="refresh" size="xs" />
              </button>
              <button class="file-action-btn remove-btn" title="从最近列表移除" @click.stop="removeFromRecent(index)">
                <Icon type="close" size="xs" />
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- 拖拽区域 -->
      <div class="drop-zone" :class="{ 'drag-over': isDragOver }" @dragover.prevent="handleDragOver" @dragleave.prevent="handleDragLeave" @drop.prevent="handleDrop">
        <Icon type="upload" size="lg" />
        <p>拖拽文件到此处开始编辑</p>
        <p class="drop-hint">支持多种文件格式</p>
        <input type="file" ref="fileInput" class="file-input" @change="handleFileSelect" multiple />
      </div>
    </div>

    <!-- 编辑器主体 -->
    <div v-if="currentFile" class="editor-main">
      <!-- 工具栏 -->
      <div class="editor-toolbar">
        <div class="toolbar-left">
          <div class="toolbar-group">
            <button class="toolbar-btn" title="撤销" :disabled="!canUndo" @click="undo">
              <Icon type="undo" size="xs" />
            </button>
            <button class="toolbar-btn" title="重做" :disabled="!canRedo" @click="redo">
              <Icon type="redo" size="xs" />
            </button>
          </div>
          <div class="toolbar-separator"></div>
          <div class="toolbar-group">
            <button class="toolbar-btn" title="查找" @click="toggleFind">
              <Icon type="search" size="xs" />
            </button>
            <button class="toolbar-btn" title="替换" @click="toggleReplace">
              <Icon type="replace" size="xs" />
            </button>
            <button class="toolbar-btn" title="转到行" @click="goToLine()">
              <Icon type="hash" size="xs" />
            </button>
          </div>
          <div class="toolbar-separator"></div>
          <div class="toolbar-group">
            <button class="toolbar-btn" title="格式化代码" @click="formatCode">
              <Icon type="align-left" size="xs" />
            </button>
            <button class="toolbar-btn" title="折叠代码" @click="toggleFold">
              <Icon type="chevron-down" size="xs" />
            </button>
            <button class="toolbar-btn" title="切换注释" @click="toggleComment">
              <Icon type="message-square" size="xs" />
            </button>
          </div>
          <div class="toolbar-separator"></div>
          <div class="toolbar-group">
            <button class="toolbar-btn" :title="showDiff ? '隐藏差异' : '显示差异'" @click="toggleDiff">
              <Icon type="git-merge" size="xs" />
            </button>
            <button class="toolbar-btn" :title="showSidebar ? '隐藏侧边栏' : '显示侧边栏'" @click="toggleSidebar">
              <Icon type="sidebar" size="xs" />
            </button>
          </div>
        </div>
        <div class="toolbar-right">
          <div class="cursor-info">
            <span>行 {{ cursorPosition.line }}, 列 {{ cursorPosition.column }}</span>
          </div>
          <div class="selection-info">
            <span v-if="selectionInfo">{{ selectionInfo }}</span>
          </div>
          <div class="file-info">
            <span class="language">{{ getLanguage(currentFile.name) }}</span>
            <span v-if="fileSize">{{ fileSize }}</span>
            <span v-if="encoding">{{ encoding }}</span>
          </div>
        </div>
      </div>

      <!-- 查找替换栏 -->
      <div v-if="showFindBar" class="find-replace-bar">
        <div class="find-input-group">
          <Icon type="search" size="sm" class="input-icon" />
          <input v-model="findQuery" ref="findInput" type="text" placeholder="查找..." class="find-input" @keydown.enter="findNext" @keydown.shift.enter="findPrevious" />
          <div class="find-controls">
            <span v-if="findResults.total > 0" class="find-results"> {{ findResults.current }} / {{ findResults.total }} </span>
            <button class="find-btn" title="上一个" :disabled="findResults.total === 0" @click="findPrevious">
              <Icon type="chevron-up" size="xs" />
            </button>
            <button class="find-btn" title="下一个" :disabled="findResults.total === 0" @click="findNext">
              <Icon type="chevron-down" size="xs" />
            </button>
            <button class="find-btn" title="区分大小写" :class="{ active: findOptions.caseSensitive }" @click="findOptions.caseSensitive = !findOptions.caseSensitive">
              <Icon type="case-sensitive" size="xs" />
            </button>
            <button class="find-btn" title="正则表达式" :class="{ active: findOptions.useRegex }" @click="findOptions.useRegex = !findOptions.useRegex">
              <Icon type="regex" size="xs" />
            </button>
            <button class="find-btn" title="全词匹配" :class="{ active: findOptions.wholeWord }" @click="findOptions.wholeWord = !findOptions.wholeWord">
              <Icon type="whole-word" size="xs" />
            </button>
          </div>
        </div>
        <div v-if="showReplace" class="replace-input-group">
          <Icon type="replace" size="sm" class="input-icon" />
          <input v-model="replaceQuery" type="text" placeholder="替换为..." class="replace-input" @keydown.enter="replaceNext" />
          <div class="replace-controls">
            <button class="replace-btn" title="替换" :disabled="findResults.total === 0" @click="replaceNext">替换</button>
            <button class="replace-btn" title="全部替换" :disabled="findResults.total === 0" @click="replaceAll">全部替换</button>
          </div>
        </div>
      </div>

      <!-- 编辑器区域 -->
      <div class="editor-container">
        <!-- 侧边栏 -->
        <div v-if="showSidebar" class="sidebar">
          <div class="sidebar-tabs">
            <button :class="['sidebar-tab', { active: activeSidebarTab === 'outline' }]" @click="activeSidebarTab = 'outline'">
              <Icon type="list" size="xs" />
              大纲
            </button>
            <button :class="['sidebar-tab', { active: activeSidebarTab === 'symbols' }]" @click="activeSidebarTab = 'symbols'">
              <Icon type="hash" size="xs" />
              符号
            </button>
            <button :class="['sidebar-tab', { active: activeSidebarTab === 'problems' }]" @click="activeSidebarTab = 'problems'">
              <Icon type="alert-circle" size="xs" />
              问题
            </button>
          </div>

          <div class="sidebar-content">
            <!-- 大纲视图 -->
            <div v-if="activeSidebarTab === 'outline'" class="outline-view">
              <div v-for="(item, index) in outlineItems" :key="index" :class="['outline-item', `level-${item.level}`]" @click="goToLine(item.line)">
                <span class="outline-icon">
                  <Icon :type="getOutlineIcon(item.type) as any" size="xs" />
                </span>
                <span class="outline-text">{{ item.text }}</span>
                <span class="outline-line">{{ item.line }}</span>
              </div>
            </div>

            <!-- 符号列表 -->
            <div v-if="activeSidebarTab === 'symbols'" class="symbols-view">
              <div v-for="(symbol, index) in symbols" :key="index" :class="['symbol-item', symbol.type]" @click="goToLine(symbol.line)">
                <Icon :type="getSymbolIcon(symbol.type) as any" size="xs" />
                <span class="symbol-name">{{ symbol.name }}</span>
                <span class="symbol-line">{{ symbol.line }}</span>
              </div>
            </div>

            <!-- 问题列表 -->
            <div v-if="activeSidebarTab === 'problems'" class="problems-view">
              <div v-for="(problem, index) in problems" :key="index" :class="['problem-item', problem.severity]" @click="goToLine(problem.line)">
                <Icon :type="getProblemIcon(problem.severity) as any" size="xs" />
                <span class="problem-message">{{ problem.message }}</span>
                <span class="problem-line">{{ problem.line }}</span>
              </div>
            </div>
          </div>
        </div>

        <!-- 编辑器内容区 -->
        <div class="editor-content">
          <!-- 差异视图 -->
          <div v-if="showDiff" class="diff-view">
            <div class="diff-header">
              <h4>文件差异对比</h4>
              <button class="diff-close" @click="showDiff = false">
                <Icon type="close" size="xs" />
              </button>
            </div>
            <div class="diff-content">
              <div class="diff-panels">
                <div class="diff-panel original">
                  <div class="panel-header">原始版本</div>
                  <div class="panel-content">
                    <pre>{{ currentFile?.originalContent }}</pre>
                  </div>
                </div>
                <div class="diff-panel modified">
                  <div class="panel-header">修改版本</div>
                  <div class="panel-content">
                    <pre>{{ fileContent }}</pre>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- 主编辑器 -->
          <div v-else class="main-editor">
            <div v-if="editorSettings.lineNumbers" class="line-numbers">
              <div v-for="(line, index) in contentLines" :key="index" :class="['line-number', { 'current-line': index === cursorPosition.line - 1, 'has-breakpoint': hasBreakpoint(index + 1) }]" @click="toggleBreakpoint(index + 1)">
                {{ index + 1 }}
              </div>
            </div>

            <div class="code-editor">
              <textarea
                ref="editorTextarea"
                v-model="fileContent"
                :style="{
                  fontSize: editorSettings.fontSize + 'px',
                  fontFamily: editorSettings.fontFamily,
                  tabSize: editorSettings.tabSize,
                  lineHeight: 1.6,
                }"
                :class="['editor-textarea', { 'word-wrap': editorSettings.wordWrap }, `theme-${editorSettings.theme}`]"
                spellcheck="false"
                @input="handleInput"
                @keydown="handleKeydown"
                @keyup="handleKeyup"
                @click="updateCursorPosition"
                @scroll="syncScroll"
              ></textarea>

              <!-- 缩略图 -->
              <div v-if="editorSettings.minimap" class="minimap">
                <div class="minimap-content">{{ fileContent }}</div>
                <div class="minimap-viewport"></div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 状态栏 -->
    <div v-if="currentFile" class="status-bar">
      <div class="status-left">
        <span class="branch">main</span>
        <span class="sync-status"> synced</span>
        <span v-if="errors.length > 0" class="errors-count"> {{ errors.length }} 错误 </span>
        <span v-if="warnings.length > 0" class="warnings-count"> {{ warnings.length }} 警告 </span>
      </div>
      <div class="status-right">
        <span class="indent-size">Spaces: {{ editorSettings.tabSize }}</span>
        <span class="encoding">{{ encoding }}</span>
        <span class="eol">LF</span>
        <span v-if="hasUnsavedChanges" class="unsaved">未保存</span>
        <span v-else class="saved">已保存</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick, onMounted, onUnmounted } from "vue";
import Icon from "../ChatUI/Icon.vue";

interface EditFile {
  name: string;
  path: string;
  content: string;
  originalContent: string;
  lastEdit: number;
  size: number;
  encoding: string;
  language?: string;
}

interface OutlineItem {
  line: number;
  level: number;
  text: string;
  type: "class" | "function" | "variable" | "import" | "export" | "comment";
}

interface Symbol {
  name: string;
  type: "function" | "class" | "variable" | "constant";
  line: number;
  description?: string;
}

interface Problem {
  line: number;
  column: number;
  severity: "error" | "warning" | "info";
  message: string;
  source?: string;
}

interface CursorPosition {
  line: number;
  column: number;
}

interface FindResults {
  current: number;
  total: number;
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
  fileSaved: [file: EditFile];
  fileClosed: [file: EditFile];
}>();

// 响应式数据
const currentFile = ref<EditFile | null>(null);
const fileContent = ref("");
const recentFiles = ref<EditFile[]>([]);
const isSaving = ref(false);
const showSettings = ref(false);
const showDiff = ref(false);
const showSidebar = ref(false);
const showFindBar = ref(false);
const showReplace = ref(false);
const activeSidebarTab = ref<"outline" | "symbols" | "problems">("outline");
const isDragOver = ref(false);

// 编辑器设置
const editorSettings = ref({
  theme: "vs-code",
  fontSize: 14,
  fontFamily: 'Consolas, Monaco, "Courier New", monospace',
  tabSize: 2,
  wordWrap: false,
  lineNumbers: true,
  minimap: false,
  autoSave: false,
  autoSaveDelay: 1000,
});

// 查找替换
const findQuery = ref("");
const replaceQuery = ref("");
const findResults = ref<FindResults>({ current: 0, total: 0 });
const findOptions = ref({
  caseSensitive: false,
  useRegex: false,
  wholeWord: false,
});

// 编辑状态
const cursorPosition = ref<CursorPosition>({ line: 1, column: 1 });
const selectionInfo = ref("");
const canUndo = ref(false);
const canRedo = ref(false);

// 编辑历史
const editHistory = ref<string[]>([]);
const historyIndex = ref(-1);

// 语法分析结果
const outlineItems = ref<OutlineItem[]>([]);
const symbols = ref<Symbol[]>([]);
const problems = ref<Problem[]>([]);
const breakpoints = ref<number[]>([]);
const errors = computed(() => problems.value.filter((p) => p.severity === "error"));
const warnings = computed(() => problems.value.filter((p) => p.severity === "warning"));

// 自动保存
const autoSaveTimer = ref<NodeJS.Timeout | null>(null);

const fileInput = ref<HTMLInputElement>();
const findInput = ref<HTMLInputElement>();
const editorTextarea = ref<HTMLTextAreaElement>();
const websocket = ref<WebSocket | null>(null);

// 计算属性
const contentLines = computed(() => {
  return fileContent.value.split("\n");
});

const hasUnsavedChanges = computed(() => {
  return currentFile.value && currentFile.value.content !== currentFile.value.originalContent;
});

const fileSize = computed(() => {
  if (!currentFile.value?.size) return "";
  const bytes = new Blob([fileContent.value]).size;
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + " " + sizes[i];
});

const encoding = computed(() => {
  return currentFile.value?.encoding || "UTF-8";
});

// WebSocket 连接
const connectWebSocket = () => {
  try {
    websocket.value = new WebSocket(`${props.wsUrl}?session=${props.sessionId}`);

    websocket.value.onopen = () => {
      console.log("EditTool WebSocket connected");
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
      console.log("EditTool WebSocket disconnected");
      setTimeout(connectWebSocket, 5000);
    };

    websocket.value.onerror = (error) => {
      console.error("EditTool WebSocket error:", error);
    };
  } catch (error) {
    console.error("Failed to connect WebSocket:", error);
  }
};

const handleWebSocketMessage = (message: any) => {
  switch (message.type) {
    case "file_saved":
      if (message.file && currentFile.value) {
        currentFile.value.originalContent = currentFile.value.content;
        isSaving.value = false;
        emit("fileSaved", currentFile.value);
      }
      break;
    case "file_loaded":
      if (message.file) {
        loadFileContent(message.file);
      }
      break;
    case "syntax_analysis":
      if (message.analysis) {
        updateSyntaxAnalysis(message.analysis);
      }
      break;
  }
};

const sendWebSocketMessage = (message: any) => {
  if (websocket.value && websocket.value.readyState === WebSocket.OPEN) {
    websocket.value.send(JSON.stringify(message));
  }
};

// 文件操作方法
const openFile = (file: EditFile) => {
  currentFile.value = { ...file, originalContent: file.content };
  fileContent.value = file.content;
  addToRecentFiles(currentFile.value);
  resetFindReplace();
  analyzeSyntax();
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

const loadFileContent = (file: EditFile) => {
  fileContent.value = file.content;
  if (currentFile.value) {
    currentFile.value.content = file.content;
    currentFile.value.originalContent = file.content;
  }
  analyzeSyntax();
};

const createNewFile = () => {
  const fileName = prompt("输入新文件名:") || "untitled.txt";
  const newFile: EditFile = {
    name: fileName,
    path: `/${fileName}`,
    content: "",
    originalContent: "",
    lastEdit: Date.now(),
    size: 0,
    encoding: "UTF-8",
    language: getLanguage(fileName),
  };
  openFile(newFile);
};

const browseFiles = () => {
  fileInput.value?.click();
};

const openFromUrl = () => {
  const url = prompt("输入文件URL:");
  if (url) {
    sendWebSocketMessage({
      type: "load_file_from_url",
      url: url,
    });
  }
};

const reloadFromFile = (file: EditFile) => {
  if (confirm(`确定要重新加载文件 ${file.name} 吗？未保存的更改将丢失。`)) {
    sendWebSocketMessage({
      type: "reload_file",
      path: file.path,
    });
  }
};

const saveFile = async () => {
  if (!currentFile.value || isSaving.value) return;

  isSaving.value = true;
  currentFile.value.content = fileContent.value;
  currentFile.value.lastEdit = Date.now();

  sendWebSocketMessage({
    type: "save_file",
    file: {
      ...currentFile.value,
      content: fileContent.value,
    },
  });
};

const saveAndClose = async () => {
  if (currentFile.value) {
    await saveFile();
    closeFile();
  }
};

const closeFile = () => {
  if (currentFile.value && hasUnsavedChanges.value) {
    if (!confirm("文件有未保存的更改，确定要关闭吗？")) {
      return;
    }
  }

  if (currentFile.value) {
    emit("fileClosed", currentFile.value);
    currentFile.value = null;
    fileContent.value = "";
    resetFindReplace();
    clearAutoSave();
    outlineItems.value = [];
    symbols.value = [];
    problems.value = [];
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
  Array.from(files).forEach((file) => {
    if (file.type.startsWith("text/") || isTextFileName(file.name)) {
      const reader = new FileReader();
      reader.onload = (e) => {
        const result = e.target?.result as string;
        const newFile: EditFile = {
          name: file.name,
          path: `/${file.name}`,
          content: result,
          originalContent: result,
          lastEdit: Date.now(),
          size: file.size,
          encoding: "UTF-8",
          language: getLanguage(file.name),
        };
        openFile(newFile);
      };
      reader.readAsText(file);
    }
  });
};

const isTextFileName = (fileName: string): boolean => {
  const extension = fileName.split(".").pop()?.toLowerCase();
  const textExtensions = ["txt", "md", "json", "js", "ts", "jsx", "tsx", "html", "css", "scss", "py", "java", "cpp", "c", "h", "hpp", "go", "rs", "sql", "sh", "bash", "yml", "yaml", "xml", "csv", "log", "vue", "svelte", "php", "rb"];
  return textExtensions.includes(extension || "");
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
  const query = createSearchQuery();

  const start = textarea.selectionEnd;
  const text = textarea.value;

  let match;
  if (findOptions.value.useRegex) {
    const regex = new RegExp(findQuery.value, findOptions.value.caseSensitive ? "g" : "gi");
    regex.lastIndex = start;
    match = regex.exec(text);
  } else {
    const searchQuery = findOptions.value.caseSensitive ? query : query.toLowerCase();
    const searchText = findOptions.value.caseSensitive ? text : text.toLowerCase();
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
  if (findOptions.value.useRegex) {
    const regex = new RegExp(findQuery.value, findOptions.value.caseSensitive ? "g" : "gi");
    const matches = [...text.matchAll(regex)];
    const currentIndex = matches.findIndex((m) => m.index >= textarea.selectionStart);
    if (currentIndex > 0) {
      match = matches[currentIndex - 1];
    } else if (matches.length > 0) {
      match = matches[matches.length - 1];
    }
  } else {
    const searchQuery = findOptions.value.caseSensitive ? query : query.toLowerCase();
    const searchText = findOptions.value.caseSensitive ? text : text.toLowerCase();
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
  const flags = findOptions.value.caseSensitive ? "g" : "gi";
  let pattern = findQuery.value;
  if (findOptions.value.wholeWord) {
    pattern = `\\b${pattern}\\b`;
  }
  return new RegExp(pattern.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"), flags);
};

const isMatch = (text: string): boolean => {
  if (findOptions.value.caseSensitive) {
    return findOptions.value.wholeWord ? new RegExp(`\\b${findQuery.value}\\b`).test(text) : text.includes(findQuery.value);
  } else {
    return findOptions.value.wholeWord ? new RegExp(`\\b${findQuery.value}\\b`, "i").test(text) : text.toLowerCase().includes(findQuery.value.toLowerCase());
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
  findQuery.value = "";
  replaceQuery.value = "";
  findResults.value = { current: 0, total: 0 };
  showFindBar.value = false;
  showReplace.value = false;
};

// 导航方法
const goToLine = (lineNumber?: number) => {
  const line = lineNumber || parseInt(prompt("转到行号:") || "1");
  if (line && line > 0 && line <= contentLines.value.length) {
    const lines = contentLines.value;
    let offset = 0;
    for (let i = 0; i < line - 1; i++) {
      offset += (lines[i]?.length ?? 0) + 1; // +1 for newline
    }

    if (editorTextarea.value) {
      editorTextarea.value.focus();
      editorTextarea.value.setSelectionRange(offset, offset);
      updateCursorPosition();
    }
  }
};

// 编辑器操作
const undo = () => {
  if (historyIndex.value > 0) {
    historyIndex.value--;
    fileContent.value = editHistory.value[historyIndex.value] ?? "";
    canUndo.value = historyIndex.value > 0;
    canRedo.value = true;
  }
};

const redo = () => {
  if (historyIndex.value < editHistory.value.length - 1) {
    historyIndex.value++;
    fileContent.value = editHistory.value[historyIndex.value] ?? "";
    canUndo.value = true;
    canRedo.value = historyIndex.value < editHistory.value.length - 1;
  }
};

const formatCode = () => {
  if (!currentFile.value) return;

  const extension = currentFile.value.name.split(".").pop()?.toLowerCase();

  if (["js", "ts", "jsx", "tsx", "json"].includes(extension || "")) {
    try {
      if (extension === "json") {
        const parsed = JSON.parse(fileContent.value);
        fileContent.value = JSON.stringify(parsed, null, 2);
      }
      // 这里可以集成 Prettier 或其他格式化工具
    } catch (error) {
      console.warn("Code formatting failed:", error);
    }
  }
};

const toggleFold = () => {
  // 代码折叠功能
  console.log("Toggle fold functionality");
};

const toggleComment = () => {
  if (!editorTextarea.value) return;

  const textarea = editorTextarea.value;
  const start = textarea.selectionStart;
  const end = textarea.selectionEnd;
  const text = textarea.value;
  const lines = text.substring(start, end).split("\n");

  // 检查是否所有行都已注释
  const allCommented = lines.every((line) => line.trim().startsWith("//"));

  const newLines = lines.map((line) => {
    const trimmed = line.trim();
    if (allCommented) {
      // 取消注释
      return trimmed.startsWith("//") ? trimmed.substring(2).trim() : line;
    } else {
      // 添加注释
      return line.trim() === "" ? line : `// ${line}`;
    }
  });

  const newContent = newLines.join("\n");
  fileContent.value = text.substring(0, start) + newContent + text.substring(end);
};

const toggleDiff = () => {
  showDiff.value = !showDiff.value;
};

const toggleSidebar = () => {
  showSidebar.value = !showSidebar.value;
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

  // 触发自动保存
  scheduleAutoSave();

  // 更新语法分析
  scheduleSyntaxAnalysis();
};

const handleKeydown = (event: KeyboardEvent) => {
  const textarea = event.target as HTMLTextAreaElement;

  // 快捷键
  if (event.ctrlKey || event.metaKey) {
    switch (event.key) {
      case "s":
        event.preventDefault();
        saveFile();
        break;
      case "f":
        event.preventDefault();
        toggleFind();
        break;
      case "h":
        event.preventDefault();
        toggleReplace();
        break;
      case "g":
        event.preventDefault();
        goToLine();
        break;
      case "d":
        event.preventDefault();
        toggleDiff();
        break;
      case "b":
        event.preventDefault();
        toggleSidebar();
        break;
      case "z":
        if (event.shiftKey) {
          event.preventDefault();
          redo();
        } else {
          event.preventDefault();
          undo();
        }
        break;
      case "y":
        event.preventDefault();
        redo();
        break;
      case "/":
        event.preventDefault();
        toggleComment();
        break;
    }
  }

  // Tab支持
  if (event.key === "Tab") {
    event.preventDefault();
    const start = textarea.selectionStart;
    const end = textarea.selectionEnd;
    const value = textarea.value;

    if (event.shiftKey) {
      // Shift+Tab: 减少缩进
      handleDecreaseIndent(start, end, value);
    } else {
      // Tab: 增加缩进
      handleIncreaseIndent(start, end, value);
    }
  }

  // 更新选择信息
  updateSelectionInfo();
};

const handleKeyup = () => {
  updateSelectionInfo();
};

const handleIncreaseIndent = (start: number, end: number, value: string) => {
  const lines = value.substring(start, end).split("\n");
  const tabString = " ".repeat(editorSettings.value.tabSize);
  const indentedLines = lines.map((line) => tabString + line);
  const newContent = indentedLines.join("\n");

  fileContent.value = value.substring(0, start) + newContent + value.substring(end);

  nextTick(() => {
    if (editorTextarea.value) {
      editorTextarea.value.selectionStart = start;
      editorTextarea.value.selectionEnd = start + newContent.length;
    }
  });
};

const handleDecreaseIndent = (start: number, end: number, value: string) => {
  const lines = value.substring(start, end).split("\n");
  const tabSize = editorSettings.value.tabSize;
  const dedentedLines = lines.map((line) => {
    if (line.startsWith(" ".repeat(tabSize))) {
      return line.substring(tabSize);
    } else if (line.startsWith("\t")) {
      return line.substring(1);
    }
    return line;
  });
  const newContent = dedentedLines.join("\n");

  fileContent.value = value.substring(0, start) + newContent + value.substring(end);

  nextTick(() => {
    if (editorTextarea.value) {
      editorTextarea.value.selectionStart = start;
      editorTextarea.value.selectionEnd = start + newContent.length;
    }
  });
};

const updateCursorPosition = () => {
  if (!editorTextarea.value) return;

  const textarea = editorTextarea.value;
  const text = textarea.value;
  const position = textarea.selectionStart;

  let line = 1;
  let column = 1;

  for (let i = 0; i < position; i++) {
    if (text[i] === "\n") {
      line++;
      column = 1;
    } else {
      column++;
    }
  }

  cursorPosition.value = { line, column };
};

const updateSelectionInfo = () => {
  if (!editorTextarea.value) return;

  const textarea = editorTextarea.value;
  const start = textarea.selectionStart;
  const end = textarea.selectionEnd;

  if (start === end) {
    selectionInfo.value = "";
  } else {
    const selectedText = textarea.value.substring(start, end);
    const lines = selectedText.split("\n");
    const charCount = selectedText.length;

    if (lines.length === 1) {
      selectionInfo.value = `选择了 ${charCount} 个字符`;
    } else {
      selectionInfo.value = `选择了 ${lines.length} 行，${charCount} 个字符`;
    }
  }
};

const syncScroll = (event: Event) => {
  const textarea = event.target as HTMLTextAreaElement;
  const minimapViewport = document.querySelector(".minimap-viewport") as HTMLElement;

  if (minimapViewport) {
    const scrollPercentage = textarea.scrollTop / (textarea.scrollHeight - textarea.clientHeight);
    const minimapContent = document.querySelector(".minimap-content") as HTMLElement;
    if (minimapContent) {
      const maxScroll = minimapContent.scrollHeight - minimapViewport.clientHeight;
      minimapViewport.scrollTop = scrollPercentage * maxScroll;
    }
  }
};

// 断点管理
const toggleBreakpoint = (lineNumber: number) => {
  const index = breakpoints.value.indexOf(lineNumber);
  if (index === -1) {
    breakpoints.value.push(lineNumber);
  } else {
    breakpoints.value.splice(index, 1);
  }
};

const hasBreakpoint = (lineNumber: number): boolean => {
  return breakpoints.value.includes(lineNumber);
};

// 语法分析
const analyzeSyntax = () => {
  if (!currentFile.value) return;

  const extension = currentFile.value.name.split(".").pop()?.toLowerCase();
  const content = fileContent.value;

  // 简化的语法分析
  const lines = content.split("\n");
  outlineItems.value = [];
  symbols.value = [];
  problems.value = [];

  lines.forEach((line, index) => {
    const lineNumber = index + 1;
    const trimmed = line.trim();

    // 分析函数定义
    if (trimmed.match(/^(function|def|func)\s+\w+/)) {
      const match = trimmed.match(/^(function|def|func)\s+(\w+)/);
      if (match) {
        outlineItems.value.push({
          line: lineNumber,
          level: 1,
          text: match[2] ?? "",
          type: "function",
        });
        symbols.value.push({
          name: match[2] ?? "",
          type: "function",
          line: lineNumber,
          description: trimmed,
        });
      }
    }

    // 分析类定义
    if (trimmed.match(/^(class|interface)\s+\w+/)) {
      const match = trimmed.match(/^(class|interface)\s+(\w+)/);
      if (match) {
        outlineItems.value.push({
          line: lineNumber,
          level: 0,
          text: match[2] ?? "",
          type: "class",
        });
        symbols.value.push({
          name: match[2] ?? "",
          type: "class",
          line: lineNumber,
          description: trimmed,
        });
      }
    }

    // 分析import语句
    if (trimmed.match(/^(import|from)\s+/)) {
      outlineItems.value.push({
        line: lineNumber,
        level: 2,
        text: trimmed,
        type: "import",
      });
    }

    // 简单的语法检查
    if (trimmed.endsWith("{") && !trimmed.includes("}")) {
      // 可能缺少闭合括号
      const openBraces = (trimmed.match(/\{/g) || []).length;
      const closeBraces = (trimmed.match(/\}/g) || []).length;
      if (openBraces > closeBraces) {
        problems.value.push({
          line: lineNumber,
          column: trimmed.length + 1,
          severity: "warning",
          message: "可能缺少闭合括号",
          source: "syntax",
        });
      }
    }
  });

  outlineItems.value.sort((a, b) => a.line - b.line);
};

const scheduleSyntaxAnalysis = () => {
  // 防抖语法分析
  setTimeout(() => {
    analyzeSyntax();
  }, 500);
};

const updateSyntaxAnalysis = (analysis: any) => {
  if (analysis.outline) {
    outlineItems.value = analysis.outline;
  }
  if (analysis.symbols) {
    symbols.value = analysis.symbols;
  }
  if (analysis.problems) {
    problems.value = analysis.problems;
  }
};

// 自动保存
const scheduleAutoSave = () => {
  if (!editorSettings.value.autoSave) return;

  clearAutoSave();
  autoSaveTimer.value = setTimeout(() => {
    if (hasUnsavedChanges.value) {
      saveFile();
    }
  }, editorSettings.value.autoSaveDelay);
};

const clearAutoSave = () => {
  if (autoSaveTimer.value) {
    clearTimeout(autoSaveTimer.value);
    autoSaveTimer.value = null;
  }
};

const setupAutoSave = () => {
  clearAutoSave();
  if (editorSettings.value.autoSave) {
    // 自动保存设置会在下次更改时生效
  }
};

// 设置管理
const toggleSettings = () => {
  showSettings.value = !showSettings.value;
};

const saveSettings = () => {
  localStorage.setItem("editor-settings", JSON.stringify(editorSettings.value));
  showSettings.value = false;
  setupAutoSave();
};

const resetSettings = () => {
  editorSettings.value = {
    theme: "vs-code",
    fontSize: 14,
    fontFamily: 'Consolas, Monaco, "Courier New", monospace',
    tabSize: 2,
    wordWrap: false,
    lineNumbers: true,
    minimap: false,
    autoSave: false,
    autoSaveDelay: 1000,
  };
  setupAutoSave();
};

// 工具方法
const getIconForFile = (file: EditFile) => {
  const extension = file.name.split(".").pop()?.toLowerCase();
  const iconMap: { [key: string]: string } = {
    js: "file-code",
    ts: "file-code",
    jsx: "file-code",
    tsx: "file-code",
    vue: "file-code",
    html: "file-code",
    css: "file-code",
    scss: "file-code",
    json: "file-code",
    md: "file-text",
    py: "file-code",
    java: "file-code",
    cpp: "file-code",
    c: "file-code",
    go: "file-code",
    rs: "file-code",
    sql: "file-code",
    sh: "file-code",
    yml: "file-code",
    yaml: "file-code",
    xml: "file-code",
  };
  return iconMap[extension || ""] || "file";
};

const getLanguage = (fileName: string): string => {
  const extension = fileName.split(".").pop()?.toLowerCase();
  const languages: { [key: string]: string } = {
    js: "JavaScript",
    ts: "TypeScript",
    jsx: "React",
    tsx: "React TS",
    vue: "Vue",
    html: "HTML",
    css: "CSS",
    scss: "SCSS",
    py: "Python",
    java: "Java",
    cpp: "C++",
    c: "C",
    go: "Go",
    rs: "Rust",
    sql: "SQL",
    json: "JSON",
    md: "Markdown",
    yml: "YAML",
    yaml: "YAML",
    xml: "XML",
    sh: "Shell",
    php: "PHP",
    rb: "Ruby",
  };
  return languages[extension || ""] || "Plain Text";
};

const getOutlineIcon = (type: string): string => {
  const iconMap: { [key: string]: string } = {
    class: "box",
    function: "code",
    variable: "tag",
    import: "download",
    export: "upload",
    comment: "message-square",
  };
  return iconMap[type] || "file";
};

const getSymbolIcon = (type: string): string => {
  const iconMap: { [key: string]: string } = {
    function: "code",
    class: "box",
    variable: "tag",
    constant: "hash",
  };
  return iconMap[type] || "file";
};

const getProblemIcon = (severity: string): string => {
  const iconMap: { [key: string]: string } = {
    error: "x-circle",
    warning: "alert-triangle",
    info: "info",
  };
  return iconMap[severity] || "info";
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
    return date.toLocaleDateString("zh-CN");
  }
};

const addToRecentFiles = (file: EditFile) => {
  const existingIndex = recentFiles.value.findIndex((f) => f.path === file.path);
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
    localStorage.setItem("edit-recent-files", JSON.stringify(recentFiles.value));
  } catch (error) {
    console.warn("Failed to save recent files:", error);
  }
};

const removeFromRecent = (index: number) => {
  recentFiles.value.splice(index, 1);
  try {
    localStorage.setItem("edit-recent-files", JSON.stringify(recentFiles.value));
  } catch (error) {
    console.warn("Failed to save recent files:", error);
  }
};

// 生命周期
onMounted(() => {
  connectWebSocket();

  // 加载设置
  try {
    const saved = localStorage.getItem("editor-settings");
    if (saved) {
      editorSettings.value = { ...editorSettings.value, ...JSON.parse(saved) };
    }
  } catch (error) {
    console.warn("Failed to load editor settings:", error);
  }

  // 加载最近文件
  try {
    const saved = localStorage.getItem("edit-recent-files");
    if (saved) {
      recentFiles.value = JSON.parse(saved);
    }
  } catch (error) {
    console.warn("Failed to load recent files:", error);
  }

  // 初始化编辑历史
  editHistory.value = [""];
  historyIndex.value = 0;

  // 全局键盘事件监听
  document.addEventListener("keydown", handleGlobalKeydown);
});

onUnmounted(() => {
  document.removeEventListener("keydown", handleGlobalKeydown);
  clearAutoSave();
});

const handleGlobalKeydown = (event: KeyboardEvent) => {
  // 全局快捷键
  if (event.ctrlKey || event.metaKey) {
    switch (event.key) {
      case "o":
        if (!currentFile.value) {
          event.preventDefault();
          browseFiles();
        }
        break;
      case "n":
        if (!currentFile.value) {
          event.preventDefault();
          createNewFile();
        }
        break;
    }
  }
};

// 监听设置变化
watch(
  editorSettings,
  () => {
    setupAutoSave();
  },
  { deep: true },
);
</script>

<style scoped>
.edit-tool {
  @apply flex flex-col h-full bg-surface dark:bg-surface-dark border border-border dark:border-border-dark rounded-lg;
}

.edit-header {
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

.setting-select,
.setting-input {
  @apply ml-2;
}

.setting-select {
  @apply px-2 py-1 text-sm border border-border dark:border-border-dark rounded focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-800 dark:text-white;
}

.setting-input {
  @apply w-16 px-2 py-1 text-sm border border-border dark:border-border-dark rounded focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-800 dark:text-white;
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

.quick-access {
  @apply w-full max-w-2xl mb-8;
}

.quick-access h4 {
  @apply text-sm font-semibold text-text dark:text-text-dark mb-3;
}

.quick-actions {
  @apply flex gap-3 justify-center;
}

.quick-action-btn {
  @apply flex items-center gap-2 px-4 py-2 bg-blue-500 hover:bg-blue-600 text-white rounded-lg transition-colors;
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

.language {
  @apply px-2 py-0.5 bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400 rounded;
}

.file-actions {
  @apply flex gap-1;
}

.file-action-btn {
  @apply p-1 text-gray-400 dark:text-gray-500 hover:text-gray-600 dark:hover:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-600 rounded transition-colors;
}

.file-action-btn.remove-btn:hover {
  @apply text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20;
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

.find-input-group,
.replace-input-group {
  @apply flex items-center gap-2 px-4 py-2;
}

.find-input,
.replace-input {
  @apply flex-1 px-3 py-1.5 border border-border dark:border-border-dark rounded focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-700 dark:text-white;
}

.find-controls,
.replace-controls {
  @apply flex items-center gap-1;
}

.find-btn,
.replace-btn {
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

.sidebar {
  @apply w-64 bg-gray-50 dark:bg-gray-800 border-r border-gray-200 dark:border-gray-600 flex flex-col;
}

.sidebar-tabs {
  @apply flex border-b border-gray-200 dark:border-gray-600;
}

.sidebar-tab {
  @apply flex items-center gap-2 px-4 py-2 text-sm text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors;
}

.sidebar-tab.active {
  @apply text-blue-600 dark:text-blue-400 bg-white dark:bg-gray-700;
}

.sidebar-content {
  @apply flex-1 overflow-y-auto p-2;
}

.outline-view,
.symbols-view,
.problems-view {
  @apply space-y-1;
}

.outline-item,
.symbol-item,
.problem-item {
  @apply flex items-center gap-2 px-2 py-1 text-sm rounded hover:bg-gray-100 dark:hover:bg-gray-600 cursor-pointer transition-colors;
}

.outline-item.level-0 {
  @apply font-semibold;
}

.outline-item.level-1 {
  @apply pl-4;
}

.outline-item.level-2 {
  @apply pl-8;
}

.outline-line,
.symbol-line,
.problem-line {
  @apply ml-auto text-xs text-gray-500 dark:text-gray-400;
}

.problem-item.error {
  @apply text-red-600 dark:text-red-400;
}

.problem-item.warning {
  @apply text-yellow-600 dark:text-yellow-400;
}

.problem-item.info {
  @apply text-blue-600 dark:text-blue-400;
}

.editor-content {
  @apply flex-1 flex relative overflow-hidden;
}

.diff-view {
  @apply absolute inset-0 bg-white dark:bg-gray-900 z-10;
}

.diff-header {
  @apply flex items-center justify-between px-4 py-2 border-b border-gray-200 dark:border-gray-600 bg-gray-50 dark:bg-gray-800;
}

.diff-header h4 {
  @apply text-sm font-semibold text-text dark:text-text-dark;
}

.diff-close {
  @apply p-1 text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-600 rounded transition-colors;
}

.diff-content {
  @apply h-full overflow-hidden;
}

.diff-panels {
  @apply flex h-full;
}

.diff-panel {
  @apply flex-1 flex flex-col;
}

.panel-header {
  @apply px-4 py-2 bg-gray-100 dark:bg-gray-700 border-b border-gray-200 dark:border-gray-600 text-sm font-medium;
}

.panel-content {
  @apply flex-1 overflow-auto p-4;
}

.panel-content pre {
  @apply text-sm font-mono whitespace-pre-wrap;
}

.main-editor {
  @apply flex h-full;
}

.line-numbers {
  @apply w-16 bg-gray-100 dark:bg-gray-800 border-r border-gray-200 dark:border-gray-600 flex-shrink-0;
}

.line-number {
  @apply px-2 py-1 text-right text-xs text-gray-400 dark:text-gray-500 leading-6 cursor-pointer hover:bg-gray-200 dark:hover:bg-gray-700 transition-colors;
}

.line-number.current-line {
  @apply bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400;
}

.line-number.has-breakpoint {
  @apply relative;
}

.line-number.has-breakpoint::before {
  @apply absolute left-0 top-1/2 transform -translate-y-1/2 w-2 h-2 bg-red-500 rounded-full;
  content: "";
}

.code-editor {
  @apply flex-1 relative overflow-hidden;
}

.editor-textarea {
  @apply w-full h-full px-3 py-2 bg-white dark:bg-gray-900 text-gray-800 dark:text-gray-200 font-mono border-0 resize-none focus:outline-none leading-6;
}

.editor-textarea.word-wrap {
  @apply whitespace-pre-wrap;
}

.editor-textarea:not(.word-wrap) {
  @apply whitespace-pre overflow-x-auto;
}

/* 编辑器主题 */
.theme-vs-code {
  @apply bg-white text-gray-800;
}

.theme-github {
  @apply bg-white text-gray-800;
}

.theme-monokai {
  @apply bg-[#272822] text-[#f8f8f2];
}

.theme-solarized {
  @apply bg-[#002b36] text-[#839496];
}

.theme-dracula {
  @apply bg-[#282a36] text-[#f8f8f2];
}

.minimap {
  @apply absolute top-0 right-0 w-32 h-full bg-gray-50 dark:bg-gray-800 border-l border-gray-200 dark:border-gray-600 overflow-hidden;
}

.minimap-content {
  @apply p-2 text-xs font-mono leading-tight whitespace-pre-wrap break-all opacity-50;
}

.minimap-viewport {
  @apply absolute top-0 left-0 right-0 bg-blue-200 dark:bg-blue-800 opacity-30;
}

.status-bar {
  @apply flex items-center justify-between px-4 py-1 border-t border-border dark:border-border-dark bg-gray-100 dark:bg-gray-800 text-xs text-gray-600 dark:text-gray-400;
}

.status-left,
.status-right {
  @apply flex items-center gap-3;
}

.branch {
  @apply font-medium text-gray-700 dark:text-gray-300;
}

.sync-status {
  @apply text-green-500;
}

.errors-count {
  @apply text-red-500;
}

.warnings-count {
  @apply text-yellow-500;
}

.unsaved {
  @apply text-orange-500;
}

.saved {
  @apply text-green-500;
}

.animate-spin {
  @apply animate-spin;
}
</style>
