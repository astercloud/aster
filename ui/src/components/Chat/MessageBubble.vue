<template>
  <div
    :class="[
      'message-bubble',
      `message-${message.role}`,
      message.status === 'error' && 'message-error'
    ]"
  >
    <!-- Avatar -->
    <div v-if="showAvatar && message.role === 'assistant'" class="message-avatar">
      <div class="w-8 h-8 rounded-full bg-primary/10 flex items-center justify-center flex-shrink-0">
        <svg class="w-4 h-4 text-primary" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z"></path>
        </svg>
      </div>
    </div>

    <!-- Content -->
    <div class="message-content-wrapper">
      <!-- Bubble -->
      <div
        :class="[
          'message-bubble-content',
          message.role === 'user' ? 'bg-primary text-white' : 'bg-surface dark:bg-surface-dark text-text dark:text-text-dark border border-border dark:border-border-dark'
        ]"
      >
        <!-- Text Content -->
        <div
          v-if="message.type === 'text'"
          class="message-text"
          v-html="renderedContent"
        ></div>

        <!-- Image Content -->
        <div v-else-if="message.type === 'image'" class="message-image">
          <img
            :src="message.content.url"
            :alt="message.content.alt"
            class="max-w-full rounded"
            loading="lazy"
          />
        </div>

        <!-- System Message -->
        <div v-else-if="message.type === 'system'" class="message-system">
          {{ message.content.text }}
        </div>

        <!-- Status Indicator -->
        <div v-if="message.status" class="message-status">
          <svg v-if="message.status === 'pending'" class="w-3 h-3 animate-spin" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"></path>
          </svg>
          <svg v-else-if="message.status === 'sent'" class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"></path>
          </svg>
          <svg v-else-if="message.status === 'error'" class="w-3 h-3 text-red-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
          </svg>
        </div>
      </div>

      <!-- ThinkingBlock (仅显示助手消息) -->
      <ThinkingBlock
        v-if="message.role === 'assistant' && thinkingSteps.length > 0"
        :message-id="message.id"
        :steps="thinkingSteps"
        :is-active="isThinking"
        :summary="thinkingSummary"
      />

      <!-- WorkflowProgressView (仅当有激活工作流时显示) -->
      <WorkflowProgressView
        v-if="showWorkflow"
        :steps="workflowSteps"
        title="工作流进度"
        :show-progress="true"
        :show-steps="true"
        :max-visible-steps="3"
      />

      <!-- Timestamp -->
      <div v-if="showTimestamp" class="message-timestamp">
        {{ formatTime(message.createdAt) }}
      </div>

      <!-- Actions (on hover) -->
      <div v-if="showActions" class="message-actions">
        <button
          @click="$emit('copy', message)"
          class="action-button"
          title="复制"
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"></path>
          </svg>
        </button>
        <button
          v-if="message.status === 'error'"
          @click="$emit('retry', message)"
          class="action-button"
          title="重试"
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"></path>
          </svg>
        </button>
        <button
          @click="$emit('delete', message)"
          class="action-button"
          title="删除"
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"></path>
          </svg>
        </button>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent, computed } from 'vue';
import { renderMarkdown } from '@/utils/markdown';
import { formatTime } from '@/utils/format';
import { useThinkingStore } from '@/stores/thinking';
import { useWorkflowStore } from '@/stores/workflow';
import { ThinkingBlock } from '@/components/Thinking';
import { WorkflowProgressView } from '@/components/Workflow';
import type { Message, TextMessage } from '@/types';

export default defineComponent({
  name: 'MessageBubble',

  components: {
    ThinkingBlock,
    WorkflowProgressView,
  },

  props: {
    message: {
      type: Object as () => Message,
      required: true,
    },
    showAvatar: {
      type: Boolean,
      default: true,
    },
    showTimestamp: {
      type: Boolean,
      default: true,
    },
    showActions: {
      type: Boolean,
      default: true,
    },
  },

  emits: {
    copy: (message: Message) => true,
    retry: (message: Message) => true,
    delete: (message: Message) => true,
  },

  setup(props) {
    const thinkingStore = useThinkingStore();
    const workflowStore = useWorkflowStore();

    const renderedContent = computed(() => {
      if (props.message.type === 'text') {
        const textMessage = props.message as TextMessage;
        return renderMarkdown(textMessage.content.text);
      }
      return '';
    });

    // 获取该消息的思维步骤
    const thinkingSteps = computed(() => {
      return thinkingStore.getSteps(props.message.id);
    });

    // 判断是否正在思考（当前消息是否是正在思考的消息）
    const isThinking = computed(() => {
      return thinkingStore.isThinking && thinkingStore.currentMessageId === props.message.id;
    });

    // 思维摘要（可选）
    const thinkingSummary = computed(() => {
      // 可以根据步骤生成摘要
      if (thinkingSteps.value.length > 0 && !isThinking.value) {
        const toolCalls = thinkingSteps.value.filter(s => s.type === 'tool_call');
        if (toolCalls.length > 0) {
          return `执行了 ${toolCalls.length} 个工具调用`;
        }
      }
      return '';
    });

    // 是否显示工作流进度（仅当有激活的工作流时）
    const showWorkflow = computed(() => {
      return props.message.role === 'assistant' && workflowStore.hasActiveWorkflow;
    });

    // 工作流步骤
    const workflowSteps = computed(() => {
      return workflowStore.steps;
    });

    return {
      renderedContent,
      thinkingSteps,
      isThinking,
      thinkingSummary,
      showWorkflow,
      workflowSteps,
      formatTime,
    };
  },
});
</script>

<style scoped>
.message-bubble {
  @apply flex gap-2 mb-4 animate-slide-in;
}

.message-user {
  @apply flex-row-reverse;
}

.message-assistant {
  @apply flex-row;
}

.message-avatar {
  @apply flex-shrink-0;
}

.message-content-wrapper {
  @apply flex flex-col gap-1 max-w-[80%];
}

.message-user .message-content-wrapper {
  @apply items-end;
}

.message-assistant .message-content-wrapper {
  @apply items-start;
}

.message-bubble-content {
  @apply rounded-lg px-4 py-3 shadow-sm relative;
}

.message-text {
  @apply text-sm leading-relaxed;
}

.message-text :deep(p) {
  @apply mb-2 last:mb-0;
}

.message-text :deep(code) {
  @apply bg-black/10 px-1.5 py-0.5 rounded text-xs font-mono;
}

.message-text :deep(pre) {
  @apply bg-stone-900 text-stone-100 p-3 rounded my-2 overflow-x-auto;
}

.message-text :deep(pre code) {
  @apply bg-transparent p-0;
}

.message-text :deep(a) {
  @apply underline hover:no-underline;
}

.message-image img {
  @apply max-h-64 object-contain;
}

.message-system {
  @apply text-xs text-center text-secondary italic;
}

.message-status {
  @apply absolute bottom-1 right-1 opacity-70;
}

.message-timestamp {
  @apply text-xs text-secondary px-2;
}

.message-actions {
  @apply hidden gap-1 absolute -top-8 right-0 bg-surface border border-border rounded-lg shadow-lg p-1;
}

.message-bubble:hover .message-actions {
  @apply flex;
}

.action-button {
  @apply p-1.5 hover:bg-background rounded transition-colors text-secondary hover:text-primary;
}

.message-error .message-bubble-content {
  @apply border-red-300 bg-red-50;
}
</style>
