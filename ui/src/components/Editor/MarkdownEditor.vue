<template>
  <div class="markdown-editor">
    <textarea
      ref="textareaRef"
      v-model="localValue"
      @input="handleInput"
      @keydown="handleKeydown"
      :placeholder="placeholder"
      :disabled="disabled"
      :class="['editor-textarea', { 'editor-disabled': disabled }]"
      :style="{ minHeight: `${minHeight}px` }"
    ></textarea>

    <!-- 工具栏 (可选) -->
    <div v-if="showToolbar" class="editor-toolbar">
      <button v-for="tool in tools" :key="tool.name" @click="applyTool(tool)" :title="tool.tooltip" class="toolbar-button" type="button">
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" :d="tool.icon" />
        </svg>
      </button>

      <!-- 字数统计 -->
      <div class="toolbar-stats">
        <span class="text-xs text-slate-500"> {{ wordCount }} 字 | {{ charCount }} 字符 </span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from "vue";

interface ToolDefinition {
  name: string;
  tooltip: string;
  icon: string;
  action: (editor: HTMLTextAreaElement) => void;
}

interface Props {
  modelValue: string;
  placeholder?: string;
  disabled?: boolean;
  minHeight?: number;
  showToolbar?: boolean;
  autoFocus?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  placeholder: "输入 Markdown 文本...",
  disabled: false,
  minHeight: 200,
  showToolbar: true,
  autoFocus: false,
});

const emit = defineEmits<{
  "update:modelValue": [value: string];
  change: [value: string];
}>();

const textareaRef = ref<HTMLTextAreaElement>();
const localValue = ref(props.modelValue);

// 同步外部更新
watch(
  () => props.modelValue,
  (newValue) => {
    if (newValue !== localValue.value) {
      localValue.value = newValue;
    }
  },
);

// 字数统计
const wordCount = computed(() => {
  return localValue.value.replace(/\s+/g, "").length;
});

const charCount = computed(() => {
  return localValue.value.length;
});

// 工具栏工具定义
const tools = computed<ToolDefinition[]>(() => [
  {
    name: "bold",
    tooltip: "粗体 (Ctrl+B)",
    icon: "M6 4h8a4 4 0 014 4 4 4 0 01-4 4H6z M6 12h9a4 4 0 014 4 4 4 0 01-4 4H6z",
    action: (editor) => wrapSelection(editor, "**", "**"),
  },
  {
    name: "italic",
    tooltip: "斜体 (Ctrl+I)",
    icon: "M19 4h-9M14 20H5M15 4L9 20",
    action: (editor) => wrapSelection(editor, "*", "*"),
  },
  {
    name: "code",
    tooltip: "代码 (Ctrl+`)",
    icon: "M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4",
    action: (editor) => wrapSelection(editor, "`", "`"),
  },
  {
    name: "link",
    tooltip: "链接 (Ctrl+K)",
    icon: "M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1",
    action: (editor) => insertLink(editor),
  },
  {
    name: "list",
    tooltip: "列表",
    icon: "M4 6h16M4 12h16M4 18h16",
    action: (editor) => insertList(editor),
  },
  {
    name: "quote",
    tooltip: "引用",
    icon: "M7 8h10M7 12h4m1 8l-4-4H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-3l-4 4z",
    action: (editor) => insertQuote(editor),
  },
]);

// 处理输入
const handleInput = () => {
  emit("update:modelValue", localValue.value);
  emit("change", localValue.value);
};

// 处理快捷键
const handleKeydown = (e: KeyboardEvent) => {
  if (e.ctrlKey || e.metaKey) {
    const boldTool = tools.value[0];
    const italicTool = tools.value[1];
    const codeTool = tools.value[2];
    const linkTool = tools.value[3];
    switch (e.key) {
      case "b":
        e.preventDefault();
        if (boldTool) applyTool(boldTool); // Bold
        break;
      case "i":
        e.preventDefault();
        if (italicTool) applyTool(italicTool); // Italic
        break;
      case "`":
        e.preventDefault();
        if (codeTool) applyTool(codeTool); // Code
        break;
      case "k":
        e.preventDefault();
        if (linkTool) applyTool(linkTool); // Link
        break;
    }
  }

  // Tab 键插入空格
  if (e.key === "Tab") {
    e.preventDefault();
    insertText(textareaRef.value!, "  ");
  }
};

// 应用工具
const applyTool = (tool: ToolDefinition) => {
  if (!textareaRef.value || props.disabled) return;
  tool.action(textareaRef.value);
};

// 包装选中文本
const wrapSelection = (editor: HTMLTextAreaElement, prefix: string, suffix: string) => {
  const start = editor.selectionStart;
  const end = editor.selectionEnd;
  const selectedText = localValue.value.substring(start, end) || "文本";
  const replacement = `${prefix}${selectedText}${suffix}`;

  localValue.value = localValue.value.substring(0, start) + replacement + localValue.value.substring(end);

  emit("update:modelValue", localValue.value);

  // 恢复光标位置
  editor.focus();
  const newCursorPos = start + prefix.length;
  editor.setSelectionRange(newCursorPos, newCursorPos + selectedText.length);
};

// 插入文本
const insertText = (editor: HTMLTextAreaElement, text: string) => {
  const start = editor.selectionStart;
  localValue.value = localValue.value.substring(0, start) + text + localValue.value.substring(start);

  emit("update:modelValue", localValue.value);

  // 恢复光标位置
  editor.focus();
  editor.setSelectionRange(start + text.length, start + text.length);
};

// 插入链接
const insertLink = (editor: HTMLTextAreaElement) => {
  const start = editor.selectionStart;
  const end = editor.selectionEnd;
  const selectedText = localValue.value.substring(start, end) || "链接文字";
  const replacement = `[${selectedText}](url)`;

  localValue.value = localValue.value.substring(0, start) + replacement + localValue.value.substring(end);

  emit("update:modelValue", localValue.value);

  // 选中 URL 部分
  editor.focus();
  const urlStart = start + selectedText.length + 3;
  editor.setSelectionRange(urlStart, urlStart + 3);
};

// 插入列表
const insertList = (editor: HTMLTextAreaElement) => {
  insertText(editor, "- ");
};

// 插入引用
const insertQuote = (editor: HTMLTextAreaElement) => {
  insertText(editor, "> ");
};

// 自动对焦
onMounted(() => {
  if (props.autoFocus && textareaRef.value) {
    textareaRef.value.focus();
  }
});

// 暴露方法给父组件
defineExpose({
  focus: () => textareaRef.value?.focus(),
  blur: () => textareaRef.value?.blur(),
  insertText: (text: string) => insertText(textareaRef.value!, text),
});
</script>

<style scoped>
.markdown-editor {
  @apply relative flex flex-col;
}

.editor-textarea {
  @apply w-full px-4 py-3 border border-slate-200 dark:border-slate-700 rounded-lg bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 font-mono text-sm leading-relaxed resize-y transition-colors;
  @apply focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent;
}

.editor-disabled {
  @apply opacity-50 cursor-not-allowed bg-slate-50 dark:bg-slate-900;
}

.editor-toolbar {
  @apply flex items-center gap-1 mt-2 p-2 bg-slate-50 dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg;
}

.toolbar-button {
  @apply p-2 rounded hover:bg-slate-200 dark:hover:bg-slate-700 text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-slate-100 transition-colors;
}

.toolbar-stats {
  @apply ml-auto text-xs text-slate-500 dark:text-slate-400;
}
</style>
