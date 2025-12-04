<template>
  <div class="doc-viewer">
    <!-- 左侧文档内容 -->
    <div class="doc-content">
      <div class="markdown-body" v-html="renderedContent"></div>
    </div>

    <!-- 右侧 Demo 展示 -->
    <div class="doc-demo">
      <div class="demo-header">
        <h3 class="demo-title">实时演示</h3>
        <div class="demo-actions">
          <button v-for="tab in demoTabs" :key="tab.key" :class="['demo-tab', { active: activeTab === tab.key }]" @click="activeTab = tab.key as 'code' | 'preview'">
            {{ tab.label }}
          </button>
        </div>
      </div>

      <div class="demo-body">
        <!-- 预览区域 -->
        <div v-show="activeTab === 'preview'" class="demo-preview">
          <slot name="demo"></slot>
        </div>

        <!-- 代码区域 -->
        <div v-show="activeTab === 'code'" class="demo-code">
          <pre><code v-html="highlightedCode"></code></pre>
          <button class="copy-btn" @click="copyCode">
            <Icon type="copy" />
            {{ copied ? "已复制" : "复制代码" }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from "vue";
import { marked } from "marked";
import Icon from "./ChatUI/Icon.vue";

interface Props {
  content: string;
  code?: string;
}

const props = defineProps<Props>();

const activeTab = ref<"preview" | "code">("preview");
const copied = ref(false);

const demoTabs = [
  { key: "preview", label: "预览" },
  { key: "code", label: "代码" },
];

const renderedContent = computed(() => {
  try {
    return marked(props.content);
  } catch {
    return props.content;
  }
});

const highlightedCode = computed(() => {
  if (!props.code) return "";
  // 简单的代码高亮（实际项目中可以使用 highlight.js）
  return props.code
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/(".*?")/g, '<span class="string">$1</span>')
    .replace(/\b(const|let|var|function|return|import|export|from)\b/g, '<span class="keyword">$1</span>');
});

const copyCode = async () => {
  if (!props.code) return;

  try {
    await navigator.clipboard.writeText(props.code);
    copied.value = true;
    setTimeout(() => {
      copied.value = false;
    }, 2000);
  } catch (error) {
    console.error("Failed to copy:", error);
  }
};
</script>

<style scoped>
.doc-viewer {
  @apply flex gap-6 h-full;
}

.doc-content {
  @apply flex-1 overflow-y-auto pr-6;
}

.markdown-body {
  @apply text-gray-900 dark:text-gray-100;
}

.markdown-body :deep(h1) {
  @apply text-3xl font-bold mt-8 mb-4 pb-2 border-b border-gray-200 dark:border-gray-700;
}

.markdown-body :deep(h2) {
  @apply text-2xl font-bold mt-6 mb-3;
}

.markdown-body :deep(h3) {
  @apply text-xl font-bold mt-4 mb-2;
}

.markdown-body :deep(p) {
  @apply my-3 leading-relaxed;
}

.markdown-body :deep(code) {
  @apply px-1.5 py-0.5 bg-gray-100 dark:bg-gray-800 rounded text-sm font-mono text-pink-600 dark:text-pink-400;
}

.markdown-body :deep(pre) {
  @apply my-4 p-4 bg-gray-900 dark:bg-gray-950 rounded-lg overflow-x-auto;
}

.markdown-body :deep(pre code) {
  @apply p-0 bg-transparent text-gray-100;
}

.markdown-body :deep(table) {
  @apply my-4 w-full border-collapse;
}

.markdown-body :deep(th) {
  @apply px-4 py-2 bg-gray-100 dark:bg-gray-800 border border-gray-300 dark:border-gray-600 font-semibold text-left;
}

.markdown-body :deep(td) {
  @apply px-4 py-2 border border-gray-300 dark:border-gray-600;
}

.doc-demo {
  @apply w-[500px] flex-shrink-0 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-xl overflow-hidden sticky top-6 h-fit max-h-[calc(100vh-100px)];
}

.demo-header {
  @apply px-4 py-3 border-b border-gray-200 dark:border-gray-700;
}

.demo-title {
  @apply text-sm font-semibold text-gray-900 dark:text-white mb-2;
}

.demo-actions {
  @apply flex gap-2;
}

.demo-tab {
  @apply px-3 py-1 text-sm font-medium text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white rounded transition-colors;
}

.demo-tab.active {
  @apply bg-blue-50 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400;
}

.demo-body {
  @apply p-4 overflow-y-auto max-h-[calc(100vh-200px)];
}

.demo-preview {
  @apply space-y-4;
}

.demo-code {
  @apply relative;
}

.demo-code pre {
  @apply bg-gray-900 dark:bg-gray-950 rounded-lg p-4 overflow-x-auto text-sm;
}

.demo-code code {
  @apply text-gray-100 font-mono;
}

.demo-code :deep(.keyword) {
  @apply text-purple-400;
}

.demo-code :deep(.string) {
  @apply text-green-400;
}

.copy-btn {
  @apply absolute top-2 right-2 flex items-center gap-1 px-3 py-1 bg-gray-800 hover:bg-gray-700 text-white text-xs rounded transition-colors;
}
</style>
